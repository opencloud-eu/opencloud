package query

import (
	"errors"
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

// IsValidationError says whether the query itself is at fault, which makes it a
// bad request and not an error of ours.
func IsValidationError(err error) bool {
	var (
		startsWithBinaryOperator *StartsWithBinaryOperatorError
		namedGroupInvalidNodes   *NamedGroupInvalidNodesError
		unsupportedTimeRange     *UnsupportedTimeRangeError
	)

	return errors.As(err, &startsWithBinaryOperator) ||
		errors.As(err, &namedGroupInvalidNodes) ||
		errors.As(err, &unsupportedTimeRange)
}
