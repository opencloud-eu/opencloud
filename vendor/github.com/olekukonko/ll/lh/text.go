package lh

import (
	"fmt"
	"github.com/olekukonko/ll/lx"
	"io"
	"sort"
	"strings"
)

// TextHandler is a handler that outputs log entries as plain text.
type TextHandler struct {
	w io.Writer
}

// NewTextHandler creates a new TextHandler writing to the specified writer.
func NewTextHandler(w io.Writer) *TextHandler {
	return &TextHandler{w: w}
}

// Handle processes a log entry and writes it as plain text.
func (h *TextHandler) Handle(e *lx.Entry) error {
	// Special handling for dump output
	if e.Class == lx.ClassDump {
		return h.handleDumpOutput(e)
	}

	if e.Class == lx.ClassRaw {
		_, err := h.w.Write([]byte(e.Message))
		return err
	}

	return h.handleRegularOutput(e)
}

// handleRegularOutput handles normal log entries
func (h *TextHandler) handleRegularOutput(e *lx.Entry) error {
	var builder strings.Builder

	// Namespace
	switch e.Style {
	case lx.NestedPath:
		if e.Namespace != "" {
			parts := strings.Split(e.Namespace, lx.Slash)
			for i, part := range parts {
				builder.WriteString(lx.LeftBracket)
				builder.WriteString(part)
				builder.WriteString(lx.RightBracket)
				if i < len(parts)-1 {
					builder.WriteString(lx.Arrow)
				}
			}
			builder.WriteString(lx.Colon)
			builder.WriteString(lx.Space)
		}
	default:
		if e.Namespace != "" {
			builder.WriteString(lx.LeftBracket)
			builder.WriteString(e.Namespace)
			builder.WriteString(lx.RightBracket)
			builder.WriteString(lx.Space)
		}
	}

	// Level and message
	builder.WriteString(e.Level.String())
	builder.WriteString(lx.Colon)
	builder.WriteString(lx.Space)
	builder.WriteString(e.Message)

	// Fields
	if len(e.Fields) > 0 {
		var keys []string
		for k := range e.Fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		builder.WriteString(lx.Space)
		builder.WriteString(lx.LeftBracket)
		for i, k := range keys {
			if i > 0 {
				builder.WriteString(lx.Space)
			}
			builder.WriteString(k)
			builder.WriteString("=")
			builder.WriteString(fmt.Sprint(e.Fields[k]))
		}
		builder.WriteString(lx.RightBracket)
	}

	// Stack (no color, just plain indent)
	if len(e.Stack) > 0 {
		h.formatStack(&builder, e.Stack)
	}

	// Newline
	if e.Level != lx.LevelNone {
		builder.WriteString(lx.Newline)
	}

	_, err := h.w.Write([]byte(builder.String()))
	return err
}

// handleDumpOutput specially formats hex dump output (plain text version)
func (h *TextHandler) handleDumpOutput(e *lx.Entry) error {
	// For text handler, we just add a newline before dump output
	var builder strings.Builder

	// Add a separator line before dump output
	builder.WriteString("---- BEGIN DUMP ----\n")
	builder.WriteString(e.Message)
	builder.WriteString("---- END DUMP ----\n")

	_, err := h.w.Write([]byte(builder.String()))
	return err
}

func (h *TextHandler) formatStack(b *strings.Builder, stack []byte) {
	lines := strings.Split(string(stack), "\n")
	if len(lines) == 0 {
		return
	}

	b.WriteString("\n[stack]\n")

	// First line: goroutine
	b.WriteString("  ┌─ ")
	b.WriteString(lines[0])
	b.WriteString("\n")

	// Iterate through remaining lines
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		if strings.Contains(line, ".go") {
			// File path lines get extra indent
			b.WriteString("  ├       ")
		} else {
			// Function names
			b.WriteString("  │   ")
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("  └\n")
}
