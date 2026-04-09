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

// providerResponse is the JSON shape returned for a single provider.
type providerResponse struct {
	Name             string                `json:"name"`
	Type             string                `json:"type"`
	BaseURL          string                `json:"base_url"`
	APIKeyEnv        string                `json:"api_key_env"`
	Timeout          string                `json:"timeout"`
	RateLimitSeconds int                   `json:"rate_limit_seconds"`
	Enabled          bool                  `json:"enabled"`
	Models           []string              `json:"models"`
	ModelConfigs     []modelConfigResponse `json:"model_configs"`
	ModelCount       int                   `json:"model_count"`
	LastTestedAt     *string               `json:"last_tested_at"`
	LastTestResult   *string               `json:"last_test_result"`
	LastTestLatencyMs *int64               `json:"last_test_latency_ms"`
}

type modelConfigResponse struct {
	Name      string  `json:"name"`
	ChatPath  *string `json:"chat_path"`
	APIKeyEnv string  `json:"api_key_env"`
	Type      *string `json:"type"`
	BaseURL   *string `json:"base_url"`
	Enabled   bool    `json:"enabled"`
}

// toProviderResponse converts a ProviderConfig + test info + model count to a response.
func (s *Server) toProviderResponse(p config.ProviderConfig, modelCount int) providerResponse {
	mcs := make([]modelConfigResponse, len(p.ModelConfigs))
	for i, mc := range p.ModelConfigs {
		mcs[i] = modelConfigResponse{
			Name:      mc.Name,
			ChatPath:  strPtrOrNil(mc.ChatPath),
			APIKeyEnv: mc.APIKeyEnv,
			Type:      strPtrOrNil(mc.Type),
			BaseURL:   strPtrOrNil(mc.BaseURL),
			Enabled:   mc.Enabled,
		}
	}

	models := p.Models
	if models == nil {
		models = []string{}
	}

	resp := providerResponse{
		Name:             p.Name,
		Type:             p.Type,
		BaseURL:          p.BaseURL,
		APIKeyEnv:        p.APIKeyEnv,
		Timeout:          p.TimeoutStr,
		RateLimitSeconds: p.RateLimitSeconds,
		Enabled:          p.Enabled,
		Models:           models,
		ModelConfigs:     mcs,
		ModelCount:       modelCount,
	}

	if info, ok := s.providerTestInfo[p.Name]; ok {
		if info.LastTestedAt != nil {
			ts := info.LastTestedAt.Format(time.RFC3339)
			resp.LastTestedAt = &ts
		}
		if info.LastTestResult != "" {
			resp.LastTestResult = &info.LastTestResult
		}
		resp.LastTestLatencyMs = info.LastTestLatencyMs
	}

	return resp
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// countModelsForProvider counts models in the in-memory collection with the given source.
func (s *Server) countModelsForProvider(providerName string) int {
	s.modelMutex.RLock()
	defer s.modelMutex.RUnlock()
	count := 0
	for _, info := range s.modelCollection.Models {
		if info.Source == providerName {
			count++
		}
	}
	return count
}

// workingDir returns the current working directory for provider persistence.
func workingDir() string {
	dir, err := os.Getwd()
	if err != nil {
		log.Printf("Warning: could not determine working directory: %v", err)
		return "."
	}
	return dir
}

// saveProvidersToDisk persists the current s.config.Providers to providers.json.
// Must be called while holding s.providerMutex (at least RLock).
func (s *Server) saveProvidersToDisk() error {
	return config.SaveProviders(workingDir(), s.config.Providers)
}

// findProviderIndex returns the index of the provider with the given name, or -1.
// Must be called while holding s.providerMutex.
func (s *Server) findProviderIndex(name string) int {
	for i, p := range s.config.Providers {
		if p.Name == name {
			return i
		}
	}
	return -1
}

// handleListProviders returns all configured providers with test status and model counts.
// Auth: admin (enforced by middleware)
func (s *Server) handleListProviders(w http.ResponseWriter, r *http.Request) {
	s.providerMutex.RLock()
	providers := make([]providerResponse, len(s.config.Providers))
	for i, p := range s.config.Providers {
		providers[i] = s.toProviderResponse(p, s.countModelsForProvider(p.Name))
	}
	s.providerMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"providers": providers,
	})
}

// handleAddProvider creates a new provider.
// Auth: admin (enforced by middleware)
func (s *Server) handleAddProvider(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Name             string              `json:"name"`
		Type             string              `json:"type"`
		BaseURL          string              `json:"base_url"`
		APIKeyEnv        string              `json:"api_key_env"`
		Timeout          string              `json:"timeout"`
		RateLimitSeconds int                 `json:"rate_limit_seconds"`
		Enabled          bool                `json:"enabled"`
		Models           []string            `json:"models"`
		ModelConfigs     []config.ModelConfig `json:"model_configs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.BaseURL == "" {
		http.Error(w, `{"error":"name and base_url are required"}`, http.StatusBadRequest)
		return
	}

	s.providerMutex.Lock()
	defer s.providerMutex.Unlock()

	if s.findProviderIndex(req.Name) != -1 {
		http.Error(w, `{"error":"provider name already exists"}`, http.StatusConflict)
		return
	}

	p := config.ProviderConfig{
		Name:             req.Name,
		Type:             req.Type,
		BaseURL:          req.BaseURL,
		APIKeyEnv:        req.APIKeyEnv,
		TimeoutStr:       req.Timeout,
		RateLimitSeconds: req.RateLimitSeconds,
		Enabled:          req.Enabled,
		Models:           req.Models,
		ModelConfigs:     req.ModelConfigs,
	}
	config.ResolveProviderKeys(&p)

	s.config.Providers = append(s.config.Providers, p)
	if err := s.saveProvidersToDisk(); err != nil {
		// Roll back in-memory change.
		s.config.Providers = s.config.Providers[:len(s.config.Providers)-1]
		log.Printf("Error saving providers: %v", err)
		http.Error(w, `{"error":"failed to save providers"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(s.toProviderResponse(p, 0))
}

// handleUpdateProvider updates an existing provider (name change not allowed).
// Auth: admin (enforced by middleware)
func (s *Server) handleUpdateProvider(w http.ResponseWriter, r *http.Request) {
	name, err := decodeMuxVar(r, "name")
	if err != nil {
		http.Error(w, `{"error":"invalid provider name"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Name             string `json:"name"`
		Type             string `json:"type"`
		BaseURL          string `json:"base_url"`
		APIKeyEnv        string `json:"api_key_env"`
		Timeout          string `json:"timeout"`
		RateLimitSeconds int    `json:"rate_limit_seconds"`
		Enabled          bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if req.Name != "" && req.Name != name {
		http.Error(w, `{"error":"provider name cannot be changed; delete and re-add instead"}`, http.StatusBadRequest)
		return
	}

	s.providerMutex.Lock()
	defer s.providerMutex.Unlock()

	idx := s.findProviderIndex(name)
	if idx == -1 {
		http.Error(w, `{"error":"provider not found"}`, http.StatusNotFound)
		return
	}

	p := &s.config.Providers[idx]
	if req.Type != "" {
		p.Type = req.Type
	}
	if req.BaseURL != "" {
		p.BaseURL = req.BaseURL
	}
	p.APIKeyEnv = req.APIKeyEnv
	p.TimeoutStr = req.Timeout
	p.RateLimitSeconds = req.RateLimitSeconds
	p.Enabled = req.Enabled
	config.ResolveProviderKeys(p)

	if err := s.saveProvidersToDisk(); err != nil {
		log.Printf("Error saving providers: %v", err)
		http.Error(w, `{"error":"failed to save providers"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.toProviderResponse(*p, s.countModelsForProvider(p.Name)))
}

// handleDeleteProvider removes a provider.
// Auth: admin (enforced by middleware)
func (s *Server) handleDeleteProvider(w http.ResponseWriter, r *http.Request) {
	name, err := decodeMuxVar(r, "name")
	if err != nil {
		http.Error(w, `{"error":"invalid provider name"}`, http.StatusBadRequest)
		return
	}

	s.providerMutex.Lock()
	defer s.providerMutex.Unlock()

	idx := s.findProviderIndex(name)
	if idx == -1 {
		http.Error(w, `{"error":"provider not found"}`, http.StatusNotFound)
		return
	}

	s.config.Providers = append(s.config.Providers[:idx], s.config.Providers[idx+1:]...)
	delete(s.providerTestInfo, name)

	if err := s.saveProvidersToDisk(); err != nil {
		log.Printf("Error saving providers: %v", err)
		http.Error(w, `{"error":"failed to save providers"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// handleToggleProvider enables or disables a provider.
// Auth: admin (enforced by middleware)
func (s *Server) handleToggleProvider(w http.ResponseWriter, r *http.Request) {
	name, err := decodeMuxVar(r, "name")
	if err != nil {
		http.Error(w, `{"error":"invalid provider name"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	s.providerMutex.Lock()
	defer s.providerMutex.Unlock()

	idx := s.findProviderIndex(name)
	if idx == -1 {
		http.Error(w, `{"error":"provider not found"}`, http.StatusNotFound)
		return
	}

	s.config.Providers[idx].Enabled = req.Enabled
	if err := s.saveProvidersToDisk(); err != nil {
		log.Printf("Error saving providers: %v", err)
		http.Error(w, `{"error":"failed to save providers"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"enabled": req.Enabled})
}

// handleTestProvider tests connectivity to a provider by listing its models.
// Auth: admin (enforced by middleware)
func (s *Server) handleTestProvider(w http.ResponseWriter, r *http.Request) {
	name, err := decodeMuxVar(r, "name")
	if err != nil {
		http.Error(w, `{"error":"invalid provider name"}`, http.StatusBadRequest)
		return
	}

	// Get a copy of the provider config under lock.
	s.providerMutex.RLock()
	idx := s.findProviderIndex(name)
	if idx == -1 {
		s.providerMutex.RUnlock()
		http.Error(w, `{"error":"provider not found"}`, http.StatusNotFound)
		return
	}
	p := s.config.Providers[idx]
	s.providerMutex.RUnlock()

	// Perform the connectivity test (may be slow — outside lock).
	start := time.Now()
	modelNames, testErr := s.llmClient.GetModelsByProvider(p)
	latency := time.Since(start).Milliseconds()

	now := time.Now()
	info := &ProviderTestInfo{
		LastTestedAt:      &now,
		LastTestLatencyMs: &latency,
	}

	result := "success"
	var errMsg string
	if testErr != nil {
		result = "error"
		errMsg = testErr.Error()
	}
	info.LastTestResult = result

	s.providerMutex.Lock()
	s.providerTestInfo[name] = info
	s.providerMutex.Unlock()

	resp := map[string]interface{}{
		"result":     result,
		"latency_ms": latency,
	}
	if testErr != nil {
		resp["error"] = errMsg
	} else {
		resp["models_found"] = len(modelNames)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleUpdateModelConfigs bulk-replaces the model configs for a provider.
// Auth: admin (enforced by middleware)
func (s *Server) handleUpdateModelConfigs(w http.ResponseWriter, r *http.Request) {
	name, err := decodeMuxVar(r, "name")
	if err != nil {
		http.Error(w, `{"error":"invalid provider name"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		ModelConfigs []config.ModelConfig `json:"model_configs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	s.providerMutex.Lock()
	defer s.providerMutex.Unlock()

	idx := s.findProviderIndex(name)
	if idx == -1 {
		http.Error(w, `{"error":"provider not found"}`, http.StatusNotFound)
		return
	}

	s.config.Providers[idx].ModelConfigs = req.ModelConfigs
	config.ResolveProviderKeys(&s.config.Providers[idx])

	if err := s.saveProvidersToDisk(); err != nil {
		log.Printf("Error saving providers: %v", err)
		http.Error(w, `{"error":"failed to save providers"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// decodeMuxVar extracts and URL-decodes a mux route variable.
func decodeMuxVar(r *http.Request, key string) (string, error) {
	vars := mux.Vars(r)
	return vars[key], nil
}
