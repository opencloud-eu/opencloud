package shm

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	not_rand "math/rand/v2"
	"strconv"
	"unsafe"
)

var _ = fmt.Print

func RandomFilename() string {
	b := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	_, err := rand.Read(b)
	if err != nil {
		return strconv.FormatUint(uint64(not_rand.Uint32()), 16)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
}

// Unsafely converts s into a byte slice.
// If you modify b, then s will also be modified. This violates the
// property that strings are immutable.
func UnsafeStringToBytes(s string) (b []byte) {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// Unsafely converts b into a string.
// If you modify b, then s will also be modified. This violates the
// property that strings are immutable.
func UnsafeBytesToString(b []byte) (s string) {
	return unsafe.String(unsafe.SliceData(b), len(b))
}
