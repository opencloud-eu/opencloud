package mapping

import (
	"fmt"
	"reflect"
)

// OpenSearchBuildMapping builds the OpenSearch "properties" map (the value
// of mappings.properties) for type t by walking the struct via reflection.
// Field names come from json tags; overrides are keyed by those names.
//
// The returned map contains plain JSON-friendly values (strings, bools,
// nested maps) and can be marshalled directly.
func OpenSearchBuildMapping(t reflect.Type, overrides map[string]FieldOpts) (map[string]any, error) {
	return buildOpenSearchProperties(t, overrides, "")
}

func buildOpenSearchProperties(t reflect.Type, overrides map[string]FieldOpts, prefix string) (map[string]any, error) {
	props := map[string]any{}
	err := walkFields(t, func(fi fieldInfo) error {
		key := fi.Name
		if prefix != "" {
			key = prefix + "." + fi.Name
		}
		opts := overrides[key]
		fieldType := opts.Type
		if fieldType == "" {
			fieldType = inferType(fi.GoField.Type)
		}

		if fieldType == TypeObject {
			sub := structType(fi.GoField.Type)
			if sub == nil {
				return fmt.Errorf("mapping: object type on non-struct field %q", key)
			}
			subProps, err := buildOpenSearchProperties(sub, overrides, key)
			if err != nil {
				return err
			}
			props[fi.Name] = map[string]any{"properties": subProps}
			return nil
		}

		if fieldType == TypeGeopoint {
			// Keep the facet object, add a sibling _geopoint field (see GeopointSuffix).
			sub := structType(fi.GoField.Type)
			if sub == nil {
				return fmt.Errorf("mapping: geopoint type on non-struct field %q", key)
			}
			subProps, err := buildOpenSearchProperties(sub, overrides, key)
			if err != nil {
				return err
			}
			props[fi.Name] = map[string]any{"properties": subProps}
			props[fi.Name+GeopointSuffix] = map[string]any{"type": "geo_point"}
			return nil
		}

		fm, err := openSearchFieldMapping(fieldType, opts, fi.GoField.Type)
		if err != nil {
			return fmt.Errorf("mapping: field %q: %w", key, err)
		}
		props[fi.Name] = fm
		return nil
	})
	return props, err
}

func openSearchFieldMapping(fieldType string, opts FieldOpts, goType reflect.Type) (map[string]any, error) {
	switch fieldType {
	case TypeKeyword:
		m := map[string]any{"type": "keyword"}
		if opts.Analyzer != "" {
			m["type"] = "text"
			m["analyzer"] = opts.Analyzer
		}
		return m, nil
	case TypeFulltext:
		m := map[string]any{
			"type":        "text",
			"term_vector": "with_positions_offsets",
		}
		if opts.Analyzer != "" {
			m["analyzer"] = opts.Analyzer
		}
		return m, nil
	case TypePath:
		m := map[string]any{"type": "text"}
		if opts.Analyzer != "" {
			m["analyzer"] = opts.Analyzer
		} else {
			m["analyzer"] = "path_hierarchy"
		}
		return m, nil
	case TypeWildcard:
		// OpenSearch stores wildcard fields with doc_values=false by
		// default, so emit it explicitly to keep local and remote
		// mappings in sync for the Apply comparison.
		return map[string]any{"type": "wildcard", "doc_values": false}, nil
	case TypeNumeric:
		return map[string]any{"type": openSearchNumericType(goType)}, nil
	case TypeBool:
		return map[string]any{"type": "boolean"}, nil
	case TypeDatetime:
		return map[string]any{"type": "date"}, nil
	case TypeGeopoint:
		return map[string]any{"type": "geo_point"}, nil
	case "":
		return nil, fmt.Errorf("no type inferred and no override")
	}
	return nil, fmt.Errorf("unsupported type %q", fieldType)
}

// openSearchNumericType maps a Go numeric type to an OpenSearch numeric
// field type.
func openSearchNumericType(t reflect.Type) string {
	t = deref(t)
	switch t.Kind() {
	case reflect.Float32:
		return "float"
	case reflect.Float64:
		return "double"
	case reflect.Int8, reflect.Uint8, reflect.Int16, reflect.Uint16:
		return "short"
	case reflect.Int32, reflect.Uint32:
		return "integer"
	}
	return "long"
}
