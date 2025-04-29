package mathutil

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gookit/goutil/comdef"
	"github.com/gookit/goutil/internal/checkfn"
	"github.com/gookit/goutil/internal/comfunc"
)

// ToIntFunc convert value to int
type ToIntFunc func(any) (int, error)

// ToInt64Func convert value to int64
type ToInt64Func func(any) (int64, error)

// ToUintFunc convert value to uint
type ToUintFunc func(any) (uint, error)

// ToUint64Func convert value to uint
type ToUint64Func func(any) (uint64, error)

// ToFloatFunc convert value to float
type ToFloatFunc func(any) (float64, error)

// ToTypeFunc convert value to defined type
type ToTypeFunc[T any] func(any) (T, error)

// ConvOption convert options
type ConvOption[T any] struct {
	// if ture: value is nil, will return convert error;
	// if false(default): value is nil, will convert to zero value
	NilAsFail bool
	// HandlePtr auto convert ptr type(int,float,string) value. eg: *int to int
	// 	- if true: will use real type try convert. default is false
	//	- NOTE: current T type's ptr is default support.
	HandlePtr bool
	// set custom fallback convert func for not supported type.
	UserConvFn ToTypeFunc[T]
}

// NewConvOption create a new ConvOption
func NewConvOption[T any](optFns ...ConvOptionFn[T]) *ConvOption[T] {
	opt := &ConvOption[T]{}
	opt.WithOption(optFns...)
	return opt
}

// WithOption set convert option
func (opt *ConvOption[T]) WithOption(optFns ...ConvOptionFn[T]) {
	for _, fn := range optFns {
		if fn != nil {
			fn(opt)
		}
	}
}

// ConvOptionFn convert option func
type ConvOptionFn[T any] func(opt *ConvOption[T])

// WithNilAsFail set ConvOption.NilAsFail option
//
// Example:
//
//	ToIntWithFunc(val, mathutil.WithNilAsFail[int])
func WithNilAsFail[T any](opt *ConvOption[T]) {
	opt.NilAsFail = true
}

// WithHandlePtr set ConvOption.HandlePtr option
func WithHandlePtr[T any](opt *ConvOption[T]) {
	opt.HandlePtr = true
}

// WithUserConvFn set ConvOption.UserConvFn option
func WithUserConvFn[T any](fn ToTypeFunc[T]) ConvOptionFn[T] {
	return func(opt *ConvOption[T]) {
		opt.UserConvFn = fn
	}
}

/*************************************************************
 * convert value to int
 *************************************************************/

// Int convert value to int
func Int(in any) (int, error) { return ToInt(in) }

// SafeInt convert value to int, will ignore error
func SafeInt(in any) int {
	val, _ := ToInt(in)
	return val
}

// QuietInt convert value to int, will ignore error
func QuietInt(in any) int { return SafeInt(in) }

// IntOrPanic convert value to int, will panic on error
func IntOrPanic(in any) int {
	val, err := ToInt(in)
	if err != nil {
		panic(err)
	}
	return val
}

// MustInt convert value to int, will panic on error
func MustInt(in any) int { return IntOrPanic(in) }

// IntOrDefault convert value to int, return defaultVal on failed
func IntOrDefault(in any, defVal int) int { return IntOr(in, defVal) }

// IntOr convert value to int, return defaultVal on failed
func IntOr(in any, defVal int) int {
	val, err := ToIntWith(in)
	if err != nil {
		return defVal
	}
	return val
}

// IntOrErr convert value to int, return error on failed
func IntOrErr(in any) (int, error) { return ToIntWith(in) }

// ToInt convert value to int, return error on failed
func ToInt(in any) (int, error) { return ToIntWith(in) }

// ToIntWith convert value to int, can with some option func.
//
// Example:
//
//	ToIntWithFunc(val, mathutil.WithNilAsFail, mathutil.WithUserConvFn(func(in any) (int, error) {
//	})
func ToIntWith(in any, optFns ...ConvOptionFn[int]) (iVal int, err error) {
	opt := NewConvOption[int](optFns...)
	if !opt.NilAsFail && in == nil {
		return 0, nil
	}

	switch tVal := in.(type) {
	case int:
		iVal = tVal
	case *int: // default support int ptr type
		iVal = *tVal
	case int8:
		iVal = int(tVal)
	case int16:
		iVal = int(tVal)
	case int32:
		iVal = int(tVal)
	case int64:
		if tVal > math.MaxInt32 {
			err = fmt.Errorf("value overflow int32. input: %v", tVal)
		} else {
			iVal = int(tVal)
		}
	case uint:
		if tVal > math.MaxInt32 {
			err = fmt.Errorf("value overflow int32. input: %v", tVal)
		} else {
			iVal = int(tVal)
		}
	case uint8:
		iVal = int(tVal)
	case uint16:
		iVal = int(tVal)
	case uint32:
		if tVal > math.MaxInt32 {
			err = fmt.Errorf("value overflow int32. input: %v", tVal)
		} else {
			iVal = int(tVal)
		}
	case uint64:
		if tVal > math.MaxInt32 {
			err = fmt.Errorf("value overflow int32. input: %v", tVal)
		} else {
			iVal = int(tVal)
		}
	case float32:
		iVal = int(tVal)
	case float64:
		iVal = int(tVal)
	case time.Duration:
		if tVal > math.MaxInt32 {
			err = fmt.Errorf("value overflow int32. input: %v", tVal)
		} else {
			iVal = int(tVal)
		}
	case string:
		sVal := strings.TrimSpace(tVal)
		iVal, err = strconv.Atoi(sVal)
		// handle the case where the string might be a float
		if err != nil && checkfn.IsNumeric(sVal) {
			var floatVal float64
			if floatVal, err = strconv.ParseFloat(sVal, 64); err == nil {
				iVal = int(math.Round(floatVal))
				err = nil
			}
		}
	case comdef.Int64able: // eg: json.Number
		var i64 int64
		if i64, err = tVal.Int64(); err == nil {
			if i64 > math.MaxInt32 {
				err = fmt.Errorf("value overflow int32. input: %v", tVal)
			} else {
				iVal = int(i64)
			}
		}
	default:
		if opt.HandlePtr {
			if rv := reflect.ValueOf(in); rv.Kind() == reflect.Pointer {
				rv = rv.Elem()
				if checkfn.IsSimpleKind(rv.Kind()) {
					return ToIntWith(rv.Interface(), optFns...)
				}
			}
		}

		if opt.UserConvFn != nil {
			return opt.UserConvFn(in)
		}
		err = comdef.ErrConvType
	}
	return
}

// StrInt convert.
func StrInt(s string) int {
	iVal, _ := strconv.Atoi(strings.TrimSpace(s))
	return iVal
}

// StrIntOr convert string to int, return default val on failed
func StrIntOr(s string, defVal int) int {
	iVal, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return defVal
	}
	return iVal
}

/*************************************************************
 * convert value to int64
 *************************************************************/

// Int64 convert value to int64, return error on failed
func Int64(in any) (int64, error) { return ToInt64(in) }

// SafeInt64 convert value to int64, will ignore error
func SafeInt64(in any) int64 {
	i64, _ := ToInt64With(in)
	return i64
}

// QuietInt64 convert value to int64, will ignore error
func QuietInt64(in any) int64 { return SafeInt64(in) }

// MustInt64 convert value to int64, will panic on error
func MustInt64(in any) int64 {
	i64, err := ToInt64With(in)
	if err != nil {
		panic(err)
	}
	return i64
}

// Int64OrDefault convert value to int64, return default val on failed
func Int64OrDefault(in any, defVal int64) int64 { return Int64Or(in, defVal) }

// Int64Or convert value to int64, return default val on failed
func Int64Or(in any, defVal int64) int64 {
	i64, err := ToInt64With(in)
	if err != nil {
		return defVal
	}
	return i64
}

// ToInt64 convert value to int64, return error on failed
func ToInt64(in any) (int64, error) { return ToInt64With(in) }

// Int64OrErr convert value to int64, return error on failed
func Int64OrErr(in any) (int64, error) { return ToInt64With(in) }

// ToInt64With try to convert value to int64. can with some option func, more see ConvOption.
func ToInt64With(in any, optFns ...ConvOptionFn[int64]) (i64 int64, err error) {
	opt := NewConvOption(optFns...)
	if !opt.NilAsFail && in == nil {
		return 0, nil
	}

	switch tVal := in.(type) {
	case string:
		sVal := strings.TrimSpace(tVal)
		i64, err = strconv.ParseInt(sVal, 10, 0)
		// handle the case where the string might be a float
		if err != nil && checkfn.IsNumeric(sVal) {
			var floatVal float64
			if floatVal, err = strconv.ParseFloat(sVal, 64); err == nil {
				i64 = int64(math.Round(floatVal))
				err = nil
			}
		}
	case int:
		i64 = int64(tVal)
	case int8:
		i64 = int64(tVal)
	case int16:
		i64 = int64(tVal)
	case int32:
		i64 = int64(tVal)
	case int64:
		i64 = tVal
	case *int64: // default support int64 ptr type
		i64 = *tVal
	case uint:
		i64 = int64(tVal)
	case uint8:
		i64 = int64(tVal)
	case uint16:
		i64 = int64(tVal)
	case uint32:
		i64 = int64(tVal)
	case uint64:
		i64 = int64(tVal)
	case float32:
		i64 = int64(tVal)
	case float64:
		i64 = int64(tVal)
	case time.Duration:
		i64 = int64(tVal)
	case comdef.Int64able: // eg: json.Number
		i64, err = tVal.Int64()
	default:
		if opt.HandlePtr {
			if rv := reflect.ValueOf(in); rv.Kind() == reflect.Pointer {
				rv = rv.Elem()
				if checkfn.IsSimpleKind(rv.Kind()) {
					return ToInt64With(rv.Interface(), optFns...)
				}
			}
		}

		if opt.UserConvFn != nil {
			i64, err = opt.UserConvFn(in)
		} else {
			err = comdef.ErrConvType
		}
	}
	return
}

/*************************************************************
 * convert value to uint
 *************************************************************/

// Uint convert any to uint, return error on failed
func Uint(in any) (uint, error) { return ToUint(in) }

// SafeUint convert any to uint, will ignore error
func SafeUint(in any) uint {
	val, _ := ToUint(in)
	return val
}

// QuietUint convert any to uint, will ignore error
func QuietUint(in any) uint { return SafeUint(in) }

// MustUint convert any to uint, will panic on error
func MustUint(in any) uint {
	val, err := ToUintWith(in)
	if err != nil {
		panic(err)
	}
	return val
}

// UintOrDefault convert any to uint, return default val on failed
func UintOrDefault(in any, defVal uint) uint { return UintOr(in, defVal) }

// UintOr convert any to uint, return default val on failed
func UintOr(in any, defVal uint) uint {
	val, err := ToUintWith(in)
	if err != nil {
		return defVal
	}
	return val
}

// UintOrErr convert value to uint, return error on failed
func UintOrErr(in any) (uint, error) { return ToUintWith(in) }

// ToUint convert value to uint, return error on failed
func ToUint(in any) (u64 uint, err error) { return ToUintWith(in) }

// ToUintWith try to convert value to uint. can with some option func, more see ConvOption.
func ToUintWith(in any, optFns ...ConvOptionFn[uint]) (uVal uint, err error) {
	opt := NewConvOption(optFns...)
	if !opt.NilAsFail && in == nil {
		return 0, nil
	}

	switch tVal := in.(type) {
	case int:
		uVal = uint(tVal)
	case int8:
		uVal = uint(tVal)
	case int16:
		uVal = uint(tVal)
	case int32:
		uVal = uint(tVal)
	case int64:
		uVal = uint(tVal)
	case uint:
		uVal = tVal
	case *uint: // default support uint ptr type
		uVal = *tVal
	case uint8:
		uVal = uint(tVal)
	case uint16:
		uVal = uint(tVal)
	case uint32:
		uVal = uint(tVal)
	case uint64:
		uVal = uint(tVal)
	case float32:
		uVal = uint(tVal)
	case float64:
		uVal = uint(tVal)
	case time.Duration:
		uVal = uint(tVal)
	case comdef.Int64able: // eg: json.Number
		var i64 int64
		i64, err = tVal.Int64()
		uVal = uint(i64)
	case string:
		var u64 uint64
		u64, err = strconv.ParseUint(strings.TrimSpace(tVal), 10, 0)
		uVal = uint(u64)
	default:
		if opt.HandlePtr {
			if rv := reflect.ValueOf(in); rv.Kind() == reflect.Pointer {
				rv = rv.Elem()
				if checkfn.IsSimpleKind(rv.Kind()) {
					return ToUintWith(rv.Interface(), optFns...)
				}
			}
		}

		if opt.UserConvFn != nil {
			uVal, err = opt.UserConvFn(in)
		} else {
			err = comdef.ErrConvType
		}
	}
	return
}

/*************************************************************
 * convert value to uint64
 *************************************************************/

// Uint64 convert any to uint64, return error on failed
func Uint64(in any) (uint64, error) { return ToUint64(in) }

// QuietUint64 convert any to uint64, will ignore error
func QuietUint64(in any) uint64 { return SafeUint64(in) }

// SafeUint64 convert any to uint64, will ignore error
func SafeUint64(in any) uint64 {
	val, _ := ToUint64(in)
	return val
}

// MustUint64 convert any to uint64, will panic on error
func MustUint64(in any) uint64 {
	val, err := ToUint64With(in)
	if err != nil {
		panic(err)
	}
	return val
}

// Uint64OrDefault convert any to uint64, return default val on failed
func Uint64OrDefault(in any, defVal uint64) uint64 { return Uint64Or(in, defVal) }

// Uint64Or convert any to uint64, return default val on failed
func Uint64Or(in any, defVal uint64) uint64 {
	val, err := ToUint64With(in)
	if err != nil {
		return defVal
	}
	return val
}

// Uint64OrErr convert value to uint64, return error on failed
func Uint64OrErr(in any) (uint64, error) { return ToUint64With(in) }

// ToUint64 convert value to uint64, return error on failed
func ToUint64(in any) (uint64, error) { return ToUint64With(in) }

// ToUint64With try to convert value to uint64. can with some option func, more see ConvOption.
func ToUint64With(in any, optFns ...ConvOptionFn[uint64]) (u64 uint64, err error) {
	opt := NewConvOption(optFns...)
	if !opt.NilAsFail && in == nil {
		return 0, nil
	}

	switch tVal := in.(type) {
	case int:
		u64 = uint64(tVal)
	case int8:
		u64 = uint64(tVal)
	case int16:
		u64 = uint64(tVal)
	case int32:
		u64 = uint64(tVal)
	case int64:
		u64 = uint64(tVal)
	case uint:
		u64 = uint64(tVal)
	case uint8:
		u64 = uint64(tVal)
	case uint16:
		u64 = uint64(tVal)
	case uint32:
		u64 = uint64(tVal)
	case uint64:
		u64 = tVal
	case *uint64: // default support uint64 ptr type
		u64 = *tVal
	case float32:
		u64 = uint64(tVal)
	case float64:
		u64 = uint64(tVal)
	case time.Duration:
		u64 = uint64(tVal)
	case comdef.Int64able: // eg: json.Number
		var i64 int64
		i64, err = tVal.Int64()
		u64 = uint64(i64)
	case string:
		u64, err = strconv.ParseUint(strings.TrimSpace(tVal), 10, 0)
	default:
		if opt.HandlePtr {
			if rv := reflect.ValueOf(in); rv.Kind() == reflect.Pointer {
				rv = rv.Elem()
				if checkfn.IsSimpleKind(rv.Kind()) {
					return ToUint64With(rv.Interface(), optFns...)
				}
			}
		}

		if opt.UserConvFn != nil {
			u64, err = opt.UserConvFn(in)
		} else {
			err = comdef.ErrConvType
		}
	}
	return
}

/*************************************************************
 * convert value to float64
 *************************************************************/

// QuietFloat convert value to float64, will ignore error. alias of SafeFloat
func QuietFloat(in any) float64 { return SafeFloat(in) }

// SafeFloat convert value to float64, will ignore error
func SafeFloat(in any) float64 {
	val, _ := ToFloatWith(in)
	return val
}

// FloatOrPanic convert value to float64, will panic on error
func FloatOrPanic(in any) float64 { return MustFloat(in) }

// MustFloat convert value to float64, will panic on error
func MustFloat(in any) float64 {
	val, err := ToFloatWith(in)
	if err != nil {
		panic(err)
	}
	return val
}

// FloatOrDefault convert value to float64, will return default value on error
func FloatOrDefault(in any, defVal float64) float64 { return FloatOr(in, defVal) }

// FloatOr convert value to float64, will return default value on error
func FloatOr(in any, defVal float64) float64 {
	val, err := ToFloatWith(in)
	if err != nil {
		return defVal
	}
	return val
}

// Float convert value to float64, return error on failed
func Float(in any) (float64, error) { return ToFloatWith(in) }

// FloatOrErr convert value to float64, return error on failed
func FloatOrErr(in any) (float64, error) { return ToFloatWith(in) }

// ToFloat convert value to float64, return error on failed
func ToFloat(in any) (float64, error) { return ToFloatWith(in) }

// ToFloatWith try to convert value to float64. can with some option func, more see ConvOption.
func ToFloatWith(in any, optFns ...ConvOptionFn[float64]) (f64 float64, err error) {
	opt := NewConvOption(optFns...)
	if !opt.NilAsFail && in == nil {
		return 0, nil
	}

	switch tVal := in.(type) {
	case string:
		f64, err = strconv.ParseFloat(strings.TrimSpace(tVal), 64)
	case int:
		f64 = float64(tVal)
	case int8:
		f64 = float64(tVal)
	case int16:
		f64 = float64(tVal)
	case int32:
		f64 = float64(tVal)
	case int64:
		f64 = float64(tVal)
	case uint:
		f64 = float64(tVal)
	case uint8:
		f64 = float64(tVal)
	case uint16:
		f64 = float64(tVal)
	case uint32:
		f64 = float64(tVal)
	case uint64:
		f64 = float64(tVal)
	case float32:
		f64 = float64(tVal)
	case float64:
		f64 = tVal
	case *float64: // default support float64 ptr type
		f64 = *tVal
	case time.Duration:
		f64 = float64(tVal)
	case comdef.Float64able: // eg: json.Number
		f64, err = tVal.Float64()
	default:
		if opt.HandlePtr {
			if rv := reflect.ValueOf(in); rv.Kind() == reflect.Pointer {
				rv = rv.Elem()
				if checkfn.IsSimpleKind(rv.Kind()) {
					return ToFloatWith(rv.Interface(), optFns...)
				}
			}
		}

		if opt.UserConvFn != nil {
			f64, err = opt.UserConvFn(in)
		} else {
			err = comdef.ErrConvType
		}
	}
	return
}

/*************************************************************
 * convert intX/floatX to string
 *************************************************************/

// MustString convert intX/floatX value to string, will panic on error
func MustString(val any) string {
	str, err := ToStringWith(val)
	if err != nil {
		panic(err)
	}
	return str
}

// StringOrPanic convert intX/floatX value to string, will panic on error
func StringOrPanic(val any) string { return MustString(val) }

// StringOrDefault convert intX/floatX value to string, will return default value on error
func StringOrDefault(val any, defVal string) string { return StringOr(val, defVal) }

// StringOr convert intX/floatX value to string, will return default value on error
func StringOr(val any, defVal string) string {
	str, err := ToStringWith(val)
	if err != nil {
		return defVal
	}
	return str
}

// ToString convert intX/floatX value to string, return error on failed
func ToString(val any) (string, error) { return ToStringWith(val) }

// StringOrErr convert intX/floatX value to string, return error on failed
func StringOrErr(val any) (string, error) { return ToStringWith(val) }

// QuietString convert intX/floatX value to string, other type convert by fmt.Sprint
func QuietString(val any) string { return SafeString(val) }

// String convert intX/floatX value to string, other type convert by fmt.Sprint
func String(val any) string {
	str, _ := TryToString(val, false)
	return str
}

// SafeString convert intX/floatX value to string, other type convert by fmt.Sprint
func SafeString(val any) string {
	str, _ := TryToString(val, false)
	return str
}

// TryToString try convert intX/floatX value to string
//
// if defaultAsErr is False, will use fmt.Sprint convert other type
func TryToString(val any, defaultAsErr bool) (string, error) {
	var optFn comfunc.ConvOptionFn
	if !defaultAsErr {
		optFn = comfunc.WithUserConvFn(comfunc.StrBySprintFn)
	}
	return ToStringWith(val, optFn)
}

// ToStringWith try to convert value to string. can with some option func, more see comfunc.ConvOption.
func ToStringWith(in any, optFns ...comfunc.ConvOptionFn) (string, error) {
	return comfunc.ToStringWith(in, optFns...)
}
