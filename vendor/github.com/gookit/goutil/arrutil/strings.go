package arrutil

import (
	"strconv"
	"strings"

	"github.com/gookit/goutil/comdef"
)

// StringsToAnys convert []string to []any
func StringsToAnys(ss []string) []any {
	args := make([]any, len(ss))
	for i, s := range ss {
		args[i] = s
	}
	return args
}

// StringsToSlice convert []string to []any. alias of StringsToAnys()
func StringsToSlice(ss []string) []any {
	return StringsToAnys(ss)
}

// StringsAsInts convert and ignore error
func StringsAsInts(ss []string) []int {
	ints, _ := StringsTryInts(ss)
	return ints
}

// StringsToInts string slice to int slice
func StringsToInts(ss []string) (ints []int, err error) {
	return StringsTryInts(ss)
}

// StringsTryInts string slice to int slice
func StringsTryInts(ss []string) (ints []int, err error) {
	for _, str := range ss {
		iVal, err := strconv.Atoi(str)
		if err != nil {
			return nil, err
		}

		ints = append(ints, iVal)
	}
	return
}

// StringsUnique unique string slice
func StringsUnique(ss []string) []string {
	if len(ss) == 0 {
		return ss
	}

	var unique []string
	for _, s := range ss {
		if !StringsContains(unique, s) {
			unique = append(unique, s)
		}
	}
	return unique
}

// StringsContains check string slice contains string
func StringsContains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// StringsRemove value form a string slice
func StringsRemove(ss []string, s string) []string {
	return StringsFilter(ss, func(el string) bool {
		return s != el
	})
}

// StringsFilter given strings, default will filter emtpy string.
//
// Usage:
//
//	// output: [a, b]
//	ss := arrutil.StringsFilter([]string{"a", "", "b", ""})
func StringsFilter(ss []string, filter ...comdef.StringMatchFunc) []string {
	var fn comdef.StringMatchFunc
	if len(filter) > 0 && filter[0] != nil {
		fn = filter[0]
	} else {
		fn = func(s string) bool {
			return s != ""
		}
	}

	ns := make([]string, 0, len(ss))
	for _, s := range ss {
		if fn(s) {
			ns = append(ns, s)
		}
	}
	return ns
}

// StringsMap handle each string item, map to new strings
func StringsMap(ss []string, mapFn func(s string) string) []string {
	ns := make([]string, 0, len(ss))
	for _, s := range ss {
		ns = append(ns, mapFn(s))
	}
	return ns
}

// TrimStrings trim string slice item.
//
// Usage:
//
//	// output: [a, b, c]
//	ss := arrutil.TrimStrings([]string{",a", "b.", ",.c,"}, ",.")
func TrimStrings(ss []string, cutSet ...string) []string {
	cutSetLn := len(cutSet)
	hasCutSet := cutSetLn > 0 && cutSet[0] != ""

	var trimSet string
	if hasCutSet {
		trimSet = cutSet[0]
	}
	if cutSetLn > 1 {
		trimSet = strings.Join(cutSet, "")
	}

	ns := make([]string, 0, len(ss))
	for _, str := range ss {
		if hasCutSet {
			ns = append(ns, strings.Trim(str, trimSet))
		} else {
			ns = append(ns, strings.TrimSpace(str))
		}
	}
	return ns
}
