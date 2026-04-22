// Package test provides shared test primitives for ast testing.
package test

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/opencloud-eu/opencloud/pkg/ast"
)

// DiffAst returns a human-readable report of the differences between two values
// by default it ignores every ast node Base field.
func DiffAst(x, y any, opts ...cmp.Option) string {
	return cmp.Diff(
		x,
		y,
		append(
			opts,
			cmpopts.IgnoreFields(ast.Ast{}, "Base"),
			cmpopts.IgnoreFields(ast.StringNode{}, "Base"),
			cmpopts.IgnoreFields(ast.OperatorNode{}, "Base"),
			cmpopts.IgnoreFields(ast.GroupNode{}, "Base"),
			cmpopts.IgnoreFields(ast.BooleanNode{}, "Base"),
			cmpopts.IgnoreFields(ast.DateTimeNode{}, "Base"),
		)...,
	)
}
