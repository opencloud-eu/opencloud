package reflects

import (
	"fmt"
	"reflect"
	"strconv"
	"unsafe"
)

// loopIndirect returns the item at the end of indirection, and a bool to indicate
// if it's nil. If the returned bool is true, the returned value's kind will be
// either a pointer or interface.
func loopIndirect(v reflect.Value) (rv reflect.Value, isNil bool) {
	for ; v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface; v = v.Elem() {
		if v.IsNil() {
			return v, true
		}
	}
	return v, false
}

// indirectInterface returns the concrete value in an interface value,
// or else the zero reflect.Value.
// That is, if v represents the interface value x, the result is the same as reflect.ValueOf(x):
// the fact that x was an interface value is forgotten.
func indirectInterface(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Interface {
		return v
	}
	if v.IsNil() {
		return emptyValue
	}
	return v.Elem()
}

// Elem returns the value that the interface v contains
// or that the pointer v points to. otherwise, will return self
func Elem(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		return v.Elem()
	}
	return v
}

// Indirect like reflect.Indirect(), but can also indirect reflect.Interface. otherwise, will return self
func Indirect(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		return v.Elem()
	}
	return v
}

// UnwrapAny unwrap reflect.Interface value. otherwise, will return self
func UnwrapAny(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Interface {
		return v.Elem()
	}

	if v.IsNil() {
		return emptyValue
	}
	return v
}

// TypeReal returns a ptr type's real type. otherwise, will return self.
func TypeReal(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Pointer {
		return t.Elem()
	}
	return t
}

// TypeElem returns the array, slice, chan, map type's element type. otherwise, will return self.
func TypeElem(t reflect.Type) reflect.Type {
	switch t.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		return t.Elem()
	default:
		return t
	}
}

// Len get reflect value length. allow: intX, uintX, floatX, string, map, array, chan, slice.
//
// Note: (u)intX use width. float to string then calc len.
func Len(v reflect.Value) int {
	v = reflect.Indirect(v)

	// (u)int use width.
	switch v.Kind() {
	case reflect.String:
		return len([]rune(v.String()))
	case reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		return v.Len()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return len(strconv.FormatInt(int64(v.Uint()), 10))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return len(strconv.FormatInt(v.Int(), 10))
	case reflect.Float32, reflect.Float64:
		return len(fmt.Sprint(v.Interface()))
	default:
		return -1 // cannot get length
	}
}

// SliceSubKind get sub-elem kind of the array, slice, variadic-var. alias SliceElemKind()
func SliceSubKind(typ reflect.Type) reflect.Kind {
	return SliceElemKind(typ)
}

// SliceElemKind get sub-elem kind of the array, slice, variadic-var.
//
// Usage:
//
//	SliceElemKind(reflect.TypeOf([]string{"abc"})) // reflect.String
func SliceElemKind(typ reflect.Type) reflect.Kind {
	if typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array {
		return typ.Elem().Kind()
	}
	return reflect.Invalid
}

// UnexportedValue quickly get unexported value by reflect.Value
//
// NOTE: this method is unsafe, use it carefully.
// should ensure rv is addressable by field.CanAddr()
//
// refer: https://stackoverflow.com/questions/42664837/how-to-access-unexported-struct-fields
func UnexportedValue(rv reflect.Value) any {
	if rv.CanAddr() {
		// create new value from addr, now can be read and set.
		return reflect.NewAt(rv.Type(), unsafe.Pointer(rv.UnsafeAddr())).Elem().Interface()
	}

	// If the rv is not addressable this trick won't work, but you can create an addressable copy like this
	rs2 := reflect.New(rv.Type()).Elem()
	rs2.Set(rv)
	rv = rs2.Field(0)
	rv = reflect.NewAt(rv.Type(), unsafe.Pointer(rv.UnsafeAddr())).Elem()
	// Now rv can be read. TIP: Setting will succeed but only affects the temporary copy.
	return rv.Interface()
}

// SetUnexportedValue quickly set unexported field value by reflect
//
// NOTE: this method is unsafe, use it carefully.
// should ensure rv is addressable by field.CanAddr()
func SetUnexportedValue(rv reflect.Value, value any) {
	reflect.NewAt(rv.Type(), unsafe.Pointer(rv.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}

// SetValue to a `reflect.Value`. will auto convert type if needed.
func SetValue(rv reflect.Value, val any) error {
	// get a real type of the ptr value
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			elemTyp := rv.Type().Elem()
			rv.Set(reflect.New(elemTyp))
		}

		// use elem for set value
		rv = reflect.Indirect(rv)
	}

	rv1, err := ValueByType(val, rv.Type())
	if err == nil {
		rv.Set(rv1)
	}
	return err
}

// SetRValue to a `reflect.Value`. will direct set value without a type convert.
func SetRValue(rv, val reflect.Value) {
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			elemTyp := rv.Type().Elem()
			rv.Set(reflect.New(elemTyp))
		}
		rv = reflect.Indirect(rv)
	}

	rv.Set(val)
}
