//go:build !vectors

package bleve

import (
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
)

// prepareVectorFields drops vector fields from the document: this build cannot
// search them, so storing them would only cost space. Enabling vectors later
// requires a reindex anyway (the faiss part of the index was never populated).
func prepareVectorFields(doc map[string]any, overrides map[string]mapping.FieldOpts) {
	for key, opts := range overrides {
		if opts.Type == mapping.TypeVector {
			delete(doc, key)
		}
	}
}
