package internal

import (
	"encoding/base64"
)

func EncodeSASL(b []byte) string {
	if len(b) == 0 {
		return "="
	} else {
		return base64.StdEncoding.EncodeToString(b)
	}
}

func DecodeSASL(s string) ([]byte, error) {
	if s == "=" {
		// go-sasl treats nil as no challenge/response, so return a non-nil
		// empty byte slice
		return []byte{}, nil
	} else {
		return base64.StdEncoding.DecodeString(s)
	}
}
