package goutil

import (
	"reflect"

	"github.com/gookit/goutil/internal/checkfn"
	"github.com/gookit/goutil/reflects"
)

// IsNil value check
func IsNil(v any) bool {
	if v == nil {
		return true
	}
	return reflects.IsNil(reflect.ValueOf(v))
}

// IsZero value check, alias of the IsEmpty()
var IsZero = IsEmpty

// IsEmpty value check
func IsEmpty(v any) bool {
	if v == nil {
		return true
	}
	return reflects.IsEmpty(reflect.ValueOf(v))
}

// Alias of the IsEmptyReal()
var IsZeroReal = IsEmptyReal

// IsEmptyReal checks for empty given value and also real empty value if the passed value is a pointer
func IsEmptyReal(v any) bool {
	if v == nil {
		return true
	}
	return reflects.IsEmptyReal(reflect.ValueOf(v))
}

// IsFunc value
func IsFunc(val any) bool {
	if val == nil {
		return false
	}
	return reflect.TypeOf(val).Kind() == reflect.Func
}

// IsEqual determines if two objects are considered equal.
//
// TIP: cannot compare function type
func IsEqual(src, dst any) bool {
	if src == nil || dst == nil {
		return src == dst
	}

	// cannot compare function type
	if IsFunc(src) || IsFunc(dst) {
		return false
	}
	return reflects.IsEqual(src, dst)
}

// Contains try loop over the data check if the data includes the element.
// alias of the IsContains
//
// TIP: only support types: string, map, array, slice
//
//	map         - check key exists
//	string 	    - check sub-string exists
//	array,slice - check sub-element exists
func Contains(data, elem any) bool {
	_, found := checkfn.Contains(data, elem)
	return found
}

// IsContains try loop over the data check if the data includes the element.
//
// TIP: only support types: string, map, array, slice
//
//	map         - check key exists
//	string 	    - check sub-string exists
//	array,slice - check sub-element exists
func IsContains(data, elem any) bool {
	_, found := checkfn.Contains(data, elem)
	return found
}
