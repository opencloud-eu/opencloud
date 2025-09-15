package ext

import (
	"io"
	"reflect"

	"github.com/shamaton/msgpack/v2/internal/common"
)

// StreamEncoder is an interface that extended encoders should implement.
// It defines methods for encoding data into a stream.
type StreamEncoder interface {
	// Code returns the unique code for the encoder.
	Code() int8
	// Type returns the reflect.Type of the value being encoded.
	Type() reflect.Type
	// Write encodes the given value and writes it to the provided StreamWriter.
	Write(w StreamWriter, value reflect.Value) error
}

// StreamWriter provides methods for writing data in extended formats.
// It wraps an io.Writer and a buffer for efficient writing.
type StreamWriter struct {
	w   io.Writer      // The underlying writer to write data to.
	buf *common.Buffer // A buffer used for temporary storage during writing.
}

// CreateStreamWriter creates and returns a new StreamWriter instance.
func CreateStreamWriter(w io.Writer, buf *common.Buffer) StreamWriter {
	return StreamWriter{w, buf}
}

// WriteByte1Int64 writes a single byte representation of an int64 value.
func (w *StreamWriter) WriteByte1Int64(value int64) error {
	return w.buf.Write(w.w,
		byte(value),
	)
}

// WriteByte2Int64 writes a two-byte representation of an int64 value.
func (w *StreamWriter) WriteByte2Int64(value int64) error {
	return w.buf.Write(w.w,
		byte(value>>8),
		byte(value),
	)
}

// WriteByte4Int64 writes a four-byte representation of an int64 value.
func (w *StreamWriter) WriteByte4Int64(value int64) error {
	return w.buf.Write(w.w,
		byte(value>>24),
		byte(value>>16),
		byte(value>>8),
		byte(value),
	)
}

// WriteByte8Int64 writes an eight-byte representation of an int64 value.
func (w *StreamWriter) WriteByte8Int64(value int64) error {
	return w.buf.Write(w.w,
		byte(value>>56),
		byte(value>>48),
		byte(value>>40),
		byte(value>>32),
		byte(value>>24),
		byte(value>>16),
		byte(value>>8),
		byte(value),
	)
}

// WriteByte1Uint64 writes a single byte representation of a uint64 value.
func (w *StreamWriter) WriteByte1Uint64(value uint64) error {
	return w.buf.Write(w.w,
		byte(value),
	)
}

// WriteByte2Uint64 writes a two-byte representation of a uint64 value.
func (w *StreamWriter) WriteByte2Uint64(value uint64) error {
	return w.buf.Write(w.w,
		byte(value>>8),
		byte(value),
	)
}

// WriteByte4Uint64 writes a four-byte representation of a uint64 value.
func (w *StreamWriter) WriteByte4Uint64(value uint64) error {
	return w.buf.Write(w.w,
		byte(value>>24),
		byte(value>>16),
		byte(value>>8),
		byte(value),
	)
}

// WriteByte8Uint64 writes an eight-byte representation of a uint64 value.
func (w *StreamWriter) WriteByte8Uint64(value uint64) error {
	return w.buf.Write(w.w,
		byte(value>>56),
		byte(value>>48),
		byte(value>>40),
		byte(value>>32),
		byte(value>>24),
		byte(value>>16),
		byte(value>>8),
		byte(value),
	)
}

// WriteByte1Int writes a single byte representation of an int value.
func (w *StreamWriter) WriteByte1Int(value int) error {
	return w.buf.Write(w.w,
		byte(value),
	)
}

// WriteByte2Int writes a two-byte representation of an int value.
func (w *StreamWriter) WriteByte2Int(value int) error {
	return w.buf.Write(w.w,
		byte(value>>8),
		byte(value),
	)
}

// WriteByte4Int writes a four-byte representation of an int value.
func (w *StreamWriter) WriteByte4Int(value int) error {
	return w.buf.Write(w.w,
		byte(value>>24),
		byte(value>>16),
		byte(value>>8),
		byte(value),
	)
}

// WriteByte4Uint32 writes a four-byte representation of a uint32 value.
func (w *StreamWriter) WriteByte4Uint32(value uint32) error {
	return w.buf.Write(w.w,
		byte(value>>24),
		byte(value>>16),
		byte(value>>8),
		byte(value),
	)
}

// WriteBytes writes a slice of bytes to the underlying writer.
func (w *StreamWriter) WriteBytes(bs []byte) error {
	return w.buf.Write(w.w, bs...)
}
