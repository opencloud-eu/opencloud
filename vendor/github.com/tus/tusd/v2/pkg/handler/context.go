package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"golang.org/x/exp/slog"
)

// httpContext is wrapper around context.Context that also carries the
// corresponding HTTP request and response writer, as well as an
// optional body reader
type httpContext struct {
	context.Context

	// res and req are the native request and response instances
	res  http.ResponseWriter
	resC *http.ResponseController
	req  *http.Request

	// body is nil by default and set by the user if the request body is consumed.
	body *bodyReader

	// cancel allows a user to cancel the internal request context, causing
	// the request body to be closed.
	cancel context.CancelCauseFunc

	// log is the logger for this request. It gets extended with more properties as the
	// request progresses and is identified.
	log *slog.Logger
}

// newContext constructs a new httpContext for the given request. This should only be done once
// per request and the context should be stored in the request, so it can be fetched with getContext.
func (h UnroutedHandler) newContext(w http.ResponseWriter, r *http.Request) *httpContext {
	// requestCtx is the context from the native request instance. It gets cancelled
	// if the connection closes, the request is cancelled (HTTP/2), ServeHTTP returns
	// or the server's base context is cancelled.
	requestCtx := r.Context()
	// On top of requestCtx, we construct a context that we can cancel, for example when
	// the post-receive hook stops an upload or if another uploads requests a lock to be released.
	cancellableCtx, cancelHandling := context.WithCancelCause(requestCtx)
	// On top of cancellableCtx, we construct a new context which gets cancelled with a delay.
	// See HookEvent.Context for more details, but the gist is that we want to give data stores
	// some more time to finish their buisness.
	delayedCtx := newDelayedContext(cancellableCtx, h.config.GracefulRequestCompletionTimeout)

	ctx := &httpContext{
		Context: delayedCtx,
		res:     w,
		resC:    http.NewResponseController(w),
		req:     r,
		body:    nil, // body can be filled later for PATCH requests
		cancel:  cancelHandling,
		log:     h.logger.With("method", r.Method, "path", r.URL.Path, "requestId", getRequestId(r)),
	}

	go func() {
		<-cancellableCtx.Done()

		// If the cause is one of our own errors, close a potential body and relay the error.
		cause := context.Cause(cancellableCtx)
		if (errors.Is(cause, ErrServerShutdown) || errors.Is(cause, ErrUploadInterrupted) || errors.Is(cause, ErrUploadStoppedByServer)) && ctx.body != nil {
			ctx.body.closeWithError(cause)
		}
	}()

	return ctx
}

// getContext tries to retrieve a httpContext from the request or constructs a new one.
func (h UnroutedHandler) getContext(w http.ResponseWriter, r *http.Request) *httpContext {
	c, ok := r.Context().(*httpContext)
	if !ok {
		c = h.newContext(w, r)
	}

	return c
}

// newDelayedContext returns a context with delayed cancellation propagation. If the parent context
// is done, the new context will also be cancelled but only after waiting the specified delay.
// Note: The parent context MUST be cancelled or otherwise this will leak resources. In the
// case of http.Request.Context, the net/http package ensures that the context is always cancelled.
func newDelayedContext(parent context.Context, delay time.Duration) context.Context {
	// Use context.WithoutCancel to preserve the values.
	ctx, cancel := context.WithCancel(context.WithoutCancel(parent))
	go func() {
		<-parent.Done()
		<-time.After(delay)
		cancel()
	}()

	return ctx
}
