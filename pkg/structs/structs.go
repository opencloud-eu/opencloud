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

func Map[E any, R any](source []E, indexer func(E) R) []R {
	if source == nil {
		var zero []R
		return zero
	}
	result := make([]R, len(source))
	for i, e := range source {
		result[i] = indexer(e)
	}
	return result
}
