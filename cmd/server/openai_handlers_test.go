package server

import (
	"bytes"
	"encoding/json"
	"net/http"

	"PiPiMink/internal/models"
)

// TestHandleOpenAIChatCompletions tests the OpenAI-compatible chat completions endpoint
func (s *ServerTestSuite) TestHandleOpenAIChatCompletions() {
	// Setup test models
	modelMap := map[string]map[string]interface{}{
		"gpt-4-turbo": {
			"source":     "openai",
			"tags":       `{"capabilities":["general"]}`,
			"response":   "Model description",
			"enabled":    true,
			"updated_at": "2023-05-01",
		},
	}

	// Configure mock behavior
	s.mockDB.On("GetAllModels").Return(modelMap, nil)

	// We need to specify modelInfo parameter more precisely for ChatWithModel
	modelInfo := models.ModelInfo{
		Source:    "openai",
		Tags:      `{"capabilities":["general"]}`,
		Enabled:   true,
		Response:  "Model description",
		UpdatedAt: "2023-05-01",
	}

	userMessage := "Tell me a joke"
	expectedMessages := []map[string]interface{}{{"role": "user", "content": userMessage}}
	s.mockLLM.On("ChatWithModel", modelInfo, "gpt-4-turbo", expectedMessages).Return("Why did the chicken cross the road? To get to the other side!", nil)

	// Load models into server
	collection := models.NewModelCollection()
	collection.FromDatabaseMap(modelMap)
	s.server.modelCollection = collection

	// Create OpenAI-compatible request
	reqBody := map[string]interface{}{
		"model": "gpt-4-turbo",
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": userMessage,
			},
		},
	}
	reqBytes, err := json.Marshal(reqBody)
	s.Require().NoError(err)

	req, err := http.NewRequest("POST", "/v1/chat/completions", bytes.NewBuffer(reqBytes))
	s.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")

	// Process the request
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	// Check response
	s.Equal(http.StatusOK, s.recorder.Code)

	var response map[string]interface{}
	err = json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.Require().NoError(err)

	// Check key fields in OpenAI format
	s.Equal("gpt-4-turbo", response["model"])
	s.Equal("chat.completion", response["object"])

	// Check choices array
	choices, ok := response["choices"].([]interface{})
	s.True(ok, "Response should have a 'choices' array")
	s.Len(choices, 1, "Should have one choice")

	// Check first choice
	choice, ok := choices[0].(map[string]interface{})
	s.True(ok, "Choice should be an object")

	message, ok := choice["message"].(map[string]interface{})
	s.True(ok, "Message should be an object")
	s.Equal("assistant", message["role"])
	s.Equal("Why did the chicken cross the road? To get to the other side!", message["content"])
}

// TestHandleOpenAIModels tests the OpenAI-compatible models endpoint
func (s *ServerTestSuite) TestHandleOpenAIModels() {
	// Setup test models
	modelMap := map[string]map[string]interface{}{
		"gpt-4-turbo": {
			"source":     "openai",
			"tags":       `{"capabilities":["general"]}`,
			"response":   "Model description",
			"enabled":    true,
			"updated_at": "2023-05-01",
		},
		"gpt-4": {
			"source":     "openai",
			"tags":       `{"capabilities":["general","coding"]}`,
			"response":   "Advanced model",
			"enabled":    true,
			"updated_at": "2023-05-02",
		},
		"disabled-model": {
			"source":     "local",
			"tags":       `{"capabilities":["general"]}`,
			"response":   "Disabled model",
			"enabled":    false,
			"updated_at": "2023-05-03",
		},
	}

	// Load models into server
	collection := models.NewModelCollection()
	collection.FromDatabaseMap(modelMap)
	s.server.modelCollection = collection

	// Create request
	req, err := http.NewRequest("GET", "/v1/models", nil)
	s.Require().NoError(err)

	// Process the request
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	// Check response
	s.Equal(http.StatusOK, s.recorder.Code)

	var response map[string]interface{}
	err = json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.Require().NoError(err)

	// Check response structure
	s.Equal("list", response["object"])

	// Check data array
	data, ok := response["data"].([]interface{})
	s.True(ok, "Response should have a 'data' array")

	// Should have 2 enabled models
	s.Len(data, 2, "Should have two enabled models")

	// Check that both enabled models are in the response
	modelNames := make([]string, 0)
	for _, item := range data {
		model, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		name, ok := model["id"].(string)
		if ok {
			modelNames = append(modelNames, name)
		}
	}

	s.Contains(modelNames, "gpt-4-turbo")
	s.Contains(modelNames, "gpt-4")
	s.NotContains(modelNames, "disabled-model")
}
