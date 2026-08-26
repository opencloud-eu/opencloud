// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package shardhash

// murmurhash3X86_32 is a pure Go implementation of MurmurHash3 x86 32-bit.
// Matches Lucene's StringHelper.murmurhash3_x86_32(byte[], offset, length, seed).
//
// Reference: Austin Appleby's original MurmurHash3 (public domain).
func murmurhash3X86_32(data []byte, seed int32) int32 {
	const (
		c1 = int32(-862048943) // 0xcc9e2d51 as signed
		c2 = int32(0x1b873593)
	)

	h1 := seed
	length := len(data)
	nblocks := length / 4

	// Body: process 4-byte blocks.
	for i := range nblocks {
		offset := i * 4
		k1 := int32(data[offset]) |
			int32(data[offset+1])<<8 |
			int32(data[offset+2])<<16 |
			int32(data[offset+3])<<24

		k1 *= c1
		k1 = (k1 << 15) | int32(uint32(k1)>>17) //nolint:gosec // intentional bit rotation via unsigned shift; G115 false positive
		k1 *= c2

		h1 ^= k1
		h1 = (h1 << 13) | int32(uint32(h1)>>19) //nolint:gosec // intentional bit rotation via unsigned shift; G115 false positive
		h1 = h1*5 + int32(-430675100)           // 0xe6546b64 as signed
	}

	// Tail: process remaining bytes.
	tail := nblocks * 4
	var k1 int32
	switch length & 3 {
	case 3:
		k1 ^= int32(data[tail+2]) << 16
		fallthrough
	case 2:
		k1 ^= int32(data[tail+1]) << 8
		fallthrough
	case 1:
		k1 ^= int32(data[tail])
		k1 *= c1
		k1 = (k1 << 15) | int32(uint32(k1)>>17) //nolint:gosec // intentional bit rotation
		k1 *= c2
		h1 ^= k1
	}

	// Finalization mix.
	h1 ^= int32(length) //nolint:gosec // length is always small (2 * string length)
	h1 = fmix32(h1)

	return h1
}

// fmix32 is the MurmurHash3 finalization mix for 32-bit.
//
//nolint:gosec // All int32<->uint32 conversions are intentional bit manipulation for hash finalization.
func fmix32(h int32) int32 {
	h ^= int32(uint32(h) >> 16)
	h *= int32(-2048144789) // 0x85ebca6b as signed
	h ^= int32(uint32(h) >> 13)
	h *= int32(-1028477387) // 0xc2b2ae35 as signed
	h ^= int32(uint32(h) >> 16)
	return h
}
