package reflects

import (
	"bytes"
	"reflect"
)

// HasChild type check. eg: array, slice, map, struct
func HasChild(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Struct:
		return true
	default:
		return false
	}
}

// IsArrayOrSlice check. eg: array, slice
func IsArrayOrSlice(k reflect.Kind) bool {
	return k == reflect.Slice || k == reflect.Array
}

// IsSimpleKind kind in: string, bool, intX, uintX, floatX
func IsSimpleKind(k reflect.Kind) bool {
	if reflect.String == k {
		return true
	}
	return k > reflect.Invalid && k <= reflect.Float64
}

// IsAnyInt check is intX or uintX type. alias of the IsIntLike()
func IsAnyInt(k reflect.Kind) bool {
	return k >= reflect.Int && k <= reflect.Uintptr
}

// IsIntLike reports whether the type is int-like(intX, uintX).
func IsIntLike(k reflect.Kind) bool {
	return k >= reflect.Int && k <= reflect.Uintptr
}

// IsIntx check is intX type
func IsIntx(k reflect.Kind) bool {
	return k >= reflect.Int && k <= reflect.Int64
}

// IsUintX check is uintX type
func IsUintX(k reflect.Kind) bool {
	return k >= reflect.Uint && k <= reflect.Uintptr
}

// IsNil reflect value
func IsNil(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

// IsValidPtr check variable is a valid pointer.
func IsValidPtr(v reflect.Value) bool {
	return v.IsValid() && (v.Kind() == reflect.Ptr) && !v.IsNil()
}

// CanBeNil reports whether an untyped nil can be assigned to the type. See reflect.Zero.
func CanBeNil(typ reflect.Type) bool {
	switch typ.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return true
	case reflect.Struct:
		return typ == reflectValueType
	}
	return false
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
// TIP: cannot compare function type
func IsEqual(src, dst any) bool {
	if src == nil || dst == nil {
		return src == dst
	}

	bs1, ok := src.([]byte)
	if !ok {
		return reflect.DeepEqual(src, dst)
	}

	bs2, ok := dst.([]byte)
	if !ok {
		return false
	}

	if bs1 == nil || bs2 == nil {
		return bs1 == nil && bs2 == nil
	}
	return bytes.Equal(bs1, bs2)
}

// IsZero reflect value check, alias of the IsEmpty()
var IsZero = IsEmpty

// IsEmpty reflect value check
func IsEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Invalid:
		return true
	case reflect.String, reflect.Array:
		return v.Len() == 0
	case reflect.Map, reflect.Slice:
		return v.Len() == 0 || v.IsNil()
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr, reflect.Func:
		return v.IsNil()
	default:
		return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
	}
}

// IsEmptyValue reflect value check, alias of the IsEmptyReal()
var IsEmptyValue = IsEmptyReal

// IsEmptyReal reflect value check.
//
// Note:
//
//	Difference the IsEmpty(), if value is ptr or interface, will check real elem.
//
// From src/pkg/encoding/json/encode.go.
func IsEmptyReal(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		if v.IsNil() {
			return true
		}
		return IsEmptyReal(v.Elem())
	case reflect.Func:
		return v.IsNil()
	case reflect.Invalid:
		return true
	default:
		return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
	}
}
