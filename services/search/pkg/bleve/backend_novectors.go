//go:build !vectors

package bleve

import (
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"

	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
)

// addSemanticKNN rejects semantic clauses: this build carries no bleve vector
// support (see the vectors build tag).
func addSemanticKNN(*bleve.SearchRequest, []float32, query.Query, int64, bool) error {
	return errtypes.BadRequest("semantic search is not supported by this build")
}
