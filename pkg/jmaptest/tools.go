package jmaptest

import "github.com/opencloud-eu/opencloud/pkg/jmap"

var (
	truep  = ptr(true)
	falsep = ptr(false)
)

// TODO remove and replace with calls to new() when upgrading to Go 1.26
func ptr[T any | int | uint | bool | string](t T) *T {
	return &t
}

func list[T jmap.Foo, GETRESP jmap.GetResponse[T]](r GETRESP) []T { return r.GetList() }
func getid[T jmap.Idable](r T) string                             { return r.GetId() }

func uintPtr[T int | uint](i T) *uint {
	return ptr(uint(i))
}

func firstKey[K comparable, V any](m map[K]V) (K, bool) {
	for k := range m {
		return k, true
	}
	var zero K
	return zero, false
}
