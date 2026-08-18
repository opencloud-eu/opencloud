package search

import (
	"reflect"
	"strings"
	"time"
)

// IsNumericField reports whether the indexed field at the dotted path holds a
// numeric or time value. Terms aggregations on those are rejected: bleve stores
// them as prefix-coded binary, so term buckets are meaningless. The set is built
// by walking the Resource type, so new facet fields are picked up automatically.
func IsNumericField(dottedPath string) bool {
	return numericFields[dottedPath]
}

var numericFields = buildNumericFieldSet()

var timeType = reflect.TypeOf(time.Time{})

func buildNumericFieldSet() map[string]bool {
	out := map[string]bool{}
	walkStruct(out, "", reflect.TypeOf(Resource{}))
	return out
}

func walkStruct(out map[string]bool, prefix string, t reflect.Type) {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Anonymous {
			// embedded: promote fields into the current prefix, like encoding/json.
			walkStruct(out, prefix, f.Type)
			continue
		}
		path := prefix + jsonFieldName(f)
		ft := f.Type
		for ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		switch ft.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			out[path] = true
		case reflect.Struct:
			if ft == timeType {
				// time.Time round-trips as RFC3339; treat as numeric.
				out[path] = true
				continue
			}
			walkStruct(out, path+".", ft)
		}
	}
}

func jsonFieldName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return f.Name
	}
	return strings.Split(tag, ",")[0]
}
