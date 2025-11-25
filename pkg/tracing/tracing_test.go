package tracing

import (
	"context"
	"errors"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestGetTraceProvider_AutoDisabled(t *testing.T) {
	tp, err := GetTraceProvider("", "", "svc", "")
	if err != nil {
		t.Fatalf("GetTraceProvider returned error: %v", err)
	}
	assertNeverSampled(t, tp)
}

func TestGetTraceProvider_AutoSelectsGRPC(t *testing.T) {
	tp, err := GetTraceProvider("localhost:4317", "", "svc", "")
	if err != nil {
		t.Fatalf("GetTraceProvider returned error: %v", err)
	}
	assertAlwaysSampled(t, tp)
}

func TestGetTraceProvider_AutoSelectsHTTP(t *testing.T) {
	tp, err := GetTraceProvider("", "http://localhost:4318", "svc", "")
	if err != nil {
		t.Fatalf("GetTraceProvider returned error: %v", err)
	}
	assertAlwaysSampled(t, tp)
}

func TestGetTraceProvider_TypeSpecific(t *testing.T) {
	tests := []struct {
		name      string
		endpoint  string
		collector string
		traceType string
		wantErr   error
	}{
		{
			name:      "otlp requires endpoint",
			traceType: "otlp",
			wantErr:   errors.New("tracing endpoint is required when trace type is 'otlp'"),
		},
		{
			name:      "otlp_grpc requires endpoint",
			traceType: "otlp_grpc",
			wantErr:   errors.New("tracing endpoint is required when trace type is 'otlp_grpc'"),
		},
		{
			name:      "otlp_http requires collector",
			traceType: "otlp_http",
			wantErr:   errors.New("tracing collector is required when trace type is 'otlp_http'"),
		},
		{
			name:      "unknown type",
			traceType: "jaeger",
			wantErr:   errors.New("unknown trace type jaeger"),
		},
		{
			name:      "otlp_grpc success",
			endpoint:  "localhost:4317",
			traceType: "otlp_grpc",
		},
		{
			name:      "otlp_http success",
			collector: "http://localhost:4318",
			traceType: "otlp_http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp, err := GetTraceProvider(tt.endpoint, tt.collector, "svc", tt.traceType)
			if tt.wantErr != nil {
				if err == nil || err.Error() != tt.wantErr.Error() {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertAlwaysSampled(t, tp)
		})
	}
}

func TestGetServiceTraceProvider_DisabledByDefault(t *testing.T) {
	tp, err := GetServiceTraceProvider(nil, "svc")
	if err != nil {
		t.Fatalf("GetServiceTraceProvider returned error: %v", err)
	}
	assertNeverSampledFromInterface(t, tp)
}

func assertAlwaysSampled(t *testing.T, tp *sdktrace.TracerProvider) {
	t.Helper()
	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "sampled")
	if !span.SpanContext().IsSampled() {
		t.Fatalf("expected span to be sampled")
	}
	span.End()
}

func assertNeverSampled(t *testing.T, tp *sdktrace.TracerProvider) {
	t.Helper()
	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "unsampled")
	if span.SpanContext().IsSampled() {
		t.Fatalf("expected span not to be sampled")
	}
	span.End()
}

func assertNeverSampledFromInterface(t *testing.T, provider trace.TracerProvider) {
	t.Helper()
	tp := provider.(*sdktrace.TracerProvider)
	assertNeverSampled(t, tp)
}
