package bleve

import (
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/v2/document"
	bleveMapping "github.com/blevesearch/bleve/v2/mapping"
	index "github.com/blevesearch/bleve_index_api"

	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
)

// Open extensions bypass the mapping: it is fixed when the index is created
// and can only infer a type from a Go value. The typed siblings are built as
// document fields and indexed with IndexAdvanced, so the stored mapping never
// changes. The stored values are stored, not indexed, so Move, Delete and
// Restore can re-index a hit without losing them.

const openExtensionIndexOptions = index.IndexField | index.DocValues

func addOpenExtensionFields(doc *document.Document, im bleveMapping.IndexMapping, values map[string]string) error {
	analyzer := im.AnalyzerNamed(keyword.Name)
	for key, raw := range values {
		doc.AddField(document.NewTextFieldCustom(mapping.OpenExtensionsStoredField+"."+key, nil, []byte(raw), index.StoreField, nil))
	}
	for _, leaf := range mapping.OpenExtensionLeaves(values) {
		positions := func(i, n int) []uint64 {
			if n == 1 {
				return nil
			}
			return []uint64{uint64(i)} //nolint:gosec // a loop index is never negative
		}
		switch leaf.Sibling {
		case mapping.SiblingKeyword, mapping.SiblingLower:
			for i, s := range leaf.Strings {
				doc.AddField(document.NewTextFieldCustom(leaf.Field, positions(i, len(leaf.Strings)), []byte(s), openExtensionIndexOptions, analyzer))
			}
		case mapping.SiblingNumber:
			for i, f := range leaf.Numbers {
				doc.AddField(document.NewNumericFieldWithIndexingOptions(leaf.Field, positions(i, len(leaf.Numbers)), f, openExtensionIndexOptions))
			}
		case mapping.SiblingBool:
			for i, b := range leaf.Bools {
				doc.AddField(document.NewBooleanFieldWithIndexingOptions(leaf.Field, positions(i, len(leaf.Bools)), b, openExtensionIndexOptions))
			}
		case mapping.SiblingDate:
			for i, t := range leaf.Times {
				f, err := document.NewDateTimeFieldWithIndexingOptions(leaf.Field, positions(i, len(leaf.Times)), t, time.RFC3339Nano, openExtensionIndexOptions)
				if err != nil {
					return err
				}
				doc.AddField(f)
			}
		case mapping.SiblingGeo:
			doc.AddField(document.NewGeoPointFieldWithIndexingOptions(leaf.Field, nil, leaf.Geo.Longitude, leaf.Geo.Latitude, openExtensionIndexOptions))
		}
	}
	return nil
}

func openExtensionsFromHit(fields map[string]any) map[string]string {
	var values map[string]string
	prefix := mapping.OpenExtensionsStoredField + "."
	for field, value := range fields {
		if !strings.HasPrefix(field, prefix) {
			continue
		}
		raw, ok := value.(string)
		if !ok {
			continue
		}
		if values == nil {
			values = map[string]string{}
		}
		values[field[len(prefix):]] = raw
	}
	return values
}
