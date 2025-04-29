package structs

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/gookit/goutil/maputil"
	"github.com/gookit/goutil/reflects"
)

// ToMap quickly convert structs to map by reflect
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

// ToSMap quickly and safe convert structs to map[string]string by reflect
func ToSMap(st any, optFns ...MapOptFunc) map[string]string {
	mp, _ := StructToMap(st, optFns...)
	return maputil.ToStringMap(mp)
}

// TryToSMap quickly convert structs to map[string]string by reflect
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

// MapOptions for convert struct to map
type MapOptions struct {
	// TagName for map filed. default is "json"
	TagName string
	// ParseDepth for parse. TODO support depth
	ParseDepth int
	// MergeAnonymous struct fields to parent map. default is true
	MergeAnonymous bool
	// ExportPrivate export private fields. default is false
	ExportPrivate bool
}

// MapOptFunc define
type MapOptFunc func(opt *MapOptions)

// WithMapTagName set tag name for map field
func WithMapTagName(tagName string) MapOptFunc {
	return func(opt *MapOptions) {
		opt.TagName = tagName
	}
}

// MergeAnonymous merge anonymous struct fields to parent map
func MergeAnonymous(opt *MapOptions) {
	opt.MergeAnonymous = true
}

// ExportPrivate merge anonymous struct fields to parent map
func ExportPrivate(opt *MapOptions) {
	opt.ExportPrivate = true
}

// StructToMap quickly convert structs to map[string]any by reflect.
// Can custom export field name by tag `json` or custom tag
func StructToMap(st any, optFns ...MapOptFunc) (map[string]any, error) {
	mp := make(map[string]any)
	if st == nil {
		return mp, nil
	}

	obj := reflect.Indirect(reflect.ValueOf(st))
	if obj.Kind() != reflect.Struct {
		return mp, errors.New("must be an struct value")
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
		ft := refType.Field(i)
		name := ft.Name
		// skip un-exported field
		if !opt.ExportPrivate && IsUnexported(name) {
			continue
		}

		tagVal, ok := ft.Tag.Lookup(opt.TagName)
		if ok && tagVal != "" {
			sMap, err := ParseTagValueDefault(name, tagVal)
			if err != nil {
				return nil, err
			}

			name = sMap.Default("name", name)
			if name == "" { // un-exported field
				continue
			}
		}

		field := reflect.Indirect(obj.Field(i))
		if !field.IsValid() {
			continue
		}

		if field.Kind() == reflect.Struct {
			// collect anonymous struct values to parent.
			if ft.Anonymous && opt.MergeAnonymous {
				_, err := structToMap(field, opt, mp)
				if err != nil {
					return nil, err
				}
			} else { // collect struct values to submap
				sub, err := structToMap(field, opt, nil)
				if err != nil {
					return nil, err
				}
				mp[name] = sub
			}
			continue
		}

		if field.CanInterface() {
			mp[name] = field.Interface()
		} else if field.CanAddr() { // for unexported field
			mp[name] = reflects.UnexportedValue(field)
		}
	}

	return mp, nil
}
