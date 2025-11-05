package middleware

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

// deadlineReadCloser wraps an io.ReadCloser to enforce a read deadline.
// It uses an http.ResponseController to set deadlines dynamically during reads.
type deadlineReadCloser struct {
	// timeout duration defines the interval for extending the read deadline.
	timeout    time.Duration
	inner      io.ReadCloser
	controller *http.ResponseController
	logger     log.Logger
}

// NewDeadlineReadCloser creates a new deadline-enforcing ReadCloser.
func NewDeadlineReadCloser(inner io.ReadCloser, rc *http.ResponseController, timeout time.Duration, logger log.Logger) (io.ReadCloser, error) {
	if inner == nil || rc == nil {
		return nil, errors.New("inner reader and controller must not be nil")
	}

	if err := rc.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}

	return &deadlineReadCloser{
		inner:      inner,
		controller: rc,
		timeout:    timeout,
		logger:     logger,
	}, nil
}

// Read proxies the Read to the inner io.ReadCloser,
// each Read cycle extends the deadline based on the configured timeout.
func (drc *deadlineReadCloser) Read(p []byte) (int, error) {
	n, err := drc.inner.Read(p)

	// don't consider errors to give the downstream operation time to recover eventually
	if n > 0 {
		deadline := time.Now().Add(drc.timeout)
		// refresh the connection readDeadline after each read
		if setErr := drc.controller.SetReadDeadline(deadline); setErr != nil {
			// Failing means that the wrapped writer does not implement the SetReadDeadline interface.
			// This should not happen because NewDeadlineReadCloser already checks the compatibility,
			// even if it errors, it's not a problem, and reading should continue.
			drc.logger.Error().Err(setErr).
				Time("deadline", deadline).
				Dur("timeout", drc.timeout).
				Msg("failed to set read deadline")
		}
	}

	return n, err
}

// Close proxies the Close call to its inner io.ReadCloser
func (drc *deadlineReadCloser) Close() error {
	return drc.inner.Close()
}

// ReadDeadline creates middleware to enforce a read deadline for each incoming HTTP request body.
// If the provided timeout is less than or equal to zero, or the body is nil, it returns a noopHandler.
// The read deadline gets refreshed on every successful read of the request body.
func ReadDeadline(timeout time.Duration, logger log.Logger) Constructor {
	if timeout <= 0 {
		return NopConstructor
	}

	mwLogger := middlewareLogger(logger, "ReadDeadline")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body == nil {
				next.ServeHTTP(w, r)
				return
			}

			rLogger := requestLogger(mwLogger, r)

			// If the controlled body can't be created, return early and pass the request and response down the line
			controlledBody, err := NewDeadlineReadCloser(r.Body, http.NewResponseController(w), timeout, rLogger)
			if err != nil {
				rLogger.Error().Err(err).
					Dur("timeout", timeout).
					Msg("failed to create reader")
				next.ServeHTTP(w, r)
				return
			}

			r.Body = controlledBody
			next.ServeHTTP(w, r)
		})
	}
}
