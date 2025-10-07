package source

// The JSF(Jenkins Small Fast) pseudo-random number generator.
// Developed by Bob Jenkins, JSF is known for its speed and efficiency, making it suitable
// for applications requiring fast, non-cryptographic quality random numbers. This implementation
// offers seamless integration with Go's math/rand package and includes an improved seeding mechanism.

// Pros:
// - Fast and efficient, ideal for high-performance requirements.
// - Good randomness quality for non-cryptographic applications.
// - Small state size and simple operations, ensuring a minimal memory footprint.

// Cons:
// - Not suitable for cryptographic purposes due to its non-cryptographic security level.
// - Quality of randomness may not match that of more complex algorithms.

type JSF struct {
	a, b, c, d uint32
}

// NewJSF creates and returns a new JSF pseudo-random number generator.
func NewJSF(seed uint64) *JSF {
	jsf := &JSF{}
	jsf.Seed(seed)
	return jsf
}

// Seed sets the seed of the JSF with an improved seeding mechanism.
func (jsf *JSF) Seed(seed uint64) {
	// Use the seed to derive initial values for a, b, c, d with better distribution
	// Splitting the 64-bit seed into parts and using different operations to diversify
	s1 := uint32(seed)
	s2 := uint32(seed >> 32)
	jsf.a = 0xf1ea5eed
	jsf.b = s1 ^ jsf.a
	jsf.c = s2 ^ jsf.b
	jsf.d = s1
}

// Uint64 generates a pseudo-random 64-bit value using the improved JSF algorithm.
func (jsf *JSF) Uint64() uint64 {
	e := jsf.a - (jsf.b<<27 | jsf.b>>(32-27))
	f := jsf.b ^ (jsf.c << 17)
	jsf.c += jsf.d
	jsf.d += e
	jsf.a = jsf.b + f
	jsf.b = jsf.c + e
	jsf.c = f + jsf.a
	return uint64(jsf.d)<<32 | uint64(jsf.a)
}
