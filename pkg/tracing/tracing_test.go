package tracing

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestGetTraceProviderSetsGlobalPropagator(t *testing.T) {
	if _, err := GetTraceProvider(context.Background(), "none", "test"); err != nil {
		t.Fatal(err)
	}

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1},
		SpanID:     trace.SpanID{1},
		TraceFlags: trace.FlagsSampled,
	})
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(trace.ContextWithSpanContext(context.Background(), sc), carrier)

	if carrier.Get("traceparent") == "" {
		t.Fatal("global propagator did not inject a traceparent")
	}
	got := trace.SpanContextFromContext(otel.GetTextMapPropagator().Extract(context.Background(), carrier))
	if got.TraceID() != sc.TraceID() {
		t.Fatalf("extracted trace id %s, want %s", got.TraceID(), sc.TraceID())
	}
}
