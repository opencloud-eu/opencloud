package ext

import (
	"reflect"
)

// StreamDecoder defines an interface for decoding streams of data.
// It provides methods to retrieve the decoder's code, check type compatibility,
// and convert raw data into a Go value of a specified kind.
type StreamDecoder interface {
	// Code returns the unique identifier for the decoder.
	Code() int8

	// IsType checks if the provided code, inner type, and data length match the expected type.
	// Returns true if the type matches, otherwise false.
	IsType(code byte, innerType int8, dataLength int) bool

	// ToValue converts the raw data into a Go value of the specified kind.
	// Takes the code, raw data, and the target kind as input.
	// Returns the decoded value or an error if the conversion fails.
	ToValue(code byte, data []byte, k reflect.Kind) (any, error)
}
