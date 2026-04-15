package server

import (
	"encoding/json"
	"net/http"
)

func (s *ServerTestSuite) TestHandleAuthLogin_NotConfigured() {
	s.server.oauthConfig = nil

	req, _ := http.NewRequest("GET", "/auth/login", nil)
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusNotFound, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleAuthCallback_NotConfigured() {
	s.server.oauthConfig = nil

	req, _ := http.NewRequest("GET", "/auth/callback", nil)
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusNotFound, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleAuthLogout() {
	req, _ := http.NewRequest("POST", "/auth/logout", nil)
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)

	// Should clear the session cookie
	cookies := s.recorder.Result().Cookies()
	for _, c := range cookies {
		if c.Name == sessionCookieName {
			s.True(c.MaxAge < 0, "session cookie should be expired")
		}
	}
}

func (s *ServerTestSuite) TestHandleAuthMe_Unauthenticated() {
	req, _ := http.NewRequest("GET", "/auth/me", nil)
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)

	var resp map[string]interface{}
	s.Require().NoError(json.Unmarshal(s.recorder.Body.Bytes(), &resp))
	s.Equal(false, resp["authenticated"])
}
