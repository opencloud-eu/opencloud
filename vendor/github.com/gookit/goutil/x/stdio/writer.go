package stdio

import (
	"fmt"
	"io"
)

// WriteWrapper warp io.Writer support more operate methods.
type WriteWrapper struct {
	Out io.Writer
}

// WrapW instance
func WrapW(w io.Writer) *WriteWrapper {
	return &WriteWrapper{Out: w}
}

// NewWriteWrapper instance
func NewWriteWrapper(w io.Writer) *WriteWrapper {
	return &WriteWrapper{Out: w}
}

// Write bytes data
func (w *WriteWrapper) Write(p []byte) (n int, err error) {
	return w.Out.Write(p)
}

// Writef data to output
func (w *WriteWrapper) Writef(tpl string, vs ...any) (n int, err error) {
	return fmt.Fprintf(w.Out, tpl, vs...)
}

// WriteByte data
func (w *WriteWrapper) WriteByte(c byte) error {
	_, err := w.Out.Write([]byte{c})
	return err
}

// WriteString data
func (w *WriteWrapper) WriteString(s string) (n int, err error) {
	if sw, ok := w.Out.(io.StringWriter); ok {
		return sw.WriteString(s)
	}
	return w.Out.Write([]byte(s))
}

// String get write data string
func (w *WriteWrapper) String() string {
	if sw, ok := w.Out.(fmt.Stringer); ok {
		return sw.String()
	}
	return ""
}
