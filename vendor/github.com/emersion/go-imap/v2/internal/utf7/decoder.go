package utf7

import (
	"errors"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// ErrInvalidUTF7 means that a decoder encountered invalid UTF-7.
var ErrInvalidUTF7 = errors.New("utf7: invalid UTF-7")

// Decode decodes a string encoded with modified UTF-7.
//
// Note, raw UTF-8 is accepted.
func Decode(src string) (string, error) {
	if !utf8.ValidString(src) {
		return "", errors.New("invalid UTF-8")
	}

	var sb strings.Builder
	sb.Grow(len(src))

	ascii := true
	for i := 0; i < len(src); i++ {
		ch := src[i]

		if ch < min || (ch > max && ch < utf8.RuneSelf) {
			// Illegal code point in ASCII mode. Note, UTF-8 codepoints are
			// always allowed.
			return "", ErrInvalidUTF7
		}

		if ch != '&' {
			sb.WriteByte(ch)
			ascii = true
			continue
		}

		// Find the end of the Base64 or "&-" segment
		start := i + 1
		for i++; i < len(src) && src[i] != '-'; i++ {
			if src[i] == '\r' || src[i] == '\n' { // base64 package ignores CR and LF
				return "", ErrInvalidUTF7
			}
		}

		if i == len(src) { // Implicit shift ("&...")
			return "", ErrInvalidUTF7
		}

		if i == start { // Escape sequence "&-"
			sb.WriteByte('&')
			ascii = true
		} else { // Control or non-ASCII code points in base64
			if !ascii { // Null shift ("&...-&...-")
				return "", ErrInvalidUTF7
			}

			b := decode([]byte(src[start:i]))
			if len(b) == 0 { // Bad encoding
				return "", ErrInvalidUTF7
			}
			sb.Write(b)

			ascii = false
		}
	}

	return sb.String(), nil
}

// Extracts UTF-16-BE bytes from base64 data and converts them to UTF-8.
// A nil slice is returned if the encoding is invalid.
func decode(b64 []byte) []byte {
	var b []byte

	// Allocate a single block of memory large enough to store the Base64 data
	// (if padding is required), UTF-16-BE bytes, and decoded UTF-8 bytes.
	// Since a 2-byte UTF-16 sequence may expand into a 3-byte UTF-8 sequence,
	// double the space allocation for UTF-8.
	if n := len(b64); b64[n-1] == '=' {
		return nil
	} else if n&3 == 0 {
		b = make([]byte, b64Enc.DecodedLen(n)*3)
	} else {
		n += 4 - n&3
		b = make([]byte, n+b64Enc.DecodedLen(n)*3)
		copy(b[copy(b, b64):n], []byte("=="))
		b64, b = b[:n], b[n:]
	}

	// Decode Base64 into the first 1/3rd of b
	n, err := b64Enc.Decode(b, b64)
	if err != nil || n&1 == 1 {
		return nil
	}

	// Decode UTF-16-BE into the remaining 2/3rds of b
	b, s := b[:n], b[n:]
	j := 0
	for i := 0; i < n; i += 2 {
		r := rune(b[i])<<8 | rune(b[i+1])
		if utf16.IsSurrogate(r) {
			if i += 2; i == n {
				return nil
			}
			r2 := rune(b[i])<<8 | rune(b[i+1])
			if r = utf16.DecodeRune(r, r2); r == utf8.RuneError {
				return nil
			}
		} else if min <= r && r <= max {
			return nil
		}
		j += utf8.EncodeRune(s[j:], r)
	}
	return s[:j]
}
