// Define interfaces that can be used for mocking
package server

import (
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
	Close() error

	// Benchmark result methods
	SaveBenchmarkResult(modelName, source, category, taskID string, score float64, latencyMs int64) error
	GetBenchmarkScores(modelName, source string) (map[string]float64, error)
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
}

// LLMInterface defines the LLM client operations needed by the server
type LLMInterface interface {
	ChatWithModel(modelInfo models.ModelInfo, model string, messages []map[string]interface{}) (string, error)
	DecideModelBasedOnCapabilities(message string, availableModels map[string]models.ModelInfo) (string, error)
	GetModelTags(model string, p config.ProviderConfig) (string, bool, bool, error)
	GetModelsByProvider(p config.ProviderConfig) ([]string, error)
	IsLocalServerUsingMLX() bool
	UpdateTaggingPrompts(systemPrompt, userPrompt, userNoSysPrompt string)
}
