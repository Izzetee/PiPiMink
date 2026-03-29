package llm

import (
	"testing"

	"PiPiMink/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	cfg := &config.Config{
		Providers: []config.ProviderConfig{
			{Name: "openai", Type: config.ProviderTypeOpenAICompatible, BaseURL: "https://api.openai.com", APIKey: "test-key"},
			{Name: "local", Type: config.ProviderTypeOpenAICompatible, BaseURL: "http://localhost:11434", RateLimitSeconds: 1},
		},
	}

	client := NewClient(cfg)
	assert.NotNil(t, client)
	assert.Equal(t, cfg, client.Config)
	// local provider has rate_limit_seconds=1, so a limiter should be registered
	assert.NotNil(t, client.rateLimiterFor("local"))
	assert.Nil(t, client.rateLimiterFor("openai"))
}
