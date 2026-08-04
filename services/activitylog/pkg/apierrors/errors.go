// apierrors package defines common business API errors that can be used across the service. It is intended to be used by both the service and the API layer to ensure consistent error handling and messaging.
package apierrors

import "errors"

var (
	ErrNotFound     = errors.New("query target not found")
	ErrBadRequest   = errors.New("bad request")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrMissingEmail = errors.New("missing email address")
)
