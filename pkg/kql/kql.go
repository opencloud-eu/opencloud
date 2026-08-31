// Package kql provides the ability to work with kql queries.
package kql

import (
	"errors"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/ast"
)

// The operator node value definition
const (
	// BoolAND connect two nodes with "AND"
	BoolAND = "AND"
	// BoolOR connect two nodes with "OR"
	BoolOR = "OR"
	// BoolNOT connect two nodes with "NOT"
	BoolNOT = "NOT"
)

// Builder implements kql Builder interface
type Builder struct{}

// Build creates an ast.Ast based on a kql query
func (b Builder) Build(q string) (*ast.Ast, error) {
	f, err := Parse("", []byte(q))
	if err != nil {
		var list errList
		errors.As(err, &list)

		for _, listError := range list {
			var parserError *parserError
			switch {
			case errors.As(listError, &parserError):
				if parserError.Inner != nil {
					return nil, parserError.Inner
				}

				return nil, listError
			}
		}
	}

	return f.(*ast.Ast), nil
}

// timeNow mirrors time.Now by default, the only reason why this exists
// is to monkey patch it from the tests. See PatchTimeNow
var timeNow = time.Now

// PatchTimeNow pins the clock the natural language dates resolve against,
// so a test can hold "today" still while it runs
func PatchTimeNow(t func() time.Time) {
	timeNow = t
}
