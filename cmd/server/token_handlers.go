package server

import (
	"encoding/json"
	"net/http"

	"PiPiMink/internal/database"

	"github.com/gorilla/mux"
)

// handleCreateToken creates a new API token for the authenticated user.
// Auth: user (enforced by middleware)
func (s *Server) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" || userID == "anonymous" || userID == "admin:api-key" {
		http.Error(w, "Token creation requires a user account (OAuth or local)", http.StatusBadRequest)
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		body.Name = ""
	}

	tokenID, plaintext, err := s.db.CreateUserAPIToken(userID, body.Name)
	if err != nil {
		http.Error(w, "error creating token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"id":    tokenID,
		"token": plaintext,
		"name":  body.Name,
	})
}

// handleListTokens lists all API tokens for the authenticated user.
// Auth: user (enforced by middleware)
func (s *Server) handleListTokens(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" || userID == "anonymous" || userID == "admin:api-key" {
		http.Error(w, "Token listing requires a user account", http.StatusBadRequest)
		return
	}

	tokens, err := s.db.ListUserAPITokens(userID)
	if err != nil {
		http.Error(w, "error listing tokens", http.StatusInternalServerError)
		return
	}
	if tokens == nil {
		tokens = []database.UserAPITokenRow{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tokens)
}

// handleRevokeToken revokes an API token. Users can revoke their own tokens;
// admins can revoke any token.
// Auth: user (enforced by middleware)
func (s *Server) handleRevokeToken(w http.ResponseWriter, r *http.Request) {
	tokenID := mux.Vars(r)["id"]
	userID := getUserID(r)

	// Admin can revoke any token; users can only revoke their own
	ownerFilter := userID
	if getAuthLevel(r) >= AuthAdmin {
		ownerFilter = "" // skip ownership check
	}

	if err := s.db.RevokeUserAPIToken(tokenID, ownerFilter); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
