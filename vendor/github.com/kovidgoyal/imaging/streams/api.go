package streams

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// BufferedReadSeeker wraps an io.ReadSeeker to provide buffering.
// It implements the io.ReadSeeker interface.
type BufferedReadSeeker struct {
	reader *bufio.Reader
	seeker io.ReadSeeker
}

// NewBufferedReadSeeker creates a new BufferedReadSeeker with a default buffer size.
func NewBufferedReadSeeker(rs io.ReadSeeker) *BufferedReadSeeker {
	return &BufferedReadSeeker{
		reader: bufio.NewReader(rs),
		seeker: rs,
	}
}

// Read reads data into p. It reads from the underlying buffered reader.
func (brs *BufferedReadSeeker) Read(p []byte) (n int, err error) {
	return brs.reader.Read(p)
}

// Seek sets the offset for the next Read. It is optimized to use the
// buffer for seeks that land within the buffered data range.
func (brs *BufferedReadSeeker) Seek(offset int64, whence int) (int64, error) {
	// Determine the current position (where the next Read would start)
	underlyingPos, err := brs.seeker.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	// The position of the stream as seen by clients
	logicalPos := underlyingPos - int64(brs.reader.Buffered())

	// 2. Calculate the absolute target position for the seek
	var absTargetPos int64
	switch whence {
	case io.SeekStart:
		absTargetPos = offset
	case io.SeekCurrent:
		absTargetPos = logicalPos + offset
	case io.SeekEnd:
		// Seeking from the end requires a fallback, as we don't know the end
		// position without invalidating the buffer's state relative to the seeker.
		return brs.fallbackSeek(offset, whence)
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}

	// 3. Check if the target position is within the current buffer
	if absTargetPos >= logicalPos && absTargetPos < underlyingPos {
		// The target is within the buffer. Calculate how many bytes to discard.
		bytesToDiscard := absTargetPos - logicalPos
		_, err := brs.reader.Discard(int(bytesToDiscard))
		if err != nil {
			// This is unlikely, but if Discard fails, fall back to a full seek
			return brs.fallbackSeek(offset, whence)
		}
		return absTargetPos, nil
	}

	// 4. If the target is outside the buffer, perform a fallback seek
	return brs.fallbackSeek(absTargetPos, io.SeekStart)
}

// fallbackSeek performs a seek on the underlying seeker and resets the buffer.
func (brs *BufferedReadSeeker) fallbackSeek(offset int64, whence int) (int64, error) {
	newOffset, err := brs.seeker.Seek(offset, whence)
	if err != nil {
		return 0, err
	}
	brs.reader.Reset(brs.seeker)
	return newOffset, nil
}

// Run the callback function with a buffered reader that supports Seek() and
// Read(). Return an io.Reader that represents all content from the original
// io.Reader.
func CallbackWithSeekable(r io.Reader, callback func(io.Reader) error) (stream io.Reader, err error) {
	switch s := r.(type) {
	case io.ReadSeeker:
		pos, err := s.Seek(0, io.SeekCurrent)
		if err == nil {
			defer func() {
				_, serr := s.Seek(pos, io.SeekStart)
				if err == nil {
					err = serr
				}
			}()
			// Add bufferring to s for efficiency
			bs := s
			switch r.(type) {
			case *BufferedReadSeeker, *bytes.Reader:
			default:
				bs = NewBufferedReadSeeker(s)
			}
			err = callback(bs)
			return s, err
		}
	case *bytes.Buffer:
		err = callback(bytes.NewReader(s.Bytes()))
		return s, err
	}
	rewindBuffer := &bytes.Buffer{}
	tee := io.TeeReader(r, rewindBuffer)
	err = callback(bufio.NewReader(tee))
	return io.MultiReader(rewindBuffer, r), err
}

// Skip reading the specified number of bytes efficiently
func Skip(r io.Reader, amt int64) (err error) {
	if s, ok := r.(io.Seeker); ok {
		if _, serr := s.Seek(amt, io.SeekCurrent); serr == nil {
			return
		}
	}
	_, err = io.CopyN(io.Discard, r, amt)
	return
}

// Read a single byte from the reader
func ReadByte(r io.Reader) (ans byte, err error) {
	var v [1]byte
	_, err = io.ReadFull(r, v[:])
	ans = v[0]
	return
}
