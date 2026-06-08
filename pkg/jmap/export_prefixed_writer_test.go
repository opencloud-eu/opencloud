package jmap

import (
	"bytes"
)

// LogLineWriter captures data chunk by chunk, extracts full lines,
// and passes them directly to log.Printf.
type LogLineWriter struct {
	buf     bytes.Buffer
	printer func(string)
	lines   *[]string
}

// NewLogLineWriter initializes the writer with a specific prefix.
func NewLogLineWriter(printer func(string), lines *[]string) *LogLineWriter {
	return &LogLineWriter{printer: printer, lines: lines}
}

// Write intercepts the byte stream and looks for complete text lines.
func (w *LogLineWriter) Write(p []byte) (n int, err error) {
	w.buf.Write(p)

	for {
		bufferedBytes := w.buf.Bytes()
		idx := bytes.IndexByte(bufferedBytes, '\n')
		if idx == -1 {
			break // Line is incomplete; wait for more data
		}

		// Slice UP TO the newline (idx), omitting the '\n' itself.
		// Go's log package handles its own line-endings.
		line := bufferedBytes[:idx]

		// Emit to log.Printf. Using %s works perfectly with []byte
		// without forcing an expensive string allocation.
		s := string(line)
		w.printer(s)
		if w.lines != nil {
			*w.lines = append(*w.lines, s)
		}

		// Advance the buffer past the processed text AND the '\n' (idx + 1)
		w.buf.Next(idx + 1)
	}

	return len(p), nil
}

// Flush ensures that any lingering text without a trailing newline
// gets safely pushed out to the log before exit.
func (w *LogLineWriter) Flush() error {
	if w.buf.Len() > 0 {
		s := w.buf.String()
		w.printer(s)
		if w.lines != nil {
			*w.lines = append(*w.lines, s)
		}
		w.buf.Reset()
	}
	return nil
}
