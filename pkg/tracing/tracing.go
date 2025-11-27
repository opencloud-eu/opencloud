package tracing

import (
	"context"
	"fmt"

	rtrace "github.com/opencloud-eu/reva/v2/pkg/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

// Propagator ensures the importer module uses the same trace propagation strategy.
var Propagator = propagation.NewCompositeTextMapPropagator(
	propagation.Baggage{},
	propagation.TraceContext{},
)

// Deprecated: GetServiceTraceProvider returns a configured open-telemetry trace provider. Use GetTraceProvider.
func GetServiceTraceProvider(exporter, serviceName string) (trace.TracerProvider, error) {
	return GetTraceProvider(context.Background(), exporter, serviceName)
}

// GetPropagator gets a configured propagator.
func GetPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.Baggage{},
		propagation.TraceContext{},
	)
}

// GetTraceProvider returns a configured open-telemetry trace provider.
func GetTraceProvider(ctx context.Context, exporter, serviceName string) (*sdktrace.TracerProvider, error) {

	// Create resource - shared across all exporters
	resources, err := createResource(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	var tp *sdktrace.TracerProvider

	switch exporter {
	case "", "none":
		// No-op exporter - never sample
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.NeverSample()),
			sdktrace.WithResource(resources),
		)

	case "console":
		// Console exporter - prints to stdout (useful for debugging)
		consoleExporter, err := stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create console exporter: %w", err)
		}

		// Use SimpleSpanProcessor for console to get immediate output
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(consoleExporter)),
			sdktrace.WithResource(resources),
		)

	case "otlp":
		// OTLP exporter - connects to collector
		// This automatically reads:
		// - OTEL_EXPORTER_OTLP_ENDPOINT
		// - OTEL_EXPORTER_OTLP_TRACES_ENDPOINT (takes precedence)
		// - OTEL_EXPORTER_OTLP_HEADERS
		// - OTEL_EXPORTER_OTLP_INSECURE
		// - OTEL_EXPORTER_OTLP_CERTIFICATE (for custom CA)
		otlpExporter, err := otlptracegrpc.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
		}

		// Create tracer provider
		// This automatically reads:
		// - OTEL_TRACES_SAMPLER
		// - OTEL_TRACES_SAMPLER_ARG
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(otlpExporter),
			sdktrace.WithResource(resources),
		)

	default:
		return nil, fmt.Errorf("unsupported trace exporter: %q (supported: none, console, otlp)", exporter)
	}

	// Set as global default
	rtrace.SetDefaultTracerProvider(tp)

	return tp, nil
}

// createResource creates a resource with service information
func createResource(ctx context.Context, serviceName string) (*resource.Resource, error) {
	return resource.New(ctx,
		// Reads OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME
		resource.WithFromEnv(),
		// Host and process information
		resource.WithHost(),
		resource.WithProcess(),
		// Service attributes
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			attribute.String("library.language", "go"),
		),
	)
}
