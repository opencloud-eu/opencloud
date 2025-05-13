package lh

import (
	"context"
	"github.com/olekukonko/ll/lx"
	"log/slog"
)

// SlogHandler adapts a slog.Handler to implement lx.Handler.
type SlogHandler struct {
	slogHandler slog.Handler
}

// NewSlogHandler creates a new SlogHandler wrapping the provided slog.Handler.
func NewSlogHandler(h slog.Handler) *SlogHandler {
	return &SlogHandler{slogHandler: h}
}

// Handle converts an lx.Entry to slog.Record and delegates to the slog.Handler.
func (h *SlogHandler) Handle(e *lx.Entry) error {
	// Convert lx.LevelType to slog.Level
	level := toSlogLevel(e.Level)

	// Create a slog.Record with the entry's data
	record := slog.NewRecord(
		e.Timestamp, // time.Time
		level,       // slog.Level
		e.Message,   // string
		0,           // pc (program counter, optional)
	)

	// Add standard fields as attributes
	record.AddAttrs(
		slog.String("namespace", e.Namespace),
		slog.String("class", e.Class.String()),
	)

	// Add stack trace if present
	if len(e.Stack) > 0 {
		record.AddAttrs(slog.String("stack", string(e.Stack)))
	}

	// Add custom fields
	for k, v := range e.Fields {
		record.AddAttrs(slog.Any(k, v))
	}

	// Handle the record with the underlying slog.Handler
	return h.slogHandler.Handle(context.Background(), record)
}

// toSlogLevel converts lx.LevelType to slog.Level.
func toSlogLevel(level lx.LevelType) slog.Level {
	switch level {
	case lx.LevelDebug:
		return slog.LevelDebug
	case lx.LevelInfo:
		return slog.LevelInfo
	case lx.LevelWarn:
		return slog.LevelWarn
	case lx.LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo // Default for unknown levels
	}
}
