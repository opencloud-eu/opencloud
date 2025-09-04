package coding

import (
	"net/url"
	"strings"
)

// FromIDHeader decodes a Content-ID or Message-ID header value (RFC 2392) into a utf-8 string.
// Example: "<foo%3fbar+baz>" becomes "foo?bar baz".
func FromIDHeader(v string) string {
	if v == "" {
		return v
	}
	v = strings.TrimLeft(v, "<")
	v = strings.TrimRight(v, ">")
	if r, err := url.QueryUnescape(v); err == nil {
		v = r
	}
	return v
}

// ToIDHeader encodes a Content-ID or Message-ID header value (RFC 2392) from a utf-8 string.
func ToIDHeader(v string) string {
	v = url.QueryEscape(v)
	return "<" + strings.ReplaceAll(v, "%40", "@") + ">"
}
