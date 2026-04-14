// Package structs provides some utility functions for dealing with structs.
package structs

import (
	"iter"
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

// Create an iterator from a slice, iterating over every single element of the slice, in order.
func Seq[T any](s []T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, elem := range s {
			if !yield(elem) {
				return
			}
		}
	}
}

// Create an iterator from a slice that yields the position and the value,
// iterating over every single element of the slice, in order.
func Seq2[T any](s []T) iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		for i, elem := range s {
			if !yield(i, elem) {
				return
			}
		}
	}
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

// Returns a slice containing the keys of the map.
func Keys[K comparable, V any](source map[K]V) []K {
	if source == nil {
		var zero []K
		return zero
	}
	return slices.Collect(maps.Keys(source))
}

// Creates a map from a slice, using the indexer func to determine the key for each value,
// and the value being as-is.
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

// Creates a slice from a slice, putting each value from the source slice through the
// mapper function to determine the value to store into the resulting slice.
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

// Wraps an iterator with a transformer function.
func MapSeq[A, B any](it iter.Seq[A], transformer func(A) B) iter.Seq[B] {
	return func(yield func(b B) bool) {
		for v := range it {
			t := transformer(v)
			if !yield(t) {
				return
			}
		}
	}
}

// Creates a slice from a slice, putting each value from the source slice through the
// mapper function to determine the value to store into the resulting slice, but skipping
// the result of the mapper function if it returns nil.
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

// Creates a slice from a slice, putting each value from the source slice through the
// mapper function to determine the value to store into the resulting slice, but skipping
// the result of the mapper function if it returns false as its second return value.
func MapO[E any, R any](source []E, indexer func(E) (R, bool)) []R {
	if source == nil {
		var zero []R
		return zero
	}
	result := []R{}
	for _, e := range source {
		if value, keep := indexer(e); keep {
			result = append(result, value)
		}
	}
	return result
}

// Creates a map from a map, keeping each key as-is, and using the mapper
// function to determine the value to store into the resulting map.
func MapValues[K comparable, S any, T any](m map[K]S, mapper func(S) T) map[K]T {
	r := make(map[K]T, len(m))
	for k, s := range m {
		r[k] = mapper(s)
	}
	return r
}

// Creates a map from a map, keeping each key as-is, and using the mapper function
// that takes both the key and the value to determine the value to store into the resulting map.
func MapValues2[K comparable, S any, T any](m map[K]S, mapper func(K, S) T) map[K]T {
	r := make(map[K]T, len(m))
	for k, s := range m {
		r[k] = mapper(k, s)
	}
	return r
}

// Creates a map from a map, keeping each value as-is, and using the mapper
// function to determine the key to store into the resulting map.
func MapKeys[S comparable, T comparable, V any](m map[S]V, mapper func(S) T) map[T]V {
	r := make(map[T]V, len(m))
	for s, v := range m {
		r[mapper(s)] = v
	}
	return r
}

// Creates a map from a map, keeping each value as-is, and using the mapper function
// that takes both the key and the value to determine the key to store into the resulting map.
func MapKeys2[S comparable, T comparable, V any](m map[S]V, mapper func(S, V) T) map[T]V {
	r := make(map[T]V, len(m))
	for s, v := range m {
		r[mapper(s, v)] = v
	}
	return r
}

// Creates a map from a slice, using the mapper function to determine the key and value
// pair to use for each slice element in the resulting map.
func ToMap[E any, K comparable, V any](source []E, mapper func(E) (K, V)) map[K]V {
	m := map[K]V{}
	for _, e := range source {
		k, v := mapper(e)
		m[k] = v
	}
	return m
}

// Creates a map of booleans, using the values of the source slice as keys in the
// resulting map.
func ToBoolMap[E comparable](source []E) map[E]bool {
	m := make(map[E]bool, len(source))
	for _, v := range source {
		m[v] = true
	}
	return m
}

// Creates a map of ints, using the values of the source slice as keys in the
// resulting map, and storing the number of occurences of every given value
// as the int value in the map.
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

// Concatenate the elements of multiple slices into a single slice.
//
// Element order is preserved.
func Concat[E any](arys ...[]E) []E {
	l := 0
	for _, ary := range arys {
		l += len(ary)
	}
	r := make([]E, l)

	i := 0
	for _, ary := range arys {
		if ary != nil {
			i += copy(r[i:], ary)
		}
	}
	return r
}

// Create a new slice from a slice, determining whether each element should
// be added to the new slice by passing it to the predicate function.
//
// When the predicate function returns true, the element is stored in the
// new slice.
// When the predicate functoin returns false, the element is skipped and not
// stored in the new slice.
func Filter[E any](s []E, predicate func(E) bool) []E {
	if s == nil {
		var zero []E
		return zero
	}
	r := []E{}
	for _, e := range s {
		if predicate(e) {
			r = append(r, e)
		}
	}
	return r
}

// Wrap an iterator with a conditional/filtering predicate function.
func FilterSeq[T any](it iter.Seq[T], predicate func(T) bool) iter.Seq[T] {
	return func(yield func(s T) bool) {
		for v := range it {
			b := predicate(v)
			if b {
				if !yield(v) {
					return
				}
			}
		}
	}
}
