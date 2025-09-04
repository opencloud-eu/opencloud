//go:build !go1.10
// +build !go1.10

package loremipsum

import "math/rand"

// int31n returns, as an int32, a non-negative pseudo-random number in [0,n).
// n must be > 0, but int31n does not check this; the caller must ensure it.
// int31n exists because Int31n is inefficient, but Go 1 compatibility
// requires that the stream of values produced by math/rand remain unchanged.
// int31n can thus only be used internally, by newly introduced APIs.
//
// For implementation details, see:
// http://lemire.me/blog/2016/06/27/a-fast-alternative-to-the-modulo-reduction
// http://lemire.me/blog/2016/06/30/fast-random-shuffling
func (li *LoremIpsum) int31n(n int32) int32 {
	v := li.rng.Uint32()
	prod := uint64(v) * uint64(n)
	low := uint32(prod)
	if low < uint32(n) {
		thresh := uint32(-n) % uint32(n)
		for low < thresh {
			v = li.rng.Uint32()
			prod = uint64(v) * uint64(n)
			low = uint32(prod)
		}
	}
	return int32(prod >> 32)
}

// Shuffle pseudo-randomizes the order of elements.
// n is the number of elements. Shuffle panics if n < 0.
// swap swaps the elements with indexes i and j.
func (li *LoremIpsum) shuffleWords(n int, swap func(i, j int)) {
	if n < 0 {
		panic("invalid argument to Shuffle")
	}
	// Fisher-Yates shuffle: https://en.wikipedia.org/wiki/Fisher%E2%80%93Yates_shuffle
	// Shuffle really ought not be called with n that doesn't fit in 32 bits.
	// Not only will it take a very long time, but with 2³¹! possible permutations,
	// there's no way that any PRNG can have a big enough internal state to
	// generate even a minuscule percentage of the possible permutations.
	// Nevertheless, the right API signature accepts an int n, so handle it as best we can.
	i := n - 1
	for ; i > 1<<31-1-1; i-- {
		j := int(rand.Int63n(int64(i + 1)))
		swap(i, j)
	}
	for ; i > 0; i-- {
		j := int(li.int31n(int32(i + 1)))
		swap(i, j)
	}
}

// Shuffle the words
func (li *LoremIpsum) shuffle() {
	var words []string

	if !li.first {
		words = make([]string, len(loremIpsumWords))
		copy(words, loremIpsumWords[:])
	} else {
		words = make([]string, len(rest))
		copy(words, rest)
	}
	li.shuffleWords(len(words), func(i int, j int) {
		words[i], words[j] = words[j], words[i]
	})
	if li.first {
		b := make([]string, len(beg))
		copy(b, beg)
		// words, b = b, words
		// words = append(words, b...)
		words = append(b, words...)
	}
	li.words = words
	li.first = false
}
