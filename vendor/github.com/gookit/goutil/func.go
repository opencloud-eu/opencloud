package goutil

import "fmt"

// Go is a basic promise implementation: it wraps calls a function in a goroutine
// and returns a channel which will later return the function's return value.
func Go(f func() error) error {
	ch := make(chan error)
	go func() {
		ch <- f()
	}()
	return <-ch
}

// ErrFunc type
type ErrFunc func() error

// CallOn call func on condition is true
func CallOn(cond bool, fn ErrFunc) error {
	if cond {
		return fn()
	}
	return nil
}

// IfElseFn call okFunc() on condition is true, else call elseFn()
func IfElseFn(cond bool, okFn, elseFn ErrFunc) error {
	if cond {
		return okFn()
	}
	return elseFn()
}

// CallOrElse call okFunc() on condition is true, else call elseFn()
func CallOrElse(cond bool, okFn, elseFn ErrFunc) error {
	if cond {
		return okFn()
	}
	return elseFn()
}

// SafeRun sync run a func. If the func panics, the panic value is returned as an error.
func SafeRun(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", r)
			}
		}
	}()
	fn()
	return err
}

// SafeRunWithError sync run a func with error.
// If the func panics, the panic value is returned as an error.
func SafeRunWithError(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", r)
			}
		}
	}()
	return fn()
}

// SafeGo async run a func.
// If the func panics, the panic value will be handle by errHandler.
func SafeGo(fn func(), errHandler func(error)) {
	go func() {
		if err := SafeRun(fn); err != nil {
			errHandler(err)
		}
	}()
}

// SafeGoWithError async run a func with error.
// If the func panics, the panic value will be handle by errHandler.
func SafeGoWithError(fn func() error, errHandler func(error)) {
	go func() {
		if err := SafeRunWithError(fn); err != nil {
			errHandler(err)
		}
	}()
}
