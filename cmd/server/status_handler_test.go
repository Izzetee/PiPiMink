package server

import (
	"encoding/json"
	"net/http"

	"PiPiMink/internal/config"
	"PiPiMink/internal/models"
)

func (s *ServerTestSuite) TestHandleAdminStatus_AllConfigured() {
	// Set up a fully configured instance
	s.server.config.AdminAPIKey = "test-admin-key"
	s.server.config.Providers = []config.ProviderConfig{
		{Name: "openai", Type: "openai-compatible", BaseURL: "https://api.openai.com"},
	}
	s.server.config.OAuthClientID = "pipimink"
	s.server.config.OAuthIssuerURL = "http://localhost:9000/application/o/pipimink/"
	s.server.config.OAuthClientSecret = "secret"
	s.server.config.OAuthRedirectURL = "http://localhost:8080/auth/callback"

	// Load models
	collection := models.NewModelCollection()
	collection.AddModel("gpt-4", models.ModelInfo{Source: "openai", Enabled: true})
	s.server.modelCollection = collection

	req, err := http.NewRequest("GET", "/admin/status", nil)
	s.Require().NoError(err)

	s.server.GetRouter().ServeHTTP(s.recorder, req)
	s.Equal(http.StatusOK, s.recorder.Code)

	var resp adminStatusResponse
	err = json.Unmarshal(s.recorder.Body.Bytes(), &resp)
	s.Require().NoError(err)

	s.True(resp.AdminKeyConfigured)
	s.Equal(1, resp.ProviderCount)
	s.Equal(1, resp.ModelCount)
	s.True(resp.OAuthEnabled)
}

func (s *ServerTestSuite) TestHandleAdminStatus_Unconfigured() {
	// Empty config — the "needs setup" state
	s.server.config.AdminAPIKey = ""
	s.server.config.Providers = nil
	s.server.modelCollection = models.NewModelCollection()

	req, err := http.NewRequest("GET", "/admin/status", nil)
	s.Require().NoError(err)

	s.server.GetRouter().ServeHTTP(s.recorder, req)
	s.Equal(http.StatusOK, s.recorder.Code)

	var resp adminStatusResponse
	err = json.Unmarshal(s.recorder.Body.Bytes(), &resp)
	s.Require().NoError(err)

	s.False(resp.AdminKeyConfigured)
	s.Equal(0, resp.ProviderCount)
	s.Equal(0, resp.ModelCount)
	s.False(resp.OAuthEnabled)
}

func (s *ServerTestSuite) TestHandleAdminStatus_PartialConfig() {
	// Admin key set but no models, no OAuth
	s.server.config.AdminAPIKey = "test-admin-key"
	s.server.config.Providers = []config.ProviderConfig{
		{Name: "openai", Type: "openai-compatible"},
		{Name: "anthropic", Type: "anthropic"},
	}
	s.server.modelCollection = models.NewModelCollection()

	req, err := http.NewRequest("GET", "/admin/status", nil)
	s.Require().NoError(err)

	s.server.GetRouter().ServeHTTP(s.recorder, req)
	s.Equal(http.StatusOK, s.recorder.Code)

	var resp adminStatusResponse
	err = json.Unmarshal(s.recorder.Body.Bytes(), &resp)
	s.Require().NoError(err)

	s.True(resp.AdminKeyConfigured)
	s.Equal(2, resp.ProviderCount)
	s.Equal(0, resp.ModelCount)
	s.False(resp.OAuthEnabled)
}
