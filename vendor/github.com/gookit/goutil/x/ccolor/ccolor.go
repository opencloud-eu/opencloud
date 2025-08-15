// Package ccolor is a simple color render library for terminal.
// Its main code is the code that is extracted and simplified from gookit/color,
//
// TIP:
//
//	If you want to render with richer colors, use the https://github.com/gookit/color package.
package ccolor

import (
	"fmt"
	"io"
	"log"
	"os"
	"regexp"

	"github.com/gookit/goutil/x/termenv"
)

// color render templates
//
// ESC 操作的表示:
//
//	"\033"(Octal 8进制) = "\x1b"(Hexadecimal 16进制) = 27 (10进制)
const (
	// StartSet chars
	StartSet = "\x1b["
	// ResetSet close all properties.
	ResetSet = "\x1b[0m"
	// SettingTpl string.
	SettingTpl = "\x1b[%sm"
	// FullColorTpl for build color code
	FullColorTpl = "\x1b[%sm%s\x1b[0m"
	// CodeSuffix string for color code.
	CodeSuffix = "[0m"
)

// CodeExpr regex to clear color codes eg "\033[1;36mText\x1b[0m"
const CodeExpr = `\033\[[\d;?]+m`

var (
	// last error
	lastErr error
	// output the default io.Writer message print
	output io.Writer = os.Stdout
	// match color codes
	codeRegex = regexp.MustCompile(CodeExpr)
)

// SetOutput set output writer
func SetOutput(w io.Writer) { output = w }

// LastErr info
func LastErr() error {
	defer func() {
		lastErr = nil
	}()
	return lastErr
}

//
// ---------------- support detect from termenv ----------------
//

// Disable color of current terminal.
func Disable() { termenv.DisableColor() }

// Level value of current terminal.
func Level() termenv.ColorLevel { return termenv.TermColorLevel() }

// IsSupportColor returns true if the terminal supports color.
func IsSupportColor() bool { return termenv.IsSupportColor() }

// IsSupport256Color returns true if the terminal supports 256 colors.
func IsSupport256Color() bool { return termenv.IsSupport256Color() }

// IsSupportTrueColor returns true if the terminal supports true color.
func IsSupportTrueColor() bool { return termenv.IsSupportTrueColor() }

//
// ---------------- for testing ----------------
//

// ForceEnableColor setting value. TIP: use for unit testing.
//
// Usage:
//
//	ccolor.ForceEnableColor()
//	defer ccolor.RevertColorSupport()
func ForceEnableColor() {
	termenv.ForceEnableColor()
}

// RevertColorSupport value
func RevertColorSupport() {
	termenv.RevertColorSupport()
}

//
// ---------------- print with color tag style ----------------
//

// Print parse color tag and print messages
func Print(v ...any) { Fprint(output, v...) }

// Printf format and print messages
func Printf(format string, v ...any) { Fprintf(output, format, v...) }

// Println messages with new line
func Println(v ...any) { Fprintln(output, v...) }

// Sprint parse color tags, return rendered string
func Sprint(v ...any) string {
	return ReplaceTag(fmt.Sprint(v...))
}

// Sprintf format and return rendered string
func Sprintf(format string, a ...any) string {
	return ReplaceTag(fmt.Sprintf(format, a...))
}

// Fprint auto parse color-tag, print rendered messages to the writer
func Fprint(w io.Writer, v ...any) {
	_, lastErr = fmt.Fprint(w, ReplaceTag(fmt.Sprint(v...)))
}

// Fprintf auto parse color-tag, print rendered messages to the writer.
func Fprintf(w io.Writer, format string, v ...any) {
	_, lastErr = fmt.Fprint(w, ReplaceTag(fmt.Sprintf(format, v...)))
}

// Fprintln auto parse color-tag, print rendered messages to the writer
func Fprintln(w io.Writer, v ...any) {
	_, lastErr = fmt.Fprintln(w, ReplaceTag(formatLikePrintln(v)))
}

// Lprint passes colored messages to a log.Logger for printing.
func Lprint(l *log.Logger, v ...any) {
	l.Print(ReplaceTag(fmt.Sprint(v...)))
}
