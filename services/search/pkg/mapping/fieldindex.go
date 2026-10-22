package mapping

import (
	"reflect"
	"strings"
)

// FieldNameIndex maps a lowercased field path to the real field name for every
// field of t, recursing into nested facets (photo.cameraMake, ...). Names come
// from json tags, so it is backend-neutral; the query layer resolves KQL keys
// case-insensitively against it.
func FieldNameIndex(t reflect.Type, overrides map[string]FieldOpts) map[string]string {
	out := map[string]string{}
	var walk func(t reflect.Type, prefix string)
	walk = func(t reflect.Type, prefix string) {
		_ = walkFields(t, func(fi fieldInfo) error {
			name := fi.Name
			if prefix != "" {
				name = prefix + "." + fi.Name
			}
			out[strings.ToLower(name)] = name

			fieldType := overrides[name].Type
			if fieldType == "" {
				fieldType = inferType(fi.GoField.Type)
			}
			// recurse into nested facets; time.Time is a struct too but a leaf.
			if sub := structType(fi.GoField.Type); sub != nil && fieldType != TypeDatetime {
				walk(sub, name)
			}
			return nil
		})
	}
	walk(t, "")
	return out
}

// FieldTypes maps every field of t (json names, nested as parent.child) to
// its effective mapping type: the override, or the type inferred from the Go
// field.
func FieldTypes(t reflect.Type, overrides map[string]FieldOpts) map[string]string {
	out := map[string]string{}
	for key, goType := range collectFields(t, "") {
		out[key] = overrides[key].Type
		if out[key] == "" {
			out[key] = inferType(goType)
		}
	}
	return out
}
