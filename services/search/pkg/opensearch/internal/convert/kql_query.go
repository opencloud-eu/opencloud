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
	q, _, err := KQLToOpenSearchBoolQueryWithSemantic(kqlQuery, false)
	return q, err
}

// KQLToOpenSearchBoolQueryWithSemantic additionally splits the semantic
// free-text clause off the parsed tree (see query.ExtractSemantic) when
// withSemantic is set. A purely semantic query yields an empty bool query.
func KQLToOpenSearchBoolQueryWithSemantic(kqlQuery string, withSemantic bool) (*osu.BoolQuery, string, error) {
	kqlAst, err := kql.Builder{}.Build(kqlQuery)
	if err != nil {
		return nil, "", fmt.Errorf("failed to build query: %w", err)
	}

	var semanticText string
	if withSemantic {
		semanticText = query.ExtractSemantic(kqlAst)
		if semanticText != "" && len(kqlAst.Nodes) == 0 {
			return osu.NewBoolQuery(), semanticText, nil
		}
	}

	// shared lowering: field resolution, media-type expansion, value lowercasing.
	kqlAst = query.Normalize(kqlAst, query.ResolveField)

	builder, err := TranspileKQLToOpenSearch(kqlAst.Nodes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to compile query: %w", err)
	}

	if q, ok := builder.(*osu.BoolQuery); !ok {
		return osu.NewBoolQuery().Must(builder), semanticText, nil
	} else {
		return q, semanticText, nil
	}
}
