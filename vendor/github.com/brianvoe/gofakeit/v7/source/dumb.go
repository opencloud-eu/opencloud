package source

import "time"

// Dumb is a deterministic pseudo-random number generator designed specifically for testing purposes.
// It offers predictable sequences of numbers based on the provided seed, making it ideal for scenarios
// where consistent and reproducible test results are critical. By default, if initialized with a seed of 0,
// Dumb uses the current timestamp to generate a starting point, ensuring some level of variability between runs.

// Pros:
// - Predictability: Ensures reproducible outcomes in tests by providing a consistent sequence of numbers for a given seed.
// - Simplicity: Easy to understand and integrate into testing frameworks, with minimal overhead.
// - Default Variability: Uses the current timestamp as the default seed, providing variability across different test runs when no seed is specified.

// Cons:
// - Not Suitable for Production: Lacks the randomness quality required for production-level cryptographic or statistical applications.
// - Limited Randomness: The simple incrementation approach does not simulate the complexity of real-world random number generation.

// Dumb is a simplistic generator for predictable testing.
type Dumb struct {
	state uint64
}

// NewDumb initializes a Dumb generator.
// If the seed is 0, initializes with the current timestamp.
func NewDumb(seed uint64) *Dumb {
	d := &Dumb{}
	d.Seed(seed)
	return d
}

// Seed sets the generator's state. If the seed is 0, it uses the current timestamp as the seed.
func (d *Dumb) Seed(seed uint64) {
	if seed == 0 {
		seed = uint64(time.Now().UnixNano())
	}
	d.state = seed
}

// Uint64 returns the next number in the sequence, incrementing the state.
func (d *Dumb) Uint64() uint64 {
	d.state += 1
	return d.state
}
