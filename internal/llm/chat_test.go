package llm

import (
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
