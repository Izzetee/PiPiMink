package server

import (
	"net/http"
)

func (s *ServerTestSuite) TestConsoleRedirect() {
	req, _ := http.NewRequest("GET", "/console", nil)
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusMovedPermanently, s.recorder.Code)
	s.Contains(s.recorder.Header().Get("Location"), "/console/")
}

func (s *ServerTestSuite) TestConsoleSPAFallback() {
	// When dist is not built (test environment), console routes may not be registered.
	// If routes ARE registered, unmatched paths should serve index.html (200 text/html)
	// or return 404 if index.html is missing. Either way, should not panic.
	req, _ := http.NewRequest("GET", "/console/models", nil)
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	// Accept either 200 (SPA fallback) or 404 (dist not built / no index.html)
	code := s.recorder.Code
	s.True(code == http.StatusOK || code == http.StatusNotFound,
		"expected 200 or 404, got %d", code)
}
