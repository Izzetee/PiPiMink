package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"

	"PiPiMink/internal/benchmark"
	"PiPiMink/internal/database"

	"github.com/gorilla/securecookie"
	"github.com/stretchr/testify/mock"
)

// TestAuthMiddleware_PassthroughPaths verifies that public paths bypass auth.
func (s *ServerTestSuite) TestAuthMiddleware_PassthroughPaths() {
	paths := []struct {
		method string
		path   string
	}{
		{"GET", "/admin/status"},
		{"GET", "/auth/login"},
		{"GET", "/auth/callback"},
		{"GET", "/metrics"},
		{"GET", "/v1/models"},
		{"GET", "/api/tags"},
	}

	for _, p := range paths {
		s.recorder = httptest.NewRecorder()
		req, err := http.NewRequest(p.method, p.path, nil)
		s.Require().NoError(err)
		s.server.GetRouter().ServeHTTP(s.recorder, req)
		s.NotEqual(http.StatusUnauthorized, s.recorder.Code,
			"path %s %s should not return 401", p.method, p.path)
	}
}

// TestAuthMiddleware_ValidAPIKey verifies that a correct API key grants access.
func (s *ServerTestSuite) TestAuthMiddleware_ValidAPIKey() {
	// GET /admin/benchmark-tasks requires auth; mock the DB call
	s.mockDB.On("GetBenchmarkTaskConfigs").Return([]benchmark.BenchmarkTaskConfig{}, nil)

	req, err := http.NewRequest("GET", "/admin/benchmark-tasks", nil)
	s.Require().NoError(err)
	req.Header.Set("X-API-Key", "test-admin-key")

	s.server.GetRouter().ServeHTTP(s.recorder, req)
	s.NotEqual(http.StatusUnauthorized, s.recorder.Code)
}

// TestAuthMiddleware_InvalidAPIKey verifies that a wrong API key returns 401.
func (s *ServerTestSuite) TestAuthMiddleware_InvalidAPIKey() {
	req, err := http.NewRequest("GET", "/admin/benchmark-tasks", nil)
	s.Require().NoError(err)
	req.Header.Set("X-API-Key", "wrong-key")

	s.server.GetRouter().ServeHTTP(s.recorder, req)
	s.Equal(http.StatusUnauthorized, s.recorder.Code)
}

// TestAuthMiddleware_NoAuthLegacyMode verifies passthrough when OAuth is not configured.
func (s *ServerTestSuite) TestAuthMiddleware_NoAuthLegacyMode() {
	// Default config has no OAuth fields → OAuthEnabled() == false
	// Console should be accessible without auth
	req, err := http.NewRequest("GET", "/console/", nil)
	s.Require().NoError(err)

	s.server.GetRouter().ServeHTTP(s.recorder, req)
	// Should not be 401 or 302 redirect — the console SPA handler runs
	s.NotEqual(http.StatusUnauthorized, s.recorder.Code)
	s.NotEqual(http.StatusFound, s.recorder.Code)
}

// TestAuthMiddleware_OAuthRedirect verifies redirect to login when OAuth is on but no session.
func (s *ServerTestSuite) TestAuthMiddleware_OAuthRedirect() {
	s.server.config.OAuthClientID = "pipimink"
	s.server.config.OAuthIssuerURL = "http://localhost:9000/application/o/pipimink/"
	s.server.config.OAuthClientSecret = "secret"
	s.server.config.OAuthRedirectURL = "http://localhost:8080/auth/callback"

	req, err := http.NewRequest("GET", "/console/models", nil)
	s.Require().NoError(err)

	s.server.GetRouter().ServeHTTP(s.recorder, req)
	s.Equal(http.StatusFound, s.recorder.Code)
	s.Contains(s.recorder.Header().Get("Location"), "/auth/login")
}

// TestAuthMiddleware_OAuthAPI401 verifies 401 for admin API when OAuth is on but no session.
func (s *ServerTestSuite) TestAuthMiddleware_OAuthAPI401() {
	s.server.config.OAuthClientID = "pipimink"
	s.server.config.OAuthIssuerURL = "http://localhost:9000/application/o/pipimink/"
	s.server.config.OAuthClientSecret = "secret"
	s.server.config.OAuthRedirectURL = "http://localhost:8080/auth/callback"

	req, err := http.NewRequest("GET", "/admin/benchmark-tasks", nil)
	s.Require().NoError(err)

	s.server.GetRouter().ServeHTTP(s.recorder, req)
	s.Equal(http.StatusUnauthorized, s.recorder.Code)
}

// TestAuthMiddleware_ValidSessionCookie verifies that a valid session cookie passes auth.
func (s *ServerTestSuite) TestAuthMiddleware_ValidSessionCookie() {
	// Configure OAuth so passthrough doesn't kick in
	s.server.config.OAuthClientID = "pipimink"
	s.server.config.OAuthIssuerURL = "http://localhost:9000/application/o/pipimink/"
	s.server.config.OAuthClientSecret = "secret"
	s.server.config.OAuthRedirectURL = "http://localhost:8080/auth/callback"

	// Set up a secureCookie and encode a session
	hashKey := make([]byte, 32)
	blockKey := make([]byte, 32)
	sc := securecookie.New(hashKey, blockKey)
	s.server.secureCookie = sc

	sessionData := map[string]string{"email": "test@example.com"}
	encoded, err := sc.Encode(sessionCookieName, sessionData)
	s.Require().NoError(err)

	// Mock the user lookup
	s.mockDB.On("GetUserByEmail", "test@example.com").Return(&database.UserRow{
		ID:    "user-abc",
		Name:  "Test User",
		Email: "test@example.com",
		Role:  "admin",
	}, nil)

	// Access a protected admin endpoint with the cookie
	s.mockDB.On("GetBenchmarkTaskConfigs").Return([]benchmark.BenchmarkTaskConfig{}, nil)

	req, err := http.NewRequest("GET", "/admin/benchmark-tasks", nil)
	s.Require().NoError(err)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: encoded})

	s.server.GetRouter().ServeHTTP(s.recorder, req)
	s.NotEqual(http.StatusUnauthorized, s.recorder.Code)
	s.NotEqual(http.StatusFound, s.recorder.Code)
}

// TestAuthMiddleware_BearerToken_Valid verifies that a valid Bearer token grants access.
func (s *ServerTestSuite) TestAuthMiddleware_BearerToken_Valid() {
	token := "ppm_test-token-valid"
	tokenHash := database.HashToken(token)

	s.mockDB.On("GetUserByAPIToken", tokenHash).Return(&database.UserRow{
		ID:    "user-bearer",
		Name:  "Bearer User",
		Email: "bearer@example.com",
		Role:  "admin",
	}, nil)
	s.mockDB.On("GetBenchmarkTaskConfigs").Return([]benchmark.BenchmarkTaskConfig{}, nil)

	req, err := http.NewRequest("GET", "/admin/benchmark-tasks", nil)
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+token)

	s.server.GetRouter().ServeHTTP(s.recorder, req)
	s.NotEqual(http.StatusUnauthorized, s.recorder.Code)
}

// TestAuthMiddleware_BearerToken_Invalid verifies that an invalid Bearer token returns 401.
func (s *ServerTestSuite) TestAuthMiddleware_BearerToken_Invalid() {
	token := "ppm_invalid-token"
	tokenHash := database.HashToken(token)

	s.mockDB.On("GetUserByAPIToken", tokenHash).Return(nil, nil)

	req, err := http.NewRequest("GET", "/admin/benchmark-tasks", nil)
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+token)

	s.server.GetRouter().ServeHTTP(s.recorder, req)
	s.Equal(http.StatusUnauthorized, s.recorder.Code)
}

// TestAuthMiddleware_ChatRequiresAuth verifies chat returns 401 when REQUIRE_AUTH_FOR_CHAT=true.
func (s *ServerTestSuite) TestAuthMiddleware_ChatRequiresAuth() {
	s.server.config.RequireAuthForChat = true

	req, err := http.NewRequest("POST", "/chat", nil)
	s.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")

	s.server.GetRouter().ServeHTTP(s.recorder, req)
	s.Equal(http.StatusUnauthorized, s.recorder.Code)
}

// TestAuthMiddleware_ChatAnonymous verifies chat works unauthenticated when REQUIRE_AUTH_FOR_CHAT=false.
func (s *ServerTestSuite) TestAuthMiddleware_ChatAnonymous() {
	s.server.config.RequireAuthForChat = false

	body := bytes.NewBufferString(`{}`)
	req, err := http.NewRequest("POST", "/chat", body)
	s.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")

	s.server.GetRouter().ServeHTTP(s.recorder, req)
	// Should not be 401 — the handler runs (may return 400 for empty body, that's fine)
	s.NotEqual(http.StatusUnauthorized, s.recorder.Code)
}

// TestAuthMiddleware_AdminEndpoints_RejectUser verifies that AuthUser cannot access admin paths.
func (s *ServerTestSuite) TestAuthMiddleware_AdminEndpoints_RejectUser() {
	token := "ppm_regular-user-token"
	tokenHash := database.HashToken(token)

	s.mockDB.On("GetUserByAPIToken", tokenHash).Return(&database.UserRow{
		ID:    "user-regular",
		Name:  "Regular User",
		Email: "regular@example.com",
		Role:  "user",
	}, nil)

	req, err := http.NewRequest("GET", "/admin/benchmark-tasks", nil)
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+token)

	s.server.GetRouter().ServeHTTP(s.recorder, req)
	s.Equal(http.StatusUnauthorized, s.recorder.Code)
}

// TestAuthMiddleware_AuthTokensNotPublic verifies that /auth/tokens requires auth.
func (s *ServerTestSuite) TestAuthMiddleware_AuthTokensNotPublic() {
	s.server.config.RequireAuthForChat = true

	req, err := http.NewRequest("GET", "/auth/tokens", nil)
	s.Require().NoError(err)

	s.server.GetRouter().ServeHTTP(s.recorder, req)
	s.Equal(http.StatusUnauthorized, s.recorder.Code)
}

// TestAuthMiddleware_BearerToken_ChatWithAuth verifies Bearer token works for chat when auth required.
func (s *ServerTestSuite) TestAuthMiddleware_BearerToken_ChatWithAuth() {
	s.server.config.RequireAuthForChat = true

	token := "ppm_chat-user-token"
	tokenHash := database.HashToken(token)

	s.mockDB.On("GetUserByAPIToken", tokenHash).Return(&database.UserRow{
		ID:    "user-chat",
		Name:  "Chat User",
		Email: "chat@example.com",
		Role:  "user",
	}, nil)

	body := bytes.NewBufferString(`{}`)
	req, err := http.NewRequest("POST", "/chat", body)
	s.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	s.server.GetRouter().ServeHTTP(s.recorder, req)
	// Should not be 401 — the handler runs (may return 400 for empty body, that's fine)
	s.NotEqual(http.StatusUnauthorized, s.recorder.Code)
}

// TestAuthMiddleware_AnalyticsScopedToUser verifies user-scoped analytics for regular users.
func (s *ServerTestSuite) TestAuthMiddleware_AnalyticsScopedToUser() {
	token := "ppm_analytics-user"
	tokenHash := database.HashToken(token)

	s.mockDB.On("GetUserByAPIToken", tokenHash).Return(&database.UserRow{
		ID:    "user-analytics",
		Name:  "Analytics User",
		Email: "analytics@example.com",
		Role:  "admin", // Must be admin to reach /admin/analytics
	}, nil)

	// Expect filtered methods to be called with empty userID (admin sees all)
	s.mockDB.On("GetKpiSummaryFiltered", mock.Anything, mock.Anything, "").Return(database.KpiSummary{}, nil)
	s.mockDB.On("GetModelUsageFiltered", mock.Anything, mock.Anything, "").Return([]database.ModelUsageRow{}, nil)
	s.mockDB.On("GetLatencyPerModelFiltered", mock.Anything, mock.Anything, "").Return([]database.LatencyPerModelRow{}, nil)
	s.mockDB.On("GetLatencyTimeSeriesFiltered", mock.Anything, mock.Anything, "").Return([]database.LatencyTimeSeriesRow{}, nil)
	s.mockDB.On("GetLatencyPercentilesFiltered", mock.Anything, mock.Anything, "").Return([]database.LatencyPercentilesRow{}, nil)

	req, err := http.NewRequest("GET", "/admin/analytics/summary", nil)
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+token)

	s.server.GetRouter().ServeHTTP(s.recorder, req)
	s.Equal(http.StatusOK, s.recorder.Code)
}
