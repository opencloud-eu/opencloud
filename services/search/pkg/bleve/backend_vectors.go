//go:build vectors

package bleve

import (
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"

	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
)

// addSemanticKNN attaches the semantic clause to the request: a KNN search on
// the _faiss sibling, pre-filtered by the regular query. With a lexical part
// (hybrid) the two rankings are fused via reciprocal rank fusion; without an
// explicit Score bleve would naively sum BM25 and vector scores, whose scales
// are not comparable. A purely semantic query keeps the raw similarity
// scores: they are comparable across the per-space searches the service layer
// merges, while rank-based fusion scores are not.
func addSemanticKNN(req *bleve.SearchRequest, vector []float32, filter query.Query, k int64, hybrid bool) error {
	req.AddKNNWithFilter("imageVector"+mapping.VectorIndexSuffix, vector, k, 1.0, filter)
	if hybrid {
		req.Score = "rrf"
		// the fusion window defaults to From+Size, which would truncate the
		// ranking depth to the page size
		window := req.From + req.Size
		if window < int(k) {
			window = int(k)
		}
		req.Params = &bleve.RequestParams{ScoreWindowSize: window}
	}
	return nil
}
