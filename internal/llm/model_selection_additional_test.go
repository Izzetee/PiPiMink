package llm

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"PiPiMink/internal/config"
	"PiPiMink/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestSelectionAndFallbackModelHelpers(t *testing.T) {
	c := &Client{}
	assert.Equal(t, defaultSelectionModel, c.selectionModel())
	assert.Equal(t, defaultSelectionModel, c.fallbackModel())

	c = &Client{Config: &config.Config{ModelSelectionModel: "  gpt-4-turbo  "}}
	assert.Equal(t, "gpt-4-turbo", c.selectionModel())
	assert.Equal(t, "gpt-4-turbo", c.fallbackModel())

	c = &Client{Config: &config.Config{ModelSelectionModel: "gpt-4", DefaultChatModel: "  fallback-fast  "}}
	assert.Equal(t, "fallback-fast", c.fallbackModel())
}

func TestDecideModelBasedOnCapabilitiesFallbackOnInvalidJSONContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"not a json block"}}]}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Providers:              []config.ProviderConfig{{Name: "openai", Type: config.ProviderTypeOpenAICompatible, BaseURL: server.URL, APIKey: "test-key", Timeout: 5 * time.Second}},
		ModelSelectionModel:    "gpt-4-turbo",
		ModelSelectionProvider: "openai",
		DefaultChatModel:       "gpt-4-turbo",
	}
	client := NewClient(cfg)

	availableModels := map[string]models.ModelInfo{
		"gpt-4-turbo": {Source: "openai", Tags: `{"strengths":["general"]}`, Enabled: true},
	}

	selectedModel, err := client.DecideModelBasedOnCapabilities("hello", availableModels)
	assert.Error(t, err)
	assert.Equal(t, "gpt-4-turbo", selectedModel)
}

func TestDecideModelBasedOnCapabilitiesFallbackWhenModelNotAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"modelname\":\"not-available\",\"reason\":\"test\"}"}}]}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Providers:              []config.ProviderConfig{{Name: "openai", Type: config.ProviderTypeOpenAICompatible, BaseURL: server.URL, APIKey: "test-key", Timeout: 5 * time.Second}},
		ModelSelectionModel:    "gpt-4-turbo",
		ModelSelectionProvider: "openai",
		DefaultChatModel:       "gpt-4-turbo",
	}
	client := NewClient(cfg)

	availableModels := map[string]models.ModelInfo{
		"gpt-4-turbo": {Source: "openai", Tags: `{"strengths":["general"]}`, Enabled: true},
	}

	selectedModel, err := client.DecideModelBasedOnCapabilities("hello", availableModels)
	assert.NoError(t, err)
	assert.Equal(t, "gpt-4-turbo", selectedModel)
}
