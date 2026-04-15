package server

import (
	"bytes"
	"encoding/json"
	"net/http"

	"PiPiMink/internal/config"

	"github.com/stretchr/testify/mock"
)

func (s *ServerTestSuite) TestHandleListProviders() {
	s.server.config.Providers = []config.ProviderConfig{
		{Name: "openai", Type: "openai-compatible", BaseURL: "https://api.openai.com", Enabled: true},
		{Name: "anthropic", Type: "anthropic", BaseURL: "https://api.anthropic.com", Enabled: true},
	}

	req, _ := http.NewRequest("GET", "/providers", nil)
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)

	var resp map[string]interface{}
	s.Require().NoError(json.Unmarshal(s.recorder.Body.Bytes(), &resp))
	providers := resp["providers"].([]interface{})
	s.Len(providers, 2)
}

func (s *ServerTestSuite) TestHandleListProviders_Unauthorized() {
	req, _ := http.NewRequest("GET", "/providers", nil)
	// No API key
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusUnauthorized, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleAddProvider_MissingFields() {
	body, _ := json.Marshal(map[string]string{"name": ""})
	req, _ := http.NewRequest("POST", "/providers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusBadRequest, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleAddProvider_DuplicateName() {
	s.server.config.Providers = []config.ProviderConfig{
		{Name: "openai", Type: "openai-compatible"},
	}

	body, _ := json.Marshal(map[string]interface{}{
		"name": "openai", "type": "openai-compatible", "base_url": "https://api.openai.com",
	})
	req, _ := http.NewRequest("POST", "/providers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusConflict, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleUpdateProvider_NotFound() {
	s.server.config.Providers = []config.ProviderConfig{}

	body, _ := json.Marshal(map[string]interface{}{"name": "nonexistent", "type": "openai-compatible"})
	req, _ := http.NewRequest("PUT", "/providers/nonexistent", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusNotFound, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleDeleteProvider_NotFound() {
	s.server.config.Providers = []config.ProviderConfig{}

	req, _ := http.NewRequest("DELETE", "/providers/nonexistent", nil)
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusNotFound, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleTestProvider() {
	s.server.config.Providers = []config.ProviderConfig{
		{Name: "openai", Type: "openai-compatible", BaseURL: "https://api.openai.com", Enabled: true},
	}
	s.mockLLM.On("GetModelsByProvider", mock.Anything).Return([]string{"gpt-4", "gpt-4o"}, nil)

	req, _ := http.NewRequest("POST", "/providers/openai/test", nil)
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)
	var resp map[string]interface{}
	s.Require().NoError(json.Unmarshal(s.recorder.Body.Bytes(), &resp))
	s.Equal("success", resp["result"])
	s.Equal(float64(2), resp["models_found"])
}

func (s *ServerTestSuite) TestHandleTestProvider_Error() {
	s.server.config.Providers = []config.ProviderConfig{
		{Name: "openai", Type: "openai-compatible", BaseURL: "https://api.openai.com", Enabled: true},
	}
	s.mockLLM.On("GetModelsByProvider", mock.Anything).Return([]string(nil), errTest)

	req, _ := http.NewRequest("POST", "/providers/openai/test", nil)
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)
	var resp map[string]interface{}
	s.Require().NoError(json.Unmarshal(s.recorder.Body.Bytes(), &resp))
	s.Equal("error", resp["result"])
}
