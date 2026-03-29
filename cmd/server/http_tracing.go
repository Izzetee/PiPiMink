package server

import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func (s *Server) tracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s == nil || s.config == nil || !s.config.OTelEnabled {
			next.ServeHTTP(w, r)
			return
		}

		prop := otel.GetTextMapPropagator()
		ctx := prop.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		route := resolveRouteTemplate(r)
		tracer := otel.Tracer(s.otelServiceName() + "/http")
		ctx, span := tracer.Start(
			ctx,
			r.Method+" "+route,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.route", route),
				attribute.String("http.target", r.URL.Path),
			),
		)
		defer span.End()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
