package convert

import (
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/kql"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/osu"
	"github.com/opencloud-eu/opencloud/services/search/pkg/query"
)

var (
	ErrUnsupportedNodeType = fmt.Errorf("unsupported node type")
)

func KQLToOpenSearchBoolQuery(kqlQuery string) (*osu.BoolQuery, error) {
	return KQLToOpenSearchBoolQueryWithFilters(kqlQuery, nil)
}

// KQLToOpenSearchBoolQueryWithFilters compiles the query together with decoded
// aggregation filters, which are ANDed in as exact case-sensitive matches.
func KQLToOpenSearchBoolQueryWithFilters(kqlQuery string, filters []string) (*osu.BoolQuery, error) {
	// shared lowering (field resolution, media-type expansion, value lowercasing)
	// plus the filters, forced to exact case-sensitive matches, ANDed in.
	kqlAst, err := query.MergeFilters(kql.Builder{}, kqlQuery, filters)
	if err != nil {
		return nil, err
	}

	builder, err := TranspileKQLToOpenSearch(kqlAst.Nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to compile query: %w", err)
	}

	if q, ok := builder.(*osu.BoolQuery); !ok {
		return osu.NewBoolQuery().Must(builder), nil
	} else {
		return q, nil
	}
}
