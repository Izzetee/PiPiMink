package database

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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

	// Analytics tables for routing decision logging.
	if err := db.initAnalyticsSchema(); err != nil {
		return fmt.Errorf("failed to init analytics schema: %w", err)
	}

	// Auth tables for users, groups, providers, audit log.
	if err := db.initAuthSchema(); err != nil {
		return fmt.Errorf("failed to init auth schema: %w", err)
	}

	// User API tokens for Bearer authentication.
	if err := db.initTokenSchema(); err != nil {
		return fmt.Errorf("failed to init token schema: %w", err)
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

	// Add judge_model column if it doesn't exist (tracks which model scored each result).
	var judgeModelColExists bool
	err = db.QueryRow(`SELECT EXISTS (
		SELECT 1
		FROM information_schema.columns
		WHERE table_name = 'benchmark_results'
		AND column_name = 'judge_model'
	)`).Scan(&judgeModelColExists)
	if err != nil {
		log.Printf("Error checking if judge_model column exists: %v", err)
	} else if !judgeModelColExists {
		_, err = db.Exec(`ALTER TABLE benchmark_results ADD COLUMN judge_model TEXT NOT NULL DEFAULT ''`)
		if err != nil {
			log.Printf("Error adding 'judge_model' column: %v", err)
			return fmt.Errorf("failed to add judge_model column: %w", err)
		}
		log.Println("Added 'judge_model' column to benchmark_results table")
	}

	// Add response column if it doesn't exist (stores model's actual answer text).
	var responseColExists bool
	err = db.QueryRow(`SELECT EXISTS (
		SELECT 1
		FROM information_schema.columns
		WHERE table_name = 'benchmark_results'
		AND column_name = 'response'
	)`).Scan(&responseColExists)
	if err != nil {
		log.Printf("Error checking if response column exists: %v", err)
	} else if !responseColExists {
		_, err = db.Exec(`ALTER TABLE benchmark_results ADD COLUMN response TEXT NOT NULL DEFAULT ''`)
		if err != nil {
			log.Printf("Error adding 'response' column: %v", err)
			return fmt.Errorf("failed to add response column: %w", err)
		}
		log.Println("Added 'response' column to benchmark_results table")
	}

	return nil
}

// SaveBenchmarkResult persists a single task result for a model.
func (db *DB) SaveBenchmarkResult(modelName, source, category, taskID string, score float64, latencyMs int64, judgeModel, response string) error {
	_, err := db.Exec(`
		INSERT INTO benchmark_results (model_name, source, category, task_id, score, latency_ms, run_at, judge_model, response)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), $7, $8)
	`, modelName, source, category, taskID, score, latencyMs, judgeModel, response)
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

// BenchmarkResult holds a single per-task benchmark result row.
type BenchmarkResult struct {
	TaskID     string  `json:"taskId"`
	Category   string  `json:"category"`
	Score      float64 `json:"score"`
	LatencyMs  int64   `json:"latencyMs"`
	RunAt      string  `json:"scoredAt"`
	JudgeModel string  `json:"judgeModel"`
	Response   string  `json:"response"`
}

// GetBenchmarkResults returns individual task-level benchmark results for a model.
func (db *DB) GetBenchmarkResults(modelName, source string) ([]BenchmarkResult, error) {
	rows, err := db.Query(`
		SELECT task_id, category, score, latency_ms, run_at, COALESCE(judge_model, ''), COALESCE(response, '')
		FROM benchmark_results
		WHERE model_name = $1 AND source = $2
		ORDER BY category, task_id
	`, modelName, source)
	if err != nil {
		return nil, fmt.Errorf("error querying benchmark results: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []BenchmarkResult
	for rows.Next() {
		var r BenchmarkResult
		var runAt time.Time
		if err := rows.Scan(&r.TaskID, &r.Category, &r.Score, &r.LatencyMs, &runAt, &r.JudgeModel, &r.Response); err != nil {
			log.Printf("Error scanning benchmark result row: %v", err)
			continue
		}
		r.RunAt = runAt.Format(time.RFC3339)
		results = append(results, r)
	}
	return results, nil
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

// ResetModel clears a model's tags, benchmark results, and routing analytics
// but keeps the model entry itself (disabled, empty tags).
func (db *DB) ResetModel(name, source string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Clear tags and disable
	if _, err := tx.Exec(`
		UPDATE models SET tags = '{}'::jsonb, enabled = FALSE, updated_at = NOW()
		WHERE name = $1 AND source = $2
	`, name, source); err != nil {
		return fmt.Errorf("error resetting model tags: %w", err)
	}

	// Delete benchmark results
	if _, err := tx.Exec(`
		DELETE FROM benchmark_results WHERE model_name = $1 AND source = $2
	`, name, source); err != nil {
		return fmt.Errorf("error deleting benchmark results: %w", err)
	}

	// Delete routing decisions
	if _, err := tx.Exec(`
		DELETE FROM routing_decisions WHERE selected_model = $1 AND provider = $2
	`, name, source); err != nil {
		return fmt.Errorf("error deleting routing decisions: %w", err)
	}

	return tx.Commit()
}

// DeleteModelFull deletes a model and all associated data (benchmarks, routing analytics).
func (db *DB) DeleteModelFull(name, source string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Delete benchmark results
	if _, err := tx.Exec(`
		DELETE FROM benchmark_results WHERE model_name = $1 AND source = $2
	`, name, source); err != nil {
		return fmt.Errorf("error deleting benchmark results: %w", err)
	}

	// Delete routing decisions
	if _, err := tx.Exec(`
		DELETE FROM routing_decisions WHERE selected_model = $1 AND provider = $2
	`, name, source); err != nil {
		return fmt.Errorf("error deleting routing decisions: %w", err)
	}

	// Delete the model itself
	if _, err := tx.Exec(`
		DELETE FROM models WHERE name = $1 AND source = $2
	`, name, source); err != nil {
		return fmt.Errorf("error deleting model: %w", err)
	}

	return tx.Commit()
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

// ── Analytics schema ─────────────────────────────────────────────────────────

// initAnalyticsSchema creates the routing_decisions table and indexes.
func (db *DB) initAnalyticsSchema() error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS routing_decisions (
		id                 SERIAL PRIMARY KEY,
		timestamp          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		prompt_snippet     TEXT NOT NULL,
		full_prompt        TEXT NOT NULL,
		analyzed_tags      JSONB NOT NULL DEFAULT '[]',
		tag_relevance      JSONB NOT NULL DEFAULT '{}',
		selected_model     TEXT NOT NULL,
		provider           TEXT NOT NULL DEFAULT '',
		routing_reason     TEXT NOT NULL DEFAULT '',
		evaluator_model    TEXT NOT NULL DEFAULT '',
		evaluation_time_ms BIGINT NOT NULL DEFAULT 0,
		cache_hit          BOOLEAN NOT NULL DEFAULT FALSE,
		latency_ms         BIGINT NOT NULL DEFAULT 0,
		status             TEXT NOT NULL DEFAULT 'success'
	);
	CREATE INDEX IF NOT EXISTS idx_routing_decisions_ts    ON routing_decisions(timestamp);
	CREATE INDEX IF NOT EXISTS idx_routing_decisions_model ON routing_decisions(selected_model);
	`)
	if err != nil {
		return fmt.Errorf("failed to create routing_decisions table: %w", err)
	}

	// Additive migration: add user_id column to routing_decisions.
	var userIDColExists bool
	err = db.QueryRow(`SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_name = 'routing_decisions' AND column_name = 'user_id'
	)`).Scan(&userIDColExists)
	if err != nil {
		log.Printf("Error checking if user_id column exists on routing_decisions: %v", err)
	} else if !userIDColExists {
		_, err = db.Exec(`ALTER TABLE routing_decisions ADD COLUMN user_id TEXT NOT NULL DEFAULT 'anonymous'`)
		if err != nil {
			log.Printf("Error adding 'user_id' column to routing_decisions: %v", err)
		} else {
			log.Println("Added 'user_id' column to routing_decisions table")
		}
	}
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_routing_decisions_user ON routing_decisions(user_id)`)

	return nil
}

// RoutingDecisionRow represents a single routing decision for persistence.
type RoutingDecisionRow struct {
	ID               int64              `json:"id"`
	Timestamp        string             `json:"timestamp"`
	PromptSnippet    string             `json:"promptSnippet"`
	FullPrompt       string             `json:"fullPrompt"`
	AnalyzedTags     []string           `json:"analyzedTags"`
	TagRelevance     map[string]float64 `json:"tagRelevance"`
	SelectedModel    string             `json:"selectedModel"`
	Provider         string             `json:"provider"`
	RoutingReason    string             `json:"routingReason"`
	EvaluatorModel   string             `json:"evaluatorModel"`
	EvaluationTimeMs int64              `json:"evaluationTimeMs"`
	CacheHit         bool               `json:"cacheHit"`
	LatencyMs        int64              `json:"latencyMs"`
	Status           string             `json:"status"`
	UserID           string             `json:"userId"`
}

// SaveRoutingDecision persists a routing decision to the database.
func (db *DB) SaveRoutingDecision(rd RoutingDecisionRow) error {
	tagsJSON, err := json.Marshal(rd.AnalyzedTags)
	if err != nil {
		return fmt.Errorf("error marshalling analyzed tags: %w", err)
	}
	relevanceJSON, err := json.Marshal(rd.TagRelevance)
	if err != nil {
		return fmt.Errorf("error marshalling tag relevance: %w", err)
	}
	userID := rd.UserID
	if userID == "" {
		userID = "anonymous"
	}
	_, err = db.Exec(`
		INSERT INTO routing_decisions
			(prompt_snippet, full_prompt, analyzed_tags, tag_relevance, selected_model,
			 provider, routing_reason, evaluator_model, evaluation_time_ms, cache_hit, latency_ms, status, user_id)
		VALUES ($1, $2, $3::jsonb, $4::jsonb, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, rd.PromptSnippet, rd.FullPrompt, string(tagsJSON), string(relevanceJSON),
		rd.SelectedModel, rd.Provider, rd.RoutingReason, rd.EvaluatorModel,
		rd.EvaluationTimeMs, rd.CacheHit, rd.LatencyMs, rd.Status, userID)
	if err != nil {
		return fmt.Errorf("error saving routing decision: %w", err)
	}
	return nil
}

// GetRoutingDecisions returns paginated routing decisions within a time range.
// Returns the rows, total count, and any error.
func (db *DB) GetRoutingDecisions(start, end time.Time, limit, offset int) ([]RoutingDecisionRow, int, error) {
	var total int
	err := db.QueryRow(`SELECT COUNT(*) FROM routing_decisions WHERE timestamp >= $1 AND timestamp <= $2`, start, end).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting routing decisions: %w", err)
	}

	rows, err := db.Query(`
		SELECT id, timestamp, prompt_snippet, full_prompt, analyzed_tags, tag_relevance,
		       selected_model, provider, routing_reason, evaluator_model,
		       evaluation_time_ms, cache_hit, latency_ms, status, COALESCE(user_id, 'anonymous')
		FROM routing_decisions
		WHERE timestamp >= $1 AND timestamp <= $2
		ORDER BY timestamp DESC
		LIMIT $3 OFFSET $4
	`, start, end, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("error querying routing decisions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []RoutingDecisionRow
	for rows.Next() {
		var rd RoutingDecisionRow
		var ts time.Time
		var tagsJSON, relevanceJSON []byte
		if err := rows.Scan(&rd.ID, &ts, &rd.PromptSnippet, &rd.FullPrompt,
			&tagsJSON, &relevanceJSON, &rd.SelectedModel, &rd.Provider,
			&rd.RoutingReason, &rd.EvaluatorModel, &rd.EvaluationTimeMs,
			&rd.CacheHit, &rd.LatencyMs, &rd.Status, &rd.UserID); err != nil {
			log.Printf("Error scanning routing decision row: %v", err)
			continue
		}
		rd.Timestamp = ts.Format(time.RFC3339)
		_ = json.Unmarshal(tagsJSON, &rd.AnalyzedTags)
		if rd.AnalyzedTags == nil {
			rd.AnalyzedTags = []string{}
		}
		rd.TagRelevance = make(map[string]float64)
		_ = json.Unmarshal(relevanceJSON, &rd.TagRelevance)
		results = append(results, rd)
	}
	return results, total, nil
}

// KpiSummary holds aggregate metrics for a time window.
type KpiSummary struct {
	TotalRequests int64   `json:"totalRequests"`
	AvgLatencyMs  int64   `json:"avgLatencyMs"`
	MostUsedModel string  `json:"mostUsedModel"`
	ErrorRate     float64 `json:"errorRate"`
}

// GetKpiSummary returns aggregate KPI metrics for a time window.
func (db *DB) GetKpiSummary(start, end time.Time) (KpiSummary, error) {
	var kpi KpiSummary
	var avgLatency sql.NullFloat64
	var mostUsed sql.NullString

	err := db.QueryRow(`
		SELECT COUNT(*),
		       COALESCE(AVG(latency_ms), 0),
		       COALESCE(SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0), 0)
		FROM routing_decisions
		WHERE timestamp >= $1 AND timestamp <= $2
	`, start, end).Scan(&kpi.TotalRequests, &avgLatency, &kpi.ErrorRate)
	if err != nil {
		return kpi, fmt.Errorf("error querying KPI summary: %w", err)
	}
	kpi.AvgLatencyMs = int64(avgLatency.Float64)

	err = db.QueryRow(`
		SELECT selected_model FROM routing_decisions
		WHERE timestamp >= $1 AND timestamp <= $2
		GROUP BY selected_model ORDER BY COUNT(*) DESC LIMIT 1
	`, start, end).Scan(&mostUsed)
	if err != nil && err != sql.ErrNoRows {
		return kpi, fmt.Errorf("error querying most used model: %w", err)
	}
	if mostUsed.Valid {
		kpi.MostUsedModel = mostUsed.String
	}

	return kpi, nil
}

// ModelUsageRow holds request count for a single model.
type ModelUsageRow struct {
	ModelName    string  `json:"modelName"`
	RequestCount int64   `json:"requestCount"`
	Percentage   float64 `json:"percentage"`
}

// GetModelUsage returns request counts per model for a time window.
func (db *DB) GetModelUsage(start, end time.Time) ([]ModelUsageRow, error) {
	rows, err := db.Query(`
		SELECT selected_model, COUNT(*) as cnt
		FROM routing_decisions
		WHERE timestamp >= $1 AND timestamp <= $2
		GROUP BY selected_model
		ORDER BY cnt DESC
	`, start, end)
	if err != nil {
		return nil, fmt.Errorf("error querying model usage: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []ModelUsageRow
	var total int64
	for rows.Next() {
		var r ModelUsageRow
		if err := rows.Scan(&r.ModelName, &r.RequestCount); err != nil {
			log.Printf("Error scanning model usage row: %v", err)
			continue
		}
		total += r.RequestCount
		results = append(results, r)
	}
	for i := range results {
		if total > 0 {
			results[i].Percentage = float64(results[i].RequestCount) * 100 / float64(total)
		}
	}
	return results, nil
}

// LatencyPerModelRow holds average latency for a single model.
type LatencyPerModelRow struct {
	ModelName    string `json:"modelName"`
	AvgLatencyMs int64  `json:"avgLatencyMs"`
}

// GetLatencyPerModel returns average latency per model for a time window.
func (db *DB) GetLatencyPerModel(start, end time.Time) ([]LatencyPerModelRow, error) {
	rows, err := db.Query(`
		SELECT selected_model, AVG(latency_ms)
		FROM routing_decisions
		WHERE timestamp >= $1 AND timestamp <= $2
		GROUP BY selected_model
		ORDER BY AVG(latency_ms) ASC
	`, start, end)
	if err != nil {
		return nil, fmt.Errorf("error querying latency per model: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []LatencyPerModelRow
	for rows.Next() {
		var r LatencyPerModelRow
		var avg float64
		if err := rows.Scan(&r.ModelName, &avg); err != nil {
			log.Printf("Error scanning latency per model row: %v", err)
			continue
		}
		r.AvgLatencyMs = int64(avg)
		results = append(results, r)
	}
	return results, nil
}

// LatencyTimeSeriesRow holds a time-bucketed latency data point.
type LatencyTimeSeriesRow struct {
	Timestamp    string `json:"timestamp"`
	AvgLatencyMs int64  `json:"avgLatencyMs"`
	P95LatencyMs int64  `json:"p95LatencyMs"`
}

// GetLatencyTimeSeries returns time-bucketed latency averages and p95 for a time range.
func (db *DB) GetLatencyTimeSeries(start, end time.Time) ([]LatencyTimeSeriesRow, error) {
	// Choose bucket size based on time range duration.
	// date_trunc() accepts unit names like 'hour', 'minute', 'day' — not intervals.
	duration := end.Sub(start)
	var bucket string
	if duration <= 2*time.Hour {
		bucket = "minute"
	} else if duration <= 48*time.Hour {
		bucket = "hour"
	} else {
		bucket = "day"
	}

	rows, err := db.Query(fmt.Sprintf(`
		SELECT date_trunc('%s', timestamp) as bucket,
		       AVG(latency_ms),
		       PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms)
		FROM routing_decisions
		WHERE timestamp >= $1 AND timestamp <= $2
		GROUP BY bucket
		ORDER BY bucket ASC
	`, bucket), start, end)
	if err != nil {
		return nil, fmt.Errorf("error querying latency time series: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []LatencyTimeSeriesRow
	for rows.Next() {
		var r LatencyTimeSeriesRow
		var ts time.Time
		var avg, p95 float64
		if err := rows.Scan(&ts, &avg, &p95); err != nil {
			log.Printf("Error scanning latency time series row: %v", err)
			continue
		}
		r.Timestamp = ts.Format(time.RFC3339)
		r.AvgLatencyMs = int64(avg)
		r.P95LatencyMs = int64(p95)
		results = append(results, r)
	}
	return results, nil
}

// LatencyPercentilesRow holds P50/P95/P99 latencies for a model.
type LatencyPercentilesRow struct {
	ModelName string `json:"modelName"`
	P50Ms     int64  `json:"p50Ms"`
	P95Ms     int64  `json:"p95Ms"`
	P99Ms     int64  `json:"p99Ms"`
}

// GetLatencyPercentiles returns P50/P95/P99 latency per model for a time window.
func (db *DB) GetLatencyPercentiles(start, end time.Time) ([]LatencyPercentilesRow, error) {
	rows, err := db.Query(`
		SELECT selected_model,
		       PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY latency_ms),
		       PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms),
		       PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY latency_ms)
		FROM routing_decisions
		WHERE timestamp >= $1 AND timestamp <= $2
		GROUP BY selected_model
		ORDER BY selected_model
	`, start, end)
	if err != nil {
		return nil, fmt.Errorf("error querying latency percentiles: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []LatencyPercentilesRow
	for rows.Next() {
		var r LatencyPercentilesRow
		var p50, p95, p99 float64
		if err := rows.Scan(&r.ModelName, &p50, &p95, &p99); err != nil {
			log.Printf("Error scanning latency percentiles row: %v", err)
			continue
		}
		r.P50Ms = int64(p50)
		r.P95Ms = int64(p95)
		r.P99Ms = int64(p99)
		results = append(results, r)
	}
	return results, nil
}

// ============================================================================
// Auth & Users
// ============================================================================

// AuthProviderRow represents a configured identity provider.
type AuthProviderRow struct {
	ID            string     `json:"id"`
	Type          string     `json:"type"`
	Name          string     `json:"name"`
	Status        string     `json:"status"`
	IssuerURL     string     `json:"issuerUrl"`
	ClientID      string     `json:"clientId"`
	ClientSecret  string     `json:"clientSecret"`
	Scopes        string     `json:"scopes"`
	RedirectURI   string     `json:"redirectUri"`
	AutoProvision bool       `json:"autoProvision"`
	ServerURL     string     `json:"serverUrl"`
	BindDN        string     `json:"bindDn"`
	BaseDN        string     `json:"baseDn"`
	SearchFilter  string     `json:"searchFilter"`
	GroupMapping  string     `json:"groupMapping"`
	LastVerified  *time.Time `json:"lastVerified"`
	CreatedAt     *time.Time `json:"createdAt"`
}

// UserRow represents a user in the system.
type UserRow struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Email            string   `json:"email"`
	Role             string   `json:"role"`
	AuthSource       string   `json:"authSource"`
	AuthProviderName *string  `json:"authProviderName"`
	Groups           []string `json:"groups"`
	LastLogin        string   `json:"lastLogin"`
	CreatedAt        string   `json:"createdAt"`
	RequestCount     int64    `json:"requestCount"`
	TokenUsage       int64    `json:"tokenUsage"`
	AvatarURL        *string  `json:"avatarUrl"`
}

// GroupRow represents a group synced from an identity provider.
type GroupRow struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Source       string           `json:"source"`
	MemberCount  int              `json:"memberCount"`
	Role         string           `json:"role"`
	RoutingRules []RoutingRuleRow `json:"routingRules"`
	SyncedAt     string           `json:"syncedAt"`
}

// RoutingRuleRow represents a per-group routing constraint.
type RoutingRuleRow struct {
	ID          string   `json:"id"`
	GroupID     string   `json:"-"`
	Type        string   `json:"type"`
	Providers   []string `json:"providers,omitempty"`
	Models      []string `json:"models,omitempty"`
	Description string   `json:"description"`
}

// AuditEntryRow represents a chronological log entry for user management actions.
type AuditEntryRow struct {
	ID        string  `json:"id"`
	Timestamp string  `json:"timestamp"`
	Actor     string  `json:"actor"`
	Action    string  `json:"action"`
	Target    string  `json:"target"`
	Details   string  `json:"details"`
	Reason    *string `json:"reason"`
}

func (db *DB) initAuthSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS auth_providers (
		id             TEXT PRIMARY KEY,
		type           TEXT NOT NULL,
		name           TEXT NOT NULL,
		status         TEXT NOT NULL DEFAULT 'not_configured',
		issuer_url     TEXT NOT NULL DEFAULT '',
		client_id      TEXT NOT NULL DEFAULT '',
		client_secret  TEXT NOT NULL DEFAULT '',
		scopes         TEXT NOT NULL DEFAULT '',
		redirect_uri   TEXT NOT NULL DEFAULT '',
		auto_provision BOOLEAN NOT NULL DEFAULT FALSE,
		server_url     TEXT NOT NULL DEFAULT '',
		bind_dn        TEXT NOT NULL DEFAULT '',
		base_dn        TEXT NOT NULL DEFAULT '',
		search_filter  TEXT NOT NULL DEFAULT '',
		group_mapping  TEXT NOT NULL DEFAULT '',
		last_verified  TIMESTAMPTZ,
		created_at     TIMESTAMPTZ
	);

	CREATE TABLE IF NOT EXISTS users (
		id                 TEXT PRIMARY KEY,
		name               TEXT NOT NULL,
		email              TEXT NOT NULL UNIQUE,
		role               TEXT NOT NULL DEFAULT 'user',
		auth_source        TEXT NOT NULL DEFAULT 'local',
		auth_provider_name TEXT,
		groups_list        JSONB NOT NULL DEFAULT '[]',
		last_login         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		request_count      BIGINT NOT NULL DEFAULT 0,
		token_usage        BIGINT NOT NULL DEFAULT 0,
		avatar_url         TEXT
	);

	CREATE TABLE IF NOT EXISTS groups (
		id           TEXT PRIMARY KEY,
		name         TEXT NOT NULL,
		source       TEXT NOT NULL,
		member_count INT NOT NULL DEFAULT 0,
		role         TEXT NOT NULL DEFAULT 'user',
		synced_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS routing_rules (
		id          TEXT PRIMARY KEY,
		group_id    TEXT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
		type        TEXT NOT NULL,
		providers   JSONB,
		models      JSONB,
		description TEXT NOT NULL DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS audit_log (
		id        TEXT PRIMARY KEY,
		timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		actor     TEXT NOT NULL,
		action    TEXT NOT NULL,
		target    TEXT NOT NULL,
		details   TEXT NOT NULL DEFAULT '',
		reason    TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_audit_log_ts ON audit_log(timestamp);
	CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log(action);
	`
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create auth tables: %w", err)
	}
	return nil
}

// SeedAuthProvidersIfEmpty inserts default OAuth + LDAP provider entries on first run.
func (db *DB) SeedAuthProvidersIfEmpty() error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_providers`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err := db.Exec(`
		INSERT INTO auth_providers (id, type, name, status)
		VALUES ('provider-oauth', 'oauth', 'Authentik', 'not_configured'),
		       ('provider-ldap', 'ldap', 'Corporate LDAP', 'not_configured')
	`)
	return err
}

// GetAuthProviders returns all configured auth providers.
func (db *DB) GetAuthProviders() ([]AuthProviderRow, error) {
	rows, err := db.Query(`
		SELECT id, type, name, status, issuer_url, client_id, client_secret, scopes,
		       redirect_uri, auto_provision, server_url, bind_dn, base_dn, search_filter,
		       group_mapping, last_verified, created_at
		FROM auth_providers ORDER BY type
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []AuthProviderRow
	for rows.Next() {
		var p AuthProviderRow
		if err := rows.Scan(&p.ID, &p.Type, &p.Name, &p.Status, &p.IssuerURL,
			&p.ClientID, &p.ClientSecret, &p.Scopes, &p.RedirectURI, &p.AutoProvision,
			&p.ServerURL, &p.BindDN, &p.BaseDN, &p.SearchFilter, &p.GroupMapping,
			&p.LastVerified, &p.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, nil
}

// SaveAuthProvider upserts an auth provider.
func (db *DB) SaveAuthProvider(p AuthProviderRow) error {
	_, err := db.Exec(`
		INSERT INTO auth_providers (id, type, name, status, issuer_url, client_id, client_secret,
			scopes, redirect_uri, auto_provision, server_url, bind_dn, base_dn, search_filter,
			group_mapping, last_verified, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		ON CONFLICT (id) DO UPDATE SET
			type=EXCLUDED.type, name=EXCLUDED.name, status=EXCLUDED.status,
			issuer_url=EXCLUDED.issuer_url, client_id=EXCLUDED.client_id,
			client_secret=EXCLUDED.client_secret, scopes=EXCLUDED.scopes,
			redirect_uri=EXCLUDED.redirect_uri, auto_provision=EXCLUDED.auto_provision,
			server_url=EXCLUDED.server_url, bind_dn=EXCLUDED.bind_dn,
			base_dn=EXCLUDED.base_dn, search_filter=EXCLUDED.search_filter,
			group_mapping=EXCLUDED.group_mapping, last_verified=EXCLUDED.last_verified,
			created_at=EXCLUDED.created_at
	`, p.ID, p.Type, p.Name, p.Status, p.IssuerURL, p.ClientID, p.ClientSecret,
		p.Scopes, p.RedirectURI, p.AutoProvision, p.ServerURL, p.BindDN, p.BaseDN,
		p.SearchFilter, p.GroupMapping, p.LastVerified, p.CreatedAt)
	return err
}

// GetUsers returns all users.
func (db *DB) GetUsers() ([]UserRow, error) {
	rows, err := db.Query(`
		SELECT id, name, email, role, auth_source, auth_provider_name, groups_list,
		       last_login, created_at, request_count, token_usage, avatar_url
		FROM users ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []UserRow
	for rows.Next() {
		var u UserRow
		var groupsJSON []byte
		var lastLogin, createdAt time.Time
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.AuthSource,
			&u.AuthProviderName, &groupsJSON, &lastLogin, &createdAt,
			&u.RequestCount, &u.TokenUsage, &u.AvatarURL); err != nil {
			return nil, err
		}
		u.LastLogin = lastLogin.Format(time.RFC3339)
		u.CreatedAt = createdAt.Format(time.RFC3339)
		if err := json.Unmarshal(groupsJSON, &u.Groups); err != nil {
			u.Groups = []string{}
		}
		result = append(result, u)
	}
	return result, nil
}

// GetUserByEmail returns a single user by email, or nil if not found.
func (db *DB) GetUserByEmail(email string) (*UserRow, error) {
	var u UserRow
	var groupsJSON []byte
	var lastLogin, createdAt time.Time
	err := db.QueryRow(`
		SELECT id, name, email, role, auth_source, auth_provider_name, groups_list,
		       last_login, created_at, request_count, token_usage, avatar_url
		FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.AuthSource,
		&u.AuthProviderName, &groupsJSON, &lastLogin, &createdAt,
		&u.RequestCount, &u.TokenUsage, &u.AvatarURL)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.LastLogin = lastLogin.Format(time.RFC3339)
	u.CreatedAt = createdAt.Format(time.RFC3339)
	if err := json.Unmarshal(groupsJSON, &u.Groups); err != nil {
		u.Groups = []string{}
	}
	return &u, nil
}

// UpsertUser creates or updates a user.
func (db *DB) UpsertUser(u UserRow) error {
	groupsJSON, _ := json.Marshal(u.Groups)
	_, err := db.Exec(`
		INSERT INTO users (id, name, email, role, auth_source, auth_provider_name,
			groups_list, last_login, created_at, request_count, token_usage, avatar_url)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (email) DO UPDATE SET
			name=EXCLUDED.name, role=EXCLUDED.role, auth_source=EXCLUDED.auth_source,
			auth_provider_name=EXCLUDED.auth_provider_name, groups_list=EXCLUDED.groups_list,
			last_login=EXCLUDED.last_login, request_count=EXCLUDED.request_count,
			token_usage=EXCLUDED.token_usage, avatar_url=EXCLUDED.avatar_url
	`, u.ID, u.Name, u.Email, u.Role, u.AuthSource, u.AuthProviderName,
		groupsJSON, u.LastLogin, u.CreatedAt, u.RequestCount, u.TokenUsage, u.AvatarURL)
	return err
}

// ChangeUserRole updates a user's role.
func (db *DB) ChangeUserRole(userID, role string) error {
	_, err := db.Exec(`UPDATE users SET role = $1 WHERE id = $2`, role, userID)
	return err
}

// DeleteUser removes a user by ID.
func (db *DB) DeleteUser(userID string) error {
	_, err := db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	return err
}

// GetGroups returns all groups with their routing rules.
func (db *DB) GetGroups() ([]GroupRow, error) {
	groupRows, err := db.Query(`SELECT id, name, source, member_count, role, synced_at FROM groups ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = groupRows.Close() }()

	var groups []GroupRow
	for groupRows.Next() {
		var g GroupRow
		var syncedAt time.Time
		if err := groupRows.Scan(&g.ID, &g.Name, &g.Source, &g.MemberCount, &g.Role, &syncedAt); err != nil {
			return nil, err
		}
		g.SyncedAt = syncedAt.Format(time.RFC3339)
		g.RoutingRules = []RoutingRuleRow{}
		groups = append(groups, g)
	}

	ruleRows, err := db.Query(`SELECT id, group_id, type, providers, models, description FROM routing_rules ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = ruleRows.Close() }()

	rulesByGroup := make(map[string][]RoutingRuleRow)
	for ruleRows.Next() {
		var r RoutingRuleRow
		var providersJSON, modelsJSON []byte
		if err := ruleRows.Scan(&r.ID, &r.GroupID, &r.Type, &providersJSON, &modelsJSON, &r.Description); err != nil {
			return nil, err
		}
		if providersJSON != nil {
			_ = json.Unmarshal(providersJSON, &r.Providers)
		}
		if modelsJSON != nil {
			_ = json.Unmarshal(modelsJSON, &r.Models)
		}
		rulesByGroup[r.GroupID] = append(rulesByGroup[r.GroupID], r)
	}

	for i := range groups {
		if rules, ok := rulesByGroup[groups[i].ID]; ok {
			groups[i].RoutingRules = rules
		}
	}
	return groups, nil
}

// SaveGroup upserts a group.
func (db *DB) SaveGroup(g GroupRow) error {
	_, err := db.Exec(`
		INSERT INTO groups (id, name, source, member_count, role, synced_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (id) DO UPDATE SET
			name=EXCLUDED.name, source=EXCLUDED.source, member_count=EXCLUDED.member_count,
			role=EXCLUDED.role, synced_at=EXCLUDED.synced_at
	`, g.ID, g.Name, g.Source, g.MemberCount, g.Role, g.SyncedAt)
	return err
}

// ChangeGroupRole updates a group's role.
func (db *DB) ChangeGroupRole(groupID, role string) error {
	_, err := db.Exec(`UPDATE groups SET role = $1 WHERE id = $2`, role, groupID)
	return err
}

// SaveRoutingRule inserts a routing rule for a group.
func (db *DB) SaveRoutingRule(groupID string, r RoutingRuleRow) error {
	var providersJSON, modelsJSON []byte
	if len(r.Providers) > 0 {
		providersJSON, _ = json.Marshal(r.Providers)
	}
	if len(r.Models) > 0 {
		modelsJSON, _ = json.Marshal(r.Models)
	}
	_, err := db.Exec(`
		INSERT INTO routing_rules (id, group_id, type, providers, models, description)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, r.ID, groupID, r.Type, providersJSON, modelsJSON, r.Description)
	return err
}

// DeleteRoutingRule removes a routing rule.
func (db *DB) DeleteRoutingRule(groupID, ruleID string) error {
	_, err := db.Exec(`DELETE FROM routing_rules WHERE id = $1 AND group_id = $2`, ruleID, groupID)
	return err
}

// GetAuditLog returns all audit entries ordered by timestamp DESC.
func (db *DB) GetAuditLog() ([]AuditEntryRow, error) {
	rows, err := db.Query(`
		SELECT id, timestamp, actor, action, target, details, reason
		FROM audit_log ORDER BY timestamp DESC
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []AuditEntryRow
	for rows.Next() {
		var e AuditEntryRow
		var ts time.Time
		if err := rows.Scan(&e.ID, &ts, &e.Actor, &e.Action, &e.Target, &e.Details, &e.Reason); err != nil {
			return nil, err
		}
		e.Timestamp = ts.Format(time.RFC3339)
		result = append(result, e)
	}
	return result, nil
}

// SaveAuditEntry inserts an audit log entry.
func (db *DB) SaveAuditEntry(e AuditEntryRow) error {
	_, err := db.Exec(`
		INSERT INTO audit_log (id, timestamp, actor, action, target, details, reason)
		VALUES ($1, NOW(), $2, $3, $4, $5, $6)
	`, e.ID, e.Actor, e.Action, e.Target, e.Details, e.Reason)
	return err
}

// ============================================================================
// User API Tokens (Bearer authentication)
// ============================================================================

// UserAPITokenRow represents a user-issued API token (plaintext is never stored).
type UserAPITokenRow struct {
	ID         string  `json:"id"`
	UserID     string  `json:"userId"`
	Name       string  `json:"name"`
	CreatedAt  string  `json:"createdAt"`
	LastUsedAt *string `json:"lastUsedAt"`
	ExpiresAt  *string `json:"expiresAt"`
	Revoked    bool    `json:"revoked"`
}

// initTokenSchema creates the user_api_tokens table.
func (db *DB) initTokenSchema() error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS user_api_tokens (
		id           TEXT PRIMARY KEY,
		user_id      TEXT NOT NULL,
		name         TEXT NOT NULL DEFAULT '',
		token_hash   TEXT NOT NULL UNIQUE,
		created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		last_used_at TIMESTAMPTZ,
		expires_at   TIMESTAMPTZ,
		revoked      BOOLEAN NOT NULL DEFAULT FALSE
	);
	CREATE INDEX IF NOT EXISTS idx_user_api_tokens_hash ON user_api_tokens(token_hash);
	CREATE INDEX IF NOT EXISTS idx_user_api_tokens_user ON user_api_tokens(user_id);
	`)
	if err != nil {
		return fmt.Errorf("failed to create user_api_tokens table: %w", err)
	}
	return nil
}

// HashToken returns the SHA-256 hex digest of a plaintext token.
func HashToken(plaintext string) string {
	h := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(h[:])
}

// CreateUserAPIToken generates a new API token for the given user.
// Returns the token ID, plaintext token (shown once), and any error.
func (db *DB) CreateUserAPIToken(userID, name string) (string, string, error) {
	// Generate random token ID and plaintext
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return "", "", fmt.Errorf("error generating token ID: %w", err)
	}
	tokenID := "tok_" + hex.EncodeToString(idBytes)

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", fmt.Errorf("error generating token: %w", err)
	}
	plaintext := "ppm_" + hex.EncodeToString(tokenBytes)
	tokenHash := HashToken(plaintext)

	_, err := db.Exec(`
		INSERT INTO user_api_tokens (id, user_id, name, token_hash)
		VALUES ($1, $2, $3, $4)
	`, tokenID, userID, name, tokenHash)
	if err != nil {
		return "", "", fmt.Errorf("error creating API token: %w", err)
	}
	return tokenID, plaintext, nil
}

// GetUserByAPIToken looks up a user by their Bearer token hash.
// Returns nil if the token is not found, revoked, or expired.
func (db *DB) GetUserByAPIToken(tokenHash string) (*UserRow, error) {
	var userID string
	err := db.QueryRow(`
		SELECT user_id FROM user_api_tokens
		WHERE token_hash = $1
		  AND revoked = FALSE
		  AND (expires_at IS NULL OR expires_at > NOW())
	`, tokenHash).Scan(&userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error looking up API token: %w", err)
	}

	// Update last_used_at asynchronously (best-effort)
	go func() {
		_, _ = db.Exec(`UPDATE user_api_tokens SET last_used_at = NOW() WHERE token_hash = $1`, tokenHash)
	}()

	// Look up the actual user
	return db.GetUserByID(userID)
}

// GetUserByID returns a user by their ID. Returns nil if not found.
func (db *DB) GetUserByID(userID string) (*UserRow, error) {
	var u UserRow
	var groups sql.NullString
	var lastLogin sql.NullTime
	var createdAt sql.NullTime
	err := db.QueryRow(`
		SELECT id, name, email, role, auth_source, auth_provider_name,
		       groups, last_login, created_at, request_count, token_usage, avatar_url
		FROM users WHERE id = $1
	`, userID).Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.AuthSource, &u.AuthProviderName,
		&groups, &lastLogin, &createdAt, &u.RequestCount, &u.TokenUsage, &u.AvatarURL)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error looking up user by ID: %w", err)
	}
	if groups.Valid && groups.String != "" {
		_ = json.Unmarshal([]byte(groups.String), &u.Groups)
	}
	if u.Groups == nil {
		u.Groups = []string{}
	}
	if lastLogin.Valid {
		u.LastLogin = lastLogin.Time.Format(time.RFC3339)
	}
	if createdAt.Valid {
		u.CreatedAt = createdAt.Time.Format(time.RFC3339)
	}
	return &u, nil
}

// ListUserAPITokens returns all tokens for a user (no plaintext).
func (db *DB) ListUserAPITokens(userID string) ([]UserAPITokenRow, error) {
	rows, err := db.Query(`
		SELECT id, user_id, name, created_at, last_used_at, expires_at, revoked
		FROM user_api_tokens
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("error listing API tokens: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []UserAPITokenRow
	for rows.Next() {
		var t UserAPITokenRow
		var createdAt time.Time
		var lastUsedAt, expiresAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &createdAt, &lastUsedAt, &expiresAt, &t.Revoked); err != nil {
			log.Printf("Error scanning API token row: %v", err)
			continue
		}
		t.CreatedAt = createdAt.Format(time.RFC3339)
		if lastUsedAt.Valid {
			s := lastUsedAt.Time.Format(time.RFC3339)
			t.LastUsedAt = &s
		}
		if expiresAt.Valid {
			s := expiresAt.Time.Format(time.RFC3339)
			t.ExpiresAt = &s
		}
		results = append(results, t)
	}
	return results, nil
}

// RevokeUserAPIToken revokes a token. The userID parameter is used for ownership
// validation — pass empty string to skip the check (admin use).
func (db *DB) RevokeUserAPIToken(tokenID, userID string) error {
	var result sql.Result
	var err error
	if userID != "" {
		result, err = db.Exec(`UPDATE user_api_tokens SET revoked = TRUE WHERE id = $1 AND user_id = $2`, tokenID, userID)
	} else {
		result, err = db.Exec(`UPDATE user_api_tokens SET revoked = TRUE WHERE id = $1`, tokenID)
	}
	if err != nil {
		return fmt.Errorf("error revoking API token: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("token not found or not owned by user")
	}
	return nil
}

// ============================================================================
// User-Scoped Analytics
// ============================================================================

// GetRoutingDecisionsFiltered returns paginated routing decisions, optionally filtered by user.
// Pass userID="" to return all users (admin view).
func (db *DB) GetRoutingDecisionsFiltered(start, end time.Time, limit, offset int, userID string) ([]RoutingDecisionRow, int, error) {
	var total int
	if userID != "" {
		err := db.QueryRow(`SELECT COUNT(*) FROM routing_decisions WHERE timestamp >= $1 AND timestamp <= $2 AND user_id = $3`, start, end, userID).Scan(&total)
		if err != nil {
			return nil, 0, fmt.Errorf("error counting routing decisions: %w", err)
		}
	} else {
		err := db.QueryRow(`SELECT COUNT(*) FROM routing_decisions WHERE timestamp >= $1 AND timestamp <= $2`, start, end).Scan(&total)
		if err != nil {
			return nil, 0, fmt.Errorf("error counting routing decisions: %w", err)
		}
	}

	var query string
	var args []interface{}
	if userID != "" {
		query = `
			SELECT id, timestamp, prompt_snippet, full_prompt, analyzed_tags, tag_relevance,
			       selected_model, provider, routing_reason, evaluator_model,
			       evaluation_time_ms, cache_hit, latency_ms, status, COALESCE(user_id, 'anonymous')
			FROM routing_decisions
			WHERE timestamp >= $1 AND timestamp <= $2 AND user_id = $3
			ORDER BY timestamp DESC
			LIMIT $4 OFFSET $5`
		args = []interface{}{start, end, userID, limit, offset}
	} else {
		query = `
			SELECT id, timestamp, prompt_snippet, full_prompt, analyzed_tags, tag_relevance,
			       selected_model, provider, routing_reason, evaluator_model,
			       evaluation_time_ms, cache_hit, latency_ms, status, COALESCE(user_id, 'anonymous')
			FROM routing_decisions
			WHERE timestamp >= $1 AND timestamp <= $2
			ORDER BY timestamp DESC
			LIMIT $3 OFFSET $4`
		args = []interface{}{start, end, limit, offset}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("error querying routing decisions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []RoutingDecisionRow
	for rows.Next() {
		var rd RoutingDecisionRow
		var ts time.Time
		var tagsJSON, relevanceJSON []byte
		if err := rows.Scan(&rd.ID, &ts, &rd.PromptSnippet, &rd.FullPrompt,
			&tagsJSON, &relevanceJSON, &rd.SelectedModel, &rd.Provider,
			&rd.RoutingReason, &rd.EvaluatorModel, &rd.EvaluationTimeMs,
			&rd.CacheHit, &rd.LatencyMs, &rd.Status, &rd.UserID); err != nil {
			log.Printf("Error scanning routing decision row: %v", err)
			continue
		}
		rd.Timestamp = ts.Format(time.RFC3339)
		_ = json.Unmarshal(tagsJSON, &rd.AnalyzedTags)
		if rd.AnalyzedTags == nil {
			rd.AnalyzedTags = []string{}
		}
		rd.TagRelevance = make(map[string]float64)
		_ = json.Unmarshal(relevanceJSON, &rd.TagRelevance)
		results = append(results, rd)
	}
	return results, total, nil
}

// userFilterSQL returns a WHERE clause fragment and args for optional user filtering.
func userFilterSQL(start, end time.Time, userID string) (string, []interface{}) {
	if userID != "" {
		return "WHERE timestamp >= $1 AND timestamp <= $2 AND user_id = $3", []interface{}{start, end, userID}
	}
	return "WHERE timestamp >= $1 AND timestamp <= $2", []interface{}{start, end}
}

// GetKpiSummaryFiltered returns aggregate KPI metrics, optionally filtered by user.
func (db *DB) GetKpiSummaryFiltered(start, end time.Time, userID string) (KpiSummary, error) {
	where, args := userFilterSQL(start, end, userID)
	var kpi KpiSummary
	var avgLatency sql.NullFloat64

	err := db.QueryRow(fmt.Sprintf(`
		SELECT COUNT(*),
		       COALESCE(AVG(latency_ms), 0),
		       COALESCE(SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0), 0)
		FROM routing_decisions %s
	`, where), args...).Scan(&kpi.TotalRequests, &avgLatency, &kpi.ErrorRate)
	if err != nil {
		return kpi, fmt.Errorf("error querying KPI summary: %w", err)
	}
	kpi.AvgLatencyMs = int64(avgLatency.Float64)

	var mostUsed sql.NullString
	err = db.QueryRow(fmt.Sprintf(`
		SELECT selected_model FROM routing_decisions %s
		GROUP BY selected_model ORDER BY COUNT(*) DESC LIMIT 1
	`, where), args...).Scan(&mostUsed)
	if err != nil && err != sql.ErrNoRows {
		return kpi, fmt.Errorf("error querying most used model: %w", err)
	}
	if mostUsed.Valid {
		kpi.MostUsedModel = mostUsed.String
	}

	return kpi, nil
}

// GetModelUsageFiltered returns request counts per model, optionally filtered by user.
func (db *DB) GetModelUsageFiltered(start, end time.Time, userID string) ([]ModelUsageRow, error) {
	where, args := userFilterSQL(start, end, userID)
	rows, err := db.Query(fmt.Sprintf(`
		SELECT selected_model, COUNT(*) as cnt
		FROM routing_decisions %s
		GROUP BY selected_model
		ORDER BY cnt DESC
	`, where), args...)
	if err != nil {
		return nil, fmt.Errorf("error querying model usage: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []ModelUsageRow
	var total int64
	for rows.Next() {
		var r ModelUsageRow
		if err := rows.Scan(&r.ModelName, &r.RequestCount); err != nil {
			log.Printf("Error scanning model usage row: %v", err)
			continue
		}
		total += r.RequestCount
		results = append(results, r)
	}
	for i := range results {
		if total > 0 {
			results[i].Percentage = float64(results[i].RequestCount) * 100 / float64(total)
		}
	}
	return results, nil
}

// GetLatencyPerModelFiltered returns average latency per model, optionally filtered by user.
func (db *DB) GetLatencyPerModelFiltered(start, end time.Time, userID string) ([]LatencyPerModelRow, error) {
	where, args := userFilterSQL(start, end, userID)
	rows, err := db.Query(fmt.Sprintf(`
		SELECT selected_model, AVG(latency_ms)
		FROM routing_decisions %s
		GROUP BY selected_model
		ORDER BY AVG(latency_ms) ASC
	`, where), args...)
	if err != nil {
		return nil, fmt.Errorf("error querying latency per model: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []LatencyPerModelRow
	for rows.Next() {
		var r LatencyPerModelRow
		var avg float64
		if err := rows.Scan(&r.ModelName, &avg); err != nil {
			log.Printf("Error scanning latency per model row: %v", err)
			continue
		}
		r.AvgLatencyMs = int64(avg)
		results = append(results, r)
	}
	return results, nil
}

// GetLatencyTimeSeriesFiltered returns time-bucketed latency data, optionally filtered by user.
func (db *DB) GetLatencyTimeSeriesFiltered(start, end time.Time, userID string) ([]LatencyTimeSeriesRow, error) {
	duration := end.Sub(start)
	bucket := "hour"
	if duration <= 2*time.Hour {
		bucket = "minute"
	} else if duration > 48*time.Hour {
		bucket = "day"
	}

	where, args := userFilterSQL(start, end, userID)
	rows, err := db.Query(fmt.Sprintf(`
		SELECT date_trunc('%s', timestamp) as bucket,
		       AVG(latency_ms),
		       PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms)
		FROM routing_decisions %s
		GROUP BY bucket
		ORDER BY bucket ASC
	`, bucket, where), args...)
	if err != nil {
		return nil, fmt.Errorf("error querying latency time series: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []LatencyTimeSeriesRow
	for rows.Next() {
		var r LatencyTimeSeriesRow
		var ts time.Time
		var avg, p95 float64
		if err := rows.Scan(&ts, &avg, &p95); err != nil {
			log.Printf("Error scanning latency time series row: %v", err)
			continue
		}
		r.Timestamp = ts.Format(time.RFC3339)
		r.AvgLatencyMs = int64(avg)
		r.P95LatencyMs = int64(p95)
		results = append(results, r)
	}
	return results, nil
}

// GetLatencyPercentilesFiltered returns P50/P95/P99 latency per model, optionally filtered by user.
func (db *DB) GetLatencyPercentilesFiltered(start, end time.Time, userID string) ([]LatencyPercentilesRow, error) {
	where, args := userFilterSQL(start, end, userID)
	rows, err := db.Query(fmt.Sprintf(`
		SELECT selected_model,
		       PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY latency_ms),
		       PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms),
		       PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY latency_ms)
		FROM routing_decisions %s
		GROUP BY selected_model
		ORDER BY selected_model
	`, where), args...)
	if err != nil {
		return nil, fmt.Errorf("error querying latency percentiles: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []LatencyPercentilesRow
	for rows.Next() {
		var r LatencyPercentilesRow
		var p50, p95, p99 float64
		if err := rows.Scan(&r.ModelName, &p50, &p95, &p99); err != nil {
			log.Printf("Error scanning latency percentiles row: %v", err)
			continue
		}
		r.P50Ms = int64(p50)
		r.P95Ms = int64(p95)
		r.P99Ms = int64(p99)
		results = append(results, r)
	}
	return results, nil
}
