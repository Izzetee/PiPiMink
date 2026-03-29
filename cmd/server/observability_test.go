package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"PiPiMink/internal/config"
)

func TestMetricsEndpointIsExposed(t *testing.T) {
	cfg := &config.Config{
		Port:            "8080",
		OTelServiceName: "pipimink-test",
	}

	mockDB := new(MockDB)
	mockLLM := new(MockLLMClient)
	mockLLM.On("IsLocalServerUsingMLX").Return(false).Maybe()

	srv := NewServer(cfg, mockDB, mockLLM)

	// Trigger one request so metrics have data points.
	req1 := httptest.NewRequest(http.MethodGet, "/models", nil)
	rr1 := httptest.NewRecorder()
	srv.GetRouter().ServeHTTP(rr1, req1)

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsReq.Header.Set("Accept", "application/openmetrics-text")
	metricsRR := httptest.NewRecorder()
	srv.GetRouter().ServeHTTP(metricsRR, metricsReq)

	if metricsRR.Code != http.StatusOK {
		t.Fatalf("expected /metrics status 200, got %d", metricsRR.Code)
	}

	body := metricsRR.Body.String()
	if body == "" {
		t.Fatalf("expected /metrics body to be non-empty")
	}
	if !containsMetricsName(body, "pipimink_http_requests_total") {
		t.Fatalf("expected /metrics to contain pipimink_http_requests_total")
	}
}

func containsMetricsName(content, metricName string) bool {
	return len(content) > 0 && len(metricName) > 0 && strings.Contains(content, metricName)
}
