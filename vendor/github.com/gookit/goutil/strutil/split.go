package strutil

import (
	"regexp"
	"strings"
)

// BeforeFirst get substring before first sep.
func BeforeFirst(s, sep string) string {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i]
	}
	return s
}

// AfterFirst get substring after first sep.
func AfterFirst(s, sep string) string {
	if i := strings.Index(s, sep); i >= 0 {
		return s[i+len(sep):]
	}
	return ""
}

// BeforeLast get substring before last sep.
func BeforeLast(s, sep string) string {
	if i := strings.LastIndex(s, sep); i >= 0 {
		return s[:i]
	}
	return s
}

// AfterLast get substring after last sep.
func AfterLast(s, sep string) string {
	if i := strings.LastIndex(s, sep); i >= 0 {
		return s[i+len(sep):]
	}
	return ""
}

/*************************************************************
 * String split operation
 *************************************************************/

// Cut same of the strings.Cut
func Cut(s, sep string) (before string, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}

// QuietCut always returns two substring.
func QuietCut(s, sep string) (before string, after string) {
	before, after, _ = Cut(s, sep)
	return
}

// MustCut always returns two substring.
func MustCut(s, sep string) (before string, after string) {
	var ok bool
	before, after, ok = Cut(s, sep)
	if !ok {
		panic("cannot split input string to two nodes")
	}
	return
}

// TrimCut always returns two substring and trim space for items.
func TrimCut(s, sep string) (string, string) {
	before, after, _ := Cut(s, sep)
	return strings.TrimSpace(before), strings.TrimSpace(after)
}

// SplitKV split string to key and value.
func SplitKV(s, sep string) (string, string) { return TrimCut(s, sep) }

// SplitValid string to slice. will trim each item and filter empty string node.
func SplitValid(s, sep string) (ss []string) { return Split(s, sep) }

// Split string to slice. will trim each item and filter empty string node.
func Split(s, sep string) (ss []string) {
	if s = strings.TrimSpace(s); s == "" {
		return
	}

	for _, val := range strings.Split(s, sep) {
		if val = strings.TrimSpace(val); val != "" {
			ss = append(ss, val)
		}
	}
	return
}

// SplitNValid string to slice. will filter empty string node.
func SplitNValid(s, sep string, n int) (ss []string) { return SplitN(s, sep, n) }

// SplitN string to slice. will filter empty string node.
func SplitN(s, sep string, n int) (ss []string) {
	if s = strings.TrimSpace(s); s == "" {
		return
	}

	rawList := strings.Split(s, sep)
	for i, val := range rawList {
		if val = strings.TrimSpace(val); val != "" {
			if len(ss) == n-1 {
				ss = append(ss, strings.TrimSpace(strings.Join(rawList[i:], sep)))
				break
			}

			ss = append(ss, val)
		}
	}
	return
}

// SplitTrimmed split string to slice.
// will trim space for each node, but not filter empty
func SplitTrimmed(s, sep string) (ss []string) {
	if s = strings.TrimSpace(s); s == "" {
		return
	}

	for _, val := range strings.Split(s, sep) {
		ss = append(ss, strings.TrimSpace(val))
	}
	return
}

// SplitNTrimmed split string to slice.
// will trim space for each node, but not filter empty
func SplitNTrimmed(s, sep string, n int) (ss []string) {
	if s = strings.TrimSpace(s); s == "" {
		return
	}

	for _, val := range strings.SplitN(s, sep, n) {
		ss = append(ss, strings.TrimSpace(val))
	}
	return
}

// 根据空白字符（空格，TAB，换行等）分隔字符串
var whitespaceRegexp = regexp.MustCompile("\\s+")

// SplitByWhitespace Separate strings by whitespace characters (space, TAB, newline, etc.)
func SplitByWhitespace(s string) []string {
	return whitespaceRegexp.Split(s, -1)
}

// Substr for a string.
// if length <= 0, return pos to end.
func Substr(s string, pos, length int) string {
	runes := []rune(s)
	strLn := len(runes)

	// pos is too large
	if pos >= strLn {
		return ""
	}

	stopIdx := pos + length
	if length == 0 || stopIdx > strLn {
		stopIdx = strLn
	} else if length < 0 {
		stopIdx = strLn + length
	}

	return string(runes[pos:stopIdx])
}

// SplitInlineComment for an inline text string.
func SplitInlineComment(val string, strict ...bool) (string, string) {
	// strict check: must with space
	if len(strict) > 0 && strict[0] {
		if pos := strings.Index(val, " #"); pos > -1 {
			return strings.TrimRight(val[0:pos], " "), val[pos+1:]
		}

		if pos := strings.Index(val, " //"); pos > -1 {
			return strings.TrimRight(val[0:pos], " "), val[pos+1:]
		}
	} else {
		if pos := strings.IndexByte(val, '#'); pos > -1 {
			return strings.TrimRight(val[0:pos], " "), val[pos:]
		}

		if pos := strings.Index(val, "//"); pos > -1 {
			return strings.TrimRight(val[0:pos], " "), val[pos:]
		}
	}

	return val, ""
}

// FirstLine from command output
func FirstLine(output string) string {
	if i := strings.IndexByte(output, '\n'); i >= 0 {
		return output[0:i]
	}
	return output
}
