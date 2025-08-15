// Package goutil 💪 Useful utils for Go: byte, int, string, array/slice, map, struct, reflect, error, time, format, CLI, ENV, filesystem,
// system, testing, debug and more.
package goutil

import (
	"fmt"
	"reflect"

	"github.com/gookit/goutil/internal/checkfn"
	"github.com/gookit/goutil/reflects"
	"github.com/gookit/goutil/structs"
	"github.com/gookit/goutil/x/basefn"
	"github.com/gookit/goutil/x/goinfo"
)

// Value alias of structs.Value
type Value = structs.Value

// Panicf format panic message use fmt.Sprintf
func Panicf(format string, v ...any) {
	panic(fmt.Sprintf(format, v...))
}

// PanicIf if cond = true, panics with an error message
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

// OrValue get. like: if cond { okVal } else { elVal }
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

//
// ------------------------- check functions -------------------------
//

// IsNil value check
func IsNil(v any) bool {
	if v == nil {
		return true
	}
	return reflects.IsNil(reflect.ValueOf(v))
}

// IsZero value check, alias of the IsEmpty()
var IsZero = IsEmpty

// IsEmpty value check
func IsEmpty(v any) bool {
	if v == nil {
		return true
	}
	return reflects.IsEmpty(reflect.ValueOf(v))
}

// IsZeroReal Alias of the IsEmptyReal()
var IsZeroReal = IsEmptyReal

// IsEmptyReal checks for empty given value and also real empty value if the passed value is a pointer
func IsEmptyReal(v any) bool {
	if v == nil {
		return true
	}
	return reflects.IsEmptyReal(reflect.ValueOf(v))
}

// IsFunc value
func IsFunc(val any) bool {
	if val == nil {
		return false
	}
	return reflect.TypeOf(val).Kind() == reflect.Func
}

// IsEqual determines if two objects are considered equal.
//
// TIP: cannot compare a function type
func IsEqual(src, dst any) bool {
	if src == nil || dst == nil {
		return src == dst
	}

	// cannot compare a function type
	if IsFunc(src) || IsFunc(dst) {
		return false
	}
	return reflects.IsEqual(src, dst)
}

// Contains try loop over the data check if the data includes the element.
// alias of the IsContains
//
// TIP: only support types: string, map, array, slice
//
//	map         - check key exists
//	string 	    - check sub-string exists
//	array,slice - check sub-element exists
func Contains(data, elem any) bool {
	_, found := checkfn.Contains(data, elem)
	return found
}

// IsContains try loop over the data check if the data includes the element.
//
// TIP: only support types: string, map, array, slice
//
//	map         - check key exists
//	string 	    - check sub-string exists
//	array,slice - check sub-element exists
func IsContains(data, elem any) bool {
	_, found := checkfn.Contains(data, elem)
	return found
}

//
// ------------------------- goinfo functions -------------------------
//

// FuncName get func name
func FuncName(f any) string {
	return goinfo.FuncName(f)
}

// PkgName get the current package name. alias of goinfo.PkgName()
//
// Usage:
//
//	funcName := goutil.FuncName(fn)
//	pgkName := goutil.PkgName(funcName)
func PkgName(funcName string) string {
	return goinfo.PkgName(funcName)
}

