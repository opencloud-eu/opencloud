package mapping

import (
	"reflect"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	timeType      = reflect.TypeFor[time.Time]()
	timestampType = reflect.TypeFor[timestamppb.Timestamp]()
)

// deref unwraps pointer and slice types to their element type.
func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
		t = t.Elem()
	}
	return t
}

// inferType returns the mapping type for a Go type. Pointers and slices are
// unwrapped to their element type. time.Time and timestamppb.Timestamp become
// datetime; other structs become object.
func inferType(t reflect.Type) string {
	t = deref(t)
	switch t.Kind() {
	case reflect.String:
		return TypeKeyword
	case reflect.Bool:
		return TypeBool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return TypeNumeric
	case reflect.Struct:
		if t == timeType || t == timestampType {
			return TypeDatetime
		}
		return TypeObject
	}
	return ""
}

// fieldInfo is the resolved metadata for one struct field.
type fieldInfo struct {
	Name     string
	GoField  reflect.StructField
	Skip     bool
	Embedded bool
}

// resolveField resolves a struct field's json-tag name and skip/embed state.
func resolveField(sf reflect.StructField) fieldInfo {
	if !sf.IsExported() {
		return fieldInfo{Skip: true}
	}
	name := sf.Name
	tag := sf.Tag.Get("json")
	if tag != "" {
		first, _, _ := strings.Cut(tag, ",")
		if first == "-" {
			return fieldInfo{Skip: true}
		}
		if first != "" {
			name = first
		}
	}
	return fieldInfo{
		Name:     name,
		GoField:  sf,
		Embedded: sf.Anonymous,
	}
}

// walkFields visits exported leaf fields of t, flattening embedded structs
// onto the enclosing level. It returns the first error returned by fn.
func walkFields(t reflect.Type, fn func(fi fieldInfo) error) error {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	for i := 0; i < t.NumField(); i++ {
		fi := resolveField(t.Field(i))
		if fi.Skip {
			continue
		}
		if fi.Embedded {
			if err := walkFields(fi.GoField.Type, fn); err != nil {
				return err
			}
			continue
		}
		if err := fn(fi); err != nil {
			return err
		}
	}
	return nil
}

// structType returns the underlying struct type, unwrapping pointers and
// slices. Returns nil when t is not a walkable struct (e.g. time.Time).
func structType(t reflect.Type) reflect.Type {
	t = deref(t)
	if t.Kind() != reflect.Struct {
		return nil
	}
	if t == timeType || t == timestampType {
		return nil
	}
	return t
}
