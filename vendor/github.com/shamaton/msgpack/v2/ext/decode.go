package ext

import (
	"reflect"

	"github.com/shamaton/msgpack/v2/def"
)

// Decoder defines an interface for decoding values from bytes.
// It provides methods to get the decoder type, check if the data matches the type,
// and convert the data into a Go value.
type Decoder interface {
	// Code returns the unique code representing the decoder type.
	Code() int8

	// IsType checks if the data at the given offset matches the expected type.
	// Returns true if the type matches, false otherwise.
	IsType(offset int, d *[]byte) bool

	// AsValue decodes the data at the given offset into a Go value of the specified kind.
	// Returns the decoded value, the new offset, and an error if decoding fails.
	AsValue(offset int, k reflect.Kind, d *[]byte) (interface{}, int, error)
}

// DecoderCommon provides common utility methods for decoding data from bytes.
type DecoderCommon struct {
}

// ReadSize1 reads a single byte from the given index in the byte slice.
// Returns the byte and the new index after reading.
func (cd *DecoderCommon) ReadSize1(index int, d *[]byte) (byte, int) {
	rb := def.Byte1
	return (*d)[index], index + rb
}

// ReadSize2 reads two bytes from the given index in the byte slice.
// Returns the bytes as a slice and the new index after reading.
func (cd *DecoderCommon) ReadSize2(index int, d *[]byte) ([]byte, int) {
	rb := def.Byte2
	return (*d)[index : index+rb], index + rb
}

// ReadSize4 reads four bytes from the given index in the byte slice.
// Returns the bytes as a slice and the new index after reading.
func (cd *DecoderCommon) ReadSize4(index int, d *[]byte) ([]byte, int) {
	rb := def.Byte4
	return (*d)[index : index+rb], index + rb
}

// ReadSize8 reads eight bytes from the given index in the byte slice.
// Returns the bytes as a slice and the new index after reading.
func (cd *DecoderCommon) ReadSize8(index int, d *[]byte) ([]byte, int) {
	rb := def.Byte8
	return (*d)[index : index+rb], index + rb
}

// ReadSizeN reads a specified number of bytes (n) from the given index in the byte slice.
// Returns the bytes as a slice and the new index after reading.
func (cd *DecoderCommon) ReadSizeN(index, n int, d *[]byte) ([]byte, int) {
	return (*d)[index : index+n], index + n
}
