package decoding

import (
	"reflect"

	"github.com/shamaton/msgpack/v2/internal/common/decodingutil"
)

const (
	// maxPreallocBytes bounds the initial allocation made from an
	// attacker-declared byte length (Bin/Str/Ext payloads) before any
	// payload byte has been read. Larger declared lengths are read
	// incrementally and grown only as bytes actually arrive.
	maxPreallocBytes = 256 << 10 // 256 KiB

	// maxPreallocMapSize bounds the entry hint used to pre-size a map
	// from an attacker-declared pair count before any pair is decoded.
	maxPreallocMapSize = 8192
)

// lengthFromUint32 converts a MessagePack-declared 32-bit length to int,
// rejecting values that are not representable as a non-negative int.
func lengthFromUint32(u uint32) (int, error) {
	return decodingutil.LengthFromUint32(u)
}

// initialByteCap returns the capacity to reserve for a declared byte length
// before its payload has been read.
func initialByteCap(n int) int {
	if n < 1 {
		return 0
	}
	if n < maxPreallocBytes {
		return n
	}
	return maxPreallocBytes
}

// initialSliceCap returns the element capacity to reserve for a declared
// slice length, capped both by the declared count and by a byte budget so
// large element types never over-commit.
func initialSliceCap(l int, elemType reflect.Type) int {
	if l < 1 {
		return 0
	}
	elemSize := elemType.Size()
	if elemSize < 1 {
		elemSize = 1
	}
	budget := maxPreallocBytes / int(elemSize)
	if budget < 1 {
		budget = 1
	}
	if l < budget {
		return l
	}
	return budget
}

// initialMapCap returns the entry hint used to pre-size a map from a
// declared pair count.
func initialMapCap(l int) int {
	if l < 1 {
		return 0
	}
	if l < maxPreallocMapSize {
		return l
	}
	return maxPreallocMapSize
}

// initialMapCapForType is initialMapCap for maps created dynamically via
// reflection, where the key/value types are only known at decode time and
// can be arbitrarily large structs. It additionally bounds the hint by a
// byte budget derived from the key/value sizes so a declared pair count
// can't force a multi-megabyte allocation before any pair has been decoded.
func initialMapCapForType(l int, key, value reflect.Type) int {
	if l < 1 {
		return 0
	}
	entrySize := key.Size() + value.Size()
	if entrySize < 1 {
		entrySize = 1
	}
	budget := maxPreallocBytes / int(entrySize)
	if budget < 1 {
		budget = 1
	}
	if l < budget {
		return initialMapCap(l)
	}
	return initialMapCap(budget)
}
