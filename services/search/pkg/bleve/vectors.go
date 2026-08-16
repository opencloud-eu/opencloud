//go:build vectors

package bleve

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
)

// prepareVectorFields copies vector fields to their _faiss siblings, which
// carry the searchable vector in this build (see mapping.VectorIndexSuffix).
func prepareVectorFields(doc map[string]any, overrides map[string]mapping.FieldOpts) {
	mapping.AddVectorIndexSiblings(doc, overrides)
}
