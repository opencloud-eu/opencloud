package goutil

import (
	"fmt"
	"math"
	"reflect"
	"strconv"

	"github.com/gookit/goutil/comdef"
	"github.com/gookit/goutil/internal/comfunc"
	"github.com/gookit/goutil/mathutil"
	"github.com/gookit/goutil/reflects"
	"github.com/gookit/goutil/strutil"
)

// Bool convert value to bool
func Bool(v any) bool {
	bl, _ := comfunc.ToBool(v)
	return bl
}

// ToBool try to convert type to bool
func ToBool(v any) (bool, error) { return comfunc.ToBool(v) }

// String func. always converts value to string, will ignore error
func String(v any) string {
	s, _ := strutil.AnyToString(v, false)
	return s
}

// ToString convert value to string, will return error on fail.
func ToString(v any) (string, error) { return strutil.AnyToString(v, true) }

// Int convert value to int
func Int(v any) int {
	iv, _ := mathutil.ToInt(v)
	return iv
}

// ToInt try to convert value to int
func ToInt(v any) (int, error) { return mathutil.ToInt(v) }

// Int64 convert value to int64
func Int64(v any) int64 {
	iv, _ := mathutil.ToInt64(v)
	return iv
}

// ToInt64 try to convert value to int64
func ToInt64(v any) (int64, error) { return mathutil.ToInt64(v) }

// Uint convert value to uint
func Uint(v any) uint {
	iv, _ := mathutil.ToUint(v)
	return iv
}

// ToUint try to convert value to uint
func ToUint(v any) (uint, error) { return mathutil.ToUint(v) }

// Uint64 convert value to uint64
func Uint64(v any) uint64 {
	iv, _ := mathutil.ToUint64(v)
	return iv
}

// ToUint64 try to convert value to uint64
func ToUint64(v any) (uint64, error) { return mathutil.ToUint64(v) }

// BoolString convert bool to string
func BoolString(bl bool) string { return strconv.FormatBool(bl) }

// BaseTypeVal convert custom type or intX,uintX,floatX to generic base type.
//
//	intX 	    => int64
//	unitX 	    => uint64
//	floatX      => float64
//	string 	    => string
//
// returns int64,uint64,string,float or error
func BaseTypeVal(val any) (value any, err error) {
	return reflects.BaseTypeVal(reflect.ValueOf(val))
}

// SafeKind convert input any value to given reflect.Kind type.
func SafeKind(val any, kind reflect.Kind) (newVal any) {
	newVal, _ = ToKind(val, kind, nil)
	return
}

// SafeConv convert input any value to given reflect.Kind type.
func SafeConv(val any, kind reflect.Kind) (newVal any) {
	newVal, _ = ToKind(val, kind, nil)
	return
}

// ConvTo convert input any value to given reflect.Kind.
func ConvTo(val any, kind reflect.Kind) (newVal any, err error) {
	return ToKind(val, kind, nil)
}

// ConvOrDefault convert input any value to given reflect.Kind.
// if fail will return default value.
func ConvOrDefault(val any, kind reflect.Kind, defVal any) any {
	newVal, err := ToKind(val, kind, nil)
	if err != nil {
		return defVal
	}
	return newVal
}

// ToType
// func ToType[T any](val any, kind reflect.Kind, fbFunc func(val any) (T, error)) (newVal T, err error)  {
// 	switch typVal.(type) { // assert ERROR
// 	case string:
// 	}
// }

// ToKind convert input any value to given reflect.Kind type.
//
// TIPs: Only support kind: string, bool, intX, uintX, floatX
//
// Examples:
//
//		val, err := ToKind("123", reflect.Int) // 123
func ToKind(val any, kind reflect.Kind, fbFunc func(val any) (any, error)) (newVal any, err error) {
	switch kind {
	case reflect.Int:
		var dstV int
		if dstV, err = mathutil.ToInt(val); err == nil {
			if dstV > math.MaxInt {
				return nil, fmt.Errorf("value overflow int. val: %v", val)
			}
			newVal = dstV
		}
	case reflect.Int8:
		var dstV int
		if dstV, err = mathutil.ToInt(val); err == nil {
			if dstV > math.MaxInt8 {
				return nil, fmt.Errorf("value overflow int8. val: %v", val)
			}
			newVal = int8(dstV)
		}
	case reflect.Int16:
		var dstV int
		if dstV, err = mathutil.ToInt(val); err == nil {
			if dstV > math.MaxInt16 {
				return nil, fmt.Errorf("value overflow int16. val: %v", val)
			}
			newVal = int16(dstV)
		}
	case reflect.Int32:
		var dstV int
		if dstV, err = mathutil.ToInt(val); err == nil {
			if dstV > math.MaxInt32 {
				return nil, fmt.Errorf("value overflow int32. val: %v", val)
			}
			newVal = int32(dstV)
		}
	case reflect.Int64:
		var dstV int64
		if dstV, err = mathutil.ToInt64(val); err == nil {
			newVal = dstV
		}
	case reflect.Uint:
		var dstV uint
		if dstV, err = mathutil.ToUint(val); err == nil {
			newVal = dstV
		}
	case reflect.Uint8:
		var dstV uint
		if dstV, err = mathutil.ToUint(val); err == nil {
			if dstV > math.MaxUint8 {
				return nil, fmt.Errorf("value overflow uint8. val: %v", val)
			}
			newVal = uint8(dstV)
		}
	case reflect.Uint16:
		var dstV uint
		if dstV, err = mathutil.ToUint(val); err == nil {
			if dstV > math.MaxUint16 {
				return nil, fmt.Errorf("value overflow uint16. val: %v", val)
			}
			newVal = uint16(dstV)
		}
	case reflect.Uint32:
		var dstV uint
		if dstV, err = mathutil.ToUint(val); err == nil {
			if dstV > math.MaxUint32 {
				return nil, fmt.Errorf("value overflow uint32. val: %v", val)
			}
			newVal = uint32(dstV)
		}
	case reflect.Uint64:
		var dstV uint64
		if dstV, err = mathutil.ToUint64(val); err == nil {
			newVal = dstV
		}
	case reflect.Float32:
		var dstV float64
		if dstV, err = mathutil.ToFloat(val); err == nil {
			if dstV > math.MaxFloat32 {
				return nil, fmt.Errorf("value overflow float32. val: %v", val)
			}
			newVal = float32(dstV)
		}
	case reflect.Float64:
		var dstV float64
		if dstV, err = mathutil.ToFloat(val); err == nil {
			newVal = dstV
		}
	case reflect.String:
		var dstV string
		if dstV, err = strutil.ToString(val); err == nil {
			newVal = dstV
		}
	case reflect.Bool:
		if bl, err1 := comfunc.ToBool(val); err1 == nil {
			newVal = bl
		} else {
			err = err1
		}
	default:
		if fbFunc != nil {
			newVal, err = fbFunc(val)
		} else {
			err = comdef.ErrConvType
		}
	}
	return
}
