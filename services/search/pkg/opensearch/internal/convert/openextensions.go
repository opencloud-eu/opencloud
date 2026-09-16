package convert

import (
	"fmt"
	"strings"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/osu"
	"github.com/opencloud-eu/opencloud/services/search/pkg/query"
)

// mirrors the bleve compiler, see query.OpenExtensionEquality

func openExtensionStringQuery(n *ast.StringNode) osu.Builder {
	if strings.ContainsAny(n.Value, "*?") {
		sibling, value := mapping.SiblingLower, strings.ToLower(n.Value)
		if n.Exact {
			sibling, value = mapping.SiblingKeyword, n.Value
		}
		return osu.NewWildcardQuery(query.OpenExtensionField(n.Key, sibling)).Value(value)
	}

	var alternatives []osu.Builder
	for _, term := range query.OpenExtensionEquality(n.Key, n.Value, n.Exact) {
		switch term.Sibling {
		case mapping.SiblingNumber:
			alternatives = append(alternatives, osu.NewRangeQuery[float64](term.Field).Gte(term.Number).Lte(term.Number))
		case mapping.SiblingBool:
			alternatives = append(alternatives, osu.NewTermQuery[bool](term.Field).Value(term.Bool))
		case mapping.SiblingDate:
			alternatives = append(alternatives, osu.NewRangeQuery[time.Time](term.Field).Gte(term.Time).Lte(term.Time))
		default:
			alternatives = append(alternatives, osu.NewTermQuery[string](term.Field).Value(term.String))
		}
	}
	if len(alternatives) == 1 {
		return alternatives[0]
	}
	return osu.NewBoolQuery().Params(&osu.BoolQueryParams{MinimumShouldMatch: 1}).Should(alternatives...)
}

// openExtensionDateQuery: an equality is the one-instant range.
func openExtensionDateQuery(n *ast.DateTimeNode) (osu.Builder, error) {
	if n.Operator == nil {
		return nil, fmt.Errorf("date time node without operator: %w", ErrUnsupportedNodeType)
	}
	q := osu.NewRangeQuery[time.Time](query.OpenExtensionField(n.Key, mapping.SiblingDate))
	switch n.Operator.Value {
	case ">":
		return q.Gt(n.Value), nil
	case ">=":
		return q.Gte(n.Value), nil
	case "<":
		return q.Lt(n.Value), nil
	case "<=":
		return q.Lte(n.Value), nil
	default:
		return q.Gte(n.Value).Lte(n.Value), nil
	}
}
