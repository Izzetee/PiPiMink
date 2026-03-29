// Package llm provides functionality for interacting with language models.
// It handles communication with OpenAI-compatible and Anthropic providers,
// including model capability analysis, chat interactions, and intelligent routing.
package llm

import (
	"net/url"
	"strings"

	"PiPiMink/internal/config"
)

// Client is an LLM client that communicates with configured providers.
type Client struct {
	Config           *config.Config
	providers        map[string]config.ProviderConfig // keyed by provider name
	providerLimiters map[string]*RateLimiter          // keyed by provider name
	decisionCache    *decisionCache

	// Overridable tagging prompts — empty string means "use the package-level default".
	taggingSystemPrompt    string
	taggingUserPrompt      string
	taggingUserNoSysPrompt string
}

// UpdateTaggingPrompts replaces the prompts used when tagging models.
// Pass an empty string for any argument to keep the current/default value.
func (c *Client) UpdateTaggingPrompts(systemPrompt, userPrompt, userNoSysPrompt string) {
	if systemPrompt != "" {
		c.taggingSystemPrompt = systemPrompt
	}
	if userPrompt != "" {
		c.taggingUserPrompt = userPrompt
	}
	if userNoSysPrompt != "" {
		c.taggingUserNoSysPrompt = userNoSysPrompt
	}
}

// NewClient creates a new LLM client from the provided configuration.
func NewClient(cfg *config.Config) *Client {
	providers := make(map[string]config.ProviderConfig, len(cfg.Providers))
	limiters := make(map[string]*RateLimiter, len(cfg.Providers))

	for _, p := range cfg.Providers {
		providers[p.Name] = p
		if p.RateLimitSeconds > 0 {
			limiters[p.Name] = NewRateLimiter(p.RateLimitSeconds)
		}
	}

	return &Client{
		Config:           cfg,
		providers:        providers,
		providerLimiters: limiters,
		decisionCache:    newDecisionCache(cfg),
	}
}

// findProvider returns the ProviderConfig for the given provider name.
func (c *Client) findProvider(name string) (config.ProviderConfig, bool) {
	p, ok := c.providers[name]
	return p, ok
}

// selectionProvider returns the ProviderConfig used for meta-routing decisions.
// It falls back to the first openai-compatible provider if MODEL_SELECTION_PROVIDER is not set.
func (c *Client) selectionProvider() (config.ProviderConfig, bool) {
	name := c.Config.ModelSelectionProvider
	if name != "" {
		if p, ok := c.providers[name]; ok {
			return p, true
		}
	}
	// Fallback: first openai-compatible provider
	for _, p := range c.providers {
		if p.Type == config.ProviderTypeOpenAICompatible {
			return p, true
		}
	}
	return config.ProviderConfig{}, false
}

// localProviderBaseURL returns the base URL of the first locally-running provider
// (i.e. a provider whose host is localhost or 127.0.0.1). Used by MLX detection.
func (c *Client) localProviderBaseURL() string {
	for _, p := range c.providers {
		u, err := url.Parse(p.BaseURL)
		if err != nil {
			continue
		}
		host := u.Hostname()
		if host == "localhost" || host == "127.0.0.1" || strings.HasPrefix(host, "192.168.") {
			return p.BaseURL
		}
	}
	return "http://localhost:11434" // sensible fallback
}

// rateLimiterFor returns the rate limiter for the given provider, or nil if none is configured.
func (c *Client) rateLimiterFor(providerName string) *RateLimiter {
	return c.providerLimiters[providerName]
}
