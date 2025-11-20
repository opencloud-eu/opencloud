package displaywidth

import (
	"unicode/utf8"

	"github.com/clipperhouse/stringish"
	"github.com/clipperhouse/uax29/v2/graphemes"
)

// String calculates the display width of a string
// using the [DefaultOptions]
func String(s string) int {
	return DefaultOptions.String(s)
}

// Bytes calculates the display width of a []byte
// using the [DefaultOptions]
func Bytes(s []byte) int {
	return DefaultOptions.Bytes(s)
}

func Rune(r rune) int {
	return DefaultOptions.Rune(r)
}

type Options struct {
	EastAsianWidth     bool
	StrictEmojiNeutral bool
}

var DefaultOptions = Options{
	EastAsianWidth:     false,
	StrictEmojiNeutral: true,
}

// String calculates the display width of a string
// for the given options
func (options Options) String(s string) int {
	if len(s) == 0 {
		return 0
	}

	total := 0
	g := graphemes.FromString(s)
	for g.Next() {
		// The first character in the grapheme cluster determines the width;
		// modifiers and joiners do not contribute to the width.
		props, _ := lookupProperties(g.Value())
		total += props.width(options)
	}
	return total
}

// BytesOptions calculates the display width of a []byte
// for the given options
func (options Options) Bytes(s []byte) int {
	if len(s) == 0 {
		return 0
	}

	total := 0
	g := graphemes.FromBytes(s)
	for g.Next() {
		// The first character in the grapheme cluster determines the width;
		// modifiers and joiners do not contribute to the width.
		props, _ := lookupProperties(g.Value())
		total += props.width(options)
	}
	return total
}

func (options Options) Rune(r rune) int {
	// Fast path for ASCII
	if r < utf8.RuneSelf {
		if isASCIIControl(byte(r)) {
			// Control (0x00-0x1F) and DEL (0x7F)
			return 0
		}
		// ASCII printable (0x20-0x7E)
		return 1
	}

	// Surrogates (U+D800-U+DFFF) are invalid UTF-8 and have zero width
	// Other packages might turn them into the replacement character (U+FFFD)
	// in which case, we won't see it.
	if r >= 0xD800 && r <= 0xDFFF {
		return 0
	}

	// Stack-allocated to avoid heap allocation
	var buf [4]byte // UTF-8 is at most 4 bytes
	n := utf8.EncodeRune(buf[:], r)
	// Skip the grapheme iterator and directly lookup properties
	props, _ := lookupProperties(buf[:n])
	return props.width(options)
}

func isASCIIControl(b byte) bool {
	return b < 0x20 || b == 0x7F
}

const defaultWidth = 1

// is returns true if the property flag is set
func (p property) is(flag property) bool {
	return p&flag != 0
}

// lookupProperties returns the properties for the first character in a string
func lookupProperties[T stringish.Interface](s T) (property, int) {
	if len(s) == 0 {
		return 0, 0
	}

	// Fast path for ASCII characters (single byte)
	b := s[0]
	if b < utf8.RuneSelf { // Single-byte ASCII
		if isASCIIControl(b) {
			// Control characters (0x00-0x1F) and DEL (0x7F) - width 0
			return _ZeroWidth, 1
		}
		// ASCII printable characters (0x20-0x7E) - width 1
		// Return 0 properties, width calculation will default to 1
		return 0, 1
	}

	// Use the generated trie for lookup
	props, size := lookup(s)
	return property(props), size
}

// width determines the display width of a character based on its properties
// and configuration options
func (p property) width(options Options) int {
	if p == 0 {
		// Character not in trie, use default behavior
		return defaultWidth
	}

	if p.is(_ZeroWidth) {
		return 0
	}

	if options.EastAsianWidth {
		if p.is(_East_Asian_Ambiguous) {
			return 2
		}
		if p.is(_East_Asian_Ambiguous|_Emoji) && !options.StrictEmojiNeutral {
			return 2
		}
	}

	if p.is(_East_Asian_Full_Wide) {
		return 2
	}

	// Default width for all other characters
	return defaultWidth
}
