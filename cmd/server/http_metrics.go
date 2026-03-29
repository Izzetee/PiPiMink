package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
)

type httpMetrics struct {
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	inflightRequests prometheus.Gauge
}

func newHTTPMetrics() *httpMetrics {
	requestsTotal := registerOrGetCounterVec(prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pipimink_http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "route", "status"},
	))

	requestDuration := registerOrGetHistogramVec(prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pipimink_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route", "status"},
	))

	inflightRequests := registerOrGetGauge(prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "pipimink_http_inflight_requests",
			Help: "Current number of inflight HTTP requests.",
		},
	))

	return &httpMetrics{
		requestsTotal:    requestsTotal,
		requestDuration:  requestDuration,
		inflightRequests: inflightRequests,
	}
}

func registerOrGetCounterVec(collector *prometheus.CounterVec) *prometheus.CounterVec {
	if err := prometheus.Register(collector); err != nil {
		if alreadyRegistered, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, castOK := alreadyRegistered.ExistingCollector.(*prometheus.CounterVec); castOK {
				return existing
			}
		}
	}
	return collector
}

func registerOrGetHistogramVec(collector *prometheus.HistogramVec) *prometheus.HistogramVec {
	if err := prometheus.Register(collector); err != nil {
		if alreadyRegistered, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, castOK := alreadyRegistered.ExistingCollector.(*prometheus.HistogramVec); castOK {
				return existing
			}
		}
	}
	return collector
}

func registerOrGetGauge(collector prometheus.Gauge) prometheus.Gauge {
	if err := prometheus.Register(collector); err != nil {
		if alreadyRegistered, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, castOK := alreadyRegistered.ExistingCollector.(prometheus.Gauge); castOK {
				return existing
			}
		}
	}
	return collector
}

func (s *Server) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s == nil || s.httpMetrics == nil {
			next.ServeHTTP(w, r)
			return
		}

		statusWriter := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()

		s.httpMetrics.inflightRequests.Inc()
		defer s.httpMetrics.inflightRequests.Dec()

		next.ServeHTTP(statusWriter, r)

		route := resolveRouteTemplate(r)
		status := strconv.Itoa(statusWriter.status)
		duration := time.Since(start).Seconds()

		s.httpMetrics.requestsTotal.WithLabelValues(r.Method, route, status).Inc()
		s.httpMetrics.requestDuration.WithLabelValues(r.Method, route, status).Observe(duration)
	})
}

func resolveRouteTemplate(r *http.Request) string {
	if r == nil {
		return "unknown"
	}

	if route := mux.CurrentRoute(r); route != nil {
		if tpl, err := route.GetPathTemplate(); err == nil && tpl != "" {
			return tpl
		}
	}
	if r.URL != nil && r.URL.Path != "" {
		return r.URL.Path
	}
	return "unknown"
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
