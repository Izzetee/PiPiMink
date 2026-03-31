package llm

import (
	"encoding/json"
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

func TestDecideModelBasedOnCapabilitiesWithAnthropicProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Anthropic API format
		assert.Equal(t, "/v1/messages", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "test-anthropic-key", r.Header.Get("x-api-key"))
		assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))
		assert.Empty(t, r.Header.Get("Authorization"), "should not set Bearer auth for Anthropic")

		// Verify payload is Anthropic format (system as top-level field, not in messages)
		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		assert.NoError(t, err)
		assert.NotEmpty(t, payload["system"], "system message should be top-level field")
		assert.Equal(t, float64(4096), payload["max_tokens"])
		msgs := payload["messages"].([]interface{})
		assert.Len(t, msgs, 1, "should have only user message, no system message in messages array")
		firstMsg := msgs[0].(map[string]interface{})
		assert.Equal(t, "user", firstMsg["role"])

		// Return Anthropic-format response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"content": [
				{
					"type": "text",
					"text": "{\"modelname\": \"gpt-4\", \"reason\": \"Best for complex reasoning\", \"matching_tags\": [\"complex-reasoning\"], \"tag_relevance\": {\"complex-reasoning\": 9}}"
				}
			]
		}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Providers: []config.ProviderConfig{
			{
				Name:    "az-foundry",
				Type:    config.ProviderTypeOpenAICompatible,
				BaseURL: "https://should-not-be-used.example.com",
				Timeout: 5 * time.Second,
				ModelConfigs: []config.ModelConfig{
					{
						Name:    "claude-opus",
						Type:    config.ProviderTypeAnthropic,
						BaseURL: server.URL,
						APIKey:  "test-anthropic-key",
					},
				},
			},
		},
		ModelSelectionModel:    "claude-opus",
		ModelSelectionProvider: "az-foundry",
		DefaultChatModel:       "gpt-4",
	}
	client := NewClient(cfg)

	availableModels := map[string]models.ModelInfo{
		"gpt-4":       {Source: "openai", Tags: `{"strengths":["complex-reasoning","code"],"weaknesses":["speed"]}`, Enabled: true},
		"gpt-4-turbo": {Source: "openai", Tags: `{"strengths":["speed","general"],"weaknesses":["complex-reasoning"]}`, Enabled: true},
	}

	model, err := client.DecideModelBasedOnCapabilities("Write a complex algorithm", availableModels)
	assert.NoError(t, err)
	assert.Equal(t, "gpt-4", model)
}

func TestDecideModelBasedOnCapabilitiesWithPerModelOverrides(t *testing.T) {
	// Verify that per-model base_url and api_key overrides are applied correctly.
	// The provider-level URL/key should NOT be used when model_configs override them.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer model-specific-key", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"modelname\":\"gpt-4\",\"reason\":\"test\"}"}}]}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Providers: []config.ProviderConfig{
			{
				Name:    "az-foundry",
				Type:    config.ProviderTypeOpenAICompatible,
				BaseURL: "https://provider-level-should-not-be-used.example.com",
				APIKey:  "provider-level-key-should-not-be-used",
				Timeout: 5 * time.Second,
				ModelConfigs: []config.ModelConfig{
					{
						Name:    "my-judge-model",
						BaseURL: server.URL,
						APIKey:  "model-specific-key",
					},
				},
			},
		},
		ModelSelectionModel:    "my-judge-model",
		ModelSelectionProvider: "az-foundry",
		DefaultChatModel:       "gpt-4",
	}
	client := NewClient(cfg)

	availableModels := map[string]models.ModelInfo{
		"gpt-4": {Source: "openai", Tags: `{"strengths":["general"]}`, Enabled: true},
	}

	model, err := client.DecideModelBasedOnCapabilities("hello", availableModels)
	assert.NoError(t, err)
	assert.Equal(t, "gpt-4", model)
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
