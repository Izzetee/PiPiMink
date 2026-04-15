package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"PiPiMink/internal/database"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// writeAuditEntry persists an audit entry asynchronously.
func (s *Server) writeAuditEntry(actor, action, target, details string, reason *string) {
	go func() {
		if err := s.db.SaveAuditEntry(database.AuditEntryRow{
			ID:      "audit-" + uuid.New().String()[:8],
			Actor:   actor,
			Action:  action,
			Target:  target,
			Details: details,
			Reason:  reason,
		}); err != nil {
			log.Printf("Error saving audit entry: %v", err)
		}
	}()
}

// actorName returns the display name of the authenticated user for audit logging.
func actorName(r *http.Request) string {
	if u := getUserFromContext(r); u != nil {
		return u.Name
	}
	return "Admin (API Key)"
}

// handleGetAuthProviders returns all configured auth providers.
func (s *Server) handleGetAuthProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := s.db.GetAuthProviders()
	if err != nil {
		log.Printf("Error fetching auth providers: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(providers)
}

// handleSaveAuthProvider saves or updates an auth provider configuration.
func (s *Server) handleSaveAuthProvider(w http.ResponseWriter, r *http.Request) {
	var p database.AuthProviderRow
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	p.ID = mux.Vars(r)["id"]

	if err := s.db.SaveAuthProvider(p); err != nil {
		log.Printf("Error saving auth provider: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	s.writeAuditEntry(actorName(r), "provider_configured", p.Name,
		"Auth provider configuration updated", nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

// handleTestAuthProvider tests connectivity to an auth provider.
func (s *Server) handleTestAuthProvider(w http.ResponseWriter, r *http.Request) {
	providerID := mux.Vars(r)["id"]

	providers, err := s.db.GetAuthProviders()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var provider *database.AuthProviderRow
	for i := range providers {
		if providers[i].ID == providerID {
			provider = &providers[i]
			break
		}
	}
	if provider == nil {
		http.Error(w, "Provider not found", http.StatusNotFound)
		return
	}

	// Test by making HTTP GET to the issuer URL / .well-known endpoint
	testURL := provider.IssuerURL
	if provider.Type == "oauth" && testURL != "" {
		if testURL[len(testURL)-1] != '/' {
			testURL += "/"
		}
		testURL += ".well-known/openid-configuration"
	} else if provider.Type == "ldap" {
		testURL = provider.ServerURL
	}

	if testURL == "" {
		http.Error(w, "No URL configured for this provider", http.StatusBadRequest)
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(testURL)

	now := time.Now()
	result := map[string]interface{}{}

	if err != nil {
		result["status"] = "disconnected"
		result["error"] = err.Error()
		provider.Status = "disconnected"
	} else {
		_ = resp.Body.Close()
		if resp.StatusCode < 400 {
			result["status"] = "connected"
			result["lastVerified"] = now.Format(time.RFC3339)
			provider.Status = "connected"
			provider.LastVerified = &now
		} else {
			result["status"] = "disconnected"
			result["error"] = resp.Status
			provider.Status = "disconnected"
		}
	}

	// Update provider status in DB
	_ = s.db.SaveAuthProvider(*provider)

	s.writeAuditEntry(actorName(r), "provider_verified", provider.Name,
		"Auth provider connection test: "+result["status"].(string), nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleGetUsers returns all users.
func (s *Server) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.db.GetUsers()
	if err != nil {
		log.Printf("Error fetching users: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if users == nil {
		users = []database.UserRow{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// handleAddLocalUser creates a local user.
func (s *Server) handleAddLocalUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Email == "" {
		http.Error(w, "Name and email are required", http.StatusBadRequest)
		return
	}
	if req.Role == "" {
		req.Role = "user"
	}

	now := time.Now().Format(time.RFC3339)
	user := database.UserRow{
		ID:         "user-" + uuid.New().String()[:8],
		Name:       req.Name,
		Email:      req.Email,
		Role:       req.Role,
		AuthSource: "local",
		Groups:     []string{},
		LastLogin:  now,
		CreatedAt:  now,
	}

	if err := s.db.UpsertUser(user); err != nil {
		log.Printf("Error creating local user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	s.writeAuditEntry(actorName(r), "user_created", req.Name,
		"Local user created with role: "+req.Role, nil)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// handleChangeUserRole changes a user's role.
func (s *Server) handleChangeUserRole(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["id"]

	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Role != "admin" && req.Role != "user" {
		http.Error(w, "Role must be 'admin' or 'user'", http.StatusBadRequest)
		return
	}

	if err := s.db.ChangeUserRole(userID, req.Role); err != nil {
		log.Printf("Error changing user role: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	s.writeAuditEntry(actorName(r), "role_changed", userID,
		"Role changed to "+req.Role, nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleDeleteUser performs GDPR-compliant user deletion.
func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["id"]

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Reason == "" {
		http.Error(w, "Reason is required for GDPR deletion", http.StatusBadRequest)
		return
	}

	if err := s.db.DeleteUser(userID); err != nil {
		log.Printf("Error deleting user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	reason := req.Reason
	s.writeAuditEntry(actorName(r), "user_deleted", userID,
		"User deleted with full data purge", &reason)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleGetGroups returns all groups with their routing rules.
func (s *Server) handleGetGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := s.db.GetGroups()
	if err != nil {
		log.Printf("Error fetching groups: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if groups == nil {
		groups = []database.GroupRow{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(groups)
}

// handleChangeGroupRole changes a group's role.
func (s *Server) handleChangeGroupRole(w http.ResponseWriter, r *http.Request) {
	groupID := mux.Vars(r)["id"]

	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Role != "admin" && req.Role != "user" {
		http.Error(w, "Role must be 'admin' or 'user'", http.StatusBadRequest)
		return
	}

	if err := s.db.ChangeGroupRole(groupID, req.Role); err != nil {
		log.Printf("Error changing group role: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	s.writeAuditEntry(actorName(r), "group_role_changed", groupID,
		"Group role changed to "+req.Role, nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleAddRoutingRule adds a routing rule to a group.
func (s *Server) handleAddRoutingRule(w http.ResponseWriter, r *http.Request) {
	groupID := mux.Vars(r)["id"]

	var rule database.RoutingRuleRow
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	rule.ID = "rule-" + uuid.New().String()[:8]

	if err := s.db.SaveRoutingRule(groupID, rule); err != nil {
		log.Printf("Error adding routing rule: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	s.writeAuditEntry(actorName(r), "group_routing_updated", groupID,
		"Added routing rule: "+rule.Description, nil)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rule)
}

// handleRemoveRoutingRule removes a routing rule from a group.
func (s *Server) handleRemoveRoutingRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["groupId"]
	ruleID := vars["ruleId"]

	if err := s.db.DeleteRoutingRule(groupID, ruleID); err != nil {
		log.Printf("Error removing routing rule: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	s.writeAuditEntry(actorName(r), "group_routing_updated", groupID,
		"Removed routing rule: "+ruleID, nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleGetAuditLog returns audit log entries.
func (s *Server) handleGetAuditLog(w http.ResponseWriter, r *http.Request) {
	entries, err := s.db.GetAuditLog()
	if err != nil {
		log.Printf("Error fetching audit log: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if entries == nil {
		entries = []database.AuditEntryRow{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}
