package displaywidth

import (
	"unicode/utf8"

	"github.com/clipperhouse/stringish"
	"github.com/clipperhouse/uax29/v2/graphemes"
)

// Options allows you to specify the treatment of ambiguous East Asian
// characters. When EastAsianWidth is false (default), ambiguous East Asian
// characters are treated as width 1. When EastAsianWidth is true, ambiguous
// East Asian characters are treated as width 2.
type Options struct {
	EastAsianWidth bool
}

// DefaultOptions is the default options for the display width
// calculation, which is EastAsianWidth: false.
var DefaultOptions = Options{EastAsianWidth: false}

// String calculates the display width of a string,
// by iterating over grapheme clusters in the string
// and summing their widths.
func String(s string) int {
	return DefaultOptions.String(s)
}

// String calculates the display width of a string, for the given options, by
// iterating over grapheme clusters in the string and summing their widths.
func (options Options) String(s string) int {
	// Optimization: no need to parse grapheme
	switch len(s) {
	case 0:
		return 0
	case 1:
		return int(asciiWidths[s[0]])
	}

	width := 0
	g := graphemes.FromString(s)
	for g.Next() {
		width += graphemeWidth(g.Value(), options)
	}
	return width
}

// Bytes calculates the display width of a []byte,
// by iterating over grapheme clusters in the byte slice
// and summing their widths.
func Bytes(s []byte) int {
	return DefaultOptions.Bytes(s)
}

// Bytes calculates the display width of a []byte, for the given options, by
// iterating over grapheme clusters in the slice and summing their widths.
func (options Options) Bytes(s []byte) int {
	// Optimization: no need to parse grapheme
	switch len(s) {
	case 0:
		return 0
	case 1:
		return int(asciiWidths[s[0]])
	}

	width := 0
	g := graphemes.FromBytes(s)
	for g.Next() {
		width += graphemeWidth(g.Value(), options)
	}
	return width
}

// Rune calculates the display width of a rune. You
// should almost certainly use [String] or [Bytes] for
// most purposes.
//
// The smallest unit of display width is a grapheme
// cluster, not a rune. Iterating over runes to measure
// width is incorrect in many cases.
func Rune(r rune) int {
	return DefaultOptions.Rune(r)
}

// Rune calculates the display width of a rune, for the given options.
//
// You should almost certainly use [String] or [Bytes] for most purposes.
//
// The smallest unit of display width is a grapheme cluster, not a rune.
// Iterating over runes to measure width is incorrect in many cases.
func (options Options) Rune(r rune) int {
	if r < utf8.RuneSelf {
		return int(asciiWidths[byte(r)])
	}

	// Surrogates (U+D800-U+DFFF) are invalid UTF-8.
	if r >= 0xD800 && r <= 0xDFFF {
		return 0
	}

	var buf [4]byte
	n := utf8.EncodeRune(buf[:], r)

	// Skip the grapheme iterator
	return lookupProperties(buf[:n]).width(options)
}

// graphemeWidth returns the display width of a grapheme cluster.
// The passed string must be a single grapheme cluster.
func graphemeWidth[T stringish.Interface](s T, options Options) int {
	// Optimization: no need to look up properties
	switch len(s) {
	case 0:
		return 0
	case 1:
		return int(asciiWidths[s[0]])
	}

	return lookupProperties(s).width(options)
}

// isRIPrefix checks if the slice matches the Regional Indicator prefix
// (F0 9F 87). It assumes len(s) >= 3.
func isRIPrefix[T stringish.Interface](s T) bool {
	return s[0] == 0xF0 && s[1] == 0x9F && s[2] == 0x87
}

// isVS16 checks if the slice matches VS16 (U+FE0F) UTF-8 encoding
// (EF B8 8F). It assumes len(s) >= 3.
func isVS16[T stringish.Interface](s T) bool {
	return s[0] == 0xEF && s[1] == 0xB8 && s[2] == 0x8F
}

// lookupProperties returns the properties for a grapheme.
// The passed string must be at least one byte long.
//
// Callers must handle zero and single-byte strings upstream, both as an
// optimization, and to reduce the scope of this function.
func lookupProperties[T stringish.Interface](s T) property {
	l := len(s)

	if s[0] < utf8.RuneSelf {
		// Check for variation selector after ASCII (e.g., keycap sequences like 1️⃣)
		if l >= 4 {
			// Subslice may help eliminate bounds checks
			vs := s[1:4]
			if isVS16(vs) {
				// VS16 requests emoji presentation (width 2)
				return _Emoji
			}
			// VS15 (0x8E) requests text presentation but does not affect width,
			// in my reading of Unicode TR51. Falls through to _Default.
		}
		return asciiProperties[s[0]]
	}

	// Regional indicator pair (flag)
	if l >= 8 {
		// Subslice may help eliminate bounds checks
		ri := s[:8]
		// First rune
		if isRIPrefix(ri[0:3]) {
			b3 := ri[3]
			if b3 >= 0xA6 && b3 <= 0xBF {
				// Second rune
				if isRIPrefix(ri[4:7]) {
					b7 := ri[7]
					if b7 >= 0xA6 && b7 <= 0xBF {
						return _Emoji
					}
				}
			}
		}
	}

	p, sz := lookup(s)

	// Variation Selectors
	if sz > 0 && l >= sz+3 {
		// Subslice may help eliminate bounds checks
		vs := s[sz : sz+3]
		if isVS16(vs) {
			// VS16 requests emoji presentation (width 2)
			return _Emoji
		}
		// VS15 (0x8E) requests text presentation but does not affect width,
		// in my reading of Unicode TR51. Falls through to return the base
		// character's property.
	}

	return property(p)
}

const _Default property = 0
const boundsCheck = property(len(propertyWidths) - 1)

// width determines the display width of a character based on its properties,
// and configuration options
func (p property) width(options Options) int {
	if options.EastAsianWidth && p == _East_Asian_Ambiguous {
		return 2
	}

	// Bounds check may help the compiler eliminate its bounds check,
	// and safety of course.
	if p > boundsCheck {
		return 1 // default width
	}

	return propertyWidths[p]
}
