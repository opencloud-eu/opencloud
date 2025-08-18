// Package syncs provides synchronization primitives util functions.
package syncs

import "sync"

// WaitGroup is a wrapper of sync.WaitGroup.
type WaitGroup struct {
	sync.WaitGroup
}

// Go runs the given function in a new goroutine. will auto call Add and Done.
func (wg *WaitGroup) Go(fn func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		fn()
	}()
}
