package bleve

import (
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"

	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/aggregation"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// fieldQuery is a bleve query on one field.
type fieldQuery interface {
	query.Query
	SetField(string)
	SetBoost(float64)
}

// aggregationFilterQueries builds one query per filter, matching any of its
// buckets the way the aggregation counted them: a key is the exact value of
// the field, a range is from-inclusive and to-exclusive. A filter narrows and
// does not rank: bleve has no filter context, a boost of zero keeps the
// filters out of the score.
func aggregationFilterQueries(filters []*searchService.AggregationFilter) ([]query.Query, error) {
	inclusive, exclusive := true, false
	out := make([]query.Query, 0, len(filters))
	for _, f := range filters {
		buckets, err := aggregation.ParseFilter(f, search.FieldType(f.GetField()))
		if err != nil {
			return nil, err
		}
		var selected []query.Query
		add := func(q fieldQuery) {
			q.SetField(f.GetField())
			q.SetBoost(0)
			selected = append(selected, q)
		}
		for _, term := range buckets.Terms {
			switch {
			case term.Number != nil:
				add(bleve.NewNumericRangeInclusiveQuery(term.Number, term.Number, &inclusive, &inclusive))
			case term.Bool != nil:
				add(bleve.NewBoolFieldQuery(*term.Bool))
			default:
				add(bleve.NewTermQuery(term.Key))
			}
		}
		for _, r := range buckets.Ranges.Numeric {
			add(bleve.NewNumericRangeInclusiveQuery(r.From, r.To, &inclusive, &exclusive))
		}
		for _, r := range buckets.Ranges.Dates {
			add(bleve.NewDateRangeInclusiveQuery(r.From, r.To, &inclusive, &exclusive))
		}
		out = append(out, bleve.NewDisjunctionQuery(selected...))
	}
	return out, nil
}
