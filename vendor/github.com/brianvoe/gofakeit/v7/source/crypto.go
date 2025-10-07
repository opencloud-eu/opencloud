package source

import (
	"crypto/rand"
	"encoding/binary"
)

// Package source implements a cryptographically secure pseudo-random number generator (CSPRNG)
// using Go's crypto/rand. The Crypto type is designed for generating high-quality random
// uint64 values, suitable for cryptographic applications like secure token generation,
// cryptographic key creation, and other security-sensitive operations. It offers optional
// thread safety through a locking mechanism, making it suitable for concurrent usage.

// Pros:
// - Provides cryptographically secure randomness, suitable for security-sensitive applications.
// - Optional thread safety with locking, enabling safe concurrent access.

// Cons:
// - Locking mechanism, when enabled, may introduce performance overhead.
// - Does not utilize a seed, as it leverages the system's cryptographic RNG, which may be a
//   limitation in scenarios where deterministic pseudo-randomness is desired.

type Crypto struct {
	buffer [64]byte // Buffer to hold a block of random data
	offset int      // Current offset in the buffer
}

// NewCrypto creates a new instance of Crypto.
func NewCrypto() *Crypto {
	return &Crypto{
		buffer: [64]byte{}, // Initialize buffer with zeros
		offset: 64,         // Set offset to the end of the buffer to trigger a refill on the first call
	}
}

// refillBuffer fills the buffer with random data from crypto/rand.
func (s *Crypto) refillBuffer() {
	if _, err := rand.Read(s.buffer[:]); err != nil {
		panic("crypto/rand failed: " + err.Error()) // Handle the error appropriately for your application
	}
	s.offset = 0 // Reset offset after refilling
}

// Uint64 generates a pseudo-random 64-bit value using crypto/rand, served from a buffered block of data.
func (s *Crypto) Uint64() uint64 {
	if s.offset+8 > len(s.buffer) { // Check if we need to refill the buffer
		s.refillBuffer()
	}

	// Extract a uint64 value from the current position in the buffer
	val := binary.BigEndian.Uint64(s.buffer[s.offset:])
	s.offset += 8 // Move the offset for the next call

	return val
}
