package enmime

import (
	"bufio"
	"bytes"
	stderrors "errors"
	"io"
	"unicode"

	"github.com/pkg/errors"
)

// This constant needs to be at least 76 for this package to work correctly.  This is because
// \r\n--separator_of_len_70- would fill the buffer and it wouldn't be safe to consume a single byte
// from it.
const peekBufferSize = 4096

var errNoBoundaryTerminator = stderrors.New("expected boundary not present")

type boundaryReader struct {
	finished    bool          // No parts remain when finished
	partsRead   int           // Number of parts read thus far
	atPartStart bool          // Whether the current part is at its beginning
	r           *bufio.Reader // Source reader
	nlPrefix    []byte        // NL + MIME boundary prefix
	prefix      []byte        // MIME boundary prefix
	final       []byte        // Final boundary prefix
	buffer      *bytes.Buffer // Content waiting to be read
	unbounded   bool          // Flag to throw errNoBoundaryTerminator
}

// newBoundaryReader returns an initialized boundaryReader
func newBoundaryReader(reader *bufio.Reader, boundary string) *boundaryReader {
	fullBoundary := []byte("\n--" + boundary + "--")
	return &boundaryReader{
		r:        reader,
		nlPrefix: fullBoundary[:len(fullBoundary)-2],
		prefix:   fullBoundary[1 : len(fullBoundary)-2],
		final:    fullBoundary[1:],
		buffer:   new(bytes.Buffer),
	}
}

// Read returns a buffer containing the content up until boundary
//
//	Excerpt from io package on io.Reader implementations:
//
//	  type Reader interface {
//	     Read(p []byte) (n int, err error)
//	  }
//
//	  Read reads up to len(p) bytes into p. It returns the number of
//	  bytes read (0 <= n <= len(p)) and any error encountered. Even
//	  if Read returns n < len(p), it may use all of p as scratch space
//	  during the call. If some data is available but not len(p) bytes,
//	  Read conventionally returns what is available instead of waiting
//	  for more.
//
//	  When Read encounters an error or end-of-file condition after
//	  successfully reading n > 0 bytes, it returns the number of bytes
//	  read. It may return the (non-nil) error from the same call or
//	  return the error (and n == 0) from a subsequent call. An instance
//	  of this general case is that a Reader returning a non-zero number
//	  of bytes at the end of the input stream may return either err == EOF
//	  or err == nil. The next Read should return 0, EOF.
//
//	  Callers should always process the n > 0 bytes returned before
//	  considering the error err. Doing so correctly handles I/O errors
//	  that happen after reading some bytes and also both of the allowed
//	  EOF behaviors.
func (b *boundaryReader) Read(dest []byte) (n int, err error) {
	if b.buffer.Len() >= len(dest) {
		// This read request can be satisfied entirely by the buffer.
		n, err = b.buffer.Read(dest)
		if b.atPartStart && n > 0 {
			b.atPartStart = false
		}

		return n, err
	}

	for i := 0; i < len(dest); i++ {
		var cs []byte
		cs, err = b.r.Peek(1)
		if err != nil && err != io.EOF {
			return 0, errors.WithStack(err)
		}
		// Ensure that we can switch on the first byte of 'cs' without panic.
		if len(cs) > 0 {
			padding := 1
			check := false

			switch cs[0] {
			// Check for carriage return as potential CRLF boundary prefix.
			case '\r':
				padding = 2
				check = true
			// Check for line feed as potential LF boundary prefix.
			case '\n':
				check = true

			default:
				if b.atPartStart {
					// If we're at the very beginning of the part (even before the headers),
					// check to see if there's a delimiter that immediately follows.
					padding = 0
					check = true
				}
			}

			if check {
				var peek []byte
				peek, err = b.r.Peek(len(b.nlPrefix) + padding + 1)
				switch err {
				case nil:
					// Check the whitespace at the head of the peek to avoid checking for a boundary early.
					if bytes.HasPrefix(peek, []byte("\n\n")) ||
						bytes.HasPrefix(peek, []byte("\n\r")) ||
						bytes.HasPrefix(peek, []byte("\r\n\r")) ||
						bytes.HasPrefix(peek, []byte("\r\n\n")) {
						break
					}
					// Check the peek buffer for a boundary delimiter or terminator.
					if b.isDelimiter(peek[padding:]) || b.isTerminator(peek[padding:]) {
						// We have found our boundary terminator, lets write out the final bytes
						// and return io.EOF to indicate that this section read is complete.
						n, err = b.buffer.Read(dest)
						switch err {
						case nil, io.EOF:
							if b.atPartStart && n > 0 {
								b.atPartStart = false
							}
							return n, io.EOF
						default:
							return 0, errors.WithStack(err)
						}
					}
				case io.EOF:
					// We have reached the end without finding a boundary,
					// so we flag the boundary reader to add an error to
					// the errors slice and write what we have to the buffer.
					b.unbounded = true
				default:
					continue
				}
			}
		}

		var next byte
		next, err = b.r.ReadByte()
		if err != nil {
			// EOF is not fatal, it just means that we have drained the reader.
			if errors.Is(err, io.EOF) {
				break
			}

			return 0, errors.WithStack(err)
		}

		if err = b.buffer.WriteByte(next); err != nil {
			return 0, errors.WithStack(err)
		}
	}

	// Read the contents of the buffer into the destination slice.
	n, err = b.buffer.Read(dest)
	if b.atPartStart && n > 0 {
		b.atPartStart = false
	}
	return n, err
}

// Next moves over the boundary to the next part, returns true if there is another part to be read.
func (b *boundaryReader) Next() (bool, error) {
	if b.finished {
		return false, nil
	}
	if b.partsRead > 0 {
		// Exhaust the current part to prevent errors when moving to the next part.
		_, _ = io.Copy(io.Discard, b)
	}
	for {
		var line []byte
		var err error
		for {
			// Read whole line, handle extra long lines in cycle
			var segment []byte
			segment, err = b.r.ReadSlice('\n')
			if line == nil {
				line = segment
			} else {
				line = append(line, segment...)
			}

			if err == nil || err == io.EOF {
				break
			} else if err != bufio.ErrBufferFull || len(segment) == 0 {
				return false, errors.WithStack(err)
			}
		}

		if len(line) > 0 && (line[0] == '\r' || line[0] == '\n') {
			// Blank line
			continue
		}
		if b.isTerminator(line) {
			b.finished = true
			return false, nil
		}
		if err != io.EOF && b.isDelimiter(line) {
			// Start of a new part.
			b.partsRead++
			b.atPartStart = true
			return true, nil
		}
		if err == io.EOF {
			// Intentionally not wrapping with stack.
			return false, io.EOF
		}
		if b.partsRead == 0 {
			// The first part didn't find the starting delimiter, burn off any preamble in front of
			// the boundary.
			continue
		}
		b.finished = true
		return false, errors.WithMessagef(errNoBoundaryTerminator, "expecting boundary %q, got %q", string(b.prefix), string(line))
	}
}

// isDelimiter returns true for --BOUNDARY\r\n but not --BOUNDARY--
func (b *boundaryReader) isDelimiter(buf []byte) bool {
	idx := bytes.Index(buf, b.prefix)
	if idx == -1 {
		return false
	}

	// Fast forward to the end of the boundary prefix.
	buf = buf[idx+len(b.prefix):]
	if len(buf) > 0 {
		if unicode.IsSpace(rune(buf[0])) {
			return true
		}
	}

	return false
}

// isTerminator returns true for --BOUNDARY--
func (b *boundaryReader) isTerminator(buf []byte) bool {
	idx := bytes.Index(buf, b.final)
	return idx != -1
}
