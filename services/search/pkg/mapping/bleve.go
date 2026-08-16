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
// The returned mapping references the "fulltext" analyzer for Fulltext fields;
// the caller registers it on the enclosing IndexMapping.
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

		if fieldType == TypeKeyword || fieldType == TypePath {
			// bleve has no path tokenizer, so a path is a plain keyword here.
			base := bleveKeywordMapping(fieldType, opts)
			doc.AddFieldMappingsAt(fi.Name, base)
			if opts.caseInsensitive() {
				doc.AddFieldMappingsAt(fi.Name+LowercaseSuffix, lowercaseSibling(base))
			}
			return nil
		}

		if fieldType == TypeVector {
			// The field itself is stored-only: it round-trips through the
			// regular deserialization (bleve cannot return vector-typed fields
			// as stored fields, see VectorIndexSuffix). Both builds.
			nf := bleve.NewNumericFieldMapping()
			nf.Index = false
			nf.Store = true
			nf.IncludeInAll = false
			nf.DocValues = false
			doc.AddFieldMappingsAt(fi.Name, nf)
			// The _faiss sibling carries the searchable vector; without the
			// vectors build tag bleve cannot index it and the field is left
			// out of the mapping entirely (nil, nil from the !vectors twin).
			fm, err := bleveVectorFieldMapping(opts)
			if err != nil {
				return fmt.Errorf("mapping: field %q: %w", key, err)
			}
			if fm != nil {
				doc.AddFieldMappingsAt(fi.Name+VectorIndexSuffix, fm)
			}
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

// bleveKeywordMapping is a case-preserving keyword field; path fields stay out
// of _all by default.
func bleveKeywordMapping(fieldType string, opts FieldOpts) *bleveMapping.FieldMapping {
	fm := bleve.NewKeywordFieldMapping()
	switch {
	case opts.IncludeInAll != nil:
		fm.IncludeInAll = *opts.IncludeInAll
	case fieldType == TypePath:
		fm.IncludeInAll = false
	}
	return fm
}

// lowercaseSibling derives the lowercased shadow of a keyword/path field from its
// base mapping: used only for case-insensitive matching, so indexed but never
// stored, kept out of _all, and without doc values, since the case-preserved base
// field is what we return and aggregate on.
func lowercaseSibling(base *bleveMapping.FieldMapping) *bleveMapping.FieldMapping {
	fm := *base
	fm.Store = false
	fm.IncludeInAll = false
	fm.DocValues = false
	return &fm
}

func bleveFieldMapping(fieldType string, opts FieldOpts) (*bleveMapping.FieldMapping, error) {
	switch fieldType {
	case TypeWildcard:
		// bleve has no wildcard type; fall back to keyword-ish text.
		fieldType = TypeKeyword
		fallthrough
	case TypeKeyword, TypeFulltext:
		fm := bleve.NewTextFieldMapping()
		if fieldType == TypeFulltext {
			fm.Analyzer = "fulltext"
		}
		switch {
		case opts.IncludeInAll != nil:
			fm.IncludeInAll = *opts.IncludeInAll
		case fieldType == TypeFulltext:
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
