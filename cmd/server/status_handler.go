package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
)

// adminStatusResponse is the JSON shape returned by GET /admin/status.
// It exposes only boolean flags and counts — no secrets.
type adminStatusResponse struct {
	AdminKeyConfigured  bool `json:"adminKeyConfigured"`
	ProvidersConfigured bool `json:"providersConfigured"`
	ProviderCount       int  `json:"providerCount"`
	ModelCount          int  `json:"modelCount"`
	OAuthEnabled        bool `json:"oauthEnabled"`
}

// handleAdminStatus returns a lightweight, unauthenticated snapshot of instance
// configuration state. The setup wizard uses this to decide whether to show.
func (s *Server) handleAdminStatus(w http.ResponseWriter, r *http.Request) {
	s.providerMutex.RLock()
	providerCount := len(s.config.Providers)
	s.providerMutex.RUnlock()

	// Check whether a providers.json file actually exists on disk (as opposed
	// to the built-in default that loadProviders returns when the file is absent).
	providersConfigured := false
	if dir, err := os.Getwd(); err == nil {
		if _, err := os.Stat(filepath.Join(dir, "providers.json")); err == nil {
			providersConfigured = true
		}
	}

	resp := adminStatusResponse{
		AdminKeyConfigured:  s.config.AdminAPIKey != "",
		ProvidersConfigured: providersConfigured,
		ProviderCount:       providerCount,
		ModelCount:          s.modelCollection.ModelCount(),
		OAuthEnabled:        s.config.OAuthEnabled(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
