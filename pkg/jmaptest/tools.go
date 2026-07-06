package jmaptest

import (
	"net"
	"sync"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

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

var freeLocalhostPortSync = sync.Mutex{}

func FreeLocalhostPort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}

	freeLocalhostPortSync.Lock()
	defer freeLocalhostPortSync.Unlock()
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}
