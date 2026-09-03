// Package query provides functions to work with the different search query flavours.
package query

import "github.com/opencloud-eu/opencloud/pkg/ast"

// Builder is the interface that wraps the basic Build method.
type Builder interface {
	Build(qs string) (*ast.Ast, error)
}

// Compiler is the interface that wraps the basic Compile method.
type Compiler[T any] interface {
	Compile(ast *ast.Ast) (T, error)
}

// Creator is the interface that wraps the basic Create method.
type Creator[T any] interface {
	Create(qs string) (T, error)

	// CreateWithSemantic splits the semantic free-text clause off the parsed
	// query (see ExtractSemantic). A purely semantic query yields the zero
	// value for the compiled query.
	CreateWithSemantic(qs string) (T, string, error)
}
