package arrutil

import (
	"reflect"

	"github.com/gookit/goutil/comdef"
)

// Reverse any T slice.
//
// eg: []string{"site", "user", "info", "0"} -> []string{"0", "info", "user", "site"}
func Reverse[T any](ls []T) {
	ln := len(ls)
	for i := 0; i < ln/2; i++ {
		li := ln - i - 1
		ls[i], ls[li] = ls[li], ls[i]
	}
}

// Remove give element from slice []T.
//
// eg: []string{"site", "user", "info", "0"} -> []string{"site", "user", "info"}
func Remove[T comdef.Compared](ls []T, val T) []T {
	return Filter(ls, func(el T) bool {
		return el != val
	})
}

// Filter given slice, default will filter zero value.
//
// Usage:
//
//	// output: [a, b]
//	ss := arrutil.Filter([]string{"a", "", "b", ""})
func Filter[T any](ls []T, filter ...comdef.MatchFunc[T]) []T {
	var fn comdef.MatchFunc[T]
	if len(filter) > 0 && filter[0] != nil {
		fn = filter[0]
	} else {
		fn = func(el T) bool {
			return !reflect.ValueOf(el).IsZero()
		}
	}

	newLs := make([]T, 0, len(ls))
	for _, el := range ls {
		if fn(el) {
			newLs = append(newLs, el)
		}
	}
	return newLs
}

// MapFn map handle function type.
type MapFn[T any, V any] func(input T) (target V, find bool)

// Map a list to new list
//
// eg: mapping [object0{},object1{},...] to flatten list [object0.someKey, object1.someKey, ...]
func Map[T any, V any](list []T, mapFn MapFn[T, V]) []V {
	flatArr := make([]V, 0, len(list))

	for _, obj := range list {
		if target, ok := mapFn(obj); ok {
			flatArr = append(flatArr, target)
		}
	}
	return flatArr
}

// Column alias of Map func
func Column[T any, V any](list []T, mapFn func(obj T) (val V, find bool)) []V {
	return Map(list, mapFn)
}

// Unique value in the given slice data.
func Unique[T comdef.NumberOrString](list []T) []T {
	if len(list) < 2 {
		return list
	}

	valMap := make(map[T]struct{}, len(list))
	uniArr := make([]T, 0, len(list))

	for _, t := range list {
		if _, ok := valMap[t]; !ok {
			valMap[t] = struct{}{}
			uniArr = append(uniArr, t)
		}
	}
	return uniArr
}

// IndexOf value in given slice.
func IndexOf[T comdef.NumberOrString](val T, list []T) int {
	for i, v := range list {
		if v == val {
			return i
		}
	}
	return -1
}

// FirstOr get first value of slice, if slice is empty, return the default value.
func FirstOr[T any](list []T, defVal ...T) T {
	if len(list) > 0 {
		return list[0]
	}

	if len(defVal) > 0 {
		return defVal[0]
	}
	var zero T
	return zero
}
