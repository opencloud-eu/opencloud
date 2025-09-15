package ext

import (
	"reflect"
)

// Encoder defines an interface for encoding values into bytes.
// It provides methods to get the encoding type, calculate the byte size of a value,
// and write the encoded value into a byte slice.
type Encoder interface {
	// Code returns the unique code representing the encoder type.
	Code() int8

	// Type returns the reflect.Type of the value that the encoder handles.
	Type() reflect.Type

	// CalcByteSize calculates the number of bytes required to encode the given value.
	// Returns the size and an error if the calculation fails.
	CalcByteSize(value reflect.Value) (int, error)

	// WriteToBytes encodes the given value into a byte slice starting at the specified offset.
	// Returns the new offset after writing the bytes.
	WriteToBytes(value reflect.Value, offset int, bytes *[]byte) int
}

// EncoderCommon provides utility methods for encoding various types of values into bytes.
// It includes methods to encode integers and unsigned integers of different sizes,
// as well as methods to write raw byte slices into a target byte slice.
type EncoderCommon struct {
}

// SetByte1Int64 encodes a single byte from the given int64 value into the byte slice at the specified offset.
// Returns the new offset after writing the byte.
func (c *EncoderCommon) SetByte1Int64(value int64, offset int, d *[]byte) int {
	(*d)[offset] = byte(value)
	return offset + 1
}

// SetByte2Int64 encodes the lower two bytes of the given int64 value into the byte slice at the specified offset.
// Returns the new offset after writing the bytes.
func (c *EncoderCommon) SetByte2Int64(value int64, offset int, d *[]byte) int {
	(*d)[offset+0] = byte(value >> 8)
	(*d)[offset+1] = byte(value)
	return offset + 2
}

// SetByte4Int64 encodes the lower four bytes of the given int64 value into the byte slice at the specified offset.
// Returns the new offset after writing the bytes.
func (c *EncoderCommon) SetByte4Int64(value int64, offset int, d *[]byte) int {
	(*d)[offset+0] = byte(value >> 24)
	(*d)[offset+1] = byte(value >> 16)
	(*d)[offset+2] = byte(value >> 8)
	(*d)[offset+3] = byte(value)
	return offset + 4
}

// SetByte8Int64 encodes all eight bytes of the given int64 value into the byte slice at the specified offset.
// Returns the new offset after writing the bytes.
func (c *EncoderCommon) SetByte8Int64(value int64, offset int, d *[]byte) int {
	(*d)[offset] = byte(value >> 56)
	(*d)[offset+1] = byte(value >> 48)
	(*d)[offset+2] = byte(value >> 40)
	(*d)[offset+3] = byte(value >> 32)
	(*d)[offset+4] = byte(value >> 24)
	(*d)[offset+5] = byte(value >> 16)
	(*d)[offset+6] = byte(value >> 8)
	(*d)[offset+7] = byte(value)
	return offset + 8
}

// SetByte1Uint64 encodes a single byte from the given uint64 value into the byte slice at the specified offset.
// Returns the new offset after writing the byte.
func (c *EncoderCommon) SetByte1Uint64(value uint64, offset int, d *[]byte) int {
	(*d)[offset] = byte(value)
	return offset + 1
}

// SetByte2Uint64 encodes the lower two bytes of the given uint64 value into the byte slice at the specified offset.
// Returns the new offset after writing the bytes.
func (c *EncoderCommon) SetByte2Uint64(value uint64, offset int, d *[]byte) int {
	(*d)[offset] = byte(value >> 8)
	(*d)[offset+1] = byte(value)
	return offset + 2
}

// SetByte4Uint64 encodes the lower four bytes of the given uint64 value into the byte slice at the specified offset.
// Returns the new offset after writing the bytes.
func (c *EncoderCommon) SetByte4Uint64(value uint64, offset int, d *[]byte) int {
	(*d)[offset] = byte(value >> 24)
	(*d)[offset+1] = byte(value >> 16)
	(*d)[offset+2] = byte(value >> 8)
	(*d)[offset+3] = byte(value)
	return offset + 4
}

// SetByte8Uint64 encodes all eight bytes of the given uint64 value into the byte slice at the specified offset.
// Returns the new offset after writing the bytes.
func (c *EncoderCommon) SetByte8Uint64(value uint64, offset int, d *[]byte) int {
	(*d)[offset] = byte(value >> 56)
	(*d)[offset+1] = byte(value >> 48)
	(*d)[offset+2] = byte(value >> 40)
	(*d)[offset+3] = byte(value >> 32)
	(*d)[offset+4] = byte(value >> 24)
	(*d)[offset+5] = byte(value >> 16)
	(*d)[offset+6] = byte(value >> 8)
	(*d)[offset+7] = byte(value)
	return offset + 8
}

// SetByte1Int encodes a single byte from the given int value into the byte slice at the specified offset.
// Returns the new offset after writing the byte.
func (c *EncoderCommon) SetByte1Int(code, offset int, d *[]byte) int {
	(*d)[offset] = byte(code)
	return offset + 1
}

// SetByte2Int encodes the lower two bytes of the given int value into the byte slice at the specified offset.
// Returns the new offset after writing the bytes.
func (c *EncoderCommon) SetByte2Int(value int, offset int, d *[]byte) int {
	(*d)[offset] = byte(value >> 8)
	(*d)[offset+1] = byte(value)
	return offset + 2
}

// SetByte4Int encodes the lower four bytes of the given int value into the byte slice at the specified offset.
// Returns the new offset after writing the bytes.
func (c *EncoderCommon) SetByte4Int(value int, offset int, d *[]byte) int {
	(*d)[offset] = byte(value >> 24)
	(*d)[offset+1] = byte(value >> 16)
	(*d)[offset+2] = byte(value >> 8)
	(*d)[offset+3] = byte(value)
	return offset + 4
}

// SetByte4Uint32 encodes the lower four bytes of the given uint32 value into the byte slice at the specified offset.
// Returns the new offset after writing the bytes.
func (c *EncoderCommon) SetByte4Uint32(value uint32, offset int, d *[]byte) int {
	(*d)[offset] = byte(value >> 24)
	(*d)[offset+1] = byte(value >> 16)
	(*d)[offset+2] = byte(value >> 8)
	(*d)[offset+3] = byte(value)
	return offset + 4
}

// SetBytes writes the given byte slice `bs` into the target byte slice at the specified offset.
// Returns the new offset after writing the bytes.
func (c *EncoderCommon) SetBytes(bs []byte, offset int, d *[]byte) int {
	for i := range bs {
		(*d)[offset+i] = bs[i]
	}
	return offset + len(bs)
}
