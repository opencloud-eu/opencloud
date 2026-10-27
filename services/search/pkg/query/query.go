// Package query provides functions to work with the different search query flavours.
package query

import "github.com/opencloud-eu/opencloud/pkg/ast"

// Compiler is the interface that wraps the basic Compile method.
type Compiler[T any] interface {
	Compile(ast *ast.Ast) (T, error)
}
