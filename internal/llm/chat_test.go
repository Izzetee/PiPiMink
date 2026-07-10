package llm

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"PiPiMink/internal/config"
	"PiPiMink/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestChatWithModel(t *testing.T) {
	// Create a mock HTTP server using our helper
	server := MockHTTPServer(t)
	defer server.Close()

	// Create a client with providers pointing at the mock server
	cfg := &config.Config{
		Providers: []config.ProviderConfig{
			{Name: "openai", Type: config.ProviderTypeOpenAICompatible, BaseURL: server.URL, APIKey: "test-key", Timeout: 5 * time.Second},
			{Name: "local", Type: config.ProviderTypeOpenAICompatible, BaseURL: server.URL, Timeout: 5 * time.Second},
		},
	}
	client := NewClient(cfg)

	singleMessage := []map[string]interface{}{{"role": "user", "content": "Hello, world!"}}

	// Test OpenAI model
	t.Run("OpenAI Model", func(t *testing.T) {
		modelInfo := models.ModelInfo{
			Source:  "openai",
			Enabled: true,
		}

		response, err := client.ChatWithModel(modelInfo, "gpt-4-turbo", singleMessage)
		assert.NoError(t, err)
		assert.Equal(t, "This is a test response", response)
	})

	// Test Local model
	t.Run("Local Model", func(t *testing.T) {
		modelInfo := models.ModelInfo{
			Source:  "local",
			Enabled: true,
		}

		response, err := client.ChatWithModel(modelInfo, "llama2", singleMessage)
		assert.NoError(t, err)
		assert.Equal(t, "This is a test response", response)
	})

	// Test disabled model
	t.Run("Disabled Model", func(t *testing.T) {
		modelInfo := models.ModelInfo{
			Source:  "openai",
			Enabled: false,
		}

		response, err := client.ChatWithModel(modelInfo, "gpt-4", singleMessage)
		assert.NoError(t, err)
		assert.Equal(t, "This model is currently disabled", response)
	})

	// Test invalid source
	t.Run("Invalid Source", func(t *testing.T) {
		modelInfo := models.ModelInfo{
			Source:  "invalid",
			Enabled: true,
		}

		_, err := client.ChatWithModel(modelInfo, "unknown", singleMessage)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown provider")
	})

	// Test multi-turn conversation history is forwarded
	t.Run("Multi-turn History", func(t *testing.T) {
		modelInfo := models.ModelInfo{
			Source:  "openai",
			Enabled: true,
		}

		history := []map[string]interface{}{
			{"role": "user", "content": "What is Go?"},
			{"role": "assistant", "content": "Go is a statically typed language."},
			{"role": "user", "content": "Give me an example."},
		}

		response, err := client.ChatWithModel(modelInfo, "gpt-4-turbo", history)
		assert.NoError(t, err)
		assert.Equal(t, "This is a test response", response)
	})
}

func TestChatWithModelResponsesAPI(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"responses answer"}]}]}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Providers: []config.ProviderConfig{
			{
				Name:    "az-foundry",
				Type:    config.ProviderTypeOpenAICompatible,
				BaseURL: server.URL,
				Timeout: 5 * time.Second,
				ModelConfigs: []config.ModelConfig{
					{Name: "gpt-5", Type: config.ProviderTypeOpenAIResponses, ChatPath: "/openai/v1/responses", APIKey: "test-key"},
				},
			},
		},
	}
	client := NewClient(cfg)

	modelInfo := models.ModelInfo{Source: "az-foundry", Enabled: true}
	response, err := client.ChatWithModel(modelInfo, "gpt-5", []map[string]interface{}{
		{"role": "user", "content": "Hello"},
	})

	assert.NoError(t, err)
	assert.Equal(t, "responses answer", response)
	assert.True(t, strings.HasSuffix(gotPath, "/openai/v1/responses"), "must hit the Responses endpoint, got %s", gotPath)
	assert.Contains(t, gotBody, "input", "request must use 'input'")
	assert.NotContains(t, gotBody, "messages", "request must not use 'messages'")
}
