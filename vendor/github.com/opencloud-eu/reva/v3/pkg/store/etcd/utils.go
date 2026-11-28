package etcd

import (
	"strings"
)

// Returns true if the limit isn't 0 AND is greater or equal to the number
// of results.
// If the limit is 0 or the number of items is less than the number of items,
// it will return false
func shouldFinish(numberOfResults, limit int64) bool {
	if limit == 0 || numberOfResults < limit {
		return false
	}
	return true
}

// Return the first key out of the prefix represented by the parameter,
// as a byte sequence. Note that it applies to byte sequences and not
// rune sequences, so it might be ill-suited for multi-byte chars
func firstKeyOutOfPrefix(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)
	var i int
	for i = len(dst) - 1; i >= 0; i-- {
		if dst[i] < 255 {
			dst[i]++
			break
		}
	}
	return dst[:i+1]
}

// Return the first key out of the prefix represented by the parameter.
// This function relies on the firstKeyOutOfPrefix one, which uses a byte
// sequence, so it might be ill-suited if the string contains multi-byte chars.
func firstKeyOutOfPrefixString(src string) string {
	srcBytes := []byte(src)
	dstBytes := firstKeyOutOfPrefix(srcBytes)
	return string(dstBytes)
}

// Reverse the string based on the containing runes
func reverseString(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

// Build a string based on the parts, to be used as a prefix. Empty string is
// expected if no part is passed as parameter.
// The string will contain all the parts separated by '/'. The last char will
// also be '/'
//
// For example `buildPrefix(P1, P2, P3)` will return  "P1/P2/P3/"
func buildPrefix(parts ...string) string {
	var b strings.Builder
	for _, part := range parts {
		b.WriteString(part)
		b.WriteRune('/')
	}
	return b.String()
}
