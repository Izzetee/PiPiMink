package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"PiPiMink/internal/benchmark"
	"PiPiMink/internal/config"
	"PiPiMink/internal/models"

	_ "github.com/lib/pq"
)

// DB represents a database connection
type DB struct {
	*sql.DB
}

// New creates a new database connection
func New(cfg *config.Config) (*DB, error) {
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{db}, nil
}

// InitSchema initializes the database schema
func (db *DB) InitSchema() error {
	// First create tables if they don't exist
	query := `
	CREATE TABLE IF NOT EXISTS models (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		source TEXT NOT NULL,
		tags JSONB DEFAULT '{}'::jsonb,
		UNIQUE(name, source)
	);
	`
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// Check if the enabled column exists, and if not, add it
	var enabledColumnExists bool
	err := db.QueryRow(`SELECT EXISTS (
		SELECT 1 
		FROM information_schema.columns 
		WHERE table_name = 'models' 
		AND column_name = 'enabled'
	)`).Scan(&enabledColumnExists)

	if err != nil {
		log.Printf("Error checking if enabled column exists: %v", err)
	} else if !enabledColumnExists {
		// Column doesn't exist, add it
		_, err = db.Exec(`ALTER TABLE models ADD COLUMN enabled BOOLEAN DEFAULT TRUE`)
		if err != nil {
			log.Printf("Error adding 'enabled' column: %v", err)
			return fmt.Errorf("failed to add enabled column: %w", err)
		}
		log.Println("Added 'enabled' column to models table")
	}

	// Check if updated_at column exists, and if not, add it
	var updatedAtColumnExists bool
	err = db.QueryRow(`SELECT EXISTS (
		SELECT 1 
		FROM information_schema.columns 
		WHERE table_name = 'models' 
		AND column_name = 'updated_at'
	)`).Scan(&updatedAtColumnExists)

	if err != nil {
		log.Printf("Error checking if updated_at column exists: %v", err)
	} else if !updatedAtColumnExists {
		// Column doesn't exist, add it
		_, err = db.Exec(`ALTER TABLE models ADD COLUMN updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()`)
		if err != nil {
			log.Printf("Error adding 'updated_at' column: %v", err)
			return fmt.Errorf("failed to add updated_at column: %w", err)
		}
		log.Println("Added 'updated_at' column to models table")
	}

	// Check if the has_reasoning column exists, and if not, add it
	var hasReasoningColumnExists bool
	err = db.QueryRow(`SELECT EXISTS (
		SELECT 1 
		FROM information_schema.columns 
		WHERE table_name = 'models' 
		AND column_name = 'has_reasoning'
	)`).Scan(&hasReasoningColumnExists)

	if err != nil {
		log.Printf("Error checking if has_reasoning column exists: %v", err)
	} else if !hasReasoningColumnExists {
		// Column doesn't exist, add it
		_, err = db.Exec(`ALTER TABLE models ADD COLUMN has_reasoning BOOLEAN DEFAULT FALSE`)
		if err != nil {
			log.Printf("Error adding 'has_reasoning' column: %v", err)
			return fmt.Errorf("failed to add has_reasoning column: %w", err)
		}
		log.Println("Added 'has_reasoning' column to models table")

		// Run migration to set reasoning capabilities for existing models
		if migrateErr := db.MigrateExistingModelsReasoning(); migrateErr != nil {
			log.Printf("Warning: Error during reasoning migration: %v", migrateErr)
		}
	}

	// Benchmark results table — additive migration, safe to run on existing databases.
	if err := db.initBenchmarkSchema(); err != nil {
		return fmt.Errorf("failed to init benchmark schema: %w", err)
	}

	// Config tables for editable benchmark tasks and system prompts.
	if err := db.initConfigSchema(); err != nil {
		return fmt.Errorf("failed to init config schema: %w", err)
	}

	return nil
}

// initBenchmarkSchema creates the benchmark_results table and indexes if they do not exist.
func (db *DB) initBenchmarkSchema() error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS benchmark_results (
		id         SERIAL PRIMARY KEY,
		model_name TEXT NOT NULL,
		source     TEXT NOT NULL,
		category   TEXT NOT NULL,
		task_id    TEXT NOT NULL,
		score      FLOAT NOT NULL,
		latency_ms BIGINT NOT NULL DEFAULT 0,
		run_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`)
	if err != nil {
		return fmt.Errorf("failed to create benchmark_results table: %w", err)
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_benchmark_results_model    ON benchmark_results(model_name, source);`)
	if err != nil {
		return fmt.Errorf("failed to create benchmark model index: %w", err)
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_benchmark_results_category ON benchmark_results(model_name, source, category);`)
	if err != nil {
		return fmt.Errorf("failed to create benchmark category index: %w", err)
	}

	return nil
}

// SaveBenchmarkResult persists a single task result for a model.
func (db *DB) SaveBenchmarkResult(modelName, source, category, taskID string, score float64, latencyMs int64) error {
	_, err := db.Exec(`
		INSERT INTO benchmark_results (model_name, source, category, task_id, score, latency_ms, run_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`, modelName, source, category, taskID, score, latencyMs)
	if err != nil {
		return fmt.Errorf("error saving benchmark result: %w", err)
	}
	return nil
}

// GetBenchmarkScores returns the average score per category for a single model.
// The returned map is keyed by category name (e.g. "coding", "reasoning").
func (db *DB) GetBenchmarkScores(modelName, source string) (map[string]float64, error) {
	rows, err := db.Query(`
		SELECT category, AVG(score)
		FROM benchmark_results
		WHERE model_name = $1 AND source = $2
		GROUP BY category
	`, modelName, source)
	if err != nil {
		return nil, fmt.Errorf("error querying benchmark scores: %w", err)
	}
	defer func() { _ = rows.Close() }()

	scores := make(map[string]float64)
	for rows.Next() {
		var category string
		var avg float64
		if err := rows.Scan(&category, &avg); err != nil {
			log.Printf("Error scanning benchmark score row: %v", err)
			continue
		}
		scores[category] = avg
	}
	return scores, nil
}

// GetAllBenchmarkScores returns average scores for every model in the database.
// The outer map is keyed by "modelname|source"; the inner map by category.
func (db *DB) GetAllBenchmarkScores() (map[string]map[string]float64, error) {
	rows, err := db.Query(`
		SELECT model_name, source, category, AVG(score)
		FROM benchmark_results
		GROUP BY model_name, source, category
	`)
	if err != nil {
		return nil, fmt.Errorf("error querying all benchmark scores: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]map[string]float64)
	for rows.Next() {
		var modelName, source, category string
		var avg float64
		if err := rows.Scan(&modelName, &source, &category, &avg); err != nil {
			log.Printf("Error scanning benchmark score row: %v", err)
			continue
		}
		key := modelName + "|" + source
		if result[key] == nil {
			result[key] = make(map[string]float64)
		}
		result[key][category] = avg
	}
	return result, nil
}

// RegisterDiscoveredModel inserts a model into the registry with empty tags if it is not
// already present. Existing rows (already tagged or manually configured) are left untouched.
func (db *DB) RegisterDiscoveredModel(name, source string) error {
	_, err := db.Exec(`
	INSERT INTO models (name, source, tags, enabled, has_reasoning, updated_at)
	VALUES ($1, $2, '{}', false, false, NOW())
	ON CONFLICT (name, source) DO NOTHING;
	`, name, source)
	if err != nil {
		return fmt.Errorf("error registering discovered model: %w", err)
	}
	return nil
}

// GetAllModelLatencies returns the average benchmark latency in milliseconds for every model
// that has benchmark results. The outer map key is "modelname|source".
func (db *DB) GetAllModelLatencies() (map[string]int64, error) {
	rows, err := db.Query(`
		SELECT model_name, source, AVG(latency_ms)
		FROM benchmark_results
		GROUP BY model_name, source
	`)
	if err != nil {
		return nil, fmt.Errorf("error querying model latencies: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]int64)
	for rows.Next() {
		var modelName, source string
		var avg float64
		if err := rows.Scan(&modelName, &source, &avg); err != nil {
			log.Printf("Error scanning latency row: %v", err)
			continue
		}
		result[modelName+"|"+source] = int64(avg)
	}
	return result, nil
}

// SaveModel saves a model to the database
func (db *DB) SaveModel(name, source, tags string, enabled bool, hasReasoning bool) error {
	// Validate JSON tags
	if tags == "" {
		tags = "{}"
	}

	query := `
	INSERT INTO models (name, source, tags, enabled, has_reasoning, updated_at)
	VALUES ($1, $2, $3::jsonb, $4, $5, NOW())
	ON CONFLICT (name, source) DO UPDATE
	SET tags = $3::jsonb, enabled = $4, has_reasoning = $5, updated_at = NOW();
	`
	_, err := db.Exec(query, name, source, tags, enabled, hasReasoning)
	if err != nil {
		return fmt.Errorf("error saving model: %w", err)
	}

	return nil
}

// GetAllModels retrieves all models from the database
func (db *DB) GetAllModels() (map[string]map[string]interface{}, error) {
	query := `
	SELECT name, source, tags, enabled, has_reasoning, updated_at
	FROM models
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying models: %w", err)
	}
	defer func() { _ = rows.Close() }()

	models := make(map[string]map[string]interface{})

	for rows.Next() {
		var name, source string
		var tags sql.NullString
		var enabled, hasReasoning bool
		var updatedAt sql.NullTime

		if err := rows.Scan(&name, &source, &tags, &enabled, &hasReasoning, &updatedAt); err != nil {
			log.Printf("Error scanning model row: %v", err)
			continue
		}

		tagsStr := "{}"
		if tags.Valid {
			tagsStr = tags.String
		}

		updatedAtStr := ""
		if updatedAt.Valid {
			updatedAtStr = updatedAt.Time.Format(time.RFC3339)
		}

		models[name] = map[string]interface{}{
			"source":        source,
			"tags":          tagsStr,
			"enabled":       enabled,
			"has_reasoning": hasReasoning,
			"updated_at":    updatedAtStr,
		}
	}

	return models, nil
}

// HasModels checks if there are any models in the database
func (db *DB) HasModels() (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM models`
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("error checking if models exist: %w", err)
	}

	return count > 0, nil
}

// EnableModel enables or disables a model
func (db *DB) EnableModel(name, source string, enabled bool) error {
	_, err := db.Exec(`
		UPDATE models
		SET enabled = $3, updated_at = NOW()
		WHERE name = $1 AND source = $2
	`, name, source, enabled)

	if err != nil {
		return fmt.Errorf("error updating model enabled status: %w", err)
	}

	return nil
}

// UpdateModelReasoning updates the reasoning capability of a model
func (db *DB) UpdateModelReasoning(name, source string, hasReasoning bool) error {
	_, err := db.Exec(`
		UPDATE models
		SET has_reasoning = $3, updated_at = NOW()
		WHERE name = $1 AND source = $2
	`, name, source, hasReasoning)

	if err != nil {
		return fmt.Errorf("error updating model reasoning status: %w", err)
	}

	return nil
}

// DeleteModel deletes a model from the database
func (db *DB) DeleteModel(name, source string) error {
	_, err := db.Exec(`
		DELETE FROM models
		WHERE name = $1 AND source = $2
	`, name, source)

	if err != nil {
		return fmt.Errorf("error deleting model: %w", err)
	}

	return nil
}

// ── Config schema ────────────────────────────────────────────────────────────

// initConfigSchema creates the benchmark_task_configs and system_prompts tables.
func (db *DB) initConfigSchema() error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS benchmark_task_configs (
		task_id        TEXT PRIMARY KEY,
		category       TEXT NOT NULL,
		prompt         TEXT NOT NULL,
		scoring_method TEXT NOT NULL,
		expected_answer TEXT,
		judge_criteria  JSONB,
		enabled        BOOLEAN NOT NULL DEFAULT TRUE,
		is_builtin     BOOLEAN NOT NULL DEFAULT TRUE,
		updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE TABLE IF NOT EXISTS system_prompts (
		key        TEXT PRIMARY KEY,
		value      TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`)
	if err != nil {
		return fmt.Errorf("failed to create config tables: %w", err)
	}
	return nil
}

// SeedBenchmarkTasksIfEmpty inserts the default task configs only when the table is empty.
// Safe to call on every startup — a no-op after the first run.
func (db *DB) SeedBenchmarkTasksIfEmpty(defaults []benchmark.BenchmarkTaskConfig) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM benchmark_task_configs`).Scan(&count); err != nil {
		return fmt.Errorf("failed to count benchmark task configs: %w", err)
	}
	if count > 0 {
		return nil
	}
	for _, cfg := range defaults {
		if err := db.UpsertBenchmarkTaskConfig(cfg); err != nil {
			log.Printf("db: error seeding benchmark task %s: %v", cfg.TaskID, err)
		}
	}
	log.Printf("db: seeded %d default benchmark task configs", len(defaults))
	return nil
}

// GetBenchmarkTaskConfigs returns all rows from benchmark_task_configs ordered by category then task_id.
func (db *DB) GetBenchmarkTaskConfigs() ([]benchmark.BenchmarkTaskConfig, error) {
	rows, err := db.Query(`
		SELECT task_id, category, prompt, scoring_method,
		       COALESCE(expected_answer,''), COALESCE(judge_criteria,'null'::jsonb),
		       enabled, is_builtin, updated_at
		FROM benchmark_task_configs
		ORDER BY category, task_id
	`)
	if err != nil {
		return nil, fmt.Errorf("error querying benchmark task configs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var cfgs []benchmark.BenchmarkTaskConfig
	for rows.Next() {
		var c benchmark.BenchmarkTaskConfig
		var criteriaJSON []byte
		var updatedAt time.Time
		if err := rows.Scan(
			&c.TaskID, &c.Category, &c.Prompt, &c.ScoringMethod,
			&c.ExpectedAnswer, &criteriaJSON,
			&c.Enabled, &c.IsBuiltin, &updatedAt,
		); err != nil {
			log.Printf("db: error scanning benchmark task config row: %v", err)
			continue
		}
		c.UpdatedAt = updatedAt.Format(time.RFC3339)
		if len(criteriaJSON) > 0 && string(criteriaJSON) != "null" {
			_ = json.Unmarshal(criteriaJSON, &c.JudgeCriteria)
		}
		cfgs = append(cfgs, c)
	}
	return cfgs, nil
}

// UpsertBenchmarkTaskConfig inserts or replaces a benchmark task config row.
func (db *DB) UpsertBenchmarkTaskConfig(cfg benchmark.BenchmarkTaskConfig) error {
	criteriaJSON, err := json.Marshal(cfg.JudgeCriteria)
	if err != nil {
		return fmt.Errorf("error marshalling judge criteria: %w", err)
	}
	_, err = db.Exec(`
		INSERT INTO benchmark_task_configs
			(task_id, category, prompt, scoring_method, expected_answer, judge_criteria, enabled, is_builtin, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8,NOW())
		ON CONFLICT (task_id) DO UPDATE SET
			category       = EXCLUDED.category,
			prompt         = EXCLUDED.prompt,
			scoring_method = EXCLUDED.scoring_method,
			expected_answer= EXCLUDED.expected_answer,
			judge_criteria = EXCLUDED.judge_criteria,
			enabled        = EXCLUDED.enabled,
			updated_at     = NOW()
	`, cfg.TaskID, cfg.Category, cfg.Prompt, cfg.ScoringMethod,
		cfg.ExpectedAnswer, string(criteriaJSON), cfg.Enabled, cfg.IsBuiltin)
	if err != nil {
		return fmt.Errorf("error upserting benchmark task config: %w", err)
	}
	return nil
}

// DeleteBenchmarkTaskConfig removes a custom (non-builtin) task or resets a builtin task to its defaults.
// If the task is builtin, defaults must be provided so the row can be restored.
func (db *DB) DeleteBenchmarkTaskConfig(taskID string, defaultCfgs []benchmark.BenchmarkTaskConfig) error {
	// Check if builtin.
	var isBuiltin bool
	err := db.QueryRow(`SELECT is_builtin FROM benchmark_task_configs WHERE task_id=$1`, taskID).Scan(&isBuiltin)
	if err == sql.ErrNoRows {
		return nil // already gone
	}
	if err != nil {
		return fmt.Errorf("error checking task builtin status: %w", err)
	}

	if !isBuiltin {
		_, err = db.Exec(`DELETE FROM benchmark_task_configs WHERE task_id=$1`, taskID)
		return err
	}

	// Builtin — restore to default values.
	for _, d := range defaultCfgs {
		if d.TaskID == taskID {
			d.Enabled = true
			return db.UpsertBenchmarkTaskConfig(d)
		}
	}
	return fmt.Errorf("default config for builtin task %q not found", taskID)
}

// ── System prompts ────────────────────────────────────────────────────────────

// SeedSystemPromptsIfEmpty inserts default system prompts only when the table is empty.
func (db *DB) SeedSystemPromptsIfEmpty(defaults map[string]string) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM system_prompts`).Scan(&count); err != nil {
		return fmt.Errorf("failed to count system prompts: %w", err)
	}
	if count > 0 {
		return nil
	}
	descriptions := map[string]string{
		"tagging_system":     "System message sent to every model during capability tagging (not used for o1/o3/o4-series).",
		"tagging_user":       "User message sent to every model during capability tagging.",
		"tagging_user_nosys": "Combined system+user message used for models that don't support system messages (o1/o3/o4-series).",
	}
	for key, value := range defaults {
		desc := descriptions[key]
		if err := db.SetSystemPrompt(key, value, desc); err != nil {
			log.Printf("db: error seeding system prompt %s: %v", key, err)
		}
	}
	log.Printf("db: seeded %d default system prompts", len(defaults))
	return nil
}

// GetSystemPrompt returns the value for the given key. found=false when the key does not exist.
func (db *DB) GetSystemPrompt(key string) (value string, found bool, err error) {
	err = db.QueryRow(`SELECT value FROM system_prompts WHERE key=$1`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("error getting system prompt %q: %w", key, err)
	}
	return value, true, nil
}

// SetSystemPrompt inserts or updates a system prompt.
func (db *DB) SetSystemPrompt(key, value, description string) error {
	_, err := db.Exec(`
		INSERT INTO system_prompts (key, value, description, updated_at)
		VALUES ($1,$2,$3,NOW())
		ON CONFLICT (key) DO UPDATE SET value=$2, description=$3, updated_at=NOW()
	`, key, value, description)
	if err != nil {
		return fmt.Errorf("error setting system prompt %q: %w", key, err)
	}
	return nil
}

// GetAllSystemPrompts returns all system prompt rows as a map[key]value.
func (db *DB) GetAllSystemPrompts() (map[string]SystemPromptRow, error) {
	rows, err := db.Query(`SELECT key, value, description, updated_at FROM system_prompts ORDER BY key`)
	if err != nil {
		return nil, fmt.Errorf("error querying system prompts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]SystemPromptRow)
	for rows.Next() {
		var r SystemPromptRow
		var updatedAt time.Time
		if err := rows.Scan(&r.Key, &r.Value, &r.Description, &updatedAt); err != nil {
			continue
		}
		r.UpdatedAt = updatedAt.Format(time.RFC3339)
		result[r.Key] = r
	}
	return result, nil
}

// SystemPromptRow is returned by GetAllSystemPrompts.
type SystemPromptRow struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
	UpdatedAt   string `json:"updated_at"`
}

// MigrateExistingModelsReasoning updates all existing models to set their reasoning capability
// based on automatic detection
func (db *DB) MigrateExistingModelsReasoning() error {
	log.Println("Starting migration to set reasoning capabilities for existing models...")

	// Get all models from database
	allModels, err := db.GetAllModels()
	if err != nil {
		return fmt.Errorf("error getting models for migration: %w", err)
	}

	count := 0
	for name := range allModels {
		hasReasoning := models.IsReasoningModel(name)

		// We need the source to update, let's get it from the model data
		if modelData, exists := allModels[name]; exists {
			if source, ok := modelData["source"].(string); ok {
				if err := db.UpdateModelReasoning(name, source, hasReasoning); err != nil {
					log.Printf("Error updating reasoning for model %s: %v", name, err)
					continue
				}
				count++
				log.Printf("Updated model %s (source: %s) reasoning capability to %v", name, source, hasReasoning)
			}
		}
	}

	log.Printf("Migration completed. Updated %d models", count)
	return nil
}
