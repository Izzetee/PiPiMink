package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetEnvHelpers(t *testing.T) {
	t.Setenv("TEST_STR", "value")
	t.Setenv("TEST_INT", "42")
	t.Setenv("TEST_BOOL", "true")
	t.Setenv("TEST_INT64", "99")

	assert.Equal(t, "value", getEnv("TEST_STR", "default"))
	assert.Equal(t, "default", getEnv("TEST_MISSING_STR", "default"))

	assert.Equal(t, 42, getEnvInt("TEST_INT", 7))
	assert.Equal(t, 7, getEnvInt("TEST_MISSING_INT", 7))
	t.Setenv("TEST_INT", "not-an-int")
	assert.Equal(t, 7, getEnvInt("TEST_INT", 7))

	assert.Equal(t, true, getEnvBool("TEST_BOOL", false))
	assert.Equal(t, false, getEnvBool("TEST_MISSING_BOOL", false))
	t.Setenv("TEST_BOOL", "not-a-bool")
	assert.Equal(t, false, getEnvBool("TEST_BOOL", false))

	assert.Equal(t, int64(99), getEnvInt64("TEST_INT64", 5))
	assert.Equal(t, int64(5), getEnvInt64("TEST_MISSING_INT64", 5))
	t.Setenv("TEST_INT64", "not-an-int64")
	assert.Equal(t, int64(5), getEnvInt64("TEST_INT64", 5))
}

func TestLoadReadsConfiguredValues(t *testing.T) {
	t.Setenv("MODEL_SELECTION_MODEL", "gpt-4-turbo")
	t.Setenv("MODEL_SELECTION_PROVIDER", "openai")
	t.Setenv("DEFAULT_CHAT_MODEL", "gpt-4-turbo")
	t.Setenv("SELECTION_CACHE_ENABLED", "true")
	t.Setenv("SELECTION_CACHE_TTL", "90s")
	t.Setenv("SELECTION_CACHE_MAX_ENTRIES", "333")
	t.Setenv("SELECTION_CACHE_STATS_LOG_INTERVAL", "30s")
	t.Setenv("OTEL_ENABLED", "true")
	t.Setenv("OTEL_SERVICE_NAME", "pipimink-test")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "tempo:4318")
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	t.Setenv("OTEL_TRACE_SAMPLE_RATIO", "0.25")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	t.Setenv("ENABLE_CORS", "false")
	t.Setenv("TRUSTED_PROXIES", "10.0.0.1,10.0.0.2")
	t.Setenv("DATABASE_MAX_CONNECTIONS", "11")
	t.Setenv("DATABASE_MAX_IDLE_CONNECTIONS", "3")
	t.Setenv("DATABASE_CONNECTION_MAX_LIFETIME", "45m")
	t.Setenv("TAGGING_MAX_TOKENS", "1536")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	assert.Equal(t, "gpt-4-turbo", cfg.ModelSelectionModel)
	assert.Equal(t, "openai", cfg.ModelSelectionProvider)
	assert.Equal(t, "gpt-4-turbo", cfg.DefaultChatModel)
	assert.True(t, cfg.SelectionCacheEnabled)
	assert.Equal(t, 90*time.Second, cfg.SelectionCacheTTL)
	assert.Equal(t, 333, cfg.SelectionCacheMaxEntries)
	assert.Equal(t, 30*time.Second, cfg.SelectionCacheStatsLogInterval)
	assert.True(t, cfg.OTelEnabled)
	assert.Equal(t, "pipimink-test", cfg.OTelServiceName)
	assert.Equal(t, "tempo:4318", cfg.OTelExporterOTLPEndpoint)
	assert.True(t, cfg.OTelExporterOTLPInsecure)
	assert.Equal(t, 0.25, cfg.OTelTraceSampleRatio)
	assert.Equal(t, "postgres://user:pass@localhost:5432/db?sslmode=disable", cfg.DatabaseURL)
	assert.False(t, cfg.EnableCORS)
	assert.Equal(t, []string{"10.0.0.1", "10.0.0.2"}, cfg.TrustedProxies)
	assert.Equal(t, 11, cfg.DatabaseMaxConnections)
	assert.Equal(t, 3, cfg.DatabaseMaxIdleConnections)
	assert.Equal(t, 45*time.Minute, cfg.DatabaseConnectionMaxLifetime)
	assert.Equal(t, 1536, cfg.TaggingMaxTokens)
	// Providers are loaded from providers.json; without the file only the default OpenAI entry is present
	assert.NotEmpty(t, cfg.Providers)
}

func TestSaveProvidersRoundTripAndOverwrite(t *testing.T) {
	dir := t.TempDir()

	first := []ProviderConfig{
		{Name: "openai", Type: ProviderTypeOpenAICompatible, BaseURL: "https://api.openai.com"},
	}
	assert.NoError(t, SaveProviders(dir, first))

	got := loadProviders(dir)
	assert.Len(t, got, 1)
	assert.Equal(t, "openai", got[0].Name)

	// Overwriting an existing providers.json must succeed and replace the content.
	second := []ProviderConfig{
		{Name: "openai", Type: ProviderTypeOpenAICompatible, BaseURL: "https://api.openai.com"},
		{
			Name:    "az-foundry",
			Type:    ProviderTypeOpenAICompatible,
			BaseURL: "https://example.cognitiveservices.azure.com",
			ModelConfigs: []ModelConfig{
				{Name: "claude-haiku-4-5", Type: ProviderTypeAnthropic, BaseURL: "https://example.services.ai.azure.com/anthropic"},
			},
		},
	}
	assert.NoError(t, SaveProviders(dir, second))

	got = loadProviders(dir)
	assert.Len(t, got, 2)
	assert.Equal(t, "az-foundry", got[1].Name)
	assert.Len(t, got[1].ModelConfigs, 1)
	assert.Equal(t, "claude-haiku-4-5", got[1].ModelConfigs[0].Name)
	assert.Equal(t, ProviderTypeAnthropic, got[1].ModelConfigs[0].Type)
}

func TestUsesResponsesAPI(t *testing.T) {
	// Explicit type.
	p := ProviderConfig{Type: ProviderTypeOpenAIResponses, BaseURL: "https://x"}
	assert.True(t, p.UsesResponsesAPI())

	// Auto-detected via chat path.
	p = ProviderConfig{Type: ProviderTypeOpenAICompatible, ChatPath: "/openai/v1/responses"}
	assert.True(t, p.UsesResponsesAPI())

	// Plain chat completions provider must not be treated as Responses API.
	p = ProviderConfig{Type: ProviderTypeOpenAICompatible, ChatPath: "/v1/chat/completions"}
	assert.False(t, p.UsesResponsesAPI())

	p = ProviderConfig{Type: ProviderTypeOpenAICompatible}
	assert.False(t, p.UsesResponsesAPI())
}

func TestResponsesURL(t *testing.T) {
	// Default path for OpenAI.
	p := ProviderConfig{Type: ProviderTypeOpenAIResponses, BaseURL: "https://api.openai.com"}
	assert.Equal(t, "https://api.openai.com/v1/responses", p.ResponsesURL())

	// Azure AI Foundry host defaults to /openai/v1/responses without an explicit chat_path.
	p = ProviderConfig{Type: ProviderTypeOpenAIResponses, BaseURL: "https://x.services.ai.azure.com"}
	assert.Equal(t, "https://x.services.ai.azure.com/openai/v1/responses", p.ResponsesURL())

	// Trailing slash on the base URL must not produce a double slash.
	p = ProviderConfig{Type: ProviderTypeOpenAIResponses, BaseURL: "https://x.services.ai.azure.com/"}
	assert.Equal(t, "https://x.services.ai.azure.com/openai/v1/responses", p.ResponsesURL())

	// Explicit chat path override wins.
	p = ProviderConfig{
		Type:     ProviderTypeOpenAIResponses,
		BaseURL:  "https://x.services.ai.azure.com",
		ChatPath: "/openai/v1/responses",
	}
	assert.Equal(t, "https://x.services.ai.azure.com/openai/v1/responses", p.ResponsesURL())
}

func TestForModelAppliesResponsesTypeOverride(t *testing.T) {
	p := ProviderConfig{
		Name:    "az-foundry",
		Type:    ProviderTypeOpenAICompatible,
		BaseURL: "https://x.services.ai.azure.com",
		ModelConfigs: []ModelConfig{
			{Name: "gpt-5", Type: ProviderTypeOpenAIResponses, ChatPath: "/openai/v1/responses"},
		},
	}
	resolved := p.ForModel("gpt-5")
	assert.True(t, resolved.UsesResponsesAPI())
	assert.Equal(t, "https://x.services.ai.azure.com/openai/v1/responses", resolved.ResponsesURL())
}
