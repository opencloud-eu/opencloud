package strutil

import (
	"github.com/gookit/goutil/byteutil"
	"github.com/gookit/goutil/encodes"
)

// some constants string chars
const (
	Numbers  = "0123456789"
	HexChars = "0123456789abcdef" // base16

	AlphaBet  = "abcdefghijklmnopqrstuvwxyz"
	AlphaBet1 = "AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUuVvWwXxYyZz"

	// AlphaNum chars, can use for base36 encode
	AlphaNum = "abcdefghijklmnopqrstuvwxyz0123456789"
	// AlphaNum2 chars, can use for base62 encode
	AlphaNum2 = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	AlphaNum3 = "0123456789AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUuVvWwXxYyZz"
)

// RandomChars generate give length random chars at `a-z`
func RandomChars(ln int) string {
	return buildRandomString(AlphaBet, ln)
}

// RandomCharsV2 generate give length random chars in `0-9a-z`
func RandomCharsV2(ln int) string {
	return buildRandomString(AlphaNum, ln)
}

// RandomCharsV3 generate give length random chars in `0-9a-zA-Z`
func RandomCharsV3(ln int) string {
	return buildRandomString(AlphaNum2, ln)
}

// RandWithTpl generate random string with give template
func RandWithTpl(n int, letters string) string {
	if len(letters) == 0 {
		letters = AlphaNum2
	}
	return buildRandomString(letters, n)
}

// RandomString generate.
//
// Example:
//
//	// this will give us a 44 byte, base64 encoded output
//	token, err := RandomString(16) // eg: "I7S4yFZddRMxQoudLZZ-eg"
func RandomString(length int) (string, error) {
	b, err := RandomBytes(length)
	return encodes.B64URL.EncodeToString(b), err
}

// RandomBytes generate
func RandomBytes(length int) ([]byte, error) {
	return byteutil.Random(length)
}
