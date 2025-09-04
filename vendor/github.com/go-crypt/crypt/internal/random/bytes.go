package random

import (
	"crypto/rand"
	"io"
)

// Bytes returns random arbitrary bytes with a length of n.
func Bytes(n int) (bytes []byte, err error) {
	bytes = make([]byte, n)

	if _, err = io.ReadFull(rand.Reader, bytes); err != nil {
		return nil, err
	}

	return bytes, nil
}

// CharSetBytes returns random bytes with a length of n from the characters in the charset.
func CharSetBytes(n int, charset string) (bytes []byte, err error) {
	bytes = make([]byte, n)

	if _, err = rand.Read(bytes); err != nil {
		return nil, err
	}

	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}

	return bytes, nil
}
