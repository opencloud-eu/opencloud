package middleware

import (
	"net/http"

	"github.com/justinas/alice"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

// Constructor wraps an http.Handler with middleware logic
type Constructor = alice.Constructor

// NopConstructor is a middleware that passes through the given http.Handler without modifying it.
var NopConstructor Constructor = func(h http.Handler) http.Handler { return h }

// middlewareLogger adds the middleware context to the logger
func middlewareLogger(logger log.Logger, middleware string) log.Logger {
	return log.Logger{Logger: logger.With().Str("middleware", middleware).Logger()}
}

// requestLogger builds a logger that contains detailed request information
func requestLogger(base log.Logger, r *http.Request) log.Logger {
	return log.Logger{Logger: base.With().
		Str("proto", r.Proto).
		Str("remote-addr", r.RemoteAddr).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Logger()}
}
