package middleware

import (
	"context"
	"net/http"
)

type ctxKeyTraceID int

const TraceIDKey ctxKeyTraceID = 0

const maxTraceIdLength = 1024

var TraceIDHeader = "Trace-Id"

func TraceID(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get(TraceIDHeader)
		if traceID != "" {
			runes := []rune(traceID)
			if len(runes) > maxTraceIdLength {
				traceID = string(runes[0:maxTraceIdLength])
			}
			w.Header().Add(TraceIDHeader, traceID)
			ctx := context.WithValue(r.Context(), TraceIDKey, traceID)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			next.ServeHTTP(w, r)
		}
	}
	return http.HandlerFunc(fn)
}

func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}
