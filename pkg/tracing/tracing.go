package tracing

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	rtrace "github.com/opencloud-eu/reva/v2/pkg/trace"
	zlog "github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

type exporterKind string

const (
	exporterNone    exporterKind = "none"
	exporterConsole exporterKind = "console"
	exporterOTLP    exporterKind = "otlp"
)

type resolvedConfig struct {
	exporter  exporterKind
	warnings  []string
	legacyEnv map[string]string
}

// Propagator ensures the importer module uses the same trace propagation strategy.
var Propagator = propagation.NewCompositeTextMapPropagator(
	propagation.Baggage{},
	propagation.TraceContext{},
)

// GetServiceTraceProvider returns a configured open-telemetry trace provider.
func GetServiceTraceProvider(c ConfigConverter, serviceName string) (trace.TracerProvider, error) {
	var cfg Config
	if c != nil && !reflect.ValueOf(c).IsNil() {
		cfg = c.Convert()
	}

	resolved, err := resolveConfig(cfg)
	if err != nil {
		return nil, err
	}

	for _, warn := range resolved.warnings {
		zlog.Warn().Str("service", serviceName).Msg(warn)
	}

	applyLegacyEnv(resolved.legacyEnv)

	switch resolved.exporter {
	case exporterNone:
		return createNoneProvider(), nil
	case exporterConsole:
		return createConsoleProvider(serviceName)
	case exporterOTLP:
		return createOTLPProvider(serviceName)
	default:
		return nil, fmt.Errorf("unsupported trace exporter %q", resolved.exporter)
	}
}

// GetPropagator gets a configured propagator.
func GetPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.Baggage{},
		propagation.TraceContext{},
	)
}

func resolveConfig(cfg Config) (resolvedConfig, error) {
	res := resolvedConfig{
		exporter:  exporterNone,
		warnings:  make([]string, 0, 4),
		legacyEnv: map[string]string{},
	}

	normalizedExporter := strings.ToLower(strings.TrimSpace(cfg.Exporter))

	if normalizedExporter == "" && strings.TrimSpace(cfg.Type) != "" {
		normalizedExporter = strings.ToLower(strings.TrimSpace(cfg.Type))
		res.warnings = append(res.warnings, "OC_TRACING_TYPE is deprecated; set OC_TRACES_EXPORTER or OTEL_TRACES_EXPORTER instead")
	}

	switch normalizedExporter {
	case "":
		// handled later based on Enabled / other legacy fields
	case string(exporterNone):
		res.exporter = exporterNone
	case string(exporterConsole):
		res.exporter = exporterConsole
	case string(exporterOTLP):
		res.exporter = exporterOTLP
	case "otlp_grpc":
		res.exporter = exporterOTLP
		res.legacyEnv["OTEL_EXPORTER_OTLP_PROTOCOL"] = "grpc"
		res.warnings = append(res.warnings, "trace exporter 'otlp_grpc' is deprecated; use OTEL_EXPORTER_OTLP_PROTOCOL=grpc instead")
	case "otlp_http":
		res.exporter = exporterOTLP
		res.legacyEnv["OTEL_EXPORTER_OTLP_PROTOCOL"] = "http/protobuf"
		res.warnings = append(res.warnings, "trace exporter 'otlp_http' is deprecated; use OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf instead")
	case "jaeger":
		res.exporter = exporterOTLP
		res.warnings = append(res.warnings, "Jaeger exporter is no longer supported; falling back to OTLP")
	default:
		return resolvedConfig{}, fmt.Errorf("unknown trace exporter %q", cfg.Exporter)
	}

	if normalizedExporter == "" {
		if cfg.Enabled {
			res.exporter = exporterOTLP
			res.warnings = append(res.warnings, "OC_TRACING_ENABLED is deprecated; defaulting to exporter 'otlp'. Set OC_TRACES_EXPORTER or OTEL_TRACES_EXPORTER explicitly.")
		} else {
			res.exporter = exporterNone
		}
	}

	if !cfg.Enabled && res.exporter != exporterNone && strings.TrimSpace(cfg.Type) != "" {
		res.warnings = append(res.warnings, "OC_TRACING_ENABLED is deprecated and ignored when an exporter is set; use exporter 'none' instead to disable tracing")
	}

	if res.exporter == exporterOTLP {
		if endpoint := strings.TrimSpace(cfg.Endpoint); endpoint != "" {
			res.legacyEnv["OTEL_EXPORTER_OTLP_ENDPOINT"] = endpoint
			res.warnings = append(res.warnings, "OC_TRACING_ENDPOINT is deprecated; configure OTEL_EXPORTER_OTLP_ENDPOINT instead")
			if _, ok := res.legacyEnv["OTEL_EXPORTER_OTLP_INSECURE"]; !ok {
				res.legacyEnv["OTEL_EXPORTER_OTLP_INSECURE"] = "true"
			}
		}
		if collector := strings.TrimSpace(cfg.Collector); collector != "" {
			res.legacyEnv["OTEL_EXPORTER_OTLP_ENDPOINT"] = collector
			res.legacyEnv["OTEL_EXPORTER_OTLP_PROTOCOL"] = "http/protobuf"
			res.warnings = append(res.warnings, "OC_TRACING_COLLECTOR is deprecated; configure OTEL_EXPORTER_OTLP_ENDPOINT/OTEL_EXPORTER_OTLP_PROTOCOL instead")
		}
	}

	return res, nil
}

func applyLegacyEnv(values map[string]string) {
	for key, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if current, ok := os.LookupEnv(key); ok && strings.TrimSpace(current) != "" {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			zlog.Debug().Err(err).Str("env", key).Msg("failed to apply legacy tracing env override")
		}
	}
}

func createNoneProvider() trace.TracerProvider {
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.NeverSample()),
	)
	rtrace.SetDefaultTracerProvider(tp)
	return tp
}

func createConsoleProvider(serviceName string) (trace.TracerProvider, error) {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("failed to create console trace exporter: %w", err)
	}

	res, err := buildResource(context.Background(), serviceName)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithResource(res),
	)
	rtrace.SetDefaultTracerProvider(tp)
	return tp, nil
}

func createOTLPProvider(serviceName string) (trace.TracerProvider, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exporter, err := newOTLPExporter(ctx)
	if err != nil {
		return nil, err
	}

	res, err := buildResource(context.Background(), serviceName)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	rtrace.SetDefaultTracerProvider(tp)
	return tp, nil
}

func buildResource(ctx context.Context, serviceName string) (*resource.Resource, error) {
	res, err := resource.New(
		ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			attribute.String("library.language", "go"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build tracing resource: %w", err)
	}
	return res, nil
}

func newOTLPExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	client, err := newOTLPClientFromEnv()
	if err != nil {
		return nil, err
	}
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}
	return exporter, nil
}

func newOTLPClientFromEnv() (otlptrace.Client, error) {
	protocol := strings.ToLower(strings.TrimSpace(firstNonEmpty(
		os.Getenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL"),
		os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"),
	)))

	switch protocol {
	case "", "grpc":
		return otlptracegrpc.NewClient(), nil
	case "http", "http/protobuf":
		return otlptracehttp.NewClient(), nil
	default:
		zlog.Warn().Str("protocol", protocol).Msg("unsupported OTLP protocol; defaulting to gRPC")
		return otlptracegrpc.NewClient(), nil
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
