package database

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockDB(t *testing.T) (*DB, sqlmock.Sqlmock) {
	t.Helper()
	sdb, mock, err := sqlmock.New()
	require.NoError(t, err)
	return &DB{DB: sdb}, mock
}

func TestSaveModel_EmptyTagsDefaultsToEmptyJSON(t *testing.T) {
	db, mock := newMockDB(t)
	defer func() { _ = db.Close() }()

	mock.ExpectExec(regexp.QuoteMeta(`
	INSERT INTO models (name, source, tags, enabled, has_reasoning, updated_at)
	VALUES ($1, $2, $3::jsonb, $4, $5, NOW())
	ON CONFLICT (name, source) DO UPDATE
	SET tags = $3::jsonb, enabled = $4, has_reasoning = $5, updated_at = NOW();
	`)).
		WithArgs("model-a", "openai", "{}", true, false).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := db.SaveModel("model-a", "openai", "", true, false)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAllModels(t *testing.T) {
	db, mock := newMockDB(t)
	defer func() { _ = db.Close() }()

	now := time.Now().UTC()
	rows := sqlmock.NewRows([]string{"name", "source", "tags", "enabled", "has_reasoning", "updated_at"}).
		AddRow("o1-mini", "openai", `{"capabilities":["reasoning"]}`, true, true, now)

	mock.ExpectQuery(regexp.QuoteMeta(`
	SELECT name, source, tags, enabled, has_reasoning, updated_at
	FROM models
	`)).WillReturnRows(rows)

	models, err := db.GetAllModels()
	require.NoError(t, err)
	require.Len(t, models, 1)

	m := models["o1-mini"]
	assert.Equal(t, "openai", m["source"])
	assert.Equal(t, true, m["enabled"])
	assert.Equal(t, true, m["has_reasoning"])
	assert.NotEmpty(t, m["updated_at"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHasModelsSQLMock(t *testing.T) {
	db, mock := newMockDB(t)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM models`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	hasModels, err := db.HasModels()
	require.NoError(t, err)
	assert.True(t, hasModels)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEnableModelAndDeleteModel(t *testing.T) {
	db, mock := newMockDB(t)
	defer func() { _ = db.Close() }()

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE models
		SET enabled = $3, updated_at = NOW()
		WHERE name = $1 AND source = $2
	`)).WithArgs("gpt-4-turbo", "openai", false).WillReturnResult(sqlmock.NewResult(0, 1))

	err := db.EnableModel("gpt-4-turbo", "openai", false)
	require.NoError(t, err)

	mock.ExpectExec(regexp.QuoteMeta(`
		DELETE FROM models
		WHERE name = $1 AND source = $2
	`)).WithArgs("gpt-4-turbo", "openai").WillReturnResult(sqlmock.NewResult(0, 1))

	err = db.DeleteModel("gpt-4-turbo", "openai")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateModelReasoning(t *testing.T) {
	db, mock := newMockDB(t)
	defer func() { _ = db.Close() }()

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE models
		SET has_reasoning = $3, updated_at = NOW()
		WHERE name = $1 AND source = $2
	`)).WithArgs("o1-mini", "openai", true).WillReturnResult(sqlmock.NewResult(0, 1))

	err := db.UpdateModelReasoning("o1-mini", "openai", true)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMigrateExistingModelsReasoning(t *testing.T) {
	db, mock := newMockDB(t)
	defer func() { _ = db.Close() }()

	// Map iteration order is non-deterministic, so allow expectations in any order.
	mock.MatchExpectationsInOrder(false)

	now := time.Now().UTC()
	rows := sqlmock.NewRows([]string{"name", "source", "tags", "enabled", "has_reasoning", "updated_at"}).
		AddRow("o1-mini", "openai", `{"capabilities":["reasoning"]}`, true, false, now).
		AddRow("gpt-4-turbo", "openai", `{"capabilities":["general"]}`, true, false, now)

	mock.ExpectQuery(regexp.QuoteMeta(`
	SELECT name, source, tags, enabled, has_reasoning, updated_at
	FROM models
	`)).WillReturnRows(rows)

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE models
		SET has_reasoning = $3, updated_at = NOW()
		WHERE name = $1 AND source = $2
	`)).WithArgs("o1-mini", "openai", true).WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE models
		SET has_reasoning = $3, updated_at = NOW()
		WHERE name = $1 AND source = $2
	`)).WithArgs("gpt-4-turbo", "openai", false).WillReturnResult(sqlmock.NewResult(0, 1))

	err := db.MigrateExistingModelsReasoning()
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInitSchema_ColumnsAlreadyExist(t *testing.T) {
	db, mock := newMockDB(t)
	defer func() { _ = db.Close() }()

	mock.ExpectExec(regexp.QuoteMeta(`
	CREATE TABLE IF NOT EXISTS models (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		source TEXT NOT NULL,
		tags JSONB DEFAULT '{}'::jsonb,
		UNIQUE(name, source)
	);
	`)).WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS (
		SELECT 1 
		FROM information_schema.columns 
		WHERE table_name = 'models' 
		AND column_name = 'enabled'
	)`)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS (
		SELECT 1 
		FROM information_schema.columns 
		WHERE table_name = 'models' 
		AND column_name = 'updated_at'
	)`)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS (
		SELECT 1
		FROM information_schema.columns
		WHERE table_name = 'models'
		AND column_name = 'has_reasoning'
	)`)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	// benchmark schema
	mock.ExpectExec(regexp.QuoteMeta(`
	CREATE TABLE IF NOT EXISTS benchmark_results (
		id         SERIAL PRIMARY KEY,
		model_name TEXT NOT NULL,
		source     TEXT NOT NULL,
		category   TEXT NOT NULL,
		task_id    TEXT NOT NULL,
		score      FLOAT NOT NULL,
		latency_ms BIGINT NOT NULL DEFAULT 0,
		run_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`)).WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectExec(regexp.QuoteMeta(`CREATE INDEX IF NOT EXISTS idx_benchmark_results_model    ON benchmark_results(model_name, source);`)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectExec(regexp.QuoteMeta(`CREATE INDEX IF NOT EXISTS idx_benchmark_results_category ON benchmark_results(model_name, source, category);`)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectExec(regexp.QuoteMeta(`
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
	);`)).WillReturnResult(sqlmock.NewResult(0, 0))

	err := db.InitSchema()
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRegisterDiscoveredModel(t *testing.T) {
	db, mock := newMockDB(t)
	defer func() { _ = db.Close() }()

	mock.ExpectExec(regexp.QuoteMeta(`
	INSERT INTO models (name, source, tags, enabled, has_reasoning, updated_at)
	VALUES ($1, $2, '{}', false, false, NOW())
	ON CONFLICT (name, source) DO NOTHING;
	`)).WithArgs("gpt-4o", "openai").WillReturnResult(sqlmock.NewResult(1, 1))

	err := db.RegisterDiscoveredModel("gpt-4o", "openai")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAllModels_QueryError(t *testing.T) {
	db, mock := newMockDB(t)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta(`
	SELECT name, source, tags, enabled, has_reasoning, updated_at
	FROM models
	`)).WillReturnError(sql.ErrConnDone)

	_, err := db.GetAllModels()
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
