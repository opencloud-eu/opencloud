// Package errorcode allows to deal with graph error codes
package errorcode

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

// Error defines a custom error struct, containing and MS Graph error code and a textual error message
type Error struct {
	errorCode ErrorCode
	msg       string
	origin    ErrorOrigin
}

// ErrorOrigin gives information about where the error originated
type ErrorOrigin int

const (
	// ErrorOriginUnknown is the default error source
	// and indicates that the error does not have any information about its origin
	ErrorOriginUnknown ErrorOrigin = iota

	// ErrorOriginCS3 indicates that the error originated from a CS3 service
	ErrorOriginCS3
)

// ErrorCode defines code as used in MS Graph - see https://docs.microsoft.com/en-us/graph/errors?context=graph%2Fapi%2F1.0&view=graph-rest-1.0
type ErrorCode int

// List taken from https://github.com/microsoft/microsoft-graph-docs-1/blob/main/concepts/errors.md#code-property
const (
	// AccessDenied defines the error if the caller doesn't have permission to perform the action.
	AccessDenied ErrorCode = iota
	// ActivityLimitReached defines the error if the app or user has been throttled.
	ActivityLimitReached
	// GeneralException defines the error if an unspecified error has occurred.
	GeneralException
	// InvalidAuthenticationToken defines the error if the access token is missing
	InvalidAuthenticationToken
	// InvalidRange defines the error if the specified byte range is invalid or unavailable.
	InvalidRange
	// InvalidRequest defines the error if the request is malformed or incorrect.
	InvalidRequest
	// ItemNotFound defines the error if the resource could not be found.
	ItemNotFound
	// MalwareDetected defines the error if malware was detected in the requested resource.
	MalwareDetected
	// NameAlreadyExists defines the error if the specified item name already exists.
	NameAlreadyExists
	// NotAllowed defines the error if the action is not allowed by the system.
	NotAllowed
	// NotSupported defines the error if the request is not supported by the system.
	NotSupported
	// ResourceModified defines the error if the resource being updated has changed since the caller last read it, usually an eTag mismatch.
	ResourceModified
	// ResyncRequired defines the error if the delta token is no longer valid, and the app must reset the sync state.
	ResyncRequired
	// ServiceNotAvailable defines the error if the service is not available. Try the request again after a delay. There may be a Retry-After header.
	ServiceNotAvailable
	// SyncStateNotFound defines the error when the sync state generation is not found. The delta token is expired and data must be synchronized again.
	SyncStateNotFound
	// QuotaLimitReached the user has reached their quota limit.
	QuotaLimitReached
	// Unauthenticated the caller is not authenticated.
	Unauthenticated
	// PreconditionFailed the request cannot be made and this error response is sent back
	PreconditionFailed
	// ItemIsLocked The item is locked by another process. Try again later.
	ItemIsLocked
)

var errorCodes = [...]string{
	"accessDenied",
	"activityLimitReached",
	"generalException",
	"InvalidAuthenticationToken",
	"invalidRange",
	"invalidRequest",
	"itemNotFound",
	"malwareDetected",
	"nameAlreadyExists",
	"notAllowed",
	"notSupported",
	"resourceModified",
	"resyncRequired",
	"serviceNotAvailable",
	"syncStateNotFound",
	"quotaLimitReached",
	"unauthenticated",
	"preconditionFailed",
	"itemIsLocked",
}

// New constructs a new errorcode.Error
func New(e ErrorCode, msg string) Error {
	return Error{
		errorCode: e,
		msg:       msg,
	}
}

// Render writes a Graph ErrorCode object to the response writer
func (e ErrorCode) Render(w http.ResponseWriter, r *http.Request, status int, msg string) {
	render.Status(r, status)
	render.JSON(w, r, e.CreateOdataError(r.Context(), msg))
}

// CreateOdataError creates and populates a Graph ErrorCode object
func (e ErrorCode) CreateOdataError(ctx context.Context, msg string) *libregraph.OdataError {
	innererror := map[string]interface{}{
		"date": time.Now().UTC().Format(time.RFC3339),
	}

	innererror["request-id"] = middleware.GetReqID(ctx)
	return &libregraph.OdataError{
		Error: libregraph.OdataErrorMain{
			Code:       e.String(),
			Message:    msg,
			Innererror: innererror,
		},
	}
}

// Render writes a Graph Error object to the response writer
func (e Error) Render(w http.ResponseWriter, r *http.Request) {
	var status int
	switch e.errorCode {
	case AccessDenied:
		status = http.StatusForbidden
	case NotSupported:
		status = http.StatusNotImplemented
	case InvalidRange:
		status = http.StatusRequestedRangeNotSatisfiable
	case InvalidRequest:
		status = http.StatusBadRequest
	case ItemNotFound:
		status = http.StatusNotFound
	case NameAlreadyExists:
		status = http.StatusConflict
	case NotAllowed:
		status = http.StatusMethodNotAllowed
	case ItemIsLocked:
		status = http.StatusLocked
	case PreconditionFailed:
		status = http.StatusPreconditionFailed
	default:
		status = http.StatusInternalServerError
	}
	e.errorCode.Render(w, r, status, e.msg)
}

// String returns the string corresponding to the ErrorCode
func (e ErrorCode) String() string {
	return errorCodes[e]
}

// Error returns the concatenation of the error string and optional message
func (e Error) Error() string {
	errString := errorCodes[e.errorCode]
	if e.msg != "" {
		errString += ": " + e.msg
	}
	return errString
}

func (e Error) GetCode() ErrorCode {
	return e.errorCode
}

// GetOrigin returns the source of the error
func (e Error) GetOrigin() ErrorOrigin {
	return e.origin
}

// WithOrigin returns a new Error with the provided origin
func (e Error) WithOrigin(o ErrorOrigin) Error {
	e.origin = o
	return e
}

// RenderError render the Graph Error based on a code or default one
func RenderError(w http.ResponseWriter, r *http.Request, err error) {
	e, ok := ToError(err)
	if !ok {
		GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	e.Render(w, r)
}

// ToError checks if the error is of type Error and returns it,
// the second parameter indicates if the error conversion was successful
func ToError(err error) (Error, bool) {
	var e Error
	if errors.As(err, &e) {
		return e, true
	}

	return Error{}, false
}
