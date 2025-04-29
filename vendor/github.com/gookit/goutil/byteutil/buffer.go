package byteutil

import (
	"bytes"
	"fmt"
)

// Buffer wrap and extends the bytes.Buffer, add some useful methods
// and implements the io.Writer, io.Closer and stdio.Flusher interfaces
type Buffer struct {
	bytes.Buffer
	// custom error for testing
	CloseErr error
	FlushErr error
	SyncErr  error
}

// NewBuffer instance
func NewBuffer() *Buffer {
	return &Buffer{}
}

// PrintByte to buffer, ignore error. alias of WriteByte()
func (b *Buffer) PrintByte(c byte) {
	_ = b.WriteByte(c)
}

// WriteStr1 quiet write one string to buffer
func (b *Buffer) WriteStr1(s string) {
	b.writeStringNl(s, false)
}

// WriteStr1Nl quiet write one string and end with newline
func (b *Buffer) WriteStr1Nl(s string) {
	b.writeStringNl(s, true)
}

// writeStringNl quiet write one string and end with newline
func (b *Buffer) writeStringNl(s string, nl bool) {
	_, _ = b.Buffer.WriteString(s)
	if nl {
		_ = b.WriteByte('\n')
	}
}

// WriteStr quiet write strings to buffer
func (b *Buffer) WriteStr(ss ...string) {
	b.writeStringsNl(ss, false)
}

// WriteStrings to buffer, ignore error.
func (b *Buffer) WriteStrings(ss []string) {
	b.writeStringsNl(ss, false)
}

// WriteStringNl write message to buffer and end with newline
func (b *Buffer) WriteStringNl(ss ...string) {
	b.writeStringsNl(ss, true)
}

// writeStringsNl to buffer, ignore error.
func (b *Buffer) writeStringsNl(ss []string, nl bool) {
	for _, s := range ss {
		_, _ = b.Buffer.WriteString(s)
	}
	if nl {
		_ = b.WriteByte('\n')
	}
}

// WriteAny type value to buffer
func (b *Buffer) WriteAny(vs ...any) {
	b.writeAnysWithNl(vs, false)
}

// Writeln write values to buffer and end with newline
func (b *Buffer) Writeln(vs ...any) {
	b.writeAnysWithNl(vs, true)
}

// WriteAnyNl type value to buffer and end with newline
func (b *Buffer) WriteAnyNl(vs ...any) {
	b.writeAnysWithNl(vs, true)
}

// WriteAnyLn type value to buffer and end with newline
func (b *Buffer) writeAnysWithNl(vs []any, nl bool) {
	for _, v := range vs {
		_, _ = b.Buffer.WriteString(fmt.Sprint(v))
	}
	if nl {
		_ = b.WriteByte('\n')
	}
}

// Printf quick write message to buffer, ignore error.
func (b *Buffer) Printf(tpl string, vs ...any) { _, _ = fmt.Fprintf(b, tpl, vs...) }

// Println quick write message with newline to buffer, will ignore error.
func (b *Buffer) Println(vs ...any) { _, _ = fmt.Fprintln(b, vs...) }

// ResetGet buffer string. alias of ResetAndGet()
func (b *Buffer) ResetGet() string {
	return b.ResetAndGet()
}

// ResetAndGet buffer string.
func (b *Buffer) ResetAndGet() string {
	s := b.String()
	b.Reset()
	return s
}

// Close buffer
func (b *Buffer) Close() error {
	return b.CloseErr
}

// Flush buffer
func (b *Buffer) Flush() error {
	return b.FlushErr
}

// Sync anf flush buffer
func (b *Buffer) Sync() error {
	return b.SyncErr
}
