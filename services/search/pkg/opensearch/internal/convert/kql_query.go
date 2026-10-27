package convert

import (
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/osu"
	"github.com/opencloud-eu/opencloud/services/search/pkg/query"
)

var (
	ErrUnsupportedNodeType = fmt.Errorf("unsupported node type")
)

func KQLToOpenSearchBoolQuery(kqlAst *ast.Ast) (*osu.BoolQuery, error) {
	// shared lowering: field resolution, media-type expansion, value lowercasing.
	kqlAst = query.Normalize(kqlAst, query.ResolveField)

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
