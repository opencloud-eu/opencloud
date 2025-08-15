package structs

import (
	"errors"
	"reflect"
)

// Wrapper struct for read or set field value
type Wrapper struct {
	// src any // source struct

	// raw reflect.Value of source struct
	rv reflect.Value

	// FieldTagName field name for read/write value. default tag: json
	FieldTagName string

	// caches for field rv and name and tag name TODO
	fieldNames []string                 //lint:ignore U1000 for unused
	fvCacheMap map[string]reflect.Value //lint:ignore U1000 for unused
}

// Wrap quick create a struct wrapper
func Wrap(src any) *Wrapper { return NewWrapper(src) }

// NewWrapper create a struct wrapper
func NewWrapper(src any) *Wrapper {
	return WrapValue(reflect.ValueOf(src))
}

// WrapValue create a struct wrapper
func WrapValue(rv reflect.Value) *Wrapper {
	rv = reflect.Indirect(rv)
	if rv.Kind() != reflect.Struct {
		panic("must be provider an struct value")
	}
	return &Wrapper{rv: rv}
}

// Get field value by name, name allows to use dot syntax.
func (r *Wrapper) Get(name string) any {
	val, ok := r.Lookup(name)
	if !ok {
		return nil
	}
	return val
}

// Lookup field value by name, name allows to use dot syntax.
func (r *Wrapper) Lookup(name string) (val any, ok bool) {
	fv := r.rv.FieldByName(name)
	if !fv.IsValid() {
		return
	}

	if fv.CanInterface() {
		return fv.Interface(), true
	}
	return
}

// Set field value by name. name allows using dot syntax.
func (r *Wrapper) Set(name string, val any) error {
	fv := r.rv.FieldByName(name)
	if !fv.IsValid() {
		return errors.New("field " + name + " not found")
	}

	if !fv.CanSet() {
		return errors.New("can not set value for field: " + name)
	}

	fv.Set(reflect.ValueOf(val))
	return nil
}
