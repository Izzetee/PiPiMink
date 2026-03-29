package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"PiPiMink/internal/config"
	"PiPiMink/internal/llm"
	"PiPiMink/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServerWithLLM(t *testing.T, upstream *httptest.Server) *Server {
	t.Helper()
	cfg := &config.Config{
		Providers: []config.ProviderConfig{
			{Name: "openai", Type: config.ProviderTypeOpenAICompatible, BaseURL: upstream.URL, APIKey: "test-key", Timeout: 2 * time.Second},
			{Name: "local", Type: config.ProviderTypeOpenAICompatible, BaseURL: upstream.URL, Timeout: 2 * time.Second},
		},
		AdminAPIKey:      "admin-key",
		DefaultChatModel: "gpt-4-turbo",
	}
	client := llm.NewClient(cfg)
	return NewServer(cfg, nil, client)
}

func TestHelperFunctions(t *testing.T) {
	assert.True(t, shouldProcessModel("", tagRefreshInterval))
	assert.True(t, shouldProcessModel("not-a-time", tagRefreshInterval))
	assert.False(t, shouldProcessModel(time.Now().Format(time.RFC3339), tagRefreshInterval))
	assert.True(t, shouldProcessModel(time.Now().Add(-2*tagRefreshInterval).Format(time.RFC3339), tagRefreshInterval))

	_, err := time.Parse(time.RFC3339, getCurrentTimeString())
	assert.NoError(t, err)

	id := generateFallbackRandomID()
	assert.Len(t, id, 10)
}

func TestHandleOpenAIModels_DefaultWhenEmpty(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer upstream.Close()

	s := newTestServerWithLLM(t, upstream)
	s.modelCollection = models.NewModelCollection()

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "gpt-4-turbo")
}

func TestHandleOpenAIChatCompletions_ValidRequest(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/chat/completions" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hello from upstream"}}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer upstream.Close()

	s := newTestServerWithLLM(t, upstream)
	s.modelCollection.AddModel("gpt-4-turbo", models.ModelInfo{Source: "openai", Enabled: true})

	payload := map[string]interface{}{
		"model":    "gpt-4-turbo",
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "chat.completion")
	assert.Contains(t, rr.Body.String(), "hello from upstream")
}

func TestHandleOpenAIChatCompletions_ValidationErrors(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	s := newTestServerWithLLM(t, upstream)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{"messages":[]}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	req = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{"messages":[{"role":"assistant","content":"x"}]}`))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
