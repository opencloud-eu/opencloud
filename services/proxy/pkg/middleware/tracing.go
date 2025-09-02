package middleware

import (
	"net/http"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Tracer provides a middleware to start traces
func Tracer(tp trace.TracerProvider) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return &tracer{
			next:          next,
			traceProvider: tp,
		}
	}
}

type tracer struct {
	next          http.Handler
	traceProvider trace.TracerProvider
}

func (m tracer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	span := trace.SpanFromContext(r.Context())
	span.SetAttributes(
		attribute.KeyValue{
			Key:   "x-request-id",
			Value: attribute.StringValue(chimiddleware.GetReqID(r.Context())),
		})

	m.next.ServeHTTP(w, r)
}
