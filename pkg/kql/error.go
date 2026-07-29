package kql

import (
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/ast"
)

// StartsWithBinaryOperatorError records an error and the operation that caused it.
type StartsWithBinaryOperatorError struct {
	Node *ast.OperatorNode
}

func (e StartsWithBinaryOperatorError) Error() string {
	return "the expression can't begin from a binary operator: '" + e.Node.Value + "'"
}

// NamedGroupInvalidNodesError records an error and the operation that caused it.
type NamedGroupInvalidNodesError struct {
	Node ast.Node
}

func (e NamedGroupInvalidNodesError) Error() string {
	return fmt.Errorf(
		"'%T' - '%v' - '%v' is not valid",
		e.Node,
		ast.NodeKey(e.Node),
		ast.NodeValue(e.Node),
	).Error()
}

// UnsupportedTimeRangeError records an error and the value that caused it.
type UnsupportedTimeRangeError struct {
	Value any
}

func (e UnsupportedTimeRangeError) Error() string {
	return fmt.Sprintf("unable to convert '%v' to a time range", e.Value)
}

// IsValidationError reports whether err is one of the KQL parse/validation
// errors produced by this package.
func IsValidationError(err error) bool {
	switch err.(type) {
	case *StartsWithBinaryOperatorError, *NamedGroupInvalidNodesError, *UnsupportedTimeRangeError:
		return true
	}
	return false
}
