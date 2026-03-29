package llm

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"PiPiMink/internal/config"
)

// TestConfig returns a test configuration with sensible defaults.
// It includes an "openai" provider and a "local" provider pointing at localhost.
func TestConfig() *config.Config {
	return &config.Config{
		Providers: []config.ProviderConfig{
			{
				Name:    "openai",
				Type:    config.ProviderTypeOpenAICompatible,
				BaseURL: "https://api.openai.com",
				APIKey:  "test-api-key",
				Timeout: 100 * time.Millisecond,
			},
			{
				Name:             "local",
				Type:             config.ProviderTypeOpenAICompatible,
				BaseURL:          "http://localhost:11434",
				Timeout:          100 * time.Millisecond,
				RateLimitSeconds: 1,
			},
		},
		ModelSelectionModel: "gpt-4-turbo",
		DefaultChatModel:    "gpt-4-turbo",
	}
}

// MockHTTPServer creates a test server with a handler that returns predefined JSON responses
// for common API endpoints used in tests
func MockHTTPServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set common headers for all responses
		w.Header().Set("Content-Type", "application/json")

		// Handle different endpoints
		switch r.URL.Path {
		case "/v1/chat/completions":
			// Return a successful chat completion response
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"choices": [
					{
						"message": {
							"role": "assistant",
							"content": "This is a test response"
						}
					}
				]
			}`))

		case "/v1/models":
			// Return a list of available models
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"data": [
					{"id": "model-1", "object": "model"},
					{"id": "model-2", "object": "model"},
					{"id": "model-3", "object": "model"}
				]
			}`))

		default:
			// Return a generic error for unhandled paths
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": {"message": "Not found"}}`))
		}
	}))
}

// MockErrorHTTPServer creates a test server that returns error responses
// This is useful for testing error handling in the client
func MockErrorHTTPServer(t *testing.T, statusCode int, errorMessage string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(`{
			"error": {
				"message": "` + errorMessage + `"
			}
		}`))
	}))
}
