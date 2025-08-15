package stdio

import (
	"fmt"
	"io"
	"strings"
)

// Fprint to writer, will ignore error
func Fprint(w io.Writer, a ...any) {
	_, _ = fmt.Fprint(w, a...)
}

// Fprintf to writer, will ignore error
func Fprintf(w io.Writer, tpl string, vs ...any) {
	_, _ = fmt.Fprintf(w, tpl, vs...)
}

// Fprintln to writer, will ignore error
func Fprintln(w io.Writer, a ...any) {
	_, _ = fmt.Fprintln(w, a...)
}

// WriteStringTo a writer, will ignore error
func WriteStringTo(w io.Writer, ss ...string) {
	if len(ss) == 1 {
		_, _ = io.WriteString(w, ss[0])
	} else if len(ss) > 1 {
		_, _ = io.WriteString(w, strings.Join(ss, ""))
	}
}
