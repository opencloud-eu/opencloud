package errorx

import (
	"errors"
	"fmt"
)

// E new a raw go error. alias of errors.New()
func E(msg string) error { return errors.New(msg) }

// Err new a raw go error. alias of errors.New()
func Err(msg string) error { return errors.New(msg) }

// Raw new a raw go error. alias of errors.New()
func Raw(msg string) error { return errors.New(msg) }

// Ef new a raw go error. alias of fmt.Errorf
func Ef(tpl string, vars ...any) error { return fmt.Errorf(tpl, vars...) }

// Errf new a raw go error. alias of fmt.Errorf
func Errf(tpl string, vars ...any) error { return fmt.Errorf(tpl, vars...) }

// Rf new a raw go error. alias of fmt.Errorf
func Rf(tpl string, vs ...any) error { return fmt.Errorf(tpl, vs...) }

// Rawf new a raw go error. alias of fmt.Errorf
func Rawf(tpl string, vs ...any) error { return fmt.Errorf(tpl, vs...) }

/*************************************************************
 * helper func for error
 *************************************************************/

// Cause returns the first cause error by call err.Cause().
// Otherwise, will returns current error.
func Cause(err error) error {
	if err == nil {
		return nil
	}

	if err, ok := err.(Causer); ok {
		return err.Cause()
	}
	return err
}

// Unwrap returns previous error by call err.Unwrap().
// Otherwise, will returns nil.
func Unwrap(err error) error {
	if err == nil {
		return nil
	}

	if err, ok := err.(Unwrapper); ok {
		return err.Unwrap()
	}
	return nil
}

// Previous alias of Unwrap()
func Previous(err error) error { return Unwrap(err) }

// IsErrorX check
func IsErrorX(err error) (ok bool) {
	_, ok = err.(*ErrorX)
	return
}

// ToErrorX convert check. like errors.As()
func ToErrorX(err error) (ex *ErrorX, ok bool) {
	ex, ok = err.(*ErrorX)
	return
}

// MustEX convert error to *ErrorX, panic if err check failed.
func MustEX(err error) *ErrorX {
	ex, ok := err.(*ErrorX)
	if !ok {
		panic("errorx: error is not *ErrorX")
	}
	return ex
}

// Has contains target error, or err is eq target.
// alias of errors.Is()
func Has(err, target error) bool {
	return errors.Is(err, target)
}

// Is alias of errors.Is()
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// To try convert err to target, returns is result.
//
// NOTICE: target must be ptr and not nil. alias of errors.As()
//
// Usage:
//
//	var ex *errorx.ErrorX
//	err := doSomething()
//	if errorx.To(err, &ex) {
//		fmt.Println(ex.GoString())
//	}
func To(err error, target any) bool {
	return errors.As(err, target)
}

// As same of the To(), alias of errors.As()
//
// NOTICE: target must be ptr and not nil
func As(err error, target any) bool {
	return errors.As(err, target)
}
