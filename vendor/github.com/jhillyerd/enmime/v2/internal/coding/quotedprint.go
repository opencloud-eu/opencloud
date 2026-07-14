package coding

import (
	"bufio"
	"fmt"
	"io"
)

// QPCleaner scans quoted printable content for invalid characters and encodes them so that
// Go's quoted-printable decoder does not abort with an error.
type QPCleaner struct {
	in       *bufio.Reader
	overflow []byte
	lineLen  int
}

// MaxQPLineLen is the maximum line length we allow before inserting `=\r\n`.  Prevents buffer
// overflows in mime/quotedprintable.Reader.
const MaxQPLineLen = 1024

var (
	_ io.Reader = &QPCleaner{} // Assert QPCleaner implements io.Reader.

	escapedEquals = []byte("=3D") // QP encoded value of an equals sign.
	lineBreak     = []byte("=\r\n")
)

// NewQPCleaner returns a QPCleaner for the specified reader.
func NewQPCleaner(r io.Reader) *QPCleaner {
	return &QPCleaner{
		in:       bufio.NewReader(r),
		overflow: nil,
		lineLen:  0,
	}
}

// Read method for io.Reader interface.
func (qp *QPCleaner) Read(dest []byte) (n int, err error) {
	destLen := len(dest)

	if len(qp.overflow) > 0 {
		// Copy bytes that didn't fit into dest buffer during previous read.
		n = copy(dest, qp.overflow)
		qp.overflow = qp.overflow[n:]
	}

	// writeByte outputs a single byte, space for which will have already been ensured by the loop
	// condition. Updates counters.
	writeByte := func(in byte) {
		dest[n] = in
		n++
		qp.lineLen++
	}

	// safeWriteByte outputs a single byte, storing overflow for next read. Updates counters.
	safeWriteByte := func(in byte) {
		if n < destLen {
			dest[n] = in
			n++
		} else {
			qp.overflow = append(qp.overflow, in)
		}
		qp.lineLen++
	}

	// writeBytes outputs multiple bytes, storing overflow for next read. Updates counters.
	writeBytes := func(in []byte) {
		nc := copy(dest[n:], in)
		if nc < len(in) {
			// Stash unwritten bytes into overflow.
			qp.overflow = append(qp.overflow, []byte(in[nc:])...)
		}
		n += nc
		qp.lineLen += len(in)
	}

	// ensureLineLen ensures there is room to write `requested` bytes, preventing a line break being
	// inserted in the middle of the escaped string.  The requested count is in addition to the
	// byte that was already reserved for this loop iteration.
	ensureLineLen := func(requested int) {
		if qp.lineLen+requested >= MaxQPLineLen {
			writeBytes(lineBreak)
			qp.lineLen = 0
		}
	}

	// Loop over bytes in qp.in ByteReader while there is space in dest.
	for n < destLen {
		var b byte
		b, err = qp.in.ReadByte()
		if err != nil {
			return n, err
		}

		if qp.lineLen >= MaxQPLineLen {
			writeBytes(lineBreak)
			qp.lineLen = 0
			if n == destLen {
				break
			}
		}

		switch {
		// Pass valid hex bytes through, otherwise escapes the equals symbol.
		case b == '=':
			ensureLineLen(2)

			var hexBytes []byte
			hexBytes, err = qp.in.Peek(2)
			if err != nil && err != io.EOF {
				return 0, err
			}
			if validHexBytes(hexBytes) {
				safeWriteByte(b)
			} else {
				writeBytes(escapedEquals)
			}

		// Valid special character.
		case b == '\t':
			writeByte(b)

		// Valid special characters that reset line length.
		case b == '\r' || b == '\n':
			writeByte(b)
			qp.lineLen = 0

		// Invalid characters, render as quoted-printable.
		case b < ' ' || '~' < b:
			ensureLineLen(2)
			writeBytes([]byte(fmt.Sprintf("=%02X", b)))

		// Acceptable characters.
		default:
			writeByte(b)
		}
	}

	return n, err
}

func validHexByte(b byte) bool {
	return '0' <= b && b <= '9' || 'A' <= b && b <= 'F' || 'a' <= b && b <= 'f'
}

// validHexBytes returns true if this byte sequence represents a valid quoted-printable escape
// sequence or line break, minus the initial equals sign.
func validHexBytes(v []byte) bool {
	if len(v) > 0 && v[0] == '\n' {
		// Soft line break.
		return true
	}
	if len(v) < 2 {
		return false
	}
	if v[0] == '\r' && v[1] == '\n' {
		// Soft line break.
		return true
	}
	return validHexByte(v[0]) && validHexByte(v[1])
}
