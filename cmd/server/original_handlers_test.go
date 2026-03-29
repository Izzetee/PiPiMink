package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"PiPiMink/internal/models"
)

// TestHandleUpdateModels tests the models/update endpoint
func (s *ServerTestSuite) TestHandleUpdateModels() {
	// s.config.Providers is empty, so fetchAndTagModels iterates nothing.
	// Just mock the database reload that happens in the post-update goroutine.
	emptyModelMap := map[string]map[string]interface{}{}
	s.mockDB.On("GetAllModels").Return(emptyModelMap, nil)

	// Create a request with admin API key
	req, err := http.NewRequest("POST", "/models/update", nil)
	s.Require().NoError(err)
	req.Header.Set("X-API-Key", s.config.AdminAPIKey)

	// Process the request
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	// Check response
	s.Equal(http.StatusOK, s.recorder.Code)

	var response map[string]string
	err = json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.Require().NoError(err)

	s.Equal("success", response["status"])
	s.Equal("Model update process started", response["message"])

	// Test unauthorized access
	s.recorder = httptest.NewRecorder() // Reset recorder
	req.Header.Set("X-API-Key", "wrong-key")

	// Process the request with invalid API key
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	// Check unauthorized response
	s.Equal(http.StatusUnauthorized, s.recorder.Code)
}

// TestHandleListModels tests the /models endpoint
func (s *ServerTestSuite) TestHandleListModels() {
	// Setup test models
	modelMap := map[string]map[string]interface{}{
		"gpt-4-turbo": {
			"source":     "openai",
			"tags":       `{"capabilities":["general"]}`,
			"response":   "Model description",
			"enabled":    true,
			"updated_at": "2023-05-01",
		},
		"llama2": {
			"source":     "local",
			"tags":       `{"capabilities":["general"]}`,
			"response":   "Model description",
			"enabled":    true,
			"updated_at": "2023-05-02",
		},
	}

	// Load models into server
	collection := models.NewModelCollection()
	collection.FromDatabaseMap(modelMap)
	s.server.modelCollection = collection

	// Create request
	req, err := http.NewRequest("GET", "/models", nil)
	s.Require().NoError(err)

	// Process the request
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	// Check response
	s.Equal(http.StatusOK, s.recorder.Code)

	var response map[string]interface{}
	err = json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.Require().NoError(err)

	// Check response structure
	modelsList, ok := response["models"].([]interface{})
	s.True(ok, "Response should have a 'models' array")
	s.Len(modelsList, 2, "Should have two models")
	s.Equal(float64(2), response["count"])

	// Verify model details in the response
	foundGPT := false
	foundLlama := false

	for _, item := range modelsList {
		model, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		name, ok := model["name"].(string)
		if !ok {
			continue
		}

		switch name {
		case "gpt-4-turbo":
			s.Equal("openai", model["source"])
			s.Equal(true, model["enabled"])
			s.Equal("2023-05-01", model["updatedAt"])
			foundGPT = true
		case "llama2":
			s.Equal("local", model["source"])
			s.Equal(true, model["enabled"])
			s.Equal("2023-05-02", model["updatedAt"])
			foundLlama = true
		}
	}

	s.True(foundGPT, "Should find gpt-4-turbo in response")
	s.True(foundLlama, "Should find llama2 in response")
}
