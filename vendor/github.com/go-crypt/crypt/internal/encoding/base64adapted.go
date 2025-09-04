package encoding

import (
	"encoding/base64"
)

const (
	encodeBase64Adapted = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789./"
)

var (
	// Base64RawAdaptedEncoding is the adapted encoding for crypt purposes without padding.
	Base64RawAdaptedEncoding = base64.NewEncoding(encodeBase64Adapted).WithPadding(base64.NoPadding)
)
