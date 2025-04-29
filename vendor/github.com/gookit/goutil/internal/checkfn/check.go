package checkfn

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// IsNil value check
func IsNil(v any) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

// IsSimpleKind kind in: string, bool, intX, uintX, floatX
func IsSimpleKind(k reflect.Kind) bool {
	if reflect.String == k {
		return true
	}
	return k > reflect.Invalid && k <= reflect.Float64
}

// IsEqual determines if two objects are considered equal.
//
// TIP: cannot compare function type
func IsEqual(src, dst any) bool {
	if src == nil || dst == nil {
		return src == dst
	}

	bs1, ok := src.([]byte)
	if !ok {
		return reflect.DeepEqual(src, dst)
	}

	bs2, ok := dst.([]byte)
	if !ok {
		return false
	}

	if bs1 == nil || bs2 == nil {
		return bs1 == nil && bs2 == nil
	}
	return bytes.Equal(bs1, bs2)
}

// Contains try loop over the data check if the data includes the element.
//
// data allow types: string, map, array, slice
//
//	map         - check key exists
//	string      - check sub-string exists
//	array,slice - check sub-element exists
//
// Returns:
//   - valid: data is valid
//   - found: element was found
//
// return (false, false) if impossible.
// return (true, false) if element was not found.
// return (true, true) if element was found.
func Contains(data, elem any) (valid, found bool) {
	if data == nil {
		return false, false
	}

	dataRv := reflect.ValueOf(data)
	dataRt := reflect.TypeOf(data)
	dataKind := dataRt.Kind()

	// string
	if dataKind == reflect.String {
		return true, strings.Contains(dataRv.String(), fmt.Sprint(elem))
	}

	// map
	if dataKind == reflect.Map {
		mapKeys := dataRv.MapKeys()
		for i := 0; i < len(mapKeys); i++ {
			if IsEqual(mapKeys[i].Interface(), elem) {
				return true, true
			}
		}
		return true, false
	}

	// array, slice - other return false
	if dataKind != reflect.Slice && dataKind != reflect.Array {
		return false, false
	}

	for i := 0; i < dataRv.Len(); i++ {
		if IsEqual(dataRv.Index(i).Interface(), elem) {
			return true, true
		}
	}
	return true, false
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

// check is number: int or float
var numReg = regexp.MustCompile(`^[-+]?\d*\.?\d+$`)

// IsNumeric returns true if the given string is a numeric, otherwise false.
func IsNumeric(s string) bool { return numReg.MatchString(s) }
