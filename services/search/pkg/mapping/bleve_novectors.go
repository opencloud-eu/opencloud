//go:build !vectors

package mapping

import (
	bleveMapping "github.com/blevesearch/bleve/v2/mapping"
)

// bleveVectorFieldMapping is the !vectors twin: bleve has no vector support in
// this build, so TypeVector fields are left out of the mapping (nil, nil means
// "skip the field").
func bleveVectorFieldMapping(FieldOpts) (*bleveMapping.FieldMapping, error) {
	return nil, nil
}
