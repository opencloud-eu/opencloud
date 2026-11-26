//go:build vectors

package mapping

import (
	"fmt"

	"github.com/blevesearch/bleve/v2"
	bleveMapping "github.com/blevesearch/bleve/v2/mapping"
)

// bleveVectorFieldMapping builds the bleve field mapping for a TypeVector
// field. Only compiled in with the vectors build tag (bleve gates its KNN
// support behind it); the !vectors twin drops the field from the mapping.
func bleveVectorFieldMapping(opts FieldOpts) (*bleveMapping.FieldMapping, error) {
	if opts.Dims <= 0 {
		return nil, fmt.Errorf("vector field needs Dims")
	}
	fm := bleve.NewVectorFieldMapping()
	fm.Dims = opts.Dims
	// cosine: bleve normalizes the vectors at index time itself
	fm.Similarity = "cosine"
	fm.VectorIndexOptimizedFor = "recall"
	return fm, nil
}
