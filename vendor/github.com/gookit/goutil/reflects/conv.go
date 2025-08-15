package reflects

import (
	"fmt"
	"math"
	"reflect"
	"strconv"

	"github.com/gookit/goutil/comdef"
	"github.com/gookit/goutil/internal/comfunc"
	"github.com/gookit/goutil/mathutil"
	"github.com/gookit/goutil/strutil"
)

// BaseTypeVal convert custom type or intX,uintX,floatX to generic base type.
func BaseTypeVal(v reflect.Value) (value any, err error) {
	return ToBaseVal(v)
}

// ToBaseVal convert custom type or intX,uintX,floatX to generic base type.
//
//	intX 	    => int64
//	unitX 	    => uint64
//	floatX      => float64
//	string 	    => string
//
// returns int64,string,float or error
func ToBaseVal(v reflect.Value) (value any, err error) {
	v = reflect.Indirect(v)

	switch v.Kind() {
	case reflect.String:
		value = v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value = v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		value = v.Uint() // always return int64
	case reflect.Float32, reflect.Float64:
		value = v.Float()
	default:
		err = comdef.ErrConvType
	}
	return
}

// ConvToType convert and create reflect.Value by give reflect.Type
func ConvToType(val any, typ reflect.Type) (rv reflect.Value, err error) {
	return ValueByType(val, typ)
}

// ValueByType create reflect.Value by give reflect.Type
func ValueByType(val any, typ reflect.Type) (rv reflect.Value, err error) {
	var ok bool
	var newRv reflect.Value
	if newRv, ok = val.(reflect.Value); !ok {
		newRv = reflect.ValueOf(val)
	}

	// fix: check newRv is valid
	if !newRv.IsValid() {
		return rv, comdef.ErrConvType
	}

	// check the same type. like map
	if newRv.Type() == typ {
		return newRv, nil
	}

	// handle kind: string, bool, intX, uintX, floatX
	if typ.Kind() == reflect.String || typ.Kind() <= reflect.Float64 {
		return ConvToKind(val, typ.Kind())
	}

	// try the auto convert slice type
	if IsArrayOrSlice(newRv.Kind()) && IsArrayOrSlice(typ.Kind()) {
		return ConvSlice(newRv, typ.Elem())
	}

	err = comdef.ErrConvType
	return
}

// ValueByKind convert and create reflect.Value by give reflect.Kind
func ValueByKind(val any, kind reflect.Kind) (reflect.Value, error) { return ConvToKind(val, kind) }

// ConvToKind convert and create reflect.Value by give reflect.Kind
//
// TIPs:
//
//	Only support kind: string, bool, intX, uintX, floatX
func ConvToKind(val any, kind reflect.Kind, fallback ...ConvFunc) (rv reflect.Value, err error) {
	if rv1, ok := val.(reflect.Value); ok {
		val = rv1.Interface()
	}

	switch kind {
	case reflect.Int:
		var dstV int
		if dstV, err = mathutil.ToInt(val); err == nil {
			rv = reflect.ValueOf(dstV)
		}
	case reflect.Int8:
		var dstV int
		if dstV, err = mathutil.ToInt(val); err == nil {
			if dstV > math.MaxInt8 {
				return rv, fmt.Errorf("value overflow int8. val: %v", val)
			}
			rv = reflect.ValueOf(int8(dstV))
		}
	case reflect.Int16:
		var dstV int
		if dstV, err = mathutil.ToInt(val); err == nil {
			if dstV > math.MaxInt16 {
				return rv, fmt.Errorf("value overflow int16. val: %v", val)
			}
			rv = reflect.ValueOf(int16(dstV))
		}
	case reflect.Int32:
		var dstV int
		if dstV, err = mathutil.ToInt(val); err == nil {
			if dstV > math.MaxInt32 {
				return rv, fmt.Errorf("value overflow int32. val: %v", val)
			}
			rv = reflect.ValueOf(int32(dstV))
		}
	case reflect.Int64:
		var dstV int64
		if dstV, err = mathutil.ToInt64(val); err == nil {
			rv = reflect.ValueOf(dstV)
		}
	case reflect.Uint:
		var dstV uint
		if dstV, err = mathutil.ToUint(val); err == nil {
			rv = reflect.ValueOf(dstV)
		}
	case reflect.Uint8:
		var dstV uint
		if dstV, err = mathutil.ToUint(val); err == nil {
			if dstV > math.MaxUint8 {
				return rv, fmt.Errorf("value overflow uint8. val: %v", val)
			}
			rv = reflect.ValueOf(uint8(dstV))
		}
	case reflect.Uint16:
		var dstV uint
		if dstV, err = mathutil.ToUint(val); err == nil {
			if dstV > math.MaxUint16 {
				return rv, fmt.Errorf("value overflow uint16. val: %v", val)
			}
			rv = reflect.ValueOf(uint16(dstV))
		}
	case reflect.Uint32:
		var dstV uint
		if dstV, err = mathutil.ToUint(val); err == nil {
			if dstV > math.MaxUint32 {
				return rv, fmt.Errorf("value overflow uint32. val: %v", val)
			}
			rv = reflect.ValueOf(uint32(dstV))
		}
	case reflect.Uint64:
		var dstV uint64
		if dstV, err = mathutil.ToUint64(val); err == nil {
			rv = reflect.ValueOf(dstV)
		}
	case reflect.Float32:
		var dstV float64
		if dstV, err = mathutil.ToFloat(val); err == nil {
			if dstV > math.MaxFloat32 {
				return rv, fmt.Errorf("value overflow float32. val: %v", val)
			}
			rv = reflect.ValueOf(float32(dstV))
		}
	case reflect.Float64:
		var dstV float64
		if dstV, err = mathutil.ToFloat(val); err == nil {
			rv = reflect.ValueOf(dstV)
		}
	case reflect.String:
		if dstV, err1 := strutil.ToString(val); err1 == nil {
			rv = reflect.ValueOf(dstV)
		} else {
			err = err1
		}
	case reflect.Bool:
		if bl, err1 := comfunc.ToBool(val); err1 == nil {
			rv = reflect.ValueOf(bl)
		} else {
			err = err1
		}
	default:
		// call fallback func
		if len(fallback) > 0 && fallback[0] != nil {
			rv, err = fallback[0](val, kind)
		} else {
			err = comdef.ErrConvType
		}
		err = comdef.ErrConvType
	}
	return
}

// ConvSlice make new type slice from old slice, will auto convert element type.
//
// TIPs:
//
//	Only support kind: string, bool, intX, uintX, floatX
func ConvSlice(oldSlRv reflect.Value, newElemTyp reflect.Type) (rv reflect.Value, err error) {
	if !IsArrayOrSlice(oldSlRv.Kind()) {
		panic("only allow array or slice type value")
	}

	// do not need convert type
	if oldSlRv.Type().Elem() == newElemTyp {
		return oldSlRv, nil
	}

	newSlTyp := reflect.SliceOf(newElemTyp)
	newSlRv := reflect.MakeSlice(newSlTyp, 0, 0)
	for i := 0; i < oldSlRv.Len(); i++ {
		newElemV, err := ValueByKind(oldSlRv.Index(i).Interface(), newElemTyp.Kind())
		if err != nil {
			return reflect.Value{}, err
		}

		newSlRv = reflect.Append(newSlRv, newElemV)
	}
	return newSlRv, nil
}

// String convert
func String(rv reflect.Value) string {
	s, _ := ValToString(rv, false)
	return s
}

// ToString convert
func ToString(rv reflect.Value) (str string, err error) {
	return ValToString(rv, true)
}

// ValToString convert handle
func ValToString(rv reflect.Value, defaultAsErr bool) (str string, err error) {
	rv = Indirect(rv)
	switch rv.Kind() {
	case reflect.Invalid:
		str = ""
	case reflect.Bool:
		str = strconv.FormatBool(rv.Bool())
	case reflect.String:
		str = rv.String()
	case reflect.Float32, reflect.Float64:
		str = strconv.FormatFloat(rv.Float(), 'f', -1, 64)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		str = strconv.FormatInt(rv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		str = strconv.FormatUint(rv.Uint(), 10)
	default:
		if defaultAsErr {
			err = comdef.ErrConvType
		} else {
			str = fmt.Sprint(rv.Interface())
		}
	}
	return
}

// ToTimeOrDuration convert string to time.Time or time.Duration type
//
// If the target type is not match, return the input string.
func ToTimeOrDuration(str string, typ reflect.Type) (any, error) {
	// datetime, time, duration string should not greater than 64
	if len(str) > 64 {
		return str, nil
	}
	var anyVal any = str

	// time.Time date string
	if len(str) > 5 && IsTimeType(typ) {
		ttVal, err := strutil.ToTime(str)
		if err != nil {
			return nil, err
		}
		anyVal = ttVal
	} else if IsDurationType(typ) {
		dVal, err := strutil.ToDuration(str)
		if err != nil {
			return nil, err
		}
		anyVal = dVal
	}

	return anyVal, nil
}
