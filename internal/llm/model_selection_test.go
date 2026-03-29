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

func TestDecideModelBasedOnCapabilities(t *testing.T) {
	// This function would ideally require real API access for complete testing
	// For proper unit testing, we should use a mock HTTP server

	t.Run("With Mock Server", func(t *testing.T) {
		// Create a test server to mock the OpenAI API response
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check request path and method
			assert.Equal(t, "/v1/chat/completions", r.URL.Path)
			assert.Equal(t, "POST", r.Method)

			// Check for authorization header
			assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

			// Return a mock response with a model decision
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"choices": [
					{
						"message": {
							"role": "assistant",
							"content": "{\"modelname\": \"gpt-4\", \"reason\": \"This model is best suited for this complex task\"}"
						}
					}
				]
			}`))
		}))
		defer server.Close()

		// Create a client with the mock server
		cfg := &config.Config{
			Providers: []config.ProviderConfig{
				{
					Name:    "openai",
					Type:    config.ProviderTypeOpenAICompatible,
					BaseURL: server.URL,
					APIKey:  "test-key",
					Timeout: 5 * time.Second,
				},
			},
			ModelSelectionModel:    "gpt-4-turbo",
			ModelSelectionProvider: "openai",
			DefaultChatModel:       "gpt-4-turbo",
		}
		client := NewClient(cfg)

		// Create test models
		availableModels := map[string]models.ModelInfo{
			"gpt-4": {
				Source:  "openai",
				Tags:    `{"strengths":["complex-reasoning","code"],"weaknesses":["speed"]}`,
				Enabled: true,
			},
			"gpt-4-turbo": {
				Source:  "openai",
				Tags:    `{"strengths":["speed","general"],"weaknesses":["complex-reasoning"]}`,
				Enabled: true,
			},
		}

		// Test the function
		model, err := client.DecideModelBasedOnCapabilities("Write a complex algorithm to solve the traveling salesman problem", availableModels)
		assert.NoError(t, err)
		assert.Equal(t, "gpt-4", model)
	})

	// Skip the test that requires actual API access
	t.Run("Real API Test", func(t *testing.T) {
		t.Skip("Skipping test that requires actual API access")
	})
}

func TestDecideModel(t *testing.T) {
	cfg := &config.Config{DefaultChatModel: "gpt-4-turbo"}
	client := NewClient(cfg)

	model := client.DecideModel("Any message")
	assert.Equal(t, "gpt-4-turbo", model)
}
