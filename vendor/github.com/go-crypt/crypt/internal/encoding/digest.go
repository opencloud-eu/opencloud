package encoding

import (
	"strings"
)

// Split an encoded digest by the encoding.Delimiter.
func Split(encodedDigest string, n int) (parts []string) {
	return strings.SplitN(encodedDigest, DelimiterStr, n)
}
