package ccolor

import (
	"fmt"
	"io"
	"strings"

	"github.com/gookit/goutil/x/termenv"
)

// ColorsToCode convert colors to code. return like "32;45;3"
func ColorsToCode(colors ...Color) string {
	if len(colors) == 0 {
		return ""
	}

	var codes []string
	for _, color := range colors {
		codes = append(codes, color.String())
	}

	return strings.Join(codes, ";")
}

/*************************************************************
 * render color code
 *************************************************************/

// RenderCode render message by color code.
//
// Usage:
//
//	msg := RenderCode("3;32;45", "some", "message")
func RenderCode(code string, args ...any) string {
	var message string
	if ln := len(args); ln == 0 {
		return ""
	}

	message = fmt.Sprint(args...)
	if len(code) == 0 {
		return message
	}

	// disabled OR not support color
	if !termenv.IsSupportColor() {
		return ClearCode(message)
	}

	// return fmt.Sprintf(FullColorTpl, code, message)
	return StartSet + code + "m" + message + ResetSet
}

// RenderString render a string with color code.
//
// Usage:
//
//	msg := RenderString("3;32;45", "a message")
func RenderString(code string, str string) string {
	if len(code) == 0 || str == "" {
		return str
	}

	// disabled OR not support color
	if !termenv.IsSupportColor() {
		return ClearCode(str)
	}

	// return fmt.Sprintf(FullColorTpl, code, str)
	return StartSet + code + "m" + str + ResetSet
}

// RenderWithSpaces Render code with spaces.
// If the number of args is > 1, a space will be added between the args
func RenderWithSpaces(code string, args ...any) string {
	msg := formatLikePrintln(args)
	if len(code) == 0 {
		return msg
	}

	// disabled OR not support color
	if !termenv.IsSupportColor() {
		return ClearCode(msg)
	}
	return StartSet + code + "m" + msg + ResetSet
}

// ClearCode clear color codes.
//
// eg:
//
//	"\033[36;1mText\x1b[0m" -> "Text"
func ClearCode(str string) string {
	if !strings.Contains(str, CodeSuffix) {
		return str
	}
	return codeRegex.ReplaceAllString(str, "")
}

/*************************************************************
 * helper methods for print
 *************************************************************/

// new implementation, support render full color code on pwsh.exe, cmd.exe
func doPrint(code, str string) {
	_, lastErr = fmt.Fprint(output, RenderString(code, str))
}

func doPrintTo(w io.Writer, code, str string) {
	_, lastErr = fmt.Fprint(w, RenderString(code, str))
}

// new implementation, support render full color code on pwsh.exe, cmd.exe
func doPrintln(code string, args []any) {
	_, lastErr = fmt.Fprintln(output, RenderString(code, formatLikePrintln(args)))
}

// use Println, will add spaces for each arg
func formatLikePrintln(args []any) (message string) {
	if ln := len(args); ln == 0 {
		message = ""
	} else if ln == 1 {
		message = fmt.Sprint(args[0])
	} else {
		message = fmt.Sprintln(args...)
		// clear last "\n"
		message = message[:len(message)-1]
	}
	return
}

