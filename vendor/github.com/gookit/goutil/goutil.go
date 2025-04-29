// Package goutil 💪 Useful utils for Go: int, string, array/slice, map, error, time, format, CLI, ENV, filesystem,
// system, testing, debug and more.
package goutil

import (
	"fmt"

	"github.com/gookit/goutil/basefn"
	"github.com/gookit/goutil/goinfo"
	"github.com/gookit/goutil/structs"
)

// Value alias of structs.Value
type Value = structs.Value

// Panicf format panic message use fmt.Sprintf
func Panicf(format string, v ...any) {
	panic(fmt.Sprintf(format, v...))
}

// PanicIf if cond = true, panics with error message
func PanicIf(cond bool, fmtAndArgs ...any) {
	basefn.PanicIf(cond, fmtAndArgs...)
}

// PanicErr if error is not empty, will panic.
// Alias of basefn.PanicErr()
func PanicErr(err error) {
	if err != nil {
		panic(err)
	}
}

// PanicIfErr if error is not empty, will panic.
// Alias of basefn.PanicErr()
func PanicIfErr(err error) { PanicErr(err) }

// MustOK if error is not empty, will panic.
// Alias of basefn.MustOK()
func MustOK(err error) { PanicErr(err) }

// MustIgnore for return like (v, error). Ignore return v and will panic on error.
//
// Useful for io, file operation func: (n int, err error)
//
// Usage:
//
//	// old
//	_, err := fn()
//	if err != nil {
//		panic(err)
//	}
//
//	// new
//	goutil.MustIgnore(fn())
func MustIgnore(_ any, err error) { PanicErr(err) }

// Must return like (v, error). will panic on error, otherwise return v.
//
// Usage:
//
//	// old
//	v, err := fn()
//	if err != nil {
//		panic(err)
//	}
//
//	// new
//	v := goutil.Must(fn())
func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// FuncName get func name
func FuncName(f any) string {
	return goinfo.FuncName(f)
}

// PkgName get current package name. alias of goinfo.PkgName()
//
// Usage:
//
//	funcName := goutil.FuncName(fn)
//	pgkName := goutil.PkgName(funcName)
func PkgName(funcName string) string {
	return goinfo.PkgName(funcName)
}

// ErrOnFail return input error on cond is false, otherwise return nil
func ErrOnFail(cond bool, err error) error {
	return OrError(cond, err)
}

// OrError return input error on cond is false, otherwise return nil
func OrError(cond bool, err error) error {
	if !cond {
		return err
	}
	return nil
}

// OrValue get
func OrValue[T any](cond bool, okVal, elVal T) T {
	if cond {
		return okVal
	}
	return elVal
}

// OrReturn call okFunc() on condition is true, else call elseFn()
func OrReturn[T any](cond bool, okFn, elseFn func() T) T {
	if cond {
		return okFn()
	}
	return elseFn()
}
