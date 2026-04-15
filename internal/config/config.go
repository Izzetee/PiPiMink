package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// ProviderType defines the API format a provider speaks.
const (
	ProviderTypeOpenAICompatible = "openai-compatible" // OpenAI, Gemini, OpenRouter, local, Azure AI Foundry
	ProviderTypeAnthropic        = "anthropic"         // Anthropic Claude models
)

// ModelConfig holds per-model overrides within a provider entry.
// Used to configure multiple models on the same provider (e.g. Azure AI Foundry)
// where each model may have its own API key, endpoint path, or even API type.
type ModelConfig struct {
	Name      string `json:"name"`                  // Model name as returned by the provider
	APIKeyEnv string `json:"api_key_env,omitempty"` // Env var holding this model's API key (overrides provider-level key)
	APIKey    string `json:"-"`                     // Resolved at load time
	ChatPath  string `json:"chat_path,omitempty"`   // Overrides provider-level chat_path for this model
	Type      string `json:"type,omitempty"`        // Overrides provider type for this model (e.g. "anthropic")
	BaseURL   string `json:"base_url,omitempty"`    // Overrides provider base_url for this model
	Enabled   bool   `json:"enabled"`               // Whether this model config is active (default true)
}

// ProviderConfig holds the configuration for a single LLM provider endpoint.
// Providers are loaded from providers.json at startup.
type ProviderConfig struct {
	Name             string        `json:"name"`                    // Unique identifier, stored as "source" in the model registry
	Type             string        `json:"type"`                    // "openai-compatible" or "anthropic"
	BaseURL          string        `json:"base_url"`                // Base URL for the provider
	ChatPath         string        `json:"chat_path,omitempty"`     // Override for the chat completions path (default: /v1/chat/completions). May include query params.
	APIKeyEnv        string        `json:"api_key_env"`             // Name of the env var holding the API key
	APIKey           string        `json:"-"`                       // Resolved at load time, never serialised
	TimeoutStr       string        `json:"timeout"`                 // Duration string, e.g. "2m"
	Timeout          time.Duration `json:"-"`                       // Resolved at load time
	RateLimitSeconds int           `json:"rate_limit_seconds"`      // Minimum seconds between requests (0 = unlimited)
	Models           []string      `json:"models"`                  // Simple static model list (no per-model overrides)
	ModelConfigs     []ModelConfig `json:"model_configs,omitempty"` // Per-model overrides; drives the model list when set
	Enabled          bool          `json:"enabled"`                 // Whether this provider is active (default true)
}

// ModelNames returns the list of model names for this provider.
// ModelConfigs takes precedence over Models.
func (p ProviderConfig) ModelNames() []string {
	if len(p.ModelConfigs) > 0 {
		names := make([]string, len(p.ModelConfigs))
		for i, mc := range p.ModelConfigs {
			names[i] = mc.Name
		}
		return names
	}
	return p.Models
}

// ForModel returns a copy of the ProviderConfig with any per-model overrides applied.
// Call this before making API calls so model-specific keys and paths are used.
func (p ProviderConfig) ForModel(name string) ProviderConfig {
	for _, mc := range p.ModelConfigs {
		if mc.Name != name {
			continue
		}
		if mc.Type != "" {
			p.Type = mc.Type
		}
		if mc.BaseURL != "" {
			p.BaseURL = mc.BaseURL
		}
		if mc.ChatPath != "" {
			p.ChatPath = mc.ChatPath
		}
		if mc.APIKey != "" {
			p.APIKey = mc.APIKey
		}
		return p
	}
	return p
}

// ChatCompletionsURL returns the full URL for the chat completions endpoint.
// If ChatPath is set it overrides the default /v1/chat/completions path.
func (p ProviderConfig) ChatCompletionsURL() string {
	if p.ChatPath != "" {
		return p.BaseURL + p.ChatPath
	}
	return p.BaseURL + "/v1/chat/completions"
}

// OAuthEnabled returns true when all required OAuth fields are configured.
func (c *Config) OAuthEnabled() bool {
	return c.OAuthIssuerURL != "" && c.OAuthClientID != "" && c.OAuthClientSecret != ""
}

// Config holds all configuration for the application.
type Config struct {
	// Provider configuration — loaded from providers.json
	Providers []ProviderConfig

	// Routing behaviour
	ModelSelectionModel    string // Model name used as the meta-router
	ModelSelectionProvider string // Provider name for the meta-router (must be openai-compatible)
	DefaultChatModel       string // Fallback model when routing fails

	// Routing decision cache
	SelectionCacheEnabled          bool
	SelectionCacheTTL              time.Duration
	SelectionCacheMaxEntries       int
	SelectionCacheStatsLogInterval time.Duration

	// OpenTelemetry
	OTelEnabled              bool
	OTelServiceName          string
	OTelExporterOTLPEndpoint string
	OTelExporterOTLPInsecure bool
	OTelTraceSampleRatio     float64

	// Benchmarking
	BenchmarkEnabled          bool
	BenchmarkScheduleEnabled  bool
	BenchmarkScheduleInterval time.Duration
	BenchmarkJudgeProvider    string // provider for LLM judge (defaults to MODEL_SELECTION_PROVIDER)
	BenchmarkJudgeModel       string // model for LLM judge (defaults to MODEL_SELECTION_MODEL)
	BenchmarkConcurrency      int    // max parallel model benchmark runs (default 3)

	// Persistence
	DatabaseURL                   string
	DatabaseMaxConnections        int
	DatabaseMaxIdleConnections    int
	DatabaseConnectionMaxLifetime time.Duration

	// Server
	AdminAPIKey      string
	Port             string
	Environment      string
	LogLevel         string
	EnableCORS       bool
	TrustedProxies   []string
	MaxRequestSize   int64
	RateLimitEnabled bool

	// OAuth / OIDC
	OAuthIssuerURL     string
	OAuthClientID      string
	OAuthClientSecret  string
	OAuthRedirectURL   string
	OAuthScopes        string // space-separated, default "openid profile email groups"
	OAuthAutoProvision bool
	SessionSecret      string // Hex-encoded key for session cookie encryption; 128 hex chars (64 bytes) recommended
	RequireAuthForChat bool   // when true, chat/API endpoints require authentication
}

// Load loads configuration from .env file, environment variables, and providers.json.
func Load() (*Config, error) {
	execDir, err := os.Getwd()
	if err != nil {
		log.Printf("Warning: Could not determine working directory: %v", err)
	}

	envPath := filepath.Join(execDir, ".env")
	if _, err := os.Stat(envPath); err == nil {
		log.Printf("Loading environment from %s", envPath)
		if err := godotenv.Load(envPath); err != nil {
			log.Printf("Warning: Error loading .env file: %v", err)
		}
	} else {
		log.Printf("No .env file found at %s, using environment variables only", envPath)
	}

	parseDurationEnv := func(key string, defaultValue time.Duration) time.Duration {
		if value, exists := os.LookupEnv(key); exists && value != "" {
			if parsed, err := time.ParseDuration(value); err == nil {
				return parsed
			} else {
				log.Printf("Warning: Could not parse duration for %s: %v, using default", key, err)
			}
		}
		return defaultValue
	}

	parseTrustedProxies := func(key string) []string {
		if value, exists := os.LookupEnv(key); exists && value != "" {
			return strings.Split(value, ",")
		}
		return []string{}
	}

	cfg := &Config{
		ModelSelectionModel:            getEnv("MODEL_SELECTION_MODEL", "gpt-4-turbo"),
		ModelSelectionProvider:         getEnv("MODEL_SELECTION_PROVIDER", "openai"),
		DefaultChatModel:               getEnv("DEFAULT_CHAT_MODEL", "gpt-4-turbo"),
		SelectionCacheEnabled:          getEnvBool("SELECTION_CACHE_ENABLED", true),
		SelectionCacheTTL:              parseDurationEnv("SELECTION_CACHE_TTL", 2*time.Minute),
		SelectionCacheMaxEntries:       getEnvInt("SELECTION_CACHE_MAX_ENTRIES", 1000),
		SelectionCacheStatsLogInterval: parseDurationEnv("SELECTION_CACHE_STATS_LOG_INTERVAL", time.Minute),
		BenchmarkEnabled:               getEnvBool("BENCHMARK_ENABLED", false),
		BenchmarkScheduleEnabled:       getEnvBool("BENCHMARK_SCHEDULE_ENABLED", false),
		BenchmarkScheduleInterval:      parseDurationEnv("BENCHMARK_SCHEDULE_INTERVAL", 24*time.Hour),
		BenchmarkJudgeProvider:         getEnv("BENCHMARK_JUDGE_PROVIDER", ""),
		BenchmarkJudgeModel:            getEnv("BENCHMARK_JUDGE_MODEL", ""),
		BenchmarkConcurrency:           getEnvInt("BENCHMARK_CONCURRENCY", 3),
		OTelEnabled:                    getEnvBool("OTEL_ENABLED", false),
		OTelServiceName:                getEnv("OTEL_SERVICE_NAME", "pipimink"),
		OTelExporterOTLPEndpoint:       getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4318"),
		OTelExporterOTLPInsecure:       getEnvBool("OTEL_EXPORTER_OTLP_INSECURE", true),
		OTelTraceSampleRatio:           getEnvFloat64("OTEL_TRACE_SAMPLE_RATIO", 1.0),
		DatabaseURL:                    getEnv("DATABASE_URL", "postgres://pipimink_user:change_me@pipimink-postgres:5432/pipimink_db?sslmode=require"),
		AdminAPIKey:                    getEnv("ADMIN_API_KEY", ""),
		Port:                           getEnv("PORT", "8080"),
		Environment:                    getEnv("ENVIRONMENT", "development"),
		LogLevel:                       getEnv("LOG_LEVEL", "info"),
		EnableCORS:                     getEnvBool("ENABLE_CORS", true),
		TrustedProxies:                 parseTrustedProxies("TRUSTED_PROXIES"),
		MaxRequestSize:                 getEnvInt64("MAX_REQUEST_SIZE", 10*1024*1024),
		RateLimitEnabled:               getEnvBool("RATE_LIMIT_ENABLED", true),
		DatabaseMaxConnections:         getEnvInt("DATABASE_MAX_CONNECTIONS", 25),
		DatabaseMaxIdleConnections:     getEnvInt("DATABASE_MAX_IDLE_CONNECTIONS", 5),
		DatabaseConnectionMaxLifetime:  parseDurationEnv("DATABASE_CONNECTION_MAX_LIFETIME", 30*time.Minute),
		OAuthIssuerURL:                 getEnv("OAUTH_ISSUER_URL", ""),
		OAuthClientID:                  getEnv("OAUTH_CLIENT_ID", ""),
		OAuthClientSecret:              getEnv("OAUTH_CLIENT_SECRET", ""),
		OAuthRedirectURL:               getEnv("OAUTH_REDIRECT_URL", ""),
		OAuthScopes:                    getEnv("OAUTH_SCOPES", "openid profile email groups"),
		OAuthAutoProvision:             getEnvBool("OAUTH_AUTO_PROVISION", true),
		SessionSecret:                  getEnv("SESSION_SECRET", ""),
		RequireAuthForChat:             getEnvBool("REQUIRE_AUTH_FOR_CHAT", false),
	}

	// Load providers from providers.json (optional — fall back to built-in defaults)
	cfg.Providers = loadProviders(execDir)

	return cfg, nil
}

// loadProviders reads providers.json and resolves API keys from environment variables.
// If the file does not exist, a minimal default OpenAI provider is returned so the
// application still starts without configuration.
func loadProviders(dir string) []ProviderConfig {
	path := filepath.Join(dir, "providers.json")
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("providers.json not found at %s — using built-in default (OpenAI only). Copy providers.example.json to providers.json to configure.", path)
		return defaultProviders()
	}

	var providers []ProviderConfig
	if err := json.Unmarshal(data, &providers); err != nil {
		log.Printf("Warning: Could not parse providers.json: %v — using built-in default", err)
		return defaultProviders()
	}

	// Detect which providers/model configs have an explicit "enabled" field set.
	// Go unmarshals missing bools as false, so we need a two-pass approach to
	// default Enabled to true when the field is absent from JSON.
	var rawProviders []json.RawMessage
	if err := json.Unmarshal(data, &rawProviders); err == nil {
		for i, raw := range rawProviders {
			if i >= len(providers) {
				break
			}
			if !jsonHasKey(raw, "enabled") {
				providers[i].Enabled = true
			}
			// Check model configs for explicit enabled field
			var mc struct {
				ModelConfigs []json.RawMessage `json:"model_configs"`
			}
			if err := json.Unmarshal(raw, &mc); err == nil {
				for j, rawMC := range mc.ModelConfigs {
					if j < len(providers[i].ModelConfigs) && !jsonHasKey(rawMC, "enabled") {
						providers[i].ModelConfigs[j].Enabled = true
					}
				}
			}
		}
	}

	for i := range providers {
		ResolveProviderKeys(&providers[i])
	}

	log.Printf("Loaded %d provider(s) from providers.json", len(providers))
	return providers
}

// jsonHasKey checks whether a JSON object contains a given top-level key.
func jsonHasKey(raw json.RawMessage, key string) bool {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}
	_, ok := m[key]
	return ok
}

// ResolveProviderKeys resolves API keys from environment variables for a provider
// and its model configs, parses the timeout string, and sets default type.
func ResolveProviderKeys(p *ProviderConfig) {
	// Resolve provider-level API key from env
	if p.APIKeyEnv != "" {
		p.APIKey = os.Getenv(p.APIKeyEnv)
		if p.APIKey == "" {
			log.Printf("Warning: provider %q: API key not set (check env configuration)", p.Name)
		}
	}

	// Resolve per-model API keys from env
	for j := range p.ModelConfigs {
		mc := &p.ModelConfigs[j]
		if mc.APIKeyEnv != "" {
			mc.APIKey = os.Getenv(mc.APIKeyEnv)
			if mc.APIKey == "" {
				log.Printf("Warning: provider %q model %q: API key not set (check env configuration)", p.Name, mc.Name)
			}
		}
	}

	// Parse timeout
	if p.TimeoutStr != "" {
		if d, err := time.ParseDuration(p.TimeoutStr); err == nil {
			p.Timeout = d
		} else {
			log.Printf("Warning: provider %q has invalid timeout %q, using 2m", p.Name, p.TimeoutStr)
			p.Timeout = 2 * time.Minute
		}
	} else {
		p.Timeout = 2 * time.Minute
	}

	// Default type
	if p.Type == "" {
		p.Type = ProviderTypeOpenAICompatible
	}
}

// SaveProviders writes the provider list to providers.json in the given directory.
// The write is atomic: data is written to a temp file then renamed.
func SaveProviders(dir string, providers []ProviderConfig) error {
	data, err := json.MarshalIndent(providers, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal providers: %w", err)
	}
	data = append(data, '\n')

	path := filepath.Join(dir, "providers.json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

// defaultProviders returns a minimal provider list used when providers.json is absent.
func defaultProviders() []ProviderConfig {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Println("Warning: OPENAI_API_KEY is not set. Set it or configure providers.json.")
	}
	return []ProviderConfig{
		{
			Name:      "openai",
			Type:      ProviderTypeOpenAICompatible,
			BaseURL:   getEnv("OPENAI_BASE_URL", "https://api.openai.com"),
			APIKeyEnv: "OPENAI_API_KEY",
			APIKey:    apiKey,
			Timeout:   2 * time.Minute,
			Enabled:   true,
		},
	}
}

// LoadConfig is maintained for backward compatibility.
// Deprecated: Use Load() instead.
func LoadConfig() *Config {
	cfg, _ := Load()
	return cfg
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		} else {
			log.Printf("Warning: Could not parse integer for %s: %v, using default", key, err)
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		} else {
			log.Printf("Warning: Could not parse boolean for %s: %v, using default", key, err)
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		} else {
			log.Printf("Warning: Could not parse int64 for %s: %v, using default", key, err)
		}
	}
	return defaultValue
}

func getEnvFloat64(key string, defaultValue float64) float64 {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		} else {
			log.Printf("Warning: Could not parse float64 for %s: %v, using default", key, err)
		}
	}
	return defaultValue
}
