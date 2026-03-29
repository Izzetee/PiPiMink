package llm

import (
	"encoding/json"
	"fmt"
	"net/http"

	"PiPiMink/internal/config"
)

// GetModelsByProvider returns the list of model names available from a provider.
//
// If the provider has a non-empty Models slice, that static list is returned directly
// (correct for Anthropic and Azure AI Foundry per-deployment endpoints).
//
// Otherwise the provider's /v1/models endpoint is queried (correct for OpenAI,
// Gemini, OpenRouter, and local servers).
func (c *Client) GetModelsByProvider(p config.ProviderConfig) ([]string, error) {
	// ModelConfigs (per-model overrides) defines the model list when set.
	// Falls back to the plain Models list, then to auto-discovery.
	if names := p.ModelNames(); len(names) > 0 {
		return names, nil
	}

	// Anthropic has no public models listing endpoint; a static list is required.
	if p.Type == config.ProviderTypeAnthropic {
		return nil, fmt.Errorf("provider %q (anthropic) has no models configured — add a static 'models' list or 'model_configs' array to providers.json", p.Name)
	}

	// Apply rate limiting if configured for this provider.
	if rl := c.rateLimiterFor(p.Name); rl != nil {
		rl.Wait()
	}

	url := fmt.Sprintf("%s/v1/models", p.BaseURL)
	client := &http.Client{Timeout: p.Timeout}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for provider %q: %w", p.Name, err)
	}

	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error querying models from provider %q: %w", p.Name, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if rl := c.rateLimiterFor(p.Name); rl != nil {
		rl.UpdateLastRequestTime()
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding models response from provider %q: %w", p.Name, err)
	}

	var names []string
	if data, ok := result["data"].([]interface{}); ok {
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				if id, ok := m["id"].(string); ok {
					names = append(names, id)
				}
			}
		}
	}

	return names, nil
}
