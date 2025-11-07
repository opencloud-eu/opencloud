// Package structs provides some utility functions for dealing with structs.
package structs

import (
	"maps"
	"slices"

	orderedmap "github.com/wk8/go-ordered-map"
)

// CopyOrZeroValue returns a copy of s if s is not nil otherwise the zero value of T will be returned.
func CopyOrZeroValue[T any](s *T) *T {
	cp := new(T)
	if s != nil {
		*cp = *s
	}
	return cp
}

// Returns a copy of an array with a unique set of elements.
//
// Element order is retained.
func Uniq[T comparable](source []T) []T {
	m := orderedmap.New()
	for _, v := range source {
		m.Set(v, true)
	}
	set := make([]T, m.Len())
	i := 0
	for pair := m.Oldest(); pair != nil; pair = pair.Next() {
		set[i] = pair.Key.(T)
		i++
	}
	return set
}

func Keys[K comparable, V any](source map[K]V) []K {
	if source == nil {
		var zero []K
		return zero
	}
	return slices.Collect(maps.Keys(source))
}

func Index[K comparable, V any](source []V, indexer func(V) K) map[K]V {
	if source == nil {
		var zero map[K]V
		return zero
	}
	result := map[K]V{}
	for _, v := range source {
		k := indexer(v)
		result[k] = v
	}
	return result
}

func Map[E any, R any](source []E, mapper func(E) R) []R {
	if source == nil {
		var zero []R
		return zero
	}
	result := make([]R, len(source))
	for i, e := range source {
		result[i] = mapper(e)
	}
	return result
}

func MapValues[K comparable, S any, T any](m map[K]S, mapper func(S) T) map[K]T {
	r := make(map[K]T, len(m))
	for k, s := range m {
		r[k] = mapper(s)
	}
	return r
}

func MapValues2[K comparable, S any, T any](m map[K]S, mapper func(K, S) T) map[K]T {
	r := make(map[K]T, len(m))
	for k, s := range m {
		r[k] = mapper(k, s)
	}
	return r
}

func MapKeys[S comparable, T comparable, V any](m map[S]V, mapper func(S) T) map[T]V {
	r := make(map[T]V, len(m))
	for s, v := range m {
		r[mapper(s)] = v
	}
	return r
}

func MapKeys2[S comparable, T comparable, V any](m map[S]V, mapper func(S, V) T) map[T]V {
	r := make(map[T]V, len(m))
	for s, v := range m {
		r[mapper(s, v)] = v
	}
	return r
}

func ToBoolMap[E comparable](source []E) map[E]bool {
	m := make(map[E]bool, len(source))
	for _, v := range source {
		m[v] = true
	}
	return m
}

func ToIntMap[E comparable](source []E) map[E]int {
	m := make(map[E]int, len(source))
	for _, v := range source {
		if e, ok := m[v]; ok {
			m[v] = e + 1
		} else {
			m[v] = 1
		}
	}
	return m
}

func MapN[E any, R any](source []E, indexer func(E) *R) []R {
	if source == nil {
		var zero []R
		return zero
	}
	result := []R{}
	for _, e := range source {
		opt := indexer(e)
		if opt != nil {
			result = append(result, *opt)
		}
	}
	return result
}

// Check whether two slices contain the same elements, ignoring order.
func SameSlices[E comparable](x, y []E) bool {
	// https://stackoverflow.com/a/36000696
	if len(x) != len(y) {
		return false
	}
	// create a map of string -> int
	diff := make(map[E]int, len(x))
	for _, _x := range x {
		// 0 value for int is 0, so just increment a counter for the string
		diff[_x]++
	}
	for _, _y := range y {
		// If the string _y is not in diff bail out early
		if _, ok := diff[_y]; !ok {
			return false
		}
		diff[_y]--
		if diff[_y] == 0 {
			delete(diff, _y)
		}
	}
	return len(diff) == 0
}

func Missing[E comparable](expected, actual []E) []E {
	missing := []E{}
	actualIndex := ToBoolMap(actual)
	for _, e := range expected {
		if _, ok := actualIndex[e]; !ok {
			missing = append(missing, e)
		}
	}
	return missing
}

func FirstKey[K comparable, V any](m map[K]V) *K {
	for k := range m {
		return &k
	}
	return nil
}
