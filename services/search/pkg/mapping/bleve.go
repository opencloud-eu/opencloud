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
// The returned mapping references analyzer names that the caller must register
// on the enclosing IndexMapping (IndexMapping.Validate catches missing ones).
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
				doc.AddFieldMappingsAt(fi.Name+LowercaseSuffix, searchSibling(base))
			}
			if fieldType == TypeKeyword && opts.wordBroken() {
				words := searchSibling(base)
				words.Analyzer = WordsAnalyzer
				doc.AddFieldMappingsAt(fi.Name+WordsSuffix, words)
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

// searchSibling derives a search-only shadow of a keyword/path field from its
// base mapping (the _lowercase and _words siblings): indexed but never stored,
// kept out of _all, and without doc values, since the case-preserved base field
// is what we return and aggregate on.
func searchSibling(base *bleveMapping.FieldMapping) *bleveMapping.FieldMapping {
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
			fm.Analyzer = WordsAnalyzer
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
