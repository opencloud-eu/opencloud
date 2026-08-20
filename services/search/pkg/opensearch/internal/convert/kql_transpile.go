package convert

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/pkg/kql"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/osu"
)

func TranspileKQLToOpenSearch(nodes []ast.Node) (osu.Builder, error) {
	return kqlOpensearchTranspiler{}.Transpile(nodes)
}

type kqlOpensearchTranspiler struct{}

func (t kqlOpensearchTranspiler) Transpile(nodes []ast.Node) (osu.Builder, error) {
	q, err := t.transpile(nodes)
	if err != nil {
		return nil, err
	}

	return q, nil
}

func (t kqlOpensearchTranspiler) transpile(nodes []ast.Node) (osu.Builder, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes to compile")
	}

	if len(nodes) == 1 {
		builder, err := t.toBuilder(nodes[0])
		if err != nil {
			return nil, fmt.Errorf("failed to get builder for single node: %w", err)
		}
		return builder, nil
	}

	boolQueryParams := &osu.BoolQueryParams{}
	boolQuery := osu.NewBoolQuery().Params(boolQueryParams)
	boolQueryAdd := boolQuery.Must
	for i, node := range nodes {
		nextOp := t.getOperatorValueAt(nodes, i+1)
		prevOp := t.getOperatorValueAt(nodes, i-1)

		switch {
		case nextOp == kql.BoolOR:
			boolQueryAdd = boolQuery.Should
		case nextOp == kql.BoolAND:
			boolQueryAdd = boolQuery.Must
		case prevOp == kql.BoolNOT:
			boolQueryAdd = boolQuery.MustNot
		}

		builder, err := t.toBuilder(node)
		switch {
		// if the node is not known, we skip it, such as an operator node
		case errors.Is(err, ErrUnsupportedNodeType):
			continue
		case err != nil:
			return nil, fmt.Errorf("failed to get builder for node %T: %w", node, err)
		}

		if _, ok := node.(*ast.OperatorNode); ok {
			// operatorNodes are not builders, so we skip them
			continue
		}

		if nextOp == kql.BoolOR {
			// if there are should clauses, we set the minimum should match to 1
			boolQueryParams.MinimumShouldMatch = 1
		}

		boolQueryAdd(builder)
	}

	return boolQuery, nil
}

func (t kqlOpensearchTranspiler) getOperatorValueAt(nodes []ast.Node, i int) string {
	if i < 0 || i >= len(nodes) {
		return ""
	}

	if opn, ok := nodes[i].(*ast.OperatorNode); ok {
		return opn.Value
	}

	return ""
}

func (t kqlOpensearchTranspiler) toBuilder(node ast.Node) (osu.Builder, error) {
	switch node := node.(type) {
	case *ast.BooleanNode:
		return osu.NewTermQuery[bool](node.Key).Value(node.Value), nil
	case *ast.StringNode:
		return stringNodeQuery(node), nil
	case *ast.DateTimeNode:
		return dateTimeNodeQuery(node)
	case *ast.GroupNode:
		group, err := t.transpile(node.Nodes)
		if err != nil {
			return nil, fmt.Errorf("failed to build group: %w", err)
		}

		return group, nil
	}

	return nil, fmt.Errorf("%w: %T", ErrUnsupportedNodeType, node)
}

// stringNodeQuery picks the query a string node turns into.
func stringNodeQuery(node *ast.StringNode) osu.Builder {
	isWildcard := strings.Contains(node.Value, "*")

	switch {
	// Name: "*oo-bar", "*oo ba*", "*OO*"
	// Title: "*rterly rep*"
	// Tags: "*spaced tag*"
	case isWildcard && slices.Contains([]string{"Name", "Title", "Tags"}, node.Key):
		return osu.NewWildcardQuery(node.Key + ".keyword").
			Value(node.Value).
			Params(&osu.WildcardQueryParams{CaseInsensitive: true})
	// Path: "./foo*", MimeType: "*plain"
	case isWildcard:
		return osu.NewWildcardQuery(node.Key).Value(node.Value)
	// Tags: "foo-bar", "spaced tag", "FOO-BAR"
	case node.Key == "Tags":
		return osu.NewTermQuery[string](node.Key + ".keyword").
			Value(node.Value).
			Params(&osu.TermQueryParams{CaseInsensitive: true})
	// Name: "foo-bar", "foo bar"
	// Title: "quarterly report"
	// Content: "foo bar"
	case slices.Contains([]string{"Name", "Title", "Content"}, node.Key):
		return osu.NewMatchPhraseQuery(node.Key).Query(node.Value)
	// Path: "./foo bar", MimeType: "text/plain"
	default:
		return osu.NewTermQuery[string](node.Key).Value(node.Value)
	}
}

// dateTimeNodeQuery turns a date time node into a range query.
func dateTimeNodeQuery(node *ast.DateTimeNode) (osu.Builder, error) {
	if node.Operator == nil {
		return nil, fmt.Errorf("date time node without operator: %w", ErrUnsupportedNodeType)
	}

	query := osu.NewRangeQuery[time.Time](node.Key)

	switch node.Operator.Value {
	case ">":
		return query.Gt(node.Value), nil
	case ">=":
		return query.Gte(node.Value), nil
	case "<":
		return query.Lt(node.Value), nil
	case "<=":
		return query.Lte(node.Value), nil
	}

	return nil, fmt.Errorf("unsupported operator %s for date time node: %w", node.Operator.Value, ErrUnsupportedNodeType)
}
