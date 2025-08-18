package structs

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/gookit/goutil/maputil"
	"github.com/gookit/goutil/reflects"
)

// ToMap quickly convert structs to map by reflection
func ToMap(st any, optFns ...MapOptFunc) map[string]any {
	mp, _ := StructToMap(st, optFns...)
	return mp
}

// MustToMap alis of TryToMap, but will panic on error
func MustToMap(st any, optFns ...MapOptFunc) map[string]any {
	mp, err := StructToMap(st, optFns...)
	if err != nil {
		panic(err)
	}
	return mp
}

// TryToMap simple convert structs to map by reflect
func TryToMap(st any, optFns ...MapOptFunc) (map[string]any, error) {
	return StructToMap(st, optFns...)
}

// ToSMap quickly and safe convert structs to map[string]string by reflection
func ToSMap(st any, optFns ...MapOptFunc) map[string]string {
	mp, _ := StructToMap(st, optFns...)
	return maputil.ToStringMap(mp)
}

// TryToSMap quickly convert structs to map[string]string by reflection
func TryToSMap(st any, optFns ...MapOptFunc) (map[string]string, error) {
	mp, err := StructToMap(st, optFns...)
	if err != nil {
		return nil, err
	}
	return maputil.ToStringMap(mp), nil
}

// MustToSMap alias of ToStringMap(), but will panic on error
func MustToSMap(st any, optFns ...MapOptFunc) map[string]string {
	mp, err := StructToMap(st, optFns...)
	if err != nil {
		panic(err)
	}
	return maputil.ToStringMap(mp)
}

// ToString quickly format struct to string
func ToString(st any, optFns ...MapOptFunc) string {
	mp, err := StructToMap(st, optFns...)
	if err == nil {
		return maputil.ToString(mp)
	}
	return fmt.Sprint(st)
}

const defaultFieldTag = "json"

// CustomUserFunc for map convert
//  - fName: raw field name in struct
//
// Returns:
//  - ok: return true to collect field, otherwise excluded.
//  - newVal: `newVal != nil` return new value to collect, otherwise collect original value.
type CustomUserFunc func(fName string, fv reflect.Value) (ok bool, newVal any)

// MapOptions for convert struct to map
type MapOptions struct {
	// TagName for map filed. default is "json"
	TagName string
	// ParseDepth for parse and collect. TODO support depth
	ParseDepth int
	// MergeAnonymous struct fields to parent map. default is true
	MergeAnonymous bool
	// ExportPrivate export private fields. default is false
	ExportPrivate bool
	// IgnoreEmpty ignore empty value item. default: false
	IgnoreEmpty bool
	// UserFunc custom interceptor for filter or handle field value.
	UserFunc CustomUserFunc
}

// MapOptFunc define
type MapOptFunc func(opt *MapOptions)

// WithMapTagName set tag name for map field
func WithMapTagName(tagName string) MapOptFunc {
	return func(opt *MapOptions) { opt.TagName = tagName }
}

// WithUserFunc custom user func
func WithUserFunc(fn CustomUserFunc) MapOptFunc {
	return func(opt *MapOptions) { opt.UserFunc = fn }
}

// MergeAnonymous merge anonymous struct fields to parent map
func MergeAnonymous(opt *MapOptions) { opt.MergeAnonymous = true }

// ExportPrivate merge anonymous struct fields to parent map
func ExportPrivate(opt *MapOptions) { opt.ExportPrivate = true }

// WithIgnoreEmpty ignore on field value is empty
func WithIgnoreEmpty(opt *MapOptions) { opt.IgnoreEmpty = true }

// StructToMap quickly convert structs to map[string]any by reflection.
//
// Can custom export field name by tag `json` or custom tag. see MapOptions
func StructToMap(st any, optFns ...MapOptFunc) (map[string]any, error) {
	mp := make(map[string]any)
	if st == nil {
		return mp, nil
	}

	obj := reflect.Indirect(reflect.ValueOf(st))
	if obj.Kind() != reflect.Struct {
		return mp, errors.New("StructToMap: must be an struct value")
	}

	opt := &MapOptions{TagName: defaultFieldTag}
	for _, fn := range optFns {
		fn(opt)
	}

	_, err := structToMap(obj, opt, mp)
	return mp, err
}

func structToMap(obj reflect.Value, opt *MapOptions, mp map[string]any) (map[string]any, error) {
	if mp == nil {
		mp = make(map[string]any)
	}

	refType := obj.Type()
	for i := 0; i < obj.NumField(); i++ {
		sf := refType.Field(i)
		name := sf.Name
		// skip un-exported field
		if !opt.ExportPrivate && IsUnexported(name) {
			continue
		}

		tagVal, ok := sf.Tag.Lookup(opt.TagName)
		if ok && tagVal != "" {
			sMap, err := ParseTagValueDefault(name, tagVal)
			if err != nil {
				return nil, err
			}

			name = sMap.Default("name", name)
			if name == "" || name == "-" { // un-exported field
				continue
			}
		}

		fv := reflect.Indirect(obj.Field(i))
		if !fv.IsValid() {
			continue
		}

		// opt: ignore empty field
		if opt.IgnoreEmpty && reflects.IsEmpty(fv) {
			continue
		}

		if fv.Kind() == reflect.Struct {
			// up: special handle time.Time field value
			if reflects.IsTimeType(fv.Type()) {
				mp[name] = fv.Interface().(time.Time).Format(time.RFC3339)
				continue
			}

			// collect anonymous struct values to parent.
			if sf.Anonymous && opt.MergeAnonymous {
				_, err := structToMap(fv, opt, mp)
				if err != nil {
					return nil, err
				}
			} else { // collect struct values to submap
				sub, err := structToMap(fv, opt, nil)
				if err != nil {
					return nil, err
				}
				mp[name] = sub
			}
			continue
		}

		// TODO support struct slice field.

		// up: support custom user func
		if opt.UserFunc != nil {
			ok1, newVal := opt.UserFunc(sf.Name, fv)
			if !ok1 {
				continue
			}

			// ok1=true, newVal != nil
			if newVal != nil {
				mp[name] = newVal
				continue
			}
		}

		if fv.CanInterface() {
			mp[name] = fv.Interface()
		} else if fv.CanAddr() { // for unexported field
			mp[name] = reflects.UnexportedValue(fv)
		}
	}

	return mp, nil
}
