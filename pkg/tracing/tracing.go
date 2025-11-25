package tracing

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	rtrace "github.com/opencloud-eu/reva/v2/pkg/trace"
	zlog "github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Propagator ensures the importer module uses the same trace propagation strategy.
var Propagator = propagation.NewCompositeTextMapPropagator(
	propagation.Baggage{},
	propagation.TraceContext{},
)

// GetServiceTraceProvider returns a configured open-telemetry trace provider.
func GetServiceTraceProvider(c ConfigConverter, serviceName string) (trace.TracerProvider, error) {
	var cfg Config
	if c == nil || reflect.ValueOf(c).IsNil() {
		cfg = Config{Enabled: false}
	} else {
		cfg = c.Convert()
	}

	if cfg.Enabled {
		return GetTraceProvider(cfg.Endpoint, cfg.Collector, serviceName, cfg.Type)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.NeverSample()),
	)
	rtrace.SetDefaultTracerProvider(tp)

	return tp, nil
}

// GetPropagator gets a configured propagator.
func GetPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.Baggage{},
		propagation.TraceContext{},
	)
}

// GetTraceProvider returns a configured open-telemetry trace provider.
func GetTraceProvider(endpoint, collector, serviceName, traceType string) (*sdktrace.TracerProvider, error) {
	normalizedType := strings.TrimSpace(strings.ToLower(traceType))

	switch normalizedType {
	case "":
		return getAutoTraceProvider(endpoint, collector, serviceName)
	case "otlp":
		if endpoint == "" {
			return nil, errors.New("tracing endpoint is required when trace type is 'otlp'")
		}
		return newOTLPGRPCProvider(endpoint, serviceName)
	case "otlp_grpc":
		if endpoint == "" {
			return nil, errors.New("tracing endpoint is required when trace type is 'otlp_grpc'")
		}
		return newOTLPGRPCProvider(endpoint, serviceName)
	case "otlp_http":
		if collector == "" {
			return nil, errors.New("tracing collector is required when trace type is 'otlp_http'")
		}
		return newOTLPHTTPProvider(collector, serviceName)
	default:
		return nil, fmt.Errorf("unknown trace type %s", traceType)
	}
}

func getAutoTraceProvider(endpoint, collector, serviceName string) (*sdktrace.TracerProvider, error) {
	switch {
	case endpoint != "":
		return newOTLPGRPCProvider(endpoint, serviceName)
	case collector != "":
		return newOTLPHTTPProvider(collector, serviceName)
	default:
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.NeverSample()),
		)
		rtrace.SetDefaultTracerProvider(tp)
		zlog.Warn().Msg("Tracing disabled: no OTLP endpoint or collector configured")
		return tp, nil
	}
}

func newOTLPGRPCProvider(endpoint, serviceName string) (*sdktrace.TracerProvider, error) {
	options := []otlptracegrpc.Option{
		otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	}

	if strings.Contains(endpoint, "://") {
		options = append(options, otlptracegrpc.WithEndpointURL(endpoint))
	} else {
		options = append(options, otlptracegrpc.WithEndpoint(endpoint))
	}

	exporter, err := otlptracegrpc.New(context.Background(), options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP gRPC exporter: %w", err)
	}

	resources, err := buildResource(serviceName)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resources),
	)
	rtrace.SetDefaultTracerProvider(tp)
	return tp, nil
}

func newOTLPHTTPProvider(endpoint, serviceName string) (*sdktrace.TracerProvider, error) {
	options := make([]otlptracehttp.Option, 0, 2)

	if strings.Contains(endpoint, "://") {
		options = append(options, otlptracehttp.WithEndpointURL(endpoint))
	} else {
		options = append(options, otlptracehttp.WithEndpoint(endpoint), otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(context.Background(), options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP HTTP exporter: %w", err)
	}

	resources, err := buildResource(serviceName)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resources),
	)
	rtrace.SetDefaultTracerProvider(tp)
	return tp, nil
}

func buildResource(serviceName string) (*resource.Resource, error) {
	return resource.New(
		context.Background(),
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			attribute.String("library.language", "go"),
		),
	)
}
