package kql

import (
	"errors"
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/ast"
)

// DanglingOperatorError records an error and the operator that caused it.
type DanglingOperatorError struct {
	Node *ast.OperatorNode
}

func (e DanglingOperatorError) Error() string {
	return "the operator '" + e.Node.Value + "' is missing an operand"
}

// SyntaxError records an error and the position that caused it.
type SyntaxError struct {
	line, col int
	Inner     error
}

func (e SyntaxError) Error() string {
	if e.line == 0 {
		return "invalid query syntax"
	}
	return fmt.Sprintf("invalid query syntax at %d:%d", e.line, e.col)
}

func (e SyntaxError) Unwrap() error {
	return e.Inner
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

func isGrammarActionError(err error) bool {
	var (
		danglingOperator       *DanglingOperatorError
		namedGroupInvalidNodes *NamedGroupInvalidNodesError
		unsupportedTimeRange   *UnsupportedTimeRangeError
	)

	return errors.As(err, &danglingOperator) ||
		errors.As(err, &namedGroupInvalidNodes) ||
		errors.As(err, &unsupportedTimeRange)
}

// IsValidationError reports whether err is one of the KQL parse/validation
// errors produced by this package, i.e. the query itself is at fault and the
// caller should treat it as a bad request.
func IsValidationError(err error) bool {
	var (
		danglingOperator       *DanglingOperatorError
		syntax                 *SyntaxError
		namedGroupInvalidNodes *NamedGroupInvalidNodesError
		unsupportedTimeRange   *UnsupportedTimeRangeError
	)

	return errors.As(err, &danglingOperator) ||
		errors.As(err, &syntax) ||
		errors.As(err, &namedGroupInvalidNodes) ||
		errors.As(err, &unsupportedTimeRange)
}
