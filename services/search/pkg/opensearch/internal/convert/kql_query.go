package convert

import (
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/kql"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/osu"
)

var (
	ErrUnsupportedNodeType = fmt.Errorf("unsupported node type")
)

func KQLToOpenSearchBoolQuery(kqlQuery string) (*osu.BoolQuery, error) {
	kqlAst, err := kql.Builder{}.Build(kqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	kqlNodes, err := ExpandKQL(kqlAst.Nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to expand KQL AST nodes: %w", err)
	}

	builder, err := TranspileKQLToOpenSearch(kqlNodes)
	if err != nil {
		return nil, fmt.Errorf("failed to compile query: %w", err)
	}

	if q, ok := builder.(*osu.BoolQuery); !ok {
		return osu.NewBoolQuery().Must(builder), nil
	} else {
		return q, nil
	}
}
