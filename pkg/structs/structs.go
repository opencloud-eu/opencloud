// Package structs provides some utility functions for dealing with structs.
package structs

import (
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
