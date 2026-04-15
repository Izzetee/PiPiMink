package server

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"PiPiMink/internal/config"

	"github.com/gorilla/mux"
)

// apiKeyResponse is the JSON shape for a stored API key.
type apiKeyResponse struct {
	ID            string `json:"id"`
	EnvVarName    string `json:"envVarName"`
	ProviderName  string `json:"providerName"`
	ProviderID    string `json:"providerId"`
	MaskedValue   string `json:"maskedValue"`
	LastUpdatedAt string `json:"lastUpdatedAt"`
}

// handleListApiKeys returns all API keys referenced by providers.
// Auth: admin (enforced by middleware)
func (s *Server) handleListApiKeys(w http.ResponseWriter, r *http.Request) {
	keys := s.collectApiKeys()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(keys)
}

// setApiKeyRequest is the body for PUT /admin/api-keys/{envVarName}.
type setApiKeyRequest struct {
	Value string `json:"value"`
}

// handleSetApiKey creates or updates an API key in .env.
// Auth: admin (enforced by middleware)
func (s *Server) handleSetApiKey(w http.ResponseWriter, r *http.Request) {
	envVarName := mux.Vars(r)["envVarName"]
	if envVarName == "" {
		http.Error(w, "Missing env var name", http.StatusBadRequest)
		return
	}

	var req setApiKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Value == "" {
		http.Error(w, "Value is required", http.StatusBadRequest)
		return
	}

	// Persist to .env
	if err := config.PatchDotEnv(map[string]string{envVarName: req.Value}); err != nil {
		log.Printf("Error setting API key in .env: %v", err)
		http.Error(w, "Failed to persist API key", http.StatusInternalServerError)
		return
	}

	// Update environment and re-resolve provider keys
	_ = os.Setenv(envVarName, req.Value)
	s.resolveAllProviderKeys()

	// Return the updated key
	key := s.buildApiKeyResponse(envVarName)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(key)
}

// handleDeleteApiKey removes an API key from .env.
// Auth: admin (enforced by middleware)
func (s *Server) handleDeleteApiKey(w http.ResponseWriter, r *http.Request) {
	envVarName := mux.Vars(r)["envVarName"]
	if envVarName == "" {
		http.Error(w, "Missing env var name", http.StatusBadRequest)
		return
	}

	// Remove from .env
	if err := config.RemoveDotEnvKey(envVarName); err != nil {
		log.Printf("Error removing API key from .env: %v", err)
		http.Error(w, "Failed to remove API key", http.StatusInternalServerError)
		return
	}

	// Unset environment and re-resolve
	_ = os.Unsetenv(envVarName)
	s.resolveAllProviderKeys()

	w.WriteHeader(http.StatusNoContent)
}

// collectApiKeys gathers all API key env vars from providers and their model configs.
func (s *Server) collectApiKeys() []apiKeyResponse {
	s.providerMutex.RLock()
	providers := s.config.Providers
	s.providerMutex.RUnlock()

	// Use a map to deduplicate env var names and track which provider they belong to.
	type keyInfo struct {
		envVar       string
		providerName string
		providerID   string
	}
	seen := make(map[string]keyInfo)

	for _, p := range providers {
		if p.APIKeyEnv != "" {
			if _, exists := seen[p.APIKeyEnv]; !exists {
				seen[p.APIKeyEnv] = keyInfo{envVar: p.APIKeyEnv, providerName: p.Name, providerID: p.Name}
			}
		}
		for _, mc := range p.ModelConfigs {
			if mc.APIKeyEnv != "" {
				if _, exists := seen[mc.APIKeyEnv]; !exists {
					seen[mc.APIKeyEnv] = keyInfo{envVar: mc.APIKeyEnv, providerName: p.Name, providerID: p.Name}
				}
			}
		}
	}

	// Approximate last-updated from .env file mtime
	mtime := envFileMtime()

	keys := make([]apiKeyResponse, 0, len(seen))
	for _, info := range seen {
		val := os.Getenv(info.envVar)
		masked := ""
		if val != "" {
			masked = maskValue(val)
		}
		keys = append(keys, apiKeyResponse{
			ID:            info.envVar,
			EnvVarName:    info.envVar,
			ProviderName:  info.providerName,
			ProviderID:    info.providerID,
			MaskedValue:   masked,
			LastUpdatedAt: mtime,
		})
	}
	return keys
}

// buildApiKeyResponse constructs a single API key response by env var name.
func (s *Server) buildApiKeyResponse(envVarName string) apiKeyResponse {
	s.providerMutex.RLock()
	providers := s.config.Providers
	s.providerMutex.RUnlock()

	providerName, providerID := "", ""
	for _, p := range providers {
		if p.APIKeyEnv == envVarName {
			providerName = p.Name
			providerID = p.Name
			break
		}
		for _, mc := range p.ModelConfigs {
			if mc.APIKeyEnv == envVarName {
				providerName = p.Name
				providerID = p.Name
				break
			}
		}
		if providerName != "" {
			break
		}
	}

	val := os.Getenv(envVarName)
	masked := ""
	if val != "" {
		masked = maskValue(val)
	}

	return apiKeyResponse{
		ID:            envVarName,
		EnvVarName:    envVarName,
		ProviderName:  providerName,
		ProviderID:    providerID,
		MaskedValue:   masked,
		LastUpdatedAt: envFileMtime(),
	}
}

// resolveAllProviderKeys re-resolves API keys from env for all providers.
func (s *Server) resolveAllProviderKeys() {
	s.providerMutex.Lock()
	defer s.providerMutex.Unlock()
	for i := range s.config.Providers {
		config.ResolveProviderKeys(&s.config.Providers[i])
	}
}

// envFileMtime returns the .env file modification time as an ISO string, or empty.
func envFileMtime() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	info, err := os.Stat(dir + "/.env")
	if err != nil {
		return ""
	}
	return info.ModTime().UTC().Truncate(time.Second).Format(time.RFC3339)
}
