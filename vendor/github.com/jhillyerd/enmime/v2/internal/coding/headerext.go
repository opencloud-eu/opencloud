package coding

import (
	"fmt"
	"io"
	"mime"
	"strings"
)

// NewExtMimeDecoder creates new MIME word decoder which allows decoding of additional charsets.
func NewExtMimeDecoder() *mime.WordDecoder {
	return &mime.WordDecoder{
		CharsetReader: NewCharsetReader,
	}
}

// DecodeExtHeader decodes a single line (per RFC 2047, aka Message Header Extensions) using Golang's
// mime.WordDecoder.
func DecodeExtHeader(input string) string {
	if !strings.Contains(input, "=?") {
		// Don't scan if there is nothing to do here
		return input
	}

	header, err := NewExtMimeDecoder().DecodeHeader(input)
	if err != nil {
		return input
	}

	return header
}

// RFC2047Decode returns a decoded string if the input uses RFC2047 encoding, otherwise it will
// return the input.
//
// RFC2047 Example: `=?UTF-8?B?bmFtZT0iw7DCn8KUwoo=?=`
func RFC2047Decode(s string) string {
	// Convert CR/LF to spaces.
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' {
			return ' '
		}
		return r
	}, s)

	var err error
	decoded := false
	for {
		s, err = rfc2047Recurse(s)
		switch err {
		case nil:
			decoded = true
			continue

		default:
			if decoded {
				key, value, found := strings.Cut(s, "=")
				if !found {
					return s
				}

				// Add quotes as needed.
				if !strings.HasPrefix(value, "\"") {
					value = `"` + value
				}
				if !strings.HasSuffix(value, "\"") {
					value += `"`
				}

				return fmt.Sprintf("%s=%s", key, value)
			}

			return s
		}
	}
}

// rfc2047Recurse is called for if the value contains content encoded in RFC2047 format and decodes
// it.
func rfc2047Recurse(s string) (string, error) {
	us := strings.ToUpper(s)
	if !strings.Contains(us, "?Q?") && !strings.Contains(us, "?B?") {
		return s, io.EOF
	}

	var val string
	if val = DecodeExtHeader(s); val == s {
		if val = DecodeExtHeader(fixRFC2047String(val)); val == s {
			return val, io.EOF
		}
	}

	return val, nil
}

// fixRFC2047String removes the following characters from charset and encoding segments of an
// RFC2047 string: '\n', '\r' and ' '
func fixRFC2047String(s string) string {
	inString := false
	isWithinTerminatingEqualSigns := false
	questionMarkCount := 0
	sb := &strings.Builder{}
	for _, v := range s {
		switch v {
		case '=':
			if questionMarkCount == 3 {
				inString = false
			} else {
				isWithinTerminatingEqualSigns = true
			}
			sb.WriteRune(v)

		case '?':
			if isWithinTerminatingEqualSigns {
				inString = true
			} else {
				questionMarkCount++
			}
			isWithinTerminatingEqualSigns = false
			sb.WriteRune(v)

		case '\n', '\r', ' ':
			if !inString {
				sb.WriteRune(v)
			}
			isWithinTerminatingEqualSigns = false

		default:
			isWithinTerminatingEqualSigns = false
			sb.WriteRune(v)
		}
	}

	return sb.String()
}
