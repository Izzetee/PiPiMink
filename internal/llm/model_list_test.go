package llm

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"PiPiMink/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestGetModelsByProvider(t *testing.T) {
	// Create a test server to mock the /v1/models API response
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/models", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"object": "list",
			"data": [
				{"id": "model-1", "object": "model"},
				{"id": "model-2", "object": "model"},
				{"id": "model-3", "object": "model"}
			]
		}`))
	}))
	defer srv.Close()

	openaiProvider := config.ProviderConfig{
		Name:    "openai",
		Type:    config.ProviderTypeOpenAICompatible,
		BaseURL: srv.URL,
		APIKey:  "test-key",
		Timeout: 5 * time.Second,
	}
	localProvider := config.ProviderConfig{
		Name:    "local",
		Type:    config.ProviderTypeOpenAICompatible,
		BaseURL: srv.URL,
		Timeout: 5 * time.Second,
	}

	cfg := &config.Config{Providers: []config.ProviderConfig{openaiProvider, localProvider}}
	client := NewClient(cfg)

	t.Run("OpenAI provider — auto-discover", func(t *testing.T) {
		models, err := client.GetModelsByProvider(openaiProvider)
		assert.NoError(t, err)
		assert.Equal(t, []string{"model-1", "model-2", "model-3"}, models)
	})

	t.Run("Local provider — auto-discover", func(t *testing.T) {
		models, err := client.GetModelsByProvider(localProvider)
		assert.NoError(t, err)
		assert.Equal(t, []string{"model-1", "model-2", "model-3"}, models)
	})

	t.Run("Static model list — no HTTP call needed", func(t *testing.T) {
		staticProvider := config.ProviderConfig{
			Name:    "az-gpt4o",
			Type:    config.ProviderTypeOpenAICompatible,
			BaseURL: srv.URL,
			Timeout: 5 * time.Second,
			Models:  []string{"gpt-4o"},
		}
		models, err := client.GetModelsByProvider(staticProvider)
		assert.NoError(t, err)
		assert.Equal(t, []string{"gpt-4o"}, models)
	})
}
