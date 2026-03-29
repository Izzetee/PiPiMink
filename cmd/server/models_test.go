package server

import (
	"testing"

	"PiPiMink/internal/config"
	"PiPiMink/internal/models"

	"github.com/stretchr/testify/assert"
)

// TestFetchAndTagModels tests the function for fetching and tagging models
func TestFetchAndTagModels(t *testing.T) {
	// Skip test as it would require real API calls
	t.Skip("Skipping as it requires actual API calls")
}

// TestLoadModelsFromDatabase tests loading models from the database
func TestLoadModelsFromDatabase(t *testing.T) {
	// Skip test as it would require a real database connection
	t.Skip("Skipping as it requires actual database connection")
}

// TestGetHandler was removed as there is no GetHandler method in the Server

// TestModelFunctionality tests various model management functions
func TestModelFunctionality(t *testing.T) {
	// Create server with test data
	cfg := &config.Config{
		AdminAPIKey: "test-admin-key",
		Port:        "8080",
	}

	server := &Server{
		config:          cfg,
		router:          nil, // Not needed for this test
		modelCollection: models.NewModelCollection(),
	}

	// Add models
	server.modelCollection.AddModel("model1", models.ModelInfo{
		Source:    "openai",
		Enabled:   true,
		Tags:      `{"capabilities":["general"]}`,
		Response:  "Model 1 description",
		UpdatedAt: "2023-05-01",
	})

	server.modelCollection.AddModel("model2", models.ModelInfo{
		Source:    "local",
		Enabled:   false,
		Tags:      `{"capabilities":["coding"]}`,
		Response:  "Model 2 description",
		UpdatedAt: "2023-05-02",
	})

	// Tests for various model functions

	// logModels should not cause any errors
	server.logModels()

	// Test model selection logic (manually)
	// This is more of a functional test than a true unit test
	enabledModels := server.modelCollection.GetEnabledModels()
	assert.Len(t, enabledModels, 1)
	_, exists := enabledModels["model1"]
	assert.True(t, exists)
}
