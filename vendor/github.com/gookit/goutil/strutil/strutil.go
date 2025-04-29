// Package strutil provide some string,char,byte util functions
package strutil

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gookit/goutil/comdef"
)

// OrCond return s1 on cond is True, OR return s2.
// Like: cond ? s1 : s2
func OrCond(cond bool, s1, s2 string) string {
	if cond {
		return s1
	}
	return s2
}

// BlankOr return default value on val is blank, otherwise return val
func BlankOr(val, defVal string) string {
	val = strings.TrimSpace(val)
	if val != "" {
		return val
	}
	return defVal
}

// ZeroOr return default value on val is zero, otherwise return val. same of OrElse()
func ZeroOr[T ~string](val, defVal T) T {
	if val != "" {
		return val
	}
	return defVal
}

// ErrorOr return default value on err is not nil, otherwise return s
// func ErrorOr(s string, err error, defVal string) string {
// 	if err != nil {
// 		return defVal
// 	}
// 	return s
// }

// OrElse return default value on s is empty, otherwise return s
func OrElse(s, orVal string) string {
	if s != "" {
		return s
	}
	return orVal
}

// OrElseNilSafe return default value on s is nil, otherwise return s
func OrElseNilSafe(s *string, orVal string) string {
	if s == nil || *s == "" {
		return orVal
	}
	return *s
}

// OrHandle return fn(s) on s is not empty.
func OrHandle(s string, fn comdef.StringHandleFunc) string {
	if s != "" {
		return fn(s)
	}
	return s
}

// Valid return first not empty element.
func Valid(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// Replaces replace multi strings
//
//	pairs: {old1: new1, old2: new2, ...}
//
// Can also use:
//
//	strings.NewReplacer("old1", "new1", "old2", "new2").Replace(str)
func Replaces(str string, pairs map[string]string) string {
	return NewReplacer(pairs).Replace(str)
}

// NewReplacer instance
func NewReplacer(pairs map[string]string) *strings.Replacer {
	ss := make([]string, len(pairs)*2)
	for old, newVal := range pairs {
		ss = append(ss, old, newVal)
	}
	return strings.NewReplacer(ss...)
}

// WrapTag for given string.
func WrapTag(s, tag string) string {
	if s == "" {
		return s
	}
	return fmt.Sprintf("<%s>%s</%s>", tag, s, tag)
}

// SubstrCount returns the number of times the substr substring occurs in the s string.
// Actually, it comes from strings.Count().
//
//   - s The string to search in
//   - substr The substring to search for
//   - params[0] The offset where to start counting.
//   - params[1] The maximum length after the specified offset to search for the substring.
func SubstrCount(s, substr string, params ...uint64) (int, error) {
	larg := len(params)
	hasArgs := larg != 0
	if hasArgs && larg > 2 {
		return 0, errors.New("too many parameters")
	}
	if !hasArgs {
		return strings.Count(s, substr), nil
	}

	strlen := len(s)
	offset := 0
	end := strlen

	if hasArgs {
		offset = int(params[0])
		if larg == 2 {
			length := int(params[1])
			end = offset + length
		}
		if end > strlen {
			end = strlen
		}
	}

	s = string([]rune(s)[offset:end])
	return strings.Count(s, substr), nil
}
