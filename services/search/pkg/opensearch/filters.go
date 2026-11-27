package opensearch

import (
	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/aggregation"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/osu"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// Keys and bounds stay strings, OpenSearch reads them as the type of the field.
func aggregationFilterQueries(filters []*searchService.AggregationFilter) ([]osu.Builder, error) {
	out := make([]osu.Builder, 0, len(filters))
	for _, f := range filters {
		if _, err := aggregation.ParseFilter(f, search.FieldType(f.GetField())); err != nil {
			return nil, err
		}
		// a bool query of only should clauses needs one of them to match
		selected := osu.NewBoolQuery()
		for _, key := range f.GetTerms() {
			selected.Should(osu.NewTermQuery[string](f.GetField()).Value(key))
		}
		for _, r := range f.GetRanges() {
			q := osu.NewRangeQuery[string](f.GetField())
			if r.GetFrom() != "" {
				q = q.Gte(r.GetFrom())
			}
			if r.GetTo() != "" {
				q = q.Lt(r.GetTo())
			}
			selected.Should(q)
		}
		out = append(out, selected)
	}
	return out, nil
}
