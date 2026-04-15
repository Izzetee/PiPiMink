// Define interfaces that can be used for mocking
package server

import (
	"time"

	"PiPiMink/internal/benchmark"
	"PiPiMink/internal/config"
	"PiPiMink/internal/database"
	"PiPiMink/internal/models"
)

// DatabaseInterface defines the database operations needed by the server
type DatabaseInterface interface {
	GetAllModels() (map[string]map[string]interface{}, error)
	SaveModel(name, source, tags string, enabled bool, hasReasoning bool) error
	RegisterDiscoveredModel(name, source string) error
	HasModels() (bool, error)
	EnableModel(name, source string, enabled bool) error
	UpdateModelReasoning(name, source string, hasReasoning bool) error
	DeleteModel(name, source string) error
	ResetModel(name, source string) error
	DeleteModelFull(name, source string) error
	Close() error

	// Benchmark result methods
	SaveBenchmarkResult(modelName, source, category, taskID string, score float64, latencyMs int64, judgeModel, response string) error
	GetBenchmarkScores(modelName, source string) (map[string]float64, error)
	GetBenchmarkResults(modelName, source string) ([]database.BenchmarkResult, error)
	GetAllBenchmarkScores() (map[string]map[string]float64, error)
	GetAllModelLatencies() (map[string]int64, error)

	// Benchmark task config methods
	SeedBenchmarkTasksIfEmpty(defaults []benchmark.BenchmarkTaskConfig) error
	GetBenchmarkTaskConfigs() ([]benchmark.BenchmarkTaskConfig, error)
	UpsertBenchmarkTaskConfig(cfg benchmark.BenchmarkTaskConfig) error
	DeleteBenchmarkTaskConfig(taskID string, defaultCfgs []benchmark.BenchmarkTaskConfig) error

	// System prompt methods
	SeedSystemPromptsIfEmpty(defaults map[string]string) error
	GetSystemPrompt(key string) (string, bool, error)
	SetSystemPrompt(key, value, description string) error
	GetAllSystemPrompts() (map[string]database.SystemPromptRow, error)

	// Analytics methods
	SaveRoutingDecision(rd database.RoutingDecisionRow) error
	GetRoutingDecisions(start, end time.Time, limit, offset int) ([]database.RoutingDecisionRow, int, error)
	GetKpiSummary(start, end time.Time) (database.KpiSummary, error)
	GetModelUsage(start, end time.Time) ([]database.ModelUsageRow, error)
	GetLatencyPerModel(start, end time.Time) ([]database.LatencyPerModelRow, error)
	GetLatencyTimeSeries(start, end time.Time) ([]database.LatencyTimeSeriesRow, error)
	GetLatencyPercentiles(start, end time.Time) ([]database.LatencyPercentilesRow, error)

	// User-scoped analytics methods
	GetRoutingDecisionsFiltered(start, end time.Time, limit, offset int, userID string) ([]database.RoutingDecisionRow, int, error)
	GetKpiSummaryFiltered(start, end time.Time, userID string) (database.KpiSummary, error)
	GetModelUsageFiltered(start, end time.Time, userID string) ([]database.ModelUsageRow, error)
	GetLatencyPerModelFiltered(start, end time.Time, userID string) ([]database.LatencyPerModelRow, error)
	GetLatencyTimeSeriesFiltered(start, end time.Time, userID string) ([]database.LatencyTimeSeriesRow, error)
	GetLatencyPercentilesFiltered(start, end time.Time, userID string) ([]database.LatencyPercentilesRow, error)

	// User API token methods
	CreateUserAPIToken(userID, name string) (id string, tokenPlaintext string, err error)
	GetUserByAPIToken(tokenHash string) (*database.UserRow, error)
	ListUserAPITokens(userID string) ([]database.UserAPITokenRow, error)
	RevokeUserAPIToken(tokenID, userID string) error

	// Auth & Users methods
	SeedAuthProvidersIfEmpty() error
	GetAuthProviders() ([]database.AuthProviderRow, error)
	SaveAuthProvider(p database.AuthProviderRow) error
	GetUsers() ([]database.UserRow, error)
	GetUserByEmail(email string) (*database.UserRow, error)
	UpsertUser(u database.UserRow) error
	ChangeUserRole(userID, role string) error
	DeleteUser(userID string) error
	GetGroups() ([]database.GroupRow, error)
	SaveGroup(g database.GroupRow) error
	ChangeGroupRole(groupID, role string) error
	SaveRoutingRule(groupID string, r database.RoutingRuleRow) error
	DeleteRoutingRule(groupID, ruleID string) error
	GetAuditLog() ([]database.AuditEntryRow, error)
	SaveAuditEntry(e database.AuditEntryRow) error
}

// LLMInterface defines the LLM client operations needed by the server
type LLMInterface interface {
	ChatWithModel(modelInfo models.ModelInfo, model string, messages []map[string]interface{}) (string, error)
	DecideModelBasedOnCapabilities(message string, availableModels map[string]models.ModelInfo) (models.RoutingResult, error)
	GetModelTags(model string, p config.ProviderConfig) (string, bool, bool, error)
	GetModelsByProvider(p config.ProviderConfig) ([]string, error)
	IsLocalServerUsingMLX() bool
	UpdateTaggingPrompts(systemPrompt, userPrompt, userNoSysPrompt string)
}
