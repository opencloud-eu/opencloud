package kql

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

// GeoFieldError records a geo predicate that was used on a field that is not
// indexed as a geopoint.
type GeoFieldError struct {
	Key string
}

func (e GeoFieldError) Error() string {
	return fmt.Sprintf("geo predicate on non-geo field %q", e.Key)
}

// IsValidationError reports whether err is one of the KQL parse/validation
// errors produced by this package, i.e. the query itself is at fault and the
// caller should treat it as a bad request.
func IsValidationError(err error) bool {
	var (
		startsWithBinaryOperator *StartsWithBinaryOperatorError
		namedGroupInvalidNodes   *NamedGroupInvalidNodesError
		unsupportedTimeRange     *UnsupportedTimeRangeError
		geoField                 *GeoFieldError
	)

	return errors.As(err, &startsWithBinaryOperator) ||
		errors.As(err, &namedGroupInvalidNodes) ||
		errors.As(err, &unsupportedTimeRange) ||
		errors.As(err, &geoField)
}
