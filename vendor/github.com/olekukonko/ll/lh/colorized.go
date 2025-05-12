package lh

import (
	"fmt"
	"github.com/olekukonko/ll/lx"
	"io"
	"os"
	"sort"
	"strings"
)

type Palette struct {
	Header    string
	Goroutine string
	Func      string
	Path      string
	FileLine  string
	Reset     string
	Pos       string
	Hex       string
	Ascii     string
	Debug     string
	Info      string
	Warn      string
	Error     string
	Title     string
}

var darkPalette = Palette{
	Header:    "\033[1;31m",     // Bold red
	Goroutine: "\033[1;36m",     // Bold cyan
	Func:      "\033[97m",       // Bright white
	Path:      "\033[38;5;245m", // Light gray
	FileLine:  "\033[38;5;111m", // Muted light blue
	Reset:     "\033[0m",

	Title: "\033[38;5;245m", // Light gray
	Pos:   "\033[38;5;117m", // Light blue
	Hex:   "\033[38;5;156m", // Light green
	Ascii: "\033[38;5;224m", // Light pink

	Debug: "\033[36m", // Cyan
	Info:  "\033[32m", // Green
	Warn:  "\033[33m", // Yellow
	Error: "\033[31m", // Red
}

var lightPalette = Palette{
	Header:    "\033[1;31m", // Same red
	Goroutine: "\033[34m",   // Blue (darker, better for light bg)
	Func:      "\033[30m",   // Black text
	Path:      "\033[90m",   // Dark gray
	FileLine:  "\033[94m",   // Blue
	Reset:     "\033[0m",

	Title: "\033[38;5;245m", // Light gray
	Pos:   "\033[38;5;117m", // Light blue
	Hex:   "\033[38;5;156m", // Light green
	Ascii: "\033[38;5;224m", // Light pink

	Debug: "\033[36m", // Cyan
	Info:  "\033[32m", // Green
	Warn:  "\033[33m", // Yellow
	Error: "\033[31m", // Red

}

// ColorizedHandler is a handler that outputs log entries with ANSI color codes.
type ColorizedHandler struct {
	w       io.Writer
	palette Palette
}

type ColorOption func(*ColorizedHandler)

func WithColorPallet(pallet Palette) ColorOption {
	return func(c *ColorizedHandler) {
		c.palette = pallet
	}
}

// NewColorizedHandler creates a new ColorizedHandler writing to the specified writer.
func NewColorizedHandler(w io.Writer, opts ...ColorOption) *ColorizedHandler {
	c := &ColorizedHandler{w: w}
	for _, opt := range opts {
		opt(c)
	}
	c.palette = c.detectPalette()
	return c
}

// Handle processes a log entry and writes it with ANSI color codes.
func (h *ColorizedHandler) Handle(e *lx.Entry) error {
	switch e.Class {
	case lx.ClassDump:
		return h.handleDumpOutput(e)
	case lx.ClassRaw:
		_, err := h.w.Write([]byte(e.Message))
		return err
	default:
		return h.handleRegularOutput(e)
	}
}

// handleRegularOutput handles normal log entries
func (h *ColorizedHandler) handleRegularOutput(e *lx.Entry) error {
	var builder strings.Builder

	// Namespace formatting
	h.formatNamespace(&builder, e)

	// Colorized level
	h.formatLevel(&builder, e)

	// Message and fields
	builder.WriteString(e.Message)
	h.formatFields(&builder, e)

	// fmt.Println("------------>", len(e.Stack))
	// Stack trace if present
	if len(e.Stack) > 0 {
		h.formatStack(&builder, e.Stack)
	}

	// Newline if needed
	if e.Level != lx.LevelNone {
		builder.WriteString(lx.Newline)
	}

	_, err := h.w.Write([]byte(builder.String()))
	return err
}

func (h *ColorizedHandler) formatNamespace(b *strings.Builder, e *lx.Entry) {
	if e.Namespace == "" {
		return
	}

	b.WriteString(lx.LeftBracket)
	switch e.Style {
	case lx.NestedPath:
		parts := strings.Split(e.Namespace, lx.Slash)
		for i, part := range parts {
			b.WriteString(part)
			b.WriteString(lx.RightBracket)
			if i < len(parts)-1 {
				b.WriteString(lx.Arrow)
				b.WriteString(lx.LeftBracket)
			}
		}
	default:
		b.WriteString(e.Namespace)
		b.WriteString(lx.RightBracket)
	}
	b.WriteString(lx.Colon)
	b.WriteString(lx.Space)
}

func (h *ColorizedHandler) formatLevel(b *strings.Builder, e *lx.Entry) {
	color := map[lx.LevelType]string{
		lx.LevelDebug: h.palette.Debug, // Cyan
		lx.LevelInfo:  h.palette.Info,  // Green
		lx.LevelWarn:  h.palette.Warn,  // Yellow
		lx.LevelError: h.palette.Error, // Red
	}[e.Level]

	b.WriteString(color)
	b.WriteString(e.Level.String())
	b.WriteString(h.palette.Reset)
	b.WriteString(lx.Colon)
	b.WriteString(lx.Space)
}

func (h *ColorizedHandler) formatFields(b *strings.Builder, e *lx.Entry) {
	if len(e.Fields) == 0 {
		return
	}

	var keys []string
	for k := range e.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	b.WriteString(lx.Space)
	b.WriteString(lx.LeftBracket)
	for i, k := range keys {
		if i > 0 {
			b.WriteString(lx.Space)
		}
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(fmt.Sprint(e.Fields[k]))
	}
	b.WriteString(lx.RightBracket)
}

func (h *ColorizedHandler) formatStack(b *strings.Builder, stack []byte) {

	b.WriteString("\n")
	b.WriteString(h.palette.Header)
	b.WriteString("[stack]")
	b.WriteString(h.palette.Reset)
	b.WriteString("\n")

	lines := strings.Split(string(stack), "\n")
	if len(lines) == 0 {
		return
	}

	b.WriteString("  ┌─ ")
	b.WriteString(h.palette.Goroutine)
	b.WriteString(lines[0])
	b.WriteString(h.palette.Reset)
	b.WriteString("\n")

	// Pair function name and file path lines
	for i := 1; i < len(lines)-1; i += 2 {
		funcLine := strings.TrimSpace(lines[i])
		pathLine := strings.TrimSpace(lines[i+1])

		if funcLine != "" {
			b.WriteString("  │   ")
			b.WriteString(h.palette.Func)
			b.WriteString(funcLine)
			b.WriteString(h.palette.Reset)
			b.WriteString("\n")
		}
		if pathLine != "" {
			b.WriteString("  │   ")

			// Look for last "/" before ".go:"
			lastSlash := strings.LastIndex(pathLine, "/")
			goIndex := strings.Index(pathLine, ".go:")

			if lastSlash >= 0 && goIndex > lastSlash {
				// Prefix path
				prefix := pathLine[:lastSlash+1]
				// File and line (e.g., ll.go:698 +0x5c)
				suffix := pathLine[lastSlash+1:]

				b.WriteString(h.palette.Path)
				b.WriteString(prefix)
				b.WriteString(h.palette.Reset)

				b.WriteString(h.palette.Path) // Use mainPath color for suffix
				b.WriteString(suffix)
				b.WriteString(h.palette.Reset)
			} else {
				// Fallback: whole line is gray
				b.WriteString(h.palette.Path)
				b.WriteString(pathLine)
				b.WriteString(h.palette.Reset)
			}

			b.WriteString("\n")
		}

	}

	// Handle any remaining unpaired line
	if len(lines)%2 == 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
		b.WriteString("  │   ")
		b.WriteString(h.palette.Func)
		b.WriteString(strings.TrimSpace(lines[len(lines)-1]))
		b.WriteString(h.palette.Reset)
		b.WriteString("\n")
	}

	b.WriteString("  └\n")
}

func (h *ColorizedHandler) handleDumpOutput(e *lx.Entry) error {
	var builder strings.Builder
	builder.WriteString(h.palette.Title)
	builder.WriteString("---- BEGIN DUMP ----")
	builder.WriteString(h.palette.Reset)
	builder.WriteString("\n")

	lines := strings.Split(e.Message, "\n")
	length := len(lines)
	for i, line := range lines {
		if strings.HasPrefix(line, "pos ") {
			parts := strings.SplitN(line, "hex:", 2)
			if len(parts) == 2 {
				builder.WriteString(h.palette.Pos)
				builder.WriteString(parts[0])
				builder.WriteString(h.palette.Reset)

				hexAscii := strings.SplitN(parts[1], "'", 2)
				builder.WriteString(h.palette.Hex)
				builder.WriteString("hex:")
				builder.WriteString(hexAscii[0])
				builder.WriteString(h.palette.Reset)

				if len(hexAscii) > 1 {
					builder.WriteString(h.palette.Ascii)
					builder.WriteString("'")
					builder.WriteString(hexAscii[1])
					builder.WriteString(h.palette.Reset)
				}
			}
		} else if strings.HasPrefix(line, "Dumping value of type:") {
			builder.WriteString(h.palette.Header)
			builder.WriteString(line)
			builder.WriteString(h.palette.Reset)
		} else {
			builder.WriteString(line)
		}

		// don't print for last line
		if i < length-1 {
			builder.WriteString("\n")
		}

	}

	builder.WriteString(h.palette.Title)
	builder.WriteString("---- END DUMP ----")
	builder.WriteString(h.palette.Reset)
	builder.WriteString("\n")

	_, err := h.w.Write([]byte(builder.String()))
	return err
}

func (h *ColorizedHandler) detectPalette() Palette {
	// Check TERM_BACKGROUND (e.g., iTerm2)
	if bg, ok := os.LookupEnv("TERM_BACKGROUND"); ok {
		if bg == "light" {
			return lightPalette
		}
		return darkPalette
	}

	// Check COLORFGBG (traditional xterm)
	if fgBg, ok := os.LookupEnv("COLORFGBG"); ok {
		parts := strings.Split(fgBg, ";")
		if len(parts) >= 2 {
			bg := parts[len(parts)-1]                    // last part (some terminals add more fields)
			if bg == "7" || bg == "15" || bg == "0;15" { // handle variations
				return lightPalette
			}
		}
	}

	// Check macOS dark mode
	if style, ok := os.LookupEnv("AppleInterfaceStyle"); ok && strings.EqualFold(style, "dark") {
		return darkPalette
	}

	// Default: dark (conservative choice for terminals)
	return darkPalette
}
