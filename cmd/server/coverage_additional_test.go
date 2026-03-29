package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"PiPiMink/internal/models"
)

func (s *ServerTestSuite) TestGetFallbackModelName() {
	enabledModels := map[string]models.ModelInfo{
		"gpt-4-turbo": {Enabled: true},
		"llama3":      {Enabled: true},
	}

	s.server.config.DefaultChatModel = "gpt-4-turbo"
	selected := s.server.getFallbackModelName(enabledModels)
	s.Equal("gpt-4-turbo", selected)

	s.server.config.DefaultChatModel = "does-not-exist"
	selected = s.server.getFallbackModelName(enabledModels)
	_, exists := enabledModels[selected]
	s.True(exists, "fallback should pick one of the enabled models")

	s.Equal("", s.server.getFallbackModelName(map[string]models.ModelInfo{}))
}

func (s *ServerTestSuite) TestHandleOpenAIModelsUsesConfiguredDefaultWhenEmpty() {
	s.server.modelCollection = models.NewModelCollection()
	s.server.config.DefaultChatModel = "my-fast-router-model"

	req, err := http.NewRequest("GET", "/v1/models", nil)
	s.Require().NoError(err)

	s.server.GetRouter().ServeHTTP(s.recorder, req)
	s.Equal(http.StatusOK, s.recorder.Code)

	var response map[string]interface{}
	err = json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.Require().NoError(err)

	data, ok := response["data"].([]interface{})
	s.True(ok)
	s.Len(data, 1)

	model, ok := data[0].(map[string]interface{})
	s.True(ok)
	s.Equal("my-fast-router-model", model["id"])
}

func (s *ServerTestSuite) TestHandleUpdateModelReasoning() {
	collection := models.NewModelCollection()
	collection.AddModel("gpt-4-turbo", models.ModelInfo{Source: "openai", Enabled: true})
	s.server.modelCollection = collection

	reqBody := map[string]interface{}{
		"model":         "gpt-4-turbo",
		"source":        "openai",
		"has_reasoning": true,
	}
	reqBytes, err := json.Marshal(reqBody)
	s.Require().NoError(err)

	s.mockDB.On("UpdateModelReasoning", "gpt-4-turbo", "openai", true).Return(nil).Once()

	req, err := http.NewRequest("POST", "/models/reasoning/update", bytes.NewBuffer(reqBytes))
	s.Require().NoError(err)
	req.Header.Set("X-API-Key", s.config.AdminAPIKey)
	req.Header.Set("Content-Type", "application/json")

	s.server.GetRouter().ServeHTTP(s.recorder, req)
	s.Equal(http.StatusOK, s.recorder.Code)

	updated, exists := s.server.modelCollection.GetModel("gpt-4-turbo")
	s.True(exists)
	s.True(updated.HasReasoning)
}

func (s *ServerTestSuite) TestHandleUpdateModelReasoningFailures() {
	reqBody := map[string]interface{}{
		"model":         "gpt-4-turbo",
		"source":        "openai",
		"has_reasoning": true,
	}
	reqBytes, err := json.Marshal(reqBody)
	s.Require().NoError(err)

	unauthorizedReq, err := http.NewRequest("POST", "/models/reasoning/update", bytes.NewBuffer(reqBytes))
	s.Require().NoError(err)
	unauthorizedReq.Header.Set("X-API-Key", "wrong-key")
	unauthorizedReq.Header.Set("Content-Type", "application/json")

	s.server.GetRouter().ServeHTTP(s.recorder, unauthorizedReq)
	s.Equal(http.StatusUnauthorized, s.recorder.Code)

	s.recorder = httptest.NewRecorder()
	s.mockDB.On("UpdateModelReasoning", "gpt-4-turbo", "openai", true).Return(errors.New("db down")).Once()

	dbFailReq, err := http.NewRequest("POST", "/models/reasoning/update", bytes.NewBuffer(reqBytes))
	s.Require().NoError(err)
	dbFailReq.Header.Set("X-API-Key", s.config.AdminAPIKey)
	dbFailReq.Header.Set("Content-Type", "application/json")

	s.server.GetRouter().ServeHTTP(s.recorder, dbFailReq)
	s.Equal(http.StatusInternalServerError, s.recorder.Code)
}
