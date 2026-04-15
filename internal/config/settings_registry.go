package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// SettingType determines the rendered control in the UI.
type SettingType string

const (
	SettingTypeText           SettingType = "text"
	SettingTypeNumber         SettingType = "number"
	SettingTypeToggle         SettingType = "toggle"
	SettingTypeDuration       SettingType = "duration"
	SettingTypeURL            SettingType = "url"
	SettingTypeSecret         SettingType = "secret"
	SettingTypeProviderSelect SettingType = "provider-select"
	SettingTypeModelSelect    SettingType = "model-select"
)

// SettingCategory groups settings into UI tabs.
type SettingCategory string

const (
	CategoryRouting       SettingCategory = "routing"
	CategoryCache         SettingCategory = "cache"
	CategoryDatabase      SettingCategory = "database"
	CategoryServer        SettingCategory = "server"
	CategoryBenchmarking  SettingCategory = "benchmarking"
	CategoryObservability SettingCategory = "observability"
)

// SettingValidation holds optional constraints for a setting.
type SettingValidation struct {
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
	Step    *float64 `json:"step,omitempty"`
	Pattern string   `json:"pattern,omitempty"`
}

// SettingDef describes a single system configuration value.
type SettingDef struct {
	Key         string             `json:"key"`
	Category    SettingCategory    `json:"category"`
	Type        SettingType        `json:"type"`
	Label       string             `json:"label"`
	Description string             `json:"description"`
	Required    bool               `json:"required"`
	DependsOn   string             `json:"dependsOn,omitempty"`
	Validation  *SettingValidation `json:"validation,omitempty"`
}

func floatPtr(f float64) *float64 { return &f }

// Registry defines all manageable settings with their metadata.
var Registry = []SettingDef{
	// ── Routing ──
	{
		Key: "MODEL_SELECTION_PROVIDER", Category: CategoryRouting, Type: SettingTypeProviderSelect,
		Label: "Routing Provider", Description: "The provider used to make routing decisions. Must match a name in your provider configuration.",
		Required: true,
	},
	{
		Key: "MODEL_SELECTION_MODEL", Category: CategoryRouting, Type: SettingTypeModelSelect,
		Label: "Routing Model", Description: "The model within the selected provider used as the meta-router for intelligent model selection.",
		Required: true, DependsOn: "MODEL_SELECTION_PROVIDER",
	},
	{
		Key: "DEFAULT_CHAT_MODEL", Category: CategoryRouting, Type: SettingTypeModelSelect,
		Label: "Fallback Model", Description: "The model used when routing fails or no suitable model is found.",
		Required: true, DependsOn: "MODEL_SELECTION_PROVIDER",
	},

	// ── Cache ──
	{
		Key: "SELECTION_CACHE_ENABLED", Category: CategoryCache, Type: SettingTypeToggle,
		Label: "Enable Routing Cache", Description: "Cache routing decisions to reduce latency and LLM calls for repeated prompt patterns.",
	},
	{
		Key: "SELECTION_CACHE_TTL", Category: CategoryCache, Type: SettingTypeDuration,
		Label: "Cache TTL", Description: "How long a cached routing decision remains valid. Go duration format (e.g., 30s, 2m, 1h).",
		Validation: &SettingValidation{Pattern: `^\d+[smh]$`},
	},
	{
		Key: "SELECTION_CACHE_MAX_ENTRIES", Category: CategoryCache, Type: SettingTypeNumber,
		Label: "Max Cache Entries", Description: "Maximum number of routing decisions to keep in cache. Oldest entries are evicted first.",
		Validation: &SettingValidation{Min: floatPtr(10), Max: floatPtr(100000)},
	},
	{
		Key: "SELECTION_CACHE_STATS_LOG_INTERVAL", Category: CategoryCache, Type: SettingTypeDuration,
		Label: "Stats Log Interval", Description: "How often cache hit/miss statistics are logged. Go duration format.",
		Validation: &SettingValidation{Pattern: `^\d+[smh]$`},
	},

	// ── Database ──
	{
		Key: "DATABASE_URL", Category: CategoryDatabase, Type: SettingTypeURL,
		Label: "Database URL", Description: "PostgreSQL connection string including credentials, host, port, database name, and SSL mode.",
		Required: true,
	},
	{
		Key: "DATABASE_MAX_CONNECTIONS", Category: CategoryDatabase, Type: SettingTypeNumber,
		Label: "Max Connections", Description: "Maximum number of open connections to the database.",
		Validation: &SettingValidation{Min: floatPtr(1), Max: floatPtr(500)},
	},
	{
		Key: "DATABASE_MAX_IDLE_CONNECTIONS", Category: CategoryDatabase, Type: SettingTypeNumber,
		Label: "Max Idle Connections", Description: "Maximum number of idle connections retained in the pool.",
		Validation: &SettingValidation{Min: floatPtr(0), Max: floatPtr(100)},
	},
	{
		Key: "DATABASE_CONNECTION_MAX_LIFETIME", Category: CategoryDatabase, Type: SettingTypeDuration,
		Label: "Connection Max Lifetime", Description: "Maximum time a connection may be reused before being closed. Go duration format.",
		Validation: &SettingValidation{Pattern: `^\d+[smh]$`},
	},

	// ── Server ──
	{
		Key: "ADMIN_API_KEY", Category: CategoryServer, Type: SettingTypeSecret,
		Label: "Admin API Key", Description: "API key required for admin endpoints. Keep this secret and rotate regularly.",
		Required: true,
	},
	{
		Key: "PORT", Category: CategoryServer, Type: SettingTypeNumber,
		Label: "Server Port", Description: "The port PiPiMink listens on for incoming requests.",
		Required: true, Validation: &SettingValidation{Min: floatPtr(1), Max: floatPtr(65535)},
	},
	{
		Key: "REQUIRE_AUTH_FOR_CHAT", Category: CategoryServer, Type: SettingTypeToggle,
		Label: "Require Auth for Chat", Description: "When enabled, chat and API endpoints require authentication (Bearer token or X-API-Key). When disabled, unauthenticated requests are allowed.",
	},

	// ── Benchmarking ──
	{
		Key: "BENCHMARK_ENABLED", Category: CategoryBenchmarking, Type: SettingTypeToggle,
		Label: "Enable Benchmarks", Description: "Enable benchmark support. Required to use the POST /models/benchmark endpoint.",
	},
	{
		Key: "BENCHMARK_SCHEDULE_ENABLED", Category: CategoryBenchmarking, Type: SettingTypeToggle,
		Label: "Scheduled Benchmarks", Description: "Automatically run benchmarks on a recurring schedule.",
	},
	{
		Key: "BENCHMARK_SCHEDULE_INTERVAL", Category: CategoryBenchmarking, Type: SettingTypeDuration,
		Label: "Schedule Interval", Description: "How often scheduled benchmarks run. Go duration format (e.g., 6h, 12h, 24h).",
		Validation: &SettingValidation{Pattern: `^\d+[smh]$`},
	},
	{
		Key: "BENCHMARK_JUDGE_PROVIDER", Category: CategoryBenchmarking, Type: SettingTypeProviderSelect,
		Label: "Judge Provider", Description: "The provider for the LLM judge used to score subjective benchmark tasks. Falls back to the routing provider if not set.",
	},
	{
		Key: "BENCHMARK_JUDGE_MODEL", Category: CategoryBenchmarking, Type: SettingTypeModelSelect,
		Label: "Judge Model", Description: "The model used as the LLM judge for subjective tasks. Falls back to the routing model if not set.",
		DependsOn: "BENCHMARK_JUDGE_PROVIDER",
	},
	{
		Key: "BENCHMARK_CONCURRENCY", Category: CategoryBenchmarking, Type: SettingTypeNumber,
		Label: "Concurrency", Description: "Maximum number of models benchmarked in parallel.",
		Validation: &SettingValidation{Min: floatPtr(1), Max: floatPtr(20)},
	},

	// ── Observability ──
	{
		Key: "OTEL_ENABLED", Category: CategoryObservability, Type: SettingTypeToggle,
		Label: "Enable OpenTelemetry", Description: "Enable OpenTelemetry tracing and metrics export.",
	},
	{
		Key: "OTEL_SERVICE_NAME", Category: CategoryObservability, Type: SettingTypeText,
		Label: "Service Name", Description: "The service name reported in traces and metrics.",
	},
	{
		Key: "OTEL_EXPORTER_OTLP_ENDPOINT", Category: CategoryObservability, Type: SettingTypeText,
		Label: "OTLP Endpoint", Description: "The OpenTelemetry collector endpoint for trace and metric export.",
	},
	{
		Key: "OTEL_EXPORTER_OTLP_INSECURE", Category: CategoryObservability, Type: SettingTypeToggle,
		Label: "Insecure Connection", Description: "Use an insecure (non-TLS) connection to the OTLP endpoint. Enable for local collectors.",
	},
	{
		Key: "OTEL_TRACE_SAMPLE_RATIO", Category: CategoryObservability, Type: SettingTypeNumber,
		Label: "Trace Sample Ratio", Description: "Fraction of traces to sample (0.0 to 1.0). Set to 1.0 to capture all traces, lower values to reduce volume.",
		Validation: &SettingValidation{Min: floatPtr(0), Max: floatPtr(1), Step: floatPtr(0.1)},
	},
}

// RegistryByCategory groups the registry entries by category.
func RegistryByCategory() map[SettingCategory][]SettingDef {
	m := make(map[SettingCategory][]SettingDef)
	for _, def := range Registry {
		m[def.Category] = append(m[def.Category], def)
	}
	return m
}

// GetSettingValue reads the current value of a setting from the Config struct.
func GetSettingValue(cfg *Config, key string) interface{} {
	switch key {
	// Routing
	case "MODEL_SELECTION_PROVIDER":
		return cfg.ModelSelectionProvider
	case "MODEL_SELECTION_MODEL":
		return cfg.ModelSelectionModel
	case "DEFAULT_CHAT_MODEL":
		return cfg.DefaultChatModel
	// Cache
	case "SELECTION_CACHE_ENABLED":
		return cfg.SelectionCacheEnabled
	case "SELECTION_CACHE_TTL":
		return cfg.SelectionCacheTTL.String()
	case "SELECTION_CACHE_MAX_ENTRIES":
		return cfg.SelectionCacheMaxEntries
	case "SELECTION_CACHE_STATS_LOG_INTERVAL":
		return cfg.SelectionCacheStatsLogInterval.String()
	// Database
	case "DATABASE_URL":
		return cfg.DatabaseURL
	case "DATABASE_MAX_CONNECTIONS":
		return cfg.DatabaseMaxConnections
	case "DATABASE_MAX_IDLE_CONNECTIONS":
		return cfg.DatabaseMaxIdleConnections
	case "DATABASE_CONNECTION_MAX_LIFETIME":
		return cfg.DatabaseConnectionMaxLifetime.String()
	// Server
	case "ADMIN_API_KEY":
		return cfg.AdminAPIKey
	case "PORT":
		return cfg.Port
	case "REQUIRE_AUTH_FOR_CHAT":
		return cfg.RequireAuthForChat
	// Benchmarking
	case "BENCHMARK_ENABLED":
		return cfg.BenchmarkEnabled
	case "BENCHMARK_SCHEDULE_ENABLED":
		return cfg.BenchmarkScheduleEnabled
	case "BENCHMARK_SCHEDULE_INTERVAL":
		return cfg.BenchmarkScheduleInterval.String()
	case "BENCHMARK_JUDGE_PROVIDER":
		return cfg.BenchmarkJudgeProvider
	case "BENCHMARK_JUDGE_MODEL":
		return cfg.BenchmarkJudgeModel
	case "BENCHMARK_CONCURRENCY":
		return cfg.BenchmarkConcurrency
	// Observability
	case "OTEL_ENABLED":
		return cfg.OTelEnabled
	case "OTEL_SERVICE_NAME":
		return cfg.OTelServiceName
	case "OTEL_EXPORTER_OTLP_ENDPOINT":
		return cfg.OTelExporterOTLPEndpoint
	case "OTEL_EXPORTER_OTLP_INSECURE":
		return cfg.OTelExporterOTLPInsecure
	case "OTEL_TRACE_SAMPLE_RATIO":
		return cfg.OTelTraceSampleRatio
	default:
		return nil
	}
}

// SetSettingValue updates a Config field in memory and sets the corresponding env var.
func SetSettingValue(cfg *Config, key string, value interface{}) error {
	strVal := fmt.Sprintf("%v", value)

	switch key {
	// Routing
	case "MODEL_SELECTION_PROVIDER":
		cfg.ModelSelectionProvider = strVal
	case "MODEL_SELECTION_MODEL":
		cfg.ModelSelectionModel = strVal
	case "DEFAULT_CHAT_MODEL":
		cfg.DefaultChatModel = strVal
	// Cache
	case "SELECTION_CACHE_ENABLED":
		b, err := parseBoolValue(value)
		if err != nil {
			return fmt.Errorf("invalid boolean for %s: %w", key, err)
		}
		cfg.SelectionCacheEnabled = b
	case "SELECTION_CACHE_TTL":
		d, err := time.ParseDuration(strVal)
		if err != nil {
			return fmt.Errorf("invalid duration for %s: %w", key, err)
		}
		cfg.SelectionCacheTTL = d
	case "SELECTION_CACHE_MAX_ENTRIES":
		n, err := parseIntValue(value)
		if err != nil {
			return fmt.Errorf("invalid integer for %s: %w", key, err)
		}
		cfg.SelectionCacheMaxEntries = n
	case "SELECTION_CACHE_STATS_LOG_INTERVAL":
		d, err := time.ParseDuration(strVal)
		if err != nil {
			return fmt.Errorf("invalid duration for %s: %w", key, err)
		}
		cfg.SelectionCacheStatsLogInterval = d
	// Database
	case "DATABASE_URL":
		cfg.DatabaseURL = strVal
	case "DATABASE_MAX_CONNECTIONS":
		n, err := parseIntValue(value)
		if err != nil {
			return fmt.Errorf("invalid integer for %s: %w", key, err)
		}
		cfg.DatabaseMaxConnections = n
	case "DATABASE_MAX_IDLE_CONNECTIONS":
		n, err := parseIntValue(value)
		if err != nil {
			return fmt.Errorf("invalid integer for %s: %w", key, err)
		}
		cfg.DatabaseMaxIdleConnections = n
	case "DATABASE_CONNECTION_MAX_LIFETIME":
		d, err := time.ParseDuration(strVal)
		if err != nil {
			return fmt.Errorf("invalid duration for %s: %w", key, err)
		}
		cfg.DatabaseConnectionMaxLifetime = d
	// Server
	case "ADMIN_API_KEY":
		cfg.AdminAPIKey = strVal
	case "PORT":
		cfg.Port = strVal
	case "REQUIRE_AUTH_FOR_CHAT":
		b, err := parseBoolValue(value)
		if err != nil {
			return fmt.Errorf("invalid boolean for %s: %w", key, err)
		}
		cfg.RequireAuthForChat = b
	// Benchmarking
	case "BENCHMARK_ENABLED":
		b, err := parseBoolValue(value)
		if err != nil {
			return fmt.Errorf("invalid boolean for %s: %w", key, err)
		}
		cfg.BenchmarkEnabled = b
	case "BENCHMARK_SCHEDULE_ENABLED":
		b, err := parseBoolValue(value)
		if err != nil {
			return fmt.Errorf("invalid boolean for %s: %w", key, err)
		}
		cfg.BenchmarkScheduleEnabled = b
	case "BENCHMARK_SCHEDULE_INTERVAL":
		d, err := time.ParseDuration(strVal)
		if err != nil {
			return fmt.Errorf("invalid duration for %s: %w", key, err)
		}
		cfg.BenchmarkScheduleInterval = d
	case "BENCHMARK_JUDGE_PROVIDER":
		cfg.BenchmarkJudgeProvider = strVal
	case "BENCHMARK_JUDGE_MODEL":
		cfg.BenchmarkJudgeModel = strVal
	case "BENCHMARK_CONCURRENCY":
		n, err := parseIntValue(value)
		if err != nil {
			return fmt.Errorf("invalid integer for %s: %w", key, err)
		}
		cfg.BenchmarkConcurrency = n
	// Observability
	case "OTEL_ENABLED":
		b, err := parseBoolValue(value)
		if err != nil {
			return fmt.Errorf("invalid boolean for %s: %w", key, err)
		}
		cfg.OTelEnabled = b
	case "OTEL_SERVICE_NAME":
		cfg.OTelServiceName = strVal
	case "OTEL_EXPORTER_OTLP_ENDPOINT":
		cfg.OTelExporterOTLPEndpoint = strVal
	case "OTEL_EXPORTER_OTLP_INSECURE":
		b, err := parseBoolValue(value)
		if err != nil {
			return fmt.Errorf("invalid boolean for %s: %w", key, err)
		}
		cfg.OTelExporterOTLPInsecure = b
	case "OTEL_TRACE_SAMPLE_RATIO":
		f, err := parseFloat64Value(value)
		if err != nil {
			return fmt.Errorf("invalid float for %s: %w", key, err)
		}
		cfg.OTelTraceSampleRatio = f
	default:
		return fmt.Errorf("unknown setting key: %s", key)
	}

	_ = os.Setenv(key, strVal)
	return nil
}

// ValueToEnvString converts a setting value to its .env string representation.
func ValueToEnvString(value interface{}) string {
	switch v := value.(type) {
	case bool:
		return strconv.FormatBool(v)
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	default:
		return fmt.Sprintf("%v", value)
	}
}

// FindSettingDef looks up a SettingDef by key, returning nil if not found.
func FindSettingDef(key string) *SettingDef {
	for i := range Registry {
		if Registry[i].Key == key {
			return &Registry[i]
		}
	}
	return nil
}

func parseBoolValue(v interface{}) (bool, error) {
	switch b := v.(type) {
	case bool:
		return b, nil
	case string:
		return strconv.ParseBool(b)
	case float64:
		return b != 0, nil
	default:
		return false, fmt.Errorf("cannot convert %T to bool", v)
	}
}

func parseIntValue(v interface{}) (int, error) {
	switch n := v.(type) {
	case float64:
		return int(n), nil
	case int:
		return n, nil
	case string:
		return strconv.Atoi(n)
	default:
		return 0, fmt.Errorf("cannot convert %T to int", v)
	}
}

func parseFloat64Value(v interface{}) (float64, error) {
	switch f := v.(type) {
	case float64:
		return f, nil
	case int:
		return float64(f), nil
	case string:
		return strconv.ParseFloat(f, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}
