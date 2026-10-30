package message

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/quotedprintable"
	"strings"
)

type UnknownEncodingError struct {
	e error
}

func (u UnknownEncodingError) Unwrap() error { return u.e }

func (u UnknownEncodingError) Error() string {
	return "encoding error: " + u.e.Error()
}

// IsUnknownEncoding returns a boolean indicating whether the error is known to
// report that the encoding advertised by the entity is unknown.
func IsUnknownEncoding(err error) bool {
	return errors.As(err, new(UnknownEncodingError))
}

func encodingReader(enc string, r io.Reader) (io.Reader, error) {
	var dec io.Reader
	switch strings.ToLower(enc) {
	case "quoted-printable":
		dec = quotedprintable.NewReader(r)
	case "base64":
		wrapped := &whitespaceReplacingReader{wrapped: r}
		dec = base64.NewDecoder(base64.StdEncoding, wrapped)
	case "7bit", "8bit", "binary", "":
		dec = r
	default:
		return nil, fmt.Errorf("unhandled encoding %q", enc)
	}
	return dec, nil
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error {
	return nil
}

func encodingWriter(enc string, w io.Writer) (io.WriteCloser, error) {
	var wc io.WriteCloser
	switch strings.ToLower(enc) {
	case "quoted-printable":
		wc = quotedprintable.NewWriter(w)
	case "base64":
		wc = base64.NewEncoder(base64.StdEncoding, &lineWrapper{w: w, maxLineLen: 76})
	case "7bit", "8bit":
		wc = nopCloser{&lineWrapper{w: w, maxLineLen: 998}}
	case "binary", "":
		wc = nopCloser{w}
	default:
		return nil, fmt.Errorf("unhandled encoding %q", enc)
	}
	return wc, nil
}

// whitespaceReplacingReader replaces space and tab characters with a LF so
// base64 bodies with a continuation indent can be decoded by the base64 decoder
// even though it is against the spec.
type whitespaceReplacingReader struct {
	wrapped io.Reader
}

func (r *whitespaceReplacingReader) Read(p []byte) (int, error) {
	n, err := r.wrapped.Read(p)

	for i := 0; i < n; i++ {
		if p[i] == ' ' || p[i] == '\t' {
			p[i] = '\n'
		}
	}

	return n, err
}

type lineWrapper struct {
	w          io.Writer
	maxLineLen int

	curLineLen int
	cr         bool
}

func (w *lineWrapper) Write(b []byte) (int, error) {
	var written int
	for len(b) > 0 {
		var l []byte
		l, b = cutLine(b, w.maxLineLen-w.curLineLen)

		lf := bytes.HasSuffix(l, []byte("\n"))
		l = bytes.TrimSuffix(l, []byte("\n"))

		n, err := w.w.Write(l)
		if err != nil {
			return written, err
		}
		written += n

		cr := bytes.HasSuffix(l, []byte("\r"))
		if len(l) == 0 {
			cr = w.cr
		}

		if !lf && len(b) == 0 {
			w.curLineLen += len(l)
			w.cr = cr
			break
		}
		w.curLineLen = 0

		ending := []byte("\r\n")
		if cr {
			ending = []byte("\n")
		}
		_, err = w.w.Write(ending)
		if err != nil {
			return written, err
		}
		w.cr = false
	}

	return written, nil
}

func cutLine(b []byte, max int) ([]byte, []byte) {
	for i := 0; i < len(b); i++ {
		if b[i] == '\r' && i == max {
			continue
		}
		if b[i] == '\n' {
			return b[:i+1], b[i+1:]
		}
		if i >= max {
			return b[:i], b[i:]
		}
	}
	return b, nil
}
