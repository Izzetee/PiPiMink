package llm

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"PiPiMink/internal/config"
	"PiPiMink/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestDecideModelBasedOnCapabilitiesUsesDecisionCache(t *testing.T) {
	var callCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"modelname\":\"gpt-4\",\"reason\":\"cached decision\"}"}}]}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Providers:                []config.ProviderConfig{{Name: "openai", Type: config.ProviderTypeOpenAICompatible, BaseURL: server.URL, APIKey: "test-key", Timeout: 5 * time.Second}},
		ModelSelectionModel:      "gpt-4-turbo",
		ModelSelectionProvider:   "openai",
		DefaultChatModel:         "gpt-4-turbo",
		SelectionCacheEnabled:    true,
		SelectionCacheTTL:        time.Minute,
		SelectionCacheMaxEntries: 10,
	}
	client := NewClient(cfg)

	availableModels := map[string]models.ModelInfo{
		"gpt-4":       {Source: "openai", Tags: `{"strengths":["reasoning"]}`, Enabled: true},
		"gpt-4-turbo": {Source: "openai", Tags: `{"strengths":["speed"]}`, Enabled: true},
	}

	model1, err1 := client.DecideModelBasedOnCapabilities("Plan a complex migration", availableModels)
	model2, err2 := client.DecideModelBasedOnCapabilities("Plan a complex migration", availableModels)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, "gpt-4", model1)
	assert.Equal(t, "gpt-4", model2)
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
}

func TestDecideModelBasedOnCapabilitiesCacheExpires(t *testing.T) {
	var callCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"modelname\":\"gpt-4-turbo\",\"reason\":\"ttl test\"}"}}]}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Providers:                []config.ProviderConfig{{Name: "openai", Type: config.ProviderTypeOpenAICompatible, BaseURL: server.URL, APIKey: "test-key", Timeout: 5 * time.Second}},
		ModelSelectionModel:      "gpt-4-turbo",
		ModelSelectionProvider:   "openai",
		DefaultChatModel:         "gpt-4-turbo",
		SelectionCacheEnabled:    true,
		SelectionCacheTTL:        20 * time.Millisecond,
		SelectionCacheMaxEntries: 10,
	}
	client := NewClient(cfg)

	availableModels := map[string]models.ModelInfo{
		"gpt-4-turbo": {Source: "openai", Tags: `{"strengths":["general"]}`, Enabled: true},
	}

	_, err1 := client.DecideModelBasedOnCapabilities("Summarize this contract", availableModels)
	assert.NoError(t, err1)

	time.Sleep(40 * time.Millisecond)

	_, err2 := client.DecideModelBasedOnCapabilities("Summarize this contract", availableModels)
	assert.NoError(t, err2)
	assert.Equal(t, int32(2), atomic.LoadInt32(&callCount))
}

func TestDecideModelBasedOnCapabilitiesCacheDisabled(t *testing.T) {
	var callCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"modelname\":\"gpt-4-turbo\",\"reason\":\"no cache\"}"}}]}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Providers:                []config.ProviderConfig{{Name: "openai", Type: config.ProviderTypeOpenAICompatible, BaseURL: server.URL, APIKey: "test-key", Timeout: 5 * time.Second}},
		ModelSelectionModel:      "gpt-4-turbo",
		ModelSelectionProvider:   "openai",
		DefaultChatModel:         "gpt-4-turbo",
		SelectionCacheEnabled:    false,
		SelectionCacheTTL:        time.Minute,
		SelectionCacheMaxEntries: 10,
	}
	client := NewClient(cfg)

	availableModels := map[string]models.ModelInfo{
		"gpt-4-turbo": {Source: "openai", Tags: `{"strengths":["general"]}`, Enabled: true},
	}

	_, err1 := client.DecideModelBasedOnCapabilities("Explain memory usage", availableModels)
	_, err2 := client.DecideModelBasedOnCapabilities("Explain memory usage", availableModels)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, int32(2), atomic.LoadInt32(&callCount))
}

func TestDecisionCacheMaybeStatsSummaryInterval(t *testing.T) {
	cfg := &config.Config{
		SelectionCacheEnabled:          true,
		SelectionCacheTTL:              time.Minute,
		SelectionCacheMaxEntries:       10,
		SelectionCacheStatsLogInterval: 50 * time.Millisecond,
	}

	cache := newDecisionCache(cfg)
	cache.set("k1", "m1")
	_, _, _ = cache.getWithStatus("k1")

	_, emit1 := cache.maybeStatsSummary()
	_, emit2 := cache.maybeStatsSummary()

	assert.True(t, emit1)
	assert.False(t, emit2)

	time.Sleep(70 * time.Millisecond)

	_, emit3 := cache.maybeStatsSummary()
	assert.True(t, emit3)
}

func TestDecisionCacheMaybeStatsSummaryDisabledByInterval(t *testing.T) {
	cfg := &config.Config{
		SelectionCacheEnabled:          true,
		SelectionCacheTTL:              time.Minute,
		SelectionCacheMaxEntries:       10,
		SelectionCacheStatsLogInterval: 0,
	}

	cache := newDecisionCache(cfg)
	cache.set("k1", "m1")

	_, emit := cache.maybeStatsSummary()
	assert.False(t, emit)
}
