package strutil

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/width"
)

// RuneIsWord char: a-zA-Z
func RuneIsWord(c rune) bool {
	return RuneIsLower(c) || RuneIsUpper(c)
}

// RuneIsLower char
func RuneIsLower(c rune) bool {
	return 'a' <= c && c <= 'z'
}

// RuneIsUpper char
func RuneIsUpper(c rune) bool {
	return 'A' <= c && c <= 'Z'
}

// RunePos alias of the strings.IndexRune
func RunePos(s string, ru rune) int { return strings.IndexRune(s, ru) }

// IsSpaceRune returns true if the given rune is a space, otherwise false.
func IsSpaceRune(r rune) bool {
	return r <= 256 && IsSpace(byte(r)) || unicode.IsSpace(r)
}

// Utf8Len count of the string
func Utf8Len(s string) int { return utf8.RuneCountInString(s) }

// Utf8len of the string
func Utf8len(s string) int { return utf8.RuneCountInString(s) }

// RuneCount of the string
func RuneCount(s string) int { return len([]rune(s)) }

// RuneWidth of the rune.
//
// Example:
//
//	RuneWidth('你') // 2
//	RuneWidth('a') // 1
//	RuneWidth('\n') // 0
func RuneWidth(r rune) int {
	p := width.LookupRune(r)
	k := p.Kind()

	// eg: "\n"
	if k == width.Neutral {
		return 0
	}

	if k == width.EastAsianFullwidth || k == width.EastAsianWide || k == width.EastAsianAmbiguous {
		return 2
	}
	return 1
}

// TextWidth utf8 string width. alias of RunesWidth()
func TextWidth(s string) int { return Utf8Width(s) }

// Utf8Width utf8 string width. alias of RunesWidth
func Utf8Width(s string) int { return RunesWidth([]rune(s)) }

// RunesWidth utf8 runes string width.
//
// Examples:
//
//	str := "hi,你好"
//
//	len(str) // 9
//	strutil.Utf8Width(str) // 7
//	len([]rune(str)) = utf8.RuneCountInString(s) // 5
func RunesWidth(rs []rune) (w int) {
	if len(rs) == 0 {
		return
	}

	for _, runeVal := range rs {
		w += RuneWidth(runeVal)
	}
	return w
}

// Truncate alias of the Utf8Truncate()
func Truncate(s string, w int, tail string) string { return Utf8Truncate(s, w, tail) }

// TextTruncate alias of the Utf8Truncate()
func TextTruncate(s string, w int, tail string) string { return Utf8Truncate(s, w, tail) }

// Utf8Truncate a string with given width.
func Utf8Truncate(s string, w int, tail string) string {
	if sw := Utf8Width(s); sw <= w {
		return s
	}

	i := 0
	r := []rune(s)
	w -= TextWidth(tail)

	tmpW := 0
	for ; i < len(r); i++ {
		cw := RuneWidth(r[i])
		if tmpW+cw > w {
			break
		}
		tmpW += cw
	}
	return string(r[0:i]) + tail
}

// Chunk split string to chunks by size.
// func Chunk[T ~string](s T, size int) []T {
// }

// TextSplit alias of the Utf8Split()
func TextSplit(s string, w int) []string { return Utf8Split(s, w) }

// Utf8Split split a string by width.
func Utf8Split(s string, w int) []string {
	sw := Utf8Width(s)
	if sw <= w {
		return []string{s}
	}

	tmpW := 0
	tmpS := ""

	ss := make([]string, 0, sw/w+1)
	for _, r := range s {
		rw := RuneWidth(r)
		if tmpW+rw == w {
			tmpS += string(r)
			ss = append(ss, tmpS)

			tmpW, tmpS = 0, "" // reset
			continue
		}

		if tmpW+rw > w {
			ss = append(ss, tmpS)

			// append to next line.
			tmpW, tmpS = rw, string(r)
			continue
		}

		tmpW += rw
		tmpS += string(r)
	}

	if tmpW > 0 {
		ss = append(ss, tmpS)
	}
	return ss
}

// TextWrap a string by "\n"
func TextWrap(s string, w int) string { return WidthWrap(s, w) }

// WidthWrap a string by "\n"
func WidthWrap(s string, w int) string {
	tmpW := 0
	out := ""

	for _, r := range s {
		cw := RuneWidth(r)
		if r == '\n' {
			out += string(r)
			tmpW = 0
			continue
		}

		if tmpW+cw > w {
			out += "\n"
			tmpW = 0
			out += string(r)
			tmpW += cw
			continue
		}

		out += string(r)
		tmpW += cw
	}
	return out
}

// WordWrap text string and limit width.
func WordWrap(s string, w int) string {
	tmpW := 0
	out := ""

	for _, sub := range strings.Split(s, " ") {
		cw := TextWidth(sub)
		if tmpW+cw > w {
			if tmpW != 0 {
				out += "\n"
			}

			tmpW = 0
			out += sub
			tmpW += cw
			continue
		}

		out += sub
		tmpW += cw
	}
	return out
}

// Runes data slice
type Runes []rune

// Padding a rune to want length and with position
func (rs Runes) Padding(pad rune, length int, pos PosFlag) []rune {
	return PadChars(rs, pad, length, pos)
}

// PadLeft a rune to want length
func (rs Runes) PadLeft(pad rune, length int) []rune {
	return rs.Padding(pad, length, PosLeft)
}

// PadRight a rune to want length
func (rs Runes) PadRight(pad rune, length int) []rune {
	return rs.Padding(pad, length, PosRight)
}
