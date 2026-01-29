package service

import (
	"context"
	"net/url"

	"github.com/opencloud-eu/opencloud/services/webfinger/pkg/webfinger"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// NewTracing returns a service that instruments traces.
func NewTracing(next Service, tp trace.TracerProvider) Service {
	return tracing{
		next: next,
		tp:   tp,
	}
}

type tracing struct {
	next Service
	tp   trace.TracerProvider
}

// Webfinger implements the Service interface.
func (t tracing) Webfinger(ctx context.Context, queryTarget *url.URL, rels []string, platform string) (webfinger.JSONResourceDescriptor, error) {
	spanOpts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.KeyValue{Key: "query_target", Value: attribute.StringValue(queryTarget.String())},
			attribute.KeyValue{Key: "rels", Value: attribute.StringSliceValue(rels)},
		),
	}
	ctx, span := t.tp.Tracer("webfinger").Start(ctx, "Webfinger", spanOpts...)
	defer span.End()

	return t.next.Webfinger(ctx, queryTarget, rels, platform)
}
