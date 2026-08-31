package convert

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
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
	case *ast.NumberNode:
		return numberNodeQuery(node)
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
	isWildcard := strings.ContainsAny(node.Value, "*?")

	switch {
	// Name: "*oo-bar", "*oo ba*", "*OO*"
	// Title: "*rterly rep*"
	// Tags: "*spaced tag*"
	case isWildcard && slices.Contains([]string{"Name", "Title"}, node.Key):
		patterns := []osu.Builder{wildcardOn(node.Key+".wildcard", node.Value)}
		if !strings.HasSuffix(node.Value, "*") {
			patterns = append(patterns, wildcardOn(node.Key+".wildcard", node.Value+".*"))
		}

		return osu.NewBoolQuery().
			Params(&osu.BoolQueryParams{MinimumShouldMatch: 1}).
			Should(patterns...)
	// Tags: "*foo*", "*paced ta*"
	case isWildcard && node.Key == "Tags":
		return wildcardOn(node.Key+".wildcard", node.Value)
	// Path: "./foo*", MimeType: "*plain"
	case isWildcard:
		return osu.NewWildcardQuery(node.Key).Value(node.Value)
	// Name: =new, Title: ="quarterly report"
	case node.Exact && slices.Contains([]string{"Name", "Title"}, node.Key):
		return osu.NewTermQuery[string](node.Key + ".wildcard").
			Value(node.Value).
			Params(&osu.TermQueryParams{CaseInsensitive: true})
	// Tags: "foo-bar", "spaced tag", "FOO-BAR"
	case node.Key == "Tags":
		return osu.NewTermQuery[string](node.Key + ".wildcard").
			Value(node.Value).
			Params(&osu.TermQueryParams{CaseInsensitive: true})
	// Name: "foo-bar", "foo bar"
	// Title: "quarterly report"
	// Content: "foo bar"
	case slices.Contains([]string{"Name", "Title", "Content"}, node.Key):
		return osu.NewMatchPhraseQuery(node.Key).Query(node.Value)
	// Size: "42", Type: "1"
	case slices.Contains([]string{"Size", "Type"}, node.Key):
		number, err := strconv.ParseInt(node.Value, 10, 64)
		if err != nil {
			return osu.NewMatchNoneQuery()
		}

		return osu.NewTermQuery[int64](node.Key).Value(number)
	// Path: "./foo bar/", the hierarchy tokens carry no trailing slash
	case node.Key == "Path":
		return osu.NewTermQuery[string](node.Key).Value(strings.TrimSuffix(node.Value, "/"))
	// Hidden: "TRUE" arrives lowered, anything that is no bool matches nothing
	case node.Key == "Hidden":
		value, err := strconv.ParseBool(node.Value)
		if err != nil {
			return osu.NewMatchNoneQuery()
		}

		return osu.NewTermQuery[bool](node.Key).Value(value)
	// MimeType: "text/plain"
	default:
		return osu.NewTermQuery[string](node.Key).Value(node.Value)
	}
}

func numberNodeQuery(node *ast.NumberNode) (osu.Builder, error) {
	if node.Operator == nil {
		return nil, fmt.Errorf("number node without operator: %w", ErrUnsupportedNodeType)
	}

	if !slices.Contains([]string{"Size", "Type"}, node.Key) {
		return osu.NewMatchNoneQuery(), nil
	}

	query := osu.NewRangeQuery[float64](node.Key)

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

	return nil, fmt.Errorf("unsupported operator %s for number node: %w", node.Operator.Value, ErrUnsupportedNodeType)
}

func wildcardOn(field, value string) osu.Builder {
	return osu.NewWildcardQuery(field).
		Value(value).
		Params(&osu.WildcardQueryParams{CaseInsensitive: true})
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
