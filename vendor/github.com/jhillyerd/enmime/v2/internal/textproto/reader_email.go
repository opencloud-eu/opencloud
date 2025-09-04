package textproto

import (
	"bytes"
	"errors"
	"math"
	"net/textproto"
)

// ReadEmailMIMEHeader reads a MIME-style header from r.
//
// This is a modified version of the stock func that better handles the characters
// we must support in email, instead of just HTTP.
func (r *Reader) ReadEmailMIMEHeader() (MIMEHeader, error) {
	return readEmailMIMEHeader(r, math.MaxInt64)
}

func readEmailMIMEHeader(r *Reader, lim int64) (MIMEHeader, error) {
	// Avoid lots of small slice allocations later by allocating one
	// large one ahead of time which we'll cut up into smaller
	// slices. If this isn't big enough later, we allocate small ones.
	var strs []string
	hint := r.upcomingHeaderNewlines()
	if hint > 0 {
		strs = make([]string, hint)
	}

	m := make(MIMEHeader, hint)

	// The first line cannot start with a leading space.
	if buf, err := r.R.Peek(1); err == nil && (buf[0] == ' ' || buf[0] == '\t') {
		line, err := r.readLineSlice()
		if err != nil {
			return m, err
		}
		return m, textproto.ProtocolError("malformed MIME header initial line: " + string(line))
	}

	for {
		kv, err := r.readContinuedLineSlice(mustHaveFieldNameColon)
		if len(kv) == 0 {
			return m, err
		}

		// Key ends at first colon.
		k, v, ok := bytes.Cut(kv, colon)
		if !ok {
			return m, textproto.ProtocolError("malformed MIME header line: " + string(kv))
		}
		key, ok := canonicalEmailMIMEHeaderKey(k)
		if !ok {
			return m, textproto.ProtocolError("malformed MIME header line: " + string(kv))
		}
		// for _, c := range v {
		// 	if !validHeaderValueByte(c) {
		// 		return m, ProtocolError("malformed MIME header line: " + string(kv))
		// 	}
		// }

		// As per RFC 7230 field-name is a token, tokens consist of one or more chars.
		// We could return a ProtocolError here, but better to be liberal in what we
		// accept, so if we get an empty key, skip it.
		if key == "" {
			continue
		}

		// Skip initial spaces in value.
		value := string(bytes.TrimLeft(v, " \t"))

		vv := m[key]
		if vv == nil {
			lim -= int64(len(key))
			lim -= 100 // map entry overhead
		}
		lim -= int64(len(value))
		if lim < 0 {
			// TODO: This should be a distinguishable error (ErrMessageTooLarge)
			// to allow mime/multipart to detect it.
			return m, errors.New("message too large")
		}
		if vv == nil && len(strs) > 0 {
			// More than likely this will be a single-element key.
			// Most headers aren't multi-valued.
			// Set the capacity on strs[0] to 1, so any future append
			// won't extend the slice into the other strings.
			vv, strs = strs[:1:1], strs[1:]
			vv[0] = value
			m[key] = vv
		} else {
			m[key] = append(vv, value)
		}

		if err != nil {
			return m, err
		}
	}
}

// CanonicalEmailMIMEHeaderKey returns the canonical format of the
// MIME header key s.
//
// This is a modified version of the stock func that better handles the characters
// we must support in email, instead of just HTTP.
func CanonicalEmailMIMEHeaderKey(s string) string {
	// Quick check for canonical encoding.
	upper := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !ValidEmailHeaderFieldByte(c) {
			return s
		}
		if upper && 'a' <= c && c <= 'z' {
			s, _ = canonicalEmailMIMEHeaderKey([]byte(s))
			return s
		}
		if !upper && 'A' <= c && c <= 'Z' {
			s, _ = canonicalEmailMIMEHeaderKey([]byte(s))
			return s
		}
		upper = c == '-'
	}
	return s
}

func canonicalEmailMIMEHeaderKey(a []byte) (_ string, ok bool) {
	noCanon := false
	for _, c := range a {
		if ValidEmailHeaderFieldByte(c) {
			continue
		}
		// Don't canonicalize.
		if c == ' ' {
			// We accept invalid headers with a space before the
			// colon, but must not canonicalize them.
			// See https://go.dev/issue/34540.
			noCanon = true
			continue
		}
		return string(a), false
	}
	if noCanon {
		return string(a), true
	}

	upper := true
	for i, c := range a {
		// Canonicalize: first letter upper case
		// and upper case after each dash.
		// (Host, User-Agent, If-Modified-Since).
		// MIME headers are ASCII only, so no Unicode issues.
		if upper && 'a' <= c && c <= 'z' {
			c -= toLower
		} else if !upper && 'A' <= c && c <= 'Z' {
			c += toLower
		}
		a[i] = c
		upper = c == '-' // for next time
	}
	commonHeaderOnce.Do(initCommonHeader)
	// The compiler recognizes m[string(byteSlice)] as a special
	// case, so a copy of a's bytes into a new string does not
	// happen in this map lookup:
	if v := commonHeader[string(a)]; v != "" {
		return v, true
	}
	return string(a), true
}

// ValidEmailHeaderFieldByte Valid characters in email header field.
//
// According to [RFC 5322](https://www.rfc-editor.org/rfc/rfc5322#section-2.2),
//
//	> A field name MUST be composed of printable US-ASCII characters (i.e.,
//	> characters that have values between 33 and 126, inclusive), except
//	> colon.
func ValidEmailHeaderFieldByte(c byte) bool {
	const mask = 0 |
		(1<<(10)-1)<<'0' |
		(1<<(26)-1)<<'a' |
		(1<<(26)-1)<<'A' |
		1<<'!' |
		1<<'"' |
		1<<'#' |
		1<<'$' |
		1<<'%' |
		1<<'&' |
		1<<'\'' |
		1<<'(' |
		1<<')' |
		1<<'*' |
		1<<'+' |
		1<<',' |
		1<<'-' |
		1<<'.' |
		1<<'/' |
		1<<';' |
		1<<'<' |
		1<<'=' |
		1<<'>' |
		1<<'?' |
		1<<'@' |
		1<<'[' |
		1<<'\\' |
		1<<']' |
		1<<'^' |
		1<<'_' |
		1<<'`' |
		1<<'{' |
		1<<'|' |
		1<<'}' |
		1<<'~'
	return ((uint64(1)<<c)&(mask&(1<<64-1)) |
		(uint64(1)<<(c-64))&(mask>>64)) != 0
}
