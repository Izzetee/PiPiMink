package server

import (
	"encoding/json"
	"log"
	"net/http"

	"PiPiMink/internal/config"
)

// settingResponse is the JSON shape for a single setting in the API response.
type settingResponse struct {
	Key         string                    `json:"key"`
	Value       interface{}               `json:"value"`
	Type        config.SettingType        `json:"type"`
	Label       string                    `json:"label"`
	Description string                    `json:"description"`
	Required    bool                      `json:"required"`
	DependsOn   string                    `json:"dependsOn,omitempty"`
	Validation  *config.SettingValidation `json:"validation,omitempty"`
}

// settingsMapResponse groups settings by category.
type settingsMapResponse struct {
	Routing       []settingResponse `json:"routing"`
	Cache         []settingResponse `json:"cache"`
	Database      []settingResponse `json:"database"`
	Server        []settingResponse `json:"server"`
	Benchmarking  []settingResponse `json:"benchmarking"`
	Observability []settingResponse `json:"observability"`
}

type providerOptionResponse struct {
	ID            string                `json:"id"`
	Name          string                `json:"name"`
	APIKeyEnvVars []string              `json:"apiKeyEnvVars"`
	Models        []modelOptionResponse `json:"models"`
}

type modelOptionResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type getSettingsResponse struct {
	Settings        settingsMapResponse      `json:"settings"`
	ProviderOptions []providerOptionResponse `json:"providerOptions"`
}

// handleGetSettings returns all settings grouped by category plus provider options.
// Auth: admin (enforced by middleware)
func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	resp := getSettingsResponse{
		Settings:        s.buildSettingsMap(),
		ProviderOptions: s.buildProviderOptions(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// patchSettingsRequest is the expected body for PATCH /admin/settings.
type patchSettingsRequest struct {
	Changes []settingChange `json:"changes"`
}

type settingChange struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// handlePatchSettings applies setting changes to .env and in-memory config.
// Auth: admin (enforced by middleware)
func (s *Server) handlePatchSettings(w http.ResponseWriter, r *http.Request) {
	var req patchSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.Changes) == 0 {
		http.Error(w, "No changes provided", http.StatusBadRequest)
		return
	}

	// Validate all keys exist in registry
	for _, ch := range req.Changes {
		if config.FindSettingDef(ch.Key) == nil {
			http.Error(w, "Unknown setting key: "+ch.Key, http.StatusBadRequest)
			return
		}
	}

	// Build env updates map
	envUpdates := make(map[string]string, len(req.Changes))
	for _, ch := range req.Changes {
		envUpdates[ch.Key] = config.ValueToEnvString(ch.Value)
	}

	// Persist to .env
	if err := config.PatchDotEnv(envUpdates); err != nil {
		log.Printf("Error patching .env: %v", err)
		http.Error(w, "Failed to persist settings", http.StatusInternalServerError)
		return
	}

	// Update in-memory config
	for _, ch := range req.Changes {
		if err := config.SetSettingValue(s.config, ch.Key, ch.Value); err != nil {
			log.Printf("Warning: failed to set in-memory config for %s: %v", ch.Key, err)
		}
	}

	// Return updated settings
	resp := getSettingsResponse{
		Settings:        s.buildSettingsMap(),
		ProviderOptions: s.buildProviderOptions(),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// buildSettingsMap constructs the settings response from the registry and current config.
func (s *Server) buildSettingsMap() settingsMapResponse {
	byCategory := config.RegistryByCategory()
	toResponses := func(cat config.SettingCategory) []settingResponse {
		defs := byCategory[cat]
		out := make([]settingResponse, 0, len(defs))
		for _, def := range defs {
			value := config.GetSettingValue(s.config, def.Key)
			// Mask secret values
			if def.Type == config.SettingTypeSecret {
				if str, ok := value.(string); ok && len(str) > 0 {
					value = maskValue(str)
				}
			}
			out = append(out, settingResponse{
				Key:         def.Key,
				Value:       value,
				Type:        def.Type,
				Label:       def.Label,
				Description: def.Description,
				Required:    def.Required,
				DependsOn:   def.DependsOn,
				Validation:  def.Validation,
			})
		}
		return out
	}
	return settingsMapResponse{
		Routing:       toResponses(config.CategoryRouting),
		Cache:         toResponses(config.CategoryCache),
		Database:      toResponses(config.CategoryDatabase),
		Server:        toResponses(config.CategoryServer),
		Benchmarking:  toResponses(config.CategoryBenchmarking),
		Observability: toResponses(config.CategoryObservability),
	}
}

// buildProviderOptions constructs the provider dropdown data from config and model collection.
func (s *Server) buildProviderOptions() []providerOptionResponse {
	s.providerMutex.RLock()
	providers := s.config.Providers
	s.providerMutex.RUnlock()

	s.modelMutex.RLock()
	enabledModels := s.modelCollection.GetEnabledModels()
	s.modelMutex.RUnlock()

	out := make([]providerOptionResponse, 0, len(providers))
	for _, p := range providers {
		if !p.Enabled {
			continue
		}

		// Collect API key env vars
		envVars := []string{}
		if p.APIKeyEnv != "" {
			envVars = append(envVars, p.APIKeyEnv)
		}
		for _, mc := range p.ModelConfigs {
			if mc.APIKeyEnv != "" {
				// Deduplicate
				found := false
				for _, ev := range envVars {
					if ev == mc.APIKeyEnv {
						found = true
						break
					}
				}
				if !found {
					envVars = append(envVars, mc.APIKeyEnv)
				}
			}
		}

		// Collect models: prefer enabled models from DB, fall back to static list
		modelOpts := []modelOptionResponse{}
		for name, info := range enabledModels {
			if info.Source == p.Name {
				modelOpts = append(modelOpts, modelOptionResponse{ID: name, Name: name})
			}
		}
		if len(modelOpts) == 0 {
			for _, name := range p.ModelNames() {
				modelOpts = append(modelOpts, modelOptionResponse{ID: name, Name: name})
			}
		}

		out = append(out, providerOptionResponse{
			ID:            p.Name,
			Name:          p.Name,
			APIKeyEnvVars: envVars,
			Models:        modelOpts,
		})
	}
	return out
}

// maskValue returns a masked version of a secret string.
func maskValue(s string) string {
	if len(s) <= 8 {
		return "••••••••"
	}
	return s[:4] + "••••••••••••••" + s[len(s)-4:]
}
