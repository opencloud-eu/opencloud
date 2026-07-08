package mapping

import (
	"fmt"
	"reflect"

	"github.com/blevesearch/bleve/v2"
	bleveMapping "github.com/blevesearch/bleve/v2/mapping"
)

// BleveBuildMapping builds a bleve DocumentMapping for t by walking the
// struct via reflection. Field names come from json tags; overrides are
// keyed by those names (or dotted paths for nested fields).
//
// The returned mapping references analyzer names (Analyzer field on the
// FieldOpts, plus "fulltext" / "path_hierarchy" for the corresponding Types);
// the caller must register every referenced analyzer on the enclosing
// IndexMapping (IndexMapping.Validate catches missing ones).
func BleveBuildMapping(t reflect.Type, overrides map[string]FieldOpts) (*bleveMapping.DocumentMapping, error) {
	return buildBleveDocMapping(t, overrides, "")
}

func buildBleveDocMapping(t reflect.Type, overrides map[string]FieldOpts, prefix string) (*bleveMapping.DocumentMapping, error) {
	doc := bleve.NewDocumentMapping()
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
			subDoc, err := buildBleveDocMapping(sub, overrides, key)
			if err != nil {
				return err
			}
			doc.AddSubDocumentMapping(fi.Name, subDoc)
			return nil
		}

		if fieldType == TypeGeopoint {
			// Keep the facet object, add a sibling _geopoint field (see GeopointSuffix).
			sub := structType(fi.GoField.Type)
			if sub == nil {
				return fmt.Errorf("mapping: geopoint type on non-struct field %q", key)
			}
			subDoc, err := buildBleveDocMapping(sub, overrides, key)
			if err != nil {
				return err
			}
			doc.AddSubDocumentMapping(fi.Name, subDoc)
			doc.AddFieldMappingsAt(fi.Name+GeopointSuffix, bleve.NewGeoPointFieldMapping())
			return nil
		}

		fm, err := bleveFieldMapping(fieldType, opts)
		if err != nil {
			return fmt.Errorf("mapping: field %q: %w", key, err)
		}
		doc.AddFieldMappingsAt(fi.Name, fm)
		return nil
	})
	return doc, err
}

func bleveFieldMapping(fieldType string, opts FieldOpts) (*bleveMapping.FieldMapping, error) {
	switch fieldType {
	case TypeWildcard:
		// bleve has no wildcard type; fall back to keyword-ish text.
		fieldType = TypeKeyword
		fallthrough
	case TypeKeyword, TypeFulltext, TypePath:
		fm := bleve.NewTextFieldMapping()
		switch {
		case opts.Analyzer != "":
			fm.Analyzer = opts.Analyzer
		case fieldType == TypeFulltext:
			fm.Analyzer = "fulltext"
		case fieldType == TypePath:
			fm.Analyzer = "path_hierarchy"
		}
		switch {
		case opts.IncludeInAll != nil:
			fm.IncludeInAll = *opts.IncludeInAll
		case fieldType == TypeFulltext, fieldType == TypePath:
			fm.IncludeInAll = false
		}
		return fm, nil
	case TypeNumeric:
		return bleve.NewNumericFieldMapping(), nil
	case TypeBool:
		return bleve.NewBooleanFieldMapping(), nil
	case TypeDatetime:
		return bleve.NewDateTimeFieldMapping(), nil
	case TypeGeopoint:
		return bleve.NewGeoPointFieldMapping(), nil
	case "":
		return nil, fmt.Errorf("no type inferred and no override")
	}
	return nil, fmt.Errorf("unsupported type %q", fieldType)
}
