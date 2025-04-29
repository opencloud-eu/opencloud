package strutil

import (
	"path"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Equal check, alias of strings.EqualFold
var Equal = strings.EqualFold

// IsNumChar returns true if the given character is a numeric, otherwise false.
func IsNumChar(c byte) bool { return c >= '0' && c <= '9' }

var numReg = regexp.MustCompile(`^\d+$`)

// IsNumeric returns true if the given string is a numeric, otherwise false.
func IsNumeric(s string) bool { return numReg.MatchString(s) }

// IsAlphabet char
func IsAlphabet(char uint8) bool {
	// A 65 -> Z 90
	if char >= 'A' && char <= 'Z' {
		return true
	}

	// a 97 -> z 122
	if char >= 'a' && char <= 'z' {
		return true
	}
	return false
}

// IsAlphaNum reports whether the byte is an ASCII letter, number, or underscore
func IsAlphaNum(c uint8) bool {
	return c == '_' || '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}

// StrPos alias of the strings.Index
func StrPos(s, sub string) int { return strings.Index(s, sub) }

// BytePos alias of the strings.IndexByte
func BytePos(s string, bt byte) int { return strings.IndexByte(s, bt) }

// IEqual ignore case check given two string is equals.
func IEqual(s1, s2 string) bool { return strings.EqualFold(s1, s2) }

// NoCaseEq check two strings is equals and case-insensitivity
func NoCaseEq(s, t string) bool { return strings.EqualFold(s, t) }

// IContains ignore case check substr in the given string.
func IContains(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

// ContainsByte in given string.
func ContainsByte(s string, c byte) bool {
	return strings.IndexByte(s, c) >= 0
}

// InArray alias of HasOneSub()
var InArray = HasOneSub

// ContainsOne substr(s) in the given string. alias of HasOneSub()
func ContainsOne(s string, subs []string) bool { return HasOneSub(s, subs) }

// HasOneSub substr(s) in the given string.
func HasOneSub(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// ContainsAll substr(s) in the given string. alias of HasAllSubs()
func ContainsAll(s string, subs []string) bool { return HasAllSubs(s, subs) }

// HasAllSubs all substr in the given string.
func HasAllSubs(s string, subs []string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

// StartsWithAny alias of the HasOnePrefix
var StartsWithAny = HasOneSuffix

// IsStartsOf alias of the HasOnePrefix
func IsStartsOf(s string, prefixes []string) bool {
	return HasOnePrefix(s, prefixes)
}

// HasOnePrefix the string start withs one of the subs
func HasOnePrefix(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

// StartsWith alias func for HasPrefix
var StartsWith = strings.HasPrefix

// HasPrefix substr in the given string.
func HasPrefix(s string, prefix string) bool { return strings.HasPrefix(s, prefix) }

// IsStartOf alias of the strings.HasPrefix
func IsStartOf(s, prefix string) bool { return strings.HasPrefix(s, prefix) }

// HasSuffix substr in the given string.
func HasSuffix(s string, suffix string) bool { return strings.HasSuffix(s, suffix) }

// IsEndOf alias of the strings.HasSuffix
func IsEndOf(s, suffix string) bool { return strings.HasSuffix(s, suffix) }

// HasOneSuffix the string end withs one of the subs
func HasOneSuffix(s string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

// IsValidUtf8 valid utf8 string check
func IsValidUtf8(s string) bool { return utf8.ValidString(s) }

// ----- refer from github.com/yuin/goldmark/util

// refer from github.com/yuin/goldmark/util
var spaceTable = [256]int8{0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

// IsSpace returns true if the given character is a space, otherwise false.
func IsSpace(c byte) bool { return spaceTable[c] == 1 }

// IsEmpty returns true if the given string is empty.
func IsEmpty(s string) bool { return len(s) == 0 }

// IsBlank returns true if the given string is all space characters.
func IsBlank(s string) bool { return IsBlankBytes([]byte(s)) }

// IsNotBlank returns true if the given string is not blank.
func IsNotBlank(s string) bool { return !IsBlankBytes([]byte(s)) }

// IsBlankBytes returns true if the given []byte is all space characters.
func IsBlankBytes(bs []byte) bool {
	for _, b := range bs {
		if !IsSpace(b) {
			return false
		}
	}
	return true
}

// IsSymbol reports whether the rune is a symbolic character.
func IsSymbol(r rune) bool { return unicode.IsSymbol(r) }

// HasEmpty value for input strings
func HasEmpty(ss ...string) bool {
	for _, s := range ss {
		if s == "" {
			return true
		}
	}
	return false
}

// IsAllEmpty for input strings
func IsAllEmpty(ss ...string) bool {
	for _, s := range ss {
		if s != "" {
			return false
		}
	}
	return true
}

var verRegex = regexp.MustCompile(`^[0-9][\d.]+(-\w+)?$`)

// IsVersion number. eg: 1.2.0
func IsVersion(s string) bool { return verRegex.MatchString(s) }

// Compare for two string.
func Compare(s1, s2, op string) bool { return VersionCompare(s1, s2, op) }

// VersionCompare for two version string.
func VersionCompare(v1, v2, op string) bool {
	switch op {
	case ">", "gt":
		return v1 > v2
	case "<", "lt":
		return v1 < v2
	case ">=", "gte":
		return v1 >= v2
	case "<=", "lte":
		return v1 <= v2
	case "!=", "ne", "neq":
		return v1 != v2
	default: // eq
		return v1 == v2
	}
}

// SimpleMatch all sub-string in the give text string.
//
// Difference the ContainsAll:
//
//   - start with ^ for exclude contains check.
//   - end with $ for check end with keyword.
func SimpleMatch(s string, keywords []string) bool {
	for _, keyword := range keywords {
		kln := len(keyword)
		if kln == 0 {
			continue
		}

		// exclude
		if kln > 1 && keyword[0] == '^' {
			if strings.Contains(s, keyword[1:]) {
				return false
			}
			continue
		}

		// end with
		if kln > 1 && keyword[kln-1] == '$' {
			return strings.HasSuffix(s, keyword[:kln-1])
		}

		// include
		if !strings.Contains(s, keyword) {
			return false
		}
	}
	return true
}

// QuickMatch check for a string. pattern can be a sub string.
func QuickMatch(pattern, s string) bool {
	if strings.ContainsRune(pattern, '*') {
		return GlobMatch(pattern, s)
	}
	return strings.Contains(s, pattern)
}

// PathMatch check for a string match the pattern. alias of the path.Match()
//
// TIP: `*` can match any char, not contain `/`.
func PathMatch(pattern, s string) bool {
	ok, err := path.Match(pattern, s)
	if err != nil {
		ok = false
	}
	return ok
}

// GlobMatch check for a string match the pattern.
//
// Difference with PathMatch() is: `*` can match any char, contain `/`.
func GlobMatch(pattern, s string) bool {
	// replace `/` to `S` for path.Match
	pattern = strings.Replace(pattern, "/", "S", -1)
	s = strings.Replace(s, "/", "S", -1)

	ok, err := path.Match(pattern, s)
	if err != nil {
		ok = false
	}
	return ok
}

// LikeMatch simple check for a string match the pattern. pattern like the SQL LIKE.
func LikeMatch(pattern, s string) bool {
	ln := len(pattern)
	if ln < 2 {
		return false
	}

	// eg `%abc` `%abc%`
	if pattern[0] == '%' {
		if ln > 2 && pattern[ln-1] == '%' {
			return strings.Contains(s, pattern[1:ln-1])
		}
		return strings.HasSuffix(s, pattern[1:])
	}

	// eg `abc%`
	if pattern[ln-1] == '%' {
		return strings.HasPrefix(s, pattern[:ln-1])
	}
	return pattern == s
}

// MatchNodePath check for a string match the pattern.
//
// Use on pattern:
//   - `*` match any to sep
//   - `**` match any to end. only allow at start or end on pattern.
//
// Example:
//
//	strutil.MatchNodePath()
func MatchNodePath(pattern, s string, sep string) bool {
	if pattern == "**" || pattern == s {
		return true
	}
	if pattern == "" {
		return len(s) == 0
	}

	if i := strings.Index(pattern, "**"); i >= 0 {
		if i == 0 { // at start
			return strings.HasSuffix(s, pattern[2:])
		}
		return strings.HasPrefix(s, pattern[:len(pattern)-2])
	}

	pattern = strings.Replace(pattern, sep, "/", -1)
	s = strings.Replace(s, sep, "/", -1)

	ok, err := path.Match(pattern, s)
	if err != nil {
		ok = false
	}
	return ok
}
