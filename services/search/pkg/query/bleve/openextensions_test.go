package bleve

import (
	"testing"
	"time"

	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/stretchr/testify/assert"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	searchquery "github.com/opencloud-eu/opencloud/services/search/pkg/query"
)

func Test_compileOpenExtensions(t *testing.T) {
	inclusive, exclusive := true, false
	term := func(field, value string) query.Query {
		q := query.NewTermQuery(value)
		q.SetField(field)
		return q
	}
	number := func(field string, lo, hi *float64, loIn, hiIn *bool) query.Query {
		q := query.NewNumericRangeInclusiveQuery(lo, hi, loIn, hiIn)
		q.SetField(field)
		return q
	}
	date := func(field string, start, end time.Time, startIn, endIn *bool) query.Query {
		q := query.NewDateRangeInclusiveQuery(start, end, startIn, endIn)
		q.SetField(field)
		return q
	}
	boolean := func(field string, v bool) query.Query {
		q := query.NewBoolFieldQuery(v)
		q.SetField(field)
		return q
	}
	three := 3.0
	oct := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		node ast.Node
		want query.Query
	}{
		{
			name: "a plain string asks the lowercased sibling",
			node: &ast.StringNode{Key: "extensions.com.example.project.state", Value: "Open"},
			want: term("ext.com.example.project.state.@lower", "open"),
		},
		{
			name: "= asks the keyword sibling as written",
			node: &ast.StringNode{Key: "extensions.com.example.project.state", Value: "Open", Exact: true},
			want: term("ext.com.example.project.state.@keyword", "Open"),
		},
		{
			name: "the prefix is case-insensitive",
			node: &ast.StringNode{Key: "Extensions.com.example.project.State", Value: "x"},
			want: term("ext.com.example.project.State.@lower", "x"),
		},
		{
			name: "a numeric literal also asks the number sibling",
			node: &ast.StringNode{Key: "extensions.com.example.project.priority", Value: "3"},
			want: query.NewDisjunctionQuery([]query.Query{
				term("ext.com.example.project.priority.@lower", "3"),
				number("ext.com.example.project.priority.@number", &three, &three, &inclusive, &inclusive),
			}),
		},
		{
			name: "a boolean literal also asks the bool sibling",
			node: &ast.StringNode{Key: "extensions.com.example.project.done", Value: "true"},
			want: query.NewDisjunctionQuery([]query.Query{
				term("ext.com.example.project.done.@lower", "true"),
				boolean("ext.com.example.project.done.@bool", true),
			}),
		},
		{
			name: "a date-time literal also asks the date sibling",
			node: &ast.StringNode{Key: "extensions.com.example.project.due", Value: "2026-10-01T00:00:00Z"},
			want: query.NewDisjunctionQuery([]query.Query{
				term("ext.com.example.project.due.@lower", "2026-10-01t00:00:00z"),
				date("ext.com.example.project.due.@date", oct, oct, &inclusive, &inclusive),
			}),
		},
		{
			name: "a wildcard runs on the lowercased sibling",
			node: &ast.StringNode{Key: "extensions.com.example.project.state", Value: "Op*"},
			want: func() query.Query {
				q := query.NewWildcardQuery("op*")
				q.SetField("ext.com.example.project.state.@lower")
				return q
			}(),
		},
		{
			name: "a number range asks the number sibling",
			node: &ast.NumberNode{Key: "extensions.com.example.project.priority", Operator: &ast.OperatorNode{Value: ">"}, Value: 3},
			want: number("ext.com.example.project.priority.@number", &three, nil, &exclusive, nil),
		},
		{
			name: "a date range asks the date sibling",
			node: &ast.DateTimeNode{Key: "extensions.com.example.project.due", Operator: &ast.OperatorNode{Value: "<="}, Value: oct},
			want: date("ext.com.example.project.due.@date", time.Time{}, oct, nil, &inclusive),
		},
		{
			name: "a boolean node asks the bool sibling",
			node: &ast.BooleanNode{Key: "extensions.com.example.project.done", Value: false},
			want: boolean("ext.com.example.project.done.@bool", false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := compile(searchquery.Normalize(&ast.Ast{Nodes: []ast.Node{tt.node}}, searchquery.ResolveField))
			assert.NoError(t, err)
			// compile wraps a single leaf in a conjunction and hands a disjunction through
			want := tt.want
			if _, ok := want.(*query.DisjunctionQuery); !ok {
				want = query.NewConjunctionQuery([]query.Query{want})
			}
			assert.Equal(t, want, got)
		})
	}
}
