package server

import (
	"context"
	"log"

	"PiPiMink/internal/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

func initOpenTelemetry(cfg *config.Config) (func(context.Context) error, error) {
	noopShutdown := func(context.Context) error { return nil }

	if cfg == nil || !cfg.OTelEnabled {
		log.Println("OpenTelemetry disabled")
		return noopShutdown, nil
	}

	opts := []otlptracehttp.Option{}
	if cfg.OTelExporterOTLPEndpoint != "" {
		opts = append(opts, otlptracehttp.WithEndpoint(cfg.OTelExporterOTLPEndpoint))
	}
	if cfg.OTelExporterOTLPInsecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(context.Background(), opts...)
	if err != nil {
		return noopShutdown, err
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.OTelServiceName),
		),
	)
	if err != nil {
		return noopShutdown, err
	}

	ttp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(cfg.OTelTraceSampleRatio))),
	)

	otel.SetTracerProvider(ttp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	log.Printf("OpenTelemetry enabled (service=%s, endpoint=%s)", cfg.OTelServiceName, cfg.OTelExporterOTLPEndpoint)
	return ttp.Shutdown, nil
}
