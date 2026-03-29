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
	// Providers are loaded from providers.json; without the file only the default OpenAI entry is present
	assert.NotEmpty(t, cfg.Providers)
}
