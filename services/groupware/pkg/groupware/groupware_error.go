package groupware

import (
	"net/http"
	"strconv"

	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

type Link struct {
	// A string whose value is a URI-reference [RFC3986 Section 4.1] pointing to the link’s target.
	Href string `json:"href"`
	// A string indicating the link’s relation type. The string MUST be a valid link relation type.
	// required: false
	Rel string `json:"rel,omitempty"`
	// A string which serves as a label for the destination of a link such that it can be used as a human-readable identifier (e.g., a menu entry).
	// required: false
	Title string `json:"title,omitempty"`
	// A string indicating the media type of the link’s target.
	// required: false
	Type string `json:"type,omitempty"`
	// A meta object containing non-standard meta-information about the link.
	// required: false
	Meta map[string]any `json:"meta,omitempty"`
}

type ErrorLinks struct {
	// A link that leads to further details about this particular occurrence of the problem.
	// When dereferenced, this URI SHOULD return a human-readable description of the error.
	// This is either a string containing an URL, or a Link object.
	About any `json:"about,omitempty"`
	// A link that identifies the type of error that this particular error is an instance of.
	// This URI SHOULD be dereferenceable to a human-readable explanation of the general error.
	// This is either a string containing an URL, or a Link object.
	Type any `json:"type"`
}

type ErrorSource struct {
	// A JSON Pointer [RFC6901] to the value in the request document that caused the error
	// (e.g. "/data" for a primary data object, or "/data/attributes/title" for a specific attribute).
	// This MUST point to a value in the request document that exists; if it doesn’t, the client SHOULD simply ignore the pointer.
	Pointer string `json:"pointer,omitempty"`
	// A string indicating which URI query parameter caused the error.
	Parameter string `json:"parameter,omitempty"`
	// A string indicating the name of a single request header which caused the error.
	Header string `json:"header,omitempty"`
}

// [Error](https://jsonapi.org/format/#error-objects)
type Error struct {
	// A unique identifier for this particular occurrence of the problem
	Id string `json:"id"`
	// Further detail links about the error.
	// required: false
	Links *ErrorLinks `json:"links,omitempty"`
	// swagger:ignore
	NumStatus int `json:"-"`
	// The HTTP status code applicable to this problem, expressed as a string value.
	Status string `json:"status"`
	// An application-specific error code, expressed as a string value.
	Code string `json:"code"`
	// A short, human-readable summary of the problem that SHOULD NOT change from occurrence to occurrence of the problem.
	Title string `json:"title,omitempty"`
	// A human-readable explanation specific to this occurrence of the problem.
	Detail string `json:"detail,omitempty"`
	// An object containing references to the primary source of the error.
	Source *ErrorSource `json:"source,omitempty"`
	// A meta object containing non-standard meta-information about the error.
	Meta map[string]any `json:"meta,omitempty"`
}

// swagger:response ErrorResponse
type ErrorResponse struct {
	// List of error objects
	Errors []Error `json:"errors"`
}

var _ render.Renderer = ErrorResponse{}

func (e ErrorResponse) Render(w http.ResponseWriter, r *http.Request) error {
	w.Header().Add("Content-Type", ContentTypeJsonApi)
	if len(e.Errors) > 0 {
		render.Status(r, e.Errors[0].NumStatus)
	} else {
		render.Status(r, http.StatusInternalServerError)
	}
	return nil
}

const (
	ContentTypeJsonApi = "application/vnd.api+json"
)

type GroupwareError struct {
	Status int
	Code   string
	Title  string
	Detail string
}

func groupwareErrorFromJmap(j jmap.Error) *GroupwareError {
	if j == nil {
		return nil
	}
	switch j.Code() {
	case jmap.JmapErrorAuthenticationFailed:
		return &ErrorForbidden
	case jmap.JmapErrorInvalidHttpRequest:
		return &ErrorInvalidRequest
	case jmap.JmapErrorServerResponse:
		return &ErrorServerResponse
	case jmap.JmapErrorReadingResponseBody:
		return &ErrorReadingResponse
	case jmap.JmapErrorDecodingResponseBody:
		return &ErrorProcessingResponse
	case jmap.JmapErrorEncodingRequestBody:
		return &ErrorEncodingRequestBody
	case jmap.JmapErrorCreatingRequest:
		return &ErrorCreatingRequest
	case jmap.JmapErrorSendingRequest:
		return &ErrorSendingRequest
	case jmap.JmapErrorInvalidSessionResponse:
		return &ErrorInvalidSessionResponse
	case jmap.JmapErrorInvalidJmapRequestPayload:
		return &ErrorInvalidRequestPayload
	case jmap.JmapErrorInvalidJmapResponsePayload:
		return &ErrorInvalidResponsePayload
	default:
		return &ErrorGeneric
	}
}

const (
	ErrorCodeGeneric                    = "ERRGEN"
	ErrorCodeMissingAuthentication      = "AUTMIS"
	ErrorCodeForbiddenGeneric           = "AUTFOR"
	ErrorCodeInvalidRequest             = "INVREQ"
	ErrorCodeServerResponse             = "SRVRSP"
	ErrorCodeServerReadingResponse      = "SRVRRE"
	ErrorCodeServerDecodingResponseBody = "SRVDRB"
	ErrorCodeEncodingRequestBody        = "ENCREQ"
	ErrorCodeCreatingRequest            = "CREREQ"
	ErrorCodeSendingRequest             = "SNDREQ"
	ErrorCodeInvalidSessionResponse     = "INVSES"
	ErrorCodeInvalidRequestPayload      = "INVRQP"
	ErrorCodeInvalidResponsePayload     = "INVRSP"
)

var (
	ErrorGeneric = GroupwareError{
		Status: http.StatusInternalServerError,
		Code:   ErrorCodeGeneric,
		Title:  "Unspecific Error",
		Detail: "Error without a specific description.",
	}
	ErrorMissingAuthentication = GroupwareError{
		Status: http.StatusUnauthorized,
		Code:   ErrorCodeMissingAuthentication,
		Title:  "Missing Authentication",
		Detail: "No authentication credentials were provided.",
	}
	ErrorForbidden = GroupwareError{
		Status: http.StatusForbidden,
		Code:   ErrorCodeForbiddenGeneric,
		Title:  "Invalid Authentication",
		Detail: "Authentication credentials were provided but are either invalid or not authorized to perform the request operation.",
	}
	ErrorInvalidRequest = GroupwareError{
		Status: http.StatusInternalServerError,
		Code:   ErrorCodeInvalidRequest,
		Title:  "Invalid Request",
		Detail: "The request that was meant to be sent to the mail server is invalid, which might be caused by configuration issues.",
	}
	ErrorServerResponse = GroupwareError{
		Status: http.StatusServiceUnavailable,
		Code:   ErrorCodeServerResponse,
		Title:  "Server responds with an Error",
		Detail: "The mail server responded with an error.",
	}
	ErrorReadingResponse = GroupwareError{
		Status: http.StatusInternalServerError,
		Code:   ErrorCodeServerResponse,
		Title:  "Server Response Body could not be decoded",
		Detail: "The mail server response body could not be decoded.",
	}
	ErrorProcessingResponse = GroupwareError{
		Status: http.StatusInternalServerError,
		Code:   ErrorCodeServerResponse,
		Title:  "Server Response Body could not be decoded",
		Detail: "The mail server response body could not be decoded.",
	}
	ErrorEncodingRequestBody = GroupwareError{
		Status: http.StatusInternalServerError,
		Code:   ErrorCodeEncodingRequestBody,
		Title:  "Failed to encode the Request Body",
		Detail: "Failed to encode the body of the request to be sent to the mail server.",
	}
	ErrorCreatingRequest = GroupwareError{
		Status: http.StatusInternalServerError,
		Code:   ErrorCodeCreatingRequest,
		Title:  "Failed to create the Request",
		Detail: "Failed to create the request to be sent to the mail server.",
	}
	ErrorSendingRequest = GroupwareError{
		Status: http.StatusInternalServerError,
		Code:   ErrorCodeSendingRequest,
		Title:  "Failed to send the Request",
		Detail: "Failed to send the request to the mail server.",
	}
	ErrorInvalidSessionResponse = GroupwareError{
		Status: http.StatusInternalServerError,
		Code:   ErrorCodeInvalidSessionResponse,
		Title:  "Invalid JMAP Session Response",
		Detail: "The JMAP session response that was provided by the mail server is invalid.",
	}
	ErrorInvalidRequestPayload = GroupwareError{
		Status: http.StatusInternalServerError,
		Code:   ErrorCodeInvalidRequestPayload,
		Title:  "Invalid Request Payload",
		Detail: "The request to the mail server is invalid.",
	}
	ErrorInvalidResponsePayload = GroupwareError{
		Status: http.StatusInternalServerError,
		Code:   ErrorCodeInvalidResponsePayload,
		Title:  "Invalid Response Payload",
		Detail: "The payload of the response received from the mail server is invalid.",
	}
)

type ErrorOpt interface {
	apply(error *Error)
}

type ErrorLinksOpt struct {
	links *ErrorLinks
}

func (o ErrorLinksOpt) apply(error *Error) {
	error.Links = o.links
}

type SourceLinksOpt struct {
	source *ErrorSource
}

func (o SourceLinksOpt) apply(error *Error) {
	error.Source = o.source
}

type MetaLinksOpt struct {
	meta map[string]any
}

func (o MetaLinksOpt) apply(error *Error) {
	error.Meta = o.meta
}

type CodeOpt struct {
	code string
}

func (o CodeOpt) apply(error *Error) {
	error.Code = o.code
}

type TitleOpt struct {
	title  string
	detail string
}

func (o TitleOpt) apply(error *Error) {
	error.Title = o.title
	error.Detail = o.detail
}

func errorResponse(id string, error GroupwareError, options ...ErrorOpt) ErrorResponse {
	err := Error{
		Id:        id,
		NumStatus: error.Status,
		Status:    strconv.Itoa(error.Status),
		Code:      error.Code,
		Title:     error.Title,
		Detail:    error.Detail,
	}

	for _, o := range options {
		o.apply(&err)
	}

	return ErrorResponse{
		Errors: []Error{err},
	}
}

func apiError(id string, error GroupwareError, options ...ErrorOpt) Error {
	err := Error{
		Id:        id,
		NumStatus: error.Status,
		Status:    strconv.Itoa(error.Status),
		Code:      error.Code,
		Title:     error.Title,
		Detail:    error.Detail,
	}

	for _, o := range options {
		o.apply(&err)
	}

	return err
}

func apiErrorFromJmap(error jmap.Error) *Error {
	if error == nil {
		return nil
	}
	gwe := groupwareErrorFromJmap(error)
	if gwe == nil {
		return nil
	}
	api := apiError(uuid.NewString(), *gwe)
	return &api
}

func errorResponses(errors ...Error) ErrorResponse {
	return ErrorResponse{Errors: errors}
}
