package server

import (
	"bytes"
	"encoding/json"
	"net/http"

	"PiPiMink/internal/database"

	"github.com/stretchr/testify/mock"
)

func (s *ServerTestSuite) TestHandleGetAuthProviders() {
	providers := []database.AuthProviderRow{
		{ID: "prov-1", Type: "oauth", Name: "Authentik", Status: "connected"},
		{ID: "prov-2", Type: "ldap", Name: "Corporate LDAP", Status: "not_configured"},
	}
	s.mockDB.On("GetAuthProviders").Return(providers, nil)

	req, _ := http.NewRequest("GET", "/admin/auth/providers", nil)
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)
	var resp []database.AuthProviderRow
	s.Require().NoError(json.Unmarshal(s.recorder.Body.Bytes(), &resp))
	s.Len(resp, 2)
}

func (s *ServerTestSuite) TestHandleGetAuthProviders_DBError() {
	s.mockDB.On("GetAuthProviders").Return([]database.AuthProviderRow{}, errTest)

	req, _ := http.NewRequest("GET", "/admin/auth/providers", nil)
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusInternalServerError, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleSaveAuthProvider() {
	s.mockDB.On("SaveAuthProvider", mock.Anything).Return(nil)

	body, _ := json.Marshal(database.AuthProviderRow{Name: "Authentik", Type: "oauth"})
	req, _ := http.NewRequest("PUT", "/admin/auth/providers/prov-1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleGetUsers() {
	users := []database.UserRow{
		{ID: "user-1", Name: "Alice", Email: "alice@example.com", Role: "admin", Groups: []string{}},
	}
	s.mockDB.On("GetUsers").Return(users, nil)

	req, _ := http.NewRequest("GET", "/admin/auth/users", nil)
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)
	var resp []database.UserRow
	s.Require().NoError(json.Unmarshal(s.recorder.Body.Bytes(), &resp))
	s.Len(resp, 1)
}

func (s *ServerTestSuite) TestHandleGetUsers_Empty() {
	s.mockDB.On("GetUsers").Return([]database.UserRow{}, nil)

	req, _ := http.NewRequest("GET", "/admin/auth/users", nil)
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)
	s.Contains(s.recorder.Body.String(), "[]")
}

func (s *ServerTestSuite) TestHandleAddLocalUser() {
	s.mockDB.On("UpsertUser", mock.Anything).Return(nil)

	body, _ := json.Marshal(map[string]string{"name": "Bob", "email": "bob@example.com", "role": "user"})
	req, _ := http.NewRequest("POST", "/admin/auth/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusCreated, s.recorder.Code)
	var resp database.UserRow
	s.Require().NoError(json.Unmarshal(s.recorder.Body.Bytes(), &resp))
	s.Equal("Bob", resp.Name)
	s.Equal("bob@example.com", resp.Email)
}

func (s *ServerTestSuite) TestHandleAddLocalUser_MissingFields() {
	body, _ := json.Marshal(map[string]string{"name": "", "email": ""})
	req, _ := http.NewRequest("POST", "/admin/auth/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusBadRequest, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleChangeUserRole() {
	s.mockDB.On("ChangeUserRole", "user-abc", "admin").Return(nil)

	body, _ := json.Marshal(map[string]string{"role": "admin"})
	req, _ := http.NewRequest("PUT", "/admin/auth/users/user-abc/role", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleChangeUserRole_InvalidRole() {
	body, _ := json.Marshal(map[string]string{"role": "superadmin"})
	req, _ := http.NewRequest("PUT", "/admin/auth/users/user-abc/role", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusBadRequest, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleDeleteUser() {
	s.mockDB.On("DeleteUser", "user-abc").Return(nil)

	body, _ := json.Marshal(map[string]string{"reason": "GDPR request"})
	req, _ := http.NewRequest("DELETE", "/admin/auth/users/user-abc", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleDeleteUser_MissingReason() {
	body, _ := json.Marshal(map[string]string{"reason": ""})
	req, _ := http.NewRequest("DELETE", "/admin/auth/users/user-abc", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusBadRequest, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleGetGroups() {
	groups := []database.GroupRow{
		{ID: "group-1", Name: "Engineering", Role: "admin", RoutingRules: []database.RoutingRuleRow{}},
	}
	s.mockDB.On("GetGroups").Return(groups, nil)

	req, _ := http.NewRequest("GET", "/admin/auth/groups", nil)
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleChangeGroupRole() {
	s.mockDB.On("ChangeGroupRole", "group-1", "user").Return(nil)

	body, _ := json.Marshal(map[string]string{"role": "user"})
	req, _ := http.NewRequest("PUT", "/admin/auth/groups/group-1/role", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleAddRoutingRule() {
	s.mockDB.On("SaveRoutingRule", "group-1", mock.Anything).Return(nil)

	body, _ := json.Marshal(database.RoutingRuleRow{Type: "allow", Providers: []string{"openai"}, Description: "Allow OpenAI"})
	req, _ := http.NewRequest("POST", "/admin/auth/groups/group-1/rules", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusCreated, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleRemoveRoutingRule() {
	s.mockDB.On("DeleteRoutingRule", "group-1", "rule-1").Return(nil)

	req, _ := http.NewRequest("DELETE", "/admin/auth/groups/group-1/rules/rule-1", nil)
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleGetAuditLog() {
	entries := []database.AuditEntryRow{
		{ID: "audit-1", Actor: "Admin", Action: "user_created", Target: "Alice"},
	}
	s.mockDB.On("GetAuditLog").Return(entries, nil)

	req, _ := http.NewRequest("GET", "/admin/auth/audit-log", nil)
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)
}
