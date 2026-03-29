package llm

import (
	"testing"
	"time"

	"PiPiMink/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDatabase implements the database interface for testing
type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) GetAllModels() (map[string]map[string]interface{}, error) {
	args := m.Called()
	return args.Get(0).(map[string]map[string]interface{}), args.Error(1)
}

func (m *MockDatabase) EnableModel(name, source string, enabled bool) error {
	args := m.Called(name, source, enabled)
	return args.Error(0)
}

func TestGetModelTags(t *testing.T) {
	// Create a mock server with standard responses
	srv := MockHTTPServer(t)
	defer srv.Close()

	openaiProvider := config.ProviderConfig{
		Name:    "openai",
		Type:    config.ProviderTypeOpenAICompatible,
		BaseURL: srv.URL,
		APIKey:  "test-api-key",
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

	t.Run("OpenAI Model", func(t *testing.T) {
		tags, shouldDisable, shouldDelete, err := client.GetModelTags("gpt-4-turbo", openaiProvider)
		assert.NoError(t, err)
		assert.False(t, shouldDisable)
		assert.False(t, shouldDelete)
		assert.NotEmpty(t, tags)
	})

	t.Run("Local Model", func(t *testing.T) {
		tags, shouldDisable, shouldDelete, err := client.GetModelTags("llama2", localProvider)
		assert.NoError(t, err)
		assert.False(t, shouldDisable)
		assert.False(t, shouldDelete)
		assert.NotEmpty(t, tags)
	})

	t.Run("Incompatible Model", func(t *testing.T) {
		errorServer := MockErrorHTTPServer(t, 400, "This model requires that either input content or output modality contain audio")
		defer errorServer.Close()

		p := config.ProviderConfig{
			Name:    "openai",
			Type:    config.ProviderTypeOpenAICompatible,
			BaseURL: errorServer.URL,
			APIKey:  "test-api-key",
			Timeout: 5 * time.Second,
		}
		errorClient := NewClient(&config.Config{Providers: []config.ProviderConfig{p}})

		tags, shouldDisable, shouldDelete, err := errorClient.GetModelTags("audio-model", p)
		assert.NoError(t, err)
		assert.True(t, shouldDisable)
		assert.False(t, shouldDelete)
		assert.Contains(t, tags, "unavailable")
	})

	t.Run("Non Chat Model", func(t *testing.T) {
		errorServer := MockErrorHTTPServer(t, 404, "This is not a chat model")
		defer errorServer.Close()

		p := config.ProviderConfig{
			Name:    "openai",
			Type:    config.ProviderTypeOpenAICompatible,
			BaseURL: errorServer.URL,
			APIKey:  "test-api-key",
			Timeout: 5 * time.Second,
		}
		errorClient := NewClient(&config.Config{Providers: []config.ProviderConfig{p}})

		_, shouldDisable, shouldDelete, err := errorClient.GetModelTags("non-chat-model", p)
		assert.NoError(t, err)
		assert.False(t, shouldDisable)
		assert.True(t, shouldDelete)
	})
}

func TestDisableModelsWithEmptyTags(t *testing.T) {
	mockDB := new(MockDatabase)

	// Setup the database mock with test data
	modelMap := map[string]map[string]interface{}{
		"model1": {
			"source": "openai",
			"tags":   `{"strengths":["general"], "weaknesses":[]}`,
		},
		"model2": {
			"source": "local",
			"tags":   `{}`, // Empty tags - should be disabled
		},
		"model3": {
			"source": "openai",
			"tags":   `{"strengths":[], "weaknesses":[]}`, // Empty arrays - should be disabled
		},
		"model4": {
			"source": "local",
			"tags":   `invalid json`, // Invalid JSON - should be disabled
		},
	}

	mockDB.On("GetAllModels").Return(modelMap, nil)
	mockDB.On("EnableModel", "model2", "local", false).Return(nil)
	mockDB.On("EnableModel", "model3", "openai", false).Return(nil)
	mockDB.On("EnableModel", "model4", "local", false).Return(nil)

	// Create the client
	cfg := TestConfig()
	client := NewClient(cfg)

	// Call the function
	err := client.DisableModelsWithEmptyTags(mockDB)
	assert.NoError(t, err)

	// Verify all expected methods were called
	mockDB.AssertExpectations(t)
}
