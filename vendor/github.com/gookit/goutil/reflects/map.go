package reflects

import (
	"errors"
	"reflect"
	"strconv"
)

// TryAnyMap convert map[TYPE1]TYPE2 to map[string]any
func TryAnyMap(mp reflect.Value) (map[string]any, error) {
	saMap := make(map[string]any)
	err := EachStrAnyMap(mp, func(key string, val any) {
		saMap[key] = val
	})

	return saMap, err
}

// EachMap process any map data
func EachMap(mp reflect.Value, fn func(key, val reflect.Value)) (err error) {
	if fn == nil {
		return
	}
	if mp.Kind() != reflect.Map {
		return errors.New("EachMap: only allow map value data")
	}

	for _, key := range mp.MapKeys() {
		fn(key, mp.MapIndex(key))
	}
	return
}

// EachStrAnyMap process any map data as string key and any value
func EachStrAnyMap(mp reflect.Value, fn func(key string, val any)) error {
	return EachMap(mp, func(key, val reflect.Value) {
		fn(String(key), val.Interface())
	})
}

// FlatFunc custom collect handle func
type FlatFunc func(path string, val reflect.Value)

// FlatMap process tree map to flat key-value map.
//
// Examples:
//
//	{"top": {"sub": "value", "sub2": "value2"} }
//	->
//	{"top.sub": "value", "top.sub2": "value2" }
func FlatMap(rv reflect.Value, fn FlatFunc) {
	if fn == nil {
		return
	}

	if rv.Kind() != reflect.Map {
		panic("only allow flat map data")
	}
	flatMap(rv, fn, "")
}

func flatMap(rv reflect.Value, fn FlatFunc, parent string) {
	for _, key := range rv.MapKeys() {
		path := String(key)
		if parent != "" {
			path = parent + "." + path
		}

		fv := Indirect(rv.MapIndex(key))
		switch fv.Kind() {
		case reflect.Map:
			flatMap(fv, fn, path)
		case reflect.Array, reflect.Slice:
			flatSlice(fv, fn, path)
		default:
			fn(path, fv)
		}
	}
}

func flatSlice(rv reflect.Value, fn FlatFunc, parent string) {
	for i := 0; i < rv.Len(); i++ {
		path := parent + "[" + strconv.Itoa(i) + "]"
		fv := Indirect(rv.Index(i))

		switch fv.Kind() {
		case reflect.Map:
			flatMap(fv, fn, path)
		case reflect.Array, reflect.Slice:
			flatSlice(fv, fn, path)
		default:
			fn(path, fv)
		}
	}
}
