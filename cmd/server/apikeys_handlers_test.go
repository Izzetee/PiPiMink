package server

import (
	"bytes"
	"encoding/json"
	"net/http"

	"PiPiMink/internal/config"
)

func (s *ServerTestSuite) TestHandleListApiKeys_Unauthorized() {
	req, _ := http.NewRequest("GET", "/admin/api-keys", nil)
	// No API key
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusUnauthorized, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleListApiKeys() {
	s.server.config.Providers = []config.ProviderConfig{
		{Name: "openai", Type: "openai-compatible", APIKeyEnv: "OPENAI_API_KEY"},
	}

	req, _ := http.NewRequest("GET", "/admin/api-keys", nil)
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)
	// Should return JSON array
	s.Contains(s.recorder.Body.String(), "[")
}

func (s *ServerTestSuite) TestHandleSetApiKey_MissingValue() {
	body, _ := json.Marshal(map[string]string{"value": ""})
	req, _ := http.NewRequest("PUT", "/admin/api-keys/OPENAI_API_KEY", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusBadRequest, s.recorder.Code)
}
