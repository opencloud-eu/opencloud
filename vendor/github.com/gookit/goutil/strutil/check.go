package strutil

import (
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gookit/goutil/internal/checkfn"
)

// Equal check, alias of strings.EqualFold
var Equal = strings.EqualFold
var IsHttpURL = checkfn.IsHttpURL

// IsNumChar returns true if the given character is a numeric, otherwise false.
func IsNumChar(c byte) bool { return c >= '0' && c <= '9' }

var (
	uintReg = regexp.MustCompile(`^\d+$`)
	intReg  = regexp.MustCompile(`^[-+]?\d+$`)

	floatReg = regexp.MustCompile(`^[-+]?\d*\.?\d+$`)
)

// IsInt check the string is an integer number
func IsInt(s string) bool {
	if s == "" {
		return false
	}
	return intReg.MatchString(s)
}

// IsUint check the string is an unsigned integer number
func IsUint(s string) bool {
	if s == "" {
		return false
	}
	return uintReg.MatchString(s)
}

// IsFloat check the string is a float number
func IsFloat(s string) bool {
	if s == "" {
		return false
	}
	return floatReg.MatchString(s)
}

// IsNumeric returns true if the given string is a numeric(int/float), otherwise false.
func IsNumeric(s string) bool { return checkfn.IsNumeric(s) }

// IsPositiveNum check the string is a positive number
func IsPositiveNum(s string) bool { return checkfn.IsPositiveNum(s) }

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

// IsUpper returns true if the given string is an uppercase, otherwise false.
func IsUpper(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			continue
		}
		return false
	}
	return true
}

// IsLower returns true if the given string is a lowercase, otherwise false.
func IsLower(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 'a' && s[i] <= 'z' {
			continue
		}
		return false
	}
	return true
}

// IsAllASCII 判断字符串是否全为可打印 ASCII（无中文等多字节字符）
func IsAllASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 0x7E || s[i] < 0x20 {
			return false
		}
	}
	return true
}

// StrPos alias of the strings.Index
func StrPos(s, sub string) int { return strings.Index(s, sub) }

// BytePos alias of the strings.IndexByte
func BytePos(s string, bt byte) int { return strings.IndexByte(s, bt) }

// IEqual ignore case check given two strings are equals.
func IEqual(s1, s2 string) bool { return strings.EqualFold(s1, s2) }

// NoCaseEq check two strings is equals and case-insensitivity
func NoCaseEq(s, t string) bool { return strings.EqualFold(s, t) }

// IContains ignore case check substr in the given string.
func IContains(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

// ContainsByte in given string.
func ContainsByte(s string, c byte) bool { return strings.IndexByte(s, c) >= 0 }

// ContainsByteOne in given string.
func ContainsByteOne(s string, bs []byte) bool {
	for _, b := range bs {
		if strings.IndexByte(s, b) >= 0 {
			return true
		}
	}
	return false
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

// IContainsOne ignore case check has one substr(s) in the given string.
func IContainsOne(s string, subs []string) bool {
	s = strings.ToLower(s)
	for _, sub := range subs {
		if strings.Contains(s, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}

// ContainsAll given string should contain all substrings. alias of HasAllSubs()
func ContainsAll(s string, subs []string) bool { return HasAllSubs(s, subs) }

// HasAllSubs given string should contain all substrings
func HasAllSubs(s string, subs []string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

// IContainsAll like ContainsAll(), but ignore case
func IContainsAll(s string, subs []string) bool {
	s = strings.ToLower(s)
	for _, sub := range subs {
		if !strings.Contains(s, strings.ToLower(sub)) {
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

// HasOnePrefix the string starts with one of the subs
func HasOnePrefix(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if len(s) >= len(prefix) && s[0:len(prefix)] == prefix {
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

// HasOneSuffix the string end with one of the subs
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
var spaceTable = [256]int8{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
}

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

var (
	// regex for check version number
	verRegex = regexp.MustCompile(`^[0-9][\d.]+(-\w+)?$`)
	// regex for check variable name
	varRegex = regexp.MustCompile(`^[a-zA-Z][\w-]*$`)
	// regex for check env var name
	envRegex = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	// IsVariableName alias for IsVarName
	IsVariableName = IsVarName
	// regex for check uuid string. format: 8-4-4-4-12
	uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)

// IsVersion number. eg: 1.2.0
func IsVersion(s string) bool { return verRegex.MatchString(s) }

// IsVarName is valid variable name.
func IsVarName(s string) bool { return varRegex.MatchString(s) }

// IsEnvName is valid ENV var name. eg: APP_NAME
func IsEnvName(s string) bool { return envRegex.MatchString(s) }

// IsUUID check if the string is a valid UUID format.
func IsUUID(s string) bool { return uuidPattern.MatchString(s) }

// Compare for two strings.
func Compare(s1, s2, op string) bool {
	switch op {
	case ">", "gt":
		return s1 > s2
	case "<", "lt":
		return s1 < s2
	case ">=", "gte":
		return s1 >= s2
	case "<=", "lte":
		return s1 <= s2
	case "!=", "ne", "neq":
		return s1 != s2
	default: // eq
		return s1 == s2
	}
}

// VersionCompare for two version strings. eg: 1.2.0 > 1.1.0
func VersionCompare(v1, v2, op string) bool {
	parts1 := parseVersion(v1)
	parts2 := parseVersion(v2)

	result := compareVersions(parts1, parts2)
	switch op {
	case ">", "gt":
		return result > 0
	case "<", "lt":
		return result < 0
	case "=", "==", "eq":
		return result == 0
	case "!=", "ne", "neq":
		return result != 0
	case ">=", "gte":
		return result >= 0
	case "<=", "lte":
		return result <= 0
	default:
		return false
	}
}

// parseVersion 将版本号字符串解析为整数数组
func parseVersion(version string) []int {
	if version[0] == 'v' || version[0] == 'V' {
		version = version[1:]
	}

	// 处理 Git 描述格式: v1.7.1-16-gc43a587
	// 格式: <tag>-<commits>-g<hash>，其中 commits 表示距离 tag 的提交数
	var extraCommits int
	var preRelease []int
	if idx := strings.Index(version, "-"); idx > 0 {
		remaining := version[idx+1:]
		version = version[:idx]
		// 尝试提取提交数（格式: 数字-g<hash> 或 数字）
		if dashIdx := strings.Index(remaining, "-"); dashIdx > 0 {
			if commits, err := strconv.Atoi(remaining[:dashIdx]); err == nil {
				extraCommits = commits
			} else {
				preRelease = parsePreRelease(remaining)
			}
		} else if commits, err := strconv.Atoi(remaining); err == nil {
			extraCommits = commits
		} else {
			preRelease = parsePreRelease(remaining)
		}
	}

	parts := strings.Split(version, ".")
	result := make([]int, len(parts))

	for i, part := range parts {
		num, _ := strconv.Atoi(part)
		result[i] = num
	}

	// 如果有额外的提交数，追加到版本号末尾
	// 这样 v1.7.1-16-gc43a587 会变成 [1, 7, 1, 16]
	// 而 v1.7.1 是 [1, 7, 1]，比较时前者更大
	if extraCommits > 0 {
		result = append(result, extraCommits)
	} else if len(preRelease) > 0 {
		result = append(result, preRelease...)
	}

	return result
}

func parsePreRelease(label string) []int {
	label = strings.ToLower(label)
	name := strings.Trim(label, ".-")
	num := 0

	for idx, char := range name {
		if char >= '0' && char <= '9' {
			if nVal, err := strconv.Atoi(strings.Trim(name[idx:], ".-")); err == nil {
				num = nVal
			}
			name = strings.Trim(name[:idx], ".-")
			break
		}
	}

	rank := 0
	switch name {
	case "a", "alpha":
		rank = 1
	case "b", "beta":
		rank = 2
	case "rc":
		rank = 3
	}
	return []int{-1, rank, num}
}

// compareVersions 比较两个版本号数组
// 返回: -1 表示 v1 < v2, 0 表示 v1 = v2, 1 表示 v1 > v2
func compareVersions(v1, v2 []int) int {
	maxLen := len(v1)
	if len(v2) > maxLen {
		maxLen = len(v2)
	}

	for i := 0; i < maxLen; i++ {
		num1 := 0
		if i < len(v1) {
			num1 = v1[i]
		}

		num2 := 0
		if i < len(v2) {
			num2 = v2[i]
		}

		if num1 > num2 {
			return 1
		} else if num1 < num2 {
			return -1
		}
	}

	return 0
}

// SimpleMatch all substring in the give text string.
//
// Difference the ContainsAll:
//
//   - start with ^ for exclude contains check.
//   - end with $ for the check end with keyword.
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

// QuickMatch check for a string. pattern can be a substring.
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
// Use on a pattern:
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
