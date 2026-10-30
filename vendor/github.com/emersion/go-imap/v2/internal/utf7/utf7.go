// Package utf7 implements modified UTF-7 encoding defined in RFC 3501 section 5.1.3
package utf7

import (
	"encoding/base64"
)

const (
	min = 0x20 // Minimum self-representing UTF-7 value
	max = 0x7E // Maximum self-representing UTF-7 value
)

var b64Enc = base64.NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+,")
