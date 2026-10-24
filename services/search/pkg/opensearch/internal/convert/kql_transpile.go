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
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/osu"
	"github.com/opencloud-eu/opencloud/services/search/pkg/query"
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

		// A preceding NOT negates this node regardless of what follows (NOT x AND y
		// is (NOT x) AND y), so it must win over nextOp. The prevOp AND/OR cases
		// give the right operand its own bucket instead of inheriting the previous
		// one, which matters right after a NOT (its MustNot must not carry over).
		switch {
		case prevOp == kql.BoolNOT:
			boolQueryAdd = boolQuery.MustNot
		case nextOp == kql.BoolOR:
			boolQueryAdd = boolQuery.Should
		case nextOp == kql.BoolAND:
			boolQueryAdd = boolQuery.Must
		case prevOp == kql.BoolOR:
			boolQueryAdd = boolQuery.Should
		case prevOp == kql.BoolAND:
			boolQueryAdd = boolQuery.Must
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

		if nextOp == kql.BoolOR || prevOp == kql.BoolOR {
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
		// hidden takes bool words only; anything else matches nothing
		if node.Key == "Hidden" {
			b, err := strconv.ParseBool(node.Value)
			if err != nil {
				return osu.NewMatchNoneQuery(), nil
			}
			return osu.NewTermQuery[bool](node.Key).Value(b), nil
		}

		field, value := node.Key, node.Value
		if query.FieldIsPath(node.Key) {
			value = strings.TrimSuffix(value, "/")
		}
		if node.CaseInsensitive {
			field += mapping.LowercaseSuffix
			value = strings.ToLower(value)
		}

		if isWildcard := strings.ContainsAny(value, "*?"); isWildcard {
			// a wildcard on a word-broken field forgives a missing extension:
			// *report also matches Report.txt
			if query.FieldIsWordBroken(node.Key) && !strings.HasSuffix(value, "*") {
				return osu.NewBoolQuery().
					Params(&osu.BoolQueryParams{MinimumShouldMatch: 1}).
					Should(
						osu.NewWildcardQuery(field).Value(value),
						osu.NewWildcardQuery(field).Value(value+".*"),
					), nil
			}
			return osu.NewWildcardQuery(field).Value(value), nil
		}

		// = matches the whole value, on the lowercased sibling for
		// case-insensitive fields
		if node.Exact {
			return osu.NewTermQuery[string](field).Value(value), nil
		}

		// a word-broken field matches the value as a phrase of its words on the
		// _words sibling, whose analyzer lowercases; wildcards stay on _lowercase
		if query.FieldIsWordBroken(node.Key) {
			return osu.NewMatchPhraseQuery(node.Key + mapping.WordsSuffix).Query(node.Value), nil
		}

		if query.FieldIsFulltext(node.Key) {
			return osu.NewMatchPhraseQuery(field).Query(value), nil
		}

		// a path value is a single term in the path_hierarchy token stream; a
		// phrase match would analyze the query into its path prefixes and match
		// everything under the root, so paths with spaces must stay term queries.
		if query.FieldIsPath(node.Key) {
			return osu.NewTermQuery[string](field).Value(value), nil
		}

		totalTerms := strings.Split(value, " ")
		isSingleTerm := len(totalTerms) == 1
		isMultiTerm := len(totalTerms) >= 1
		switch {
		case isSingleTerm:
			return osu.NewTermQuery[string](field).Value(value), nil
		case isMultiTerm:
			return osu.NewMatchPhraseQuery(field).Query(value), nil
		}

		return nil, fmt.Errorf("unsupported string node value: %s", value)
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
