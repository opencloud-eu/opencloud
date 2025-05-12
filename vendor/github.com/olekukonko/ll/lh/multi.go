package lh

import (
	"errors"
	"fmt"
	"github.com/olekukonko/ll/lx"
)

// MultiHandler combines multiple handlers to process log entries concurrently.
type MultiHandler struct {
	Handlers []lx.Handler // List of handlers to process each log entry
}

// NewMultiHandler creates a new MultiHandler with the specified handlers.
// It accepts a variadic list of handlers to be executed in order.
func NewMultiHandler(h ...lx.Handler) *MultiHandler {
	return &MultiHandler{
		Handlers: h,
	}
}

// Handle implements the Handler interface, calling Handle on each handler in sequence.
// It collects any errors from handlers and combines them into a single error using errors.Join.
func (h *MultiHandler) Handle(e *lx.Entry) error {
	var errs []error
	for i, handler := range h.Handlers {
		// Execute each handler and capture any error with its index
		if err := handler.Handle(e); err != nil {
			errs = append(errs, fmt.Errorf("handler %d: %w", i, err))
		}
	}
	// Return a combined error if any handlers failed, or nil if all succeeded
	return errors.Join(errs...)
}
