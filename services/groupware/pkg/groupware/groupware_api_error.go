package groupware

import (
	"net/http"
	"strconv"

	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

type Link struct {
	Href  string         `json:"href"`
	Rel   string         `json:"rel,omitempty"`
	Title string         `json:"title,omitempty"`
	Type  string         `json:"type,omitempty"`
	Meta  map[string]any `json:"meta,omitempty"`
}

type ErrorLinks struct {
	About any `json:"about,omitempty"`
	Type  any `json:"type"` // either a string containing an URL, or a Link object
}

type ErrorSource struct {
	Pointer   string `json:"pointer,omitempty"`   // a JSON Pointer [RFC6901] to the value in the request document that caused the error
	Parameter string `json:"parameter,omitempty"` // a string indicating which URI query parameter caused the error
	Header    string `json:"header,omitempty"`    // a string indicating the name of a single request header which caused the error
}

type ApiError struct {
	Id        string         `json:"id"` // a unique identifier for this particular occurrence of the problem
	Links     *ErrorLinks    `json:"links,omitempty"`
	NumStatus int            `json:"-"`
	Status    string         `json:"status"`           // the HTTP status code applicable to this problem, expressed as a string value
	Code      string         `json:"code"`             // an application-specific error code, expressed as a string value
	Title     string         `json:"title,omitempty"`  // a short, human-readable summary of the problem that SHOULD NOT change from occurrence to occurrence of the problem
	Detail    string         `json:"detail,omitempty"` // a human-readable explanation specific to this occurrence of the problem
	Source    *ErrorSource   `json:"source,omitempty"` // an object containing references to the primary source of the error
	Meta      map[string]any `json:"meta,omitempty"`   // a meta object containing non-standard meta-information about the error
}

type ErrorResponse struct {
	Errors []ApiError `json:"errors"`
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
	apply(error *ApiError)
}

type ErrorLinksOpt struct {
	links *ErrorLinks
}

func (o ErrorLinksOpt) apply(error *ApiError) {
	error.Links = o.links
}

type SourceLinksOpt struct {
	source *ErrorSource
}

func (o SourceLinksOpt) apply(error *ApiError) {
	error.Source = o.source
}

type MetaLinksOpt struct {
	meta map[string]any
}

func (o MetaLinksOpt) apply(error *ApiError) {
	error.Meta = o.meta
}

type CodeOpt struct {
	code string
}

func (o CodeOpt) apply(error *ApiError) {
	error.Code = o.code
}

type TitleOpt struct {
	title  string
	detail string
}

func (o TitleOpt) apply(error *ApiError) {
	error.Title = o.title
	error.Detail = o.detail
}

func errorResponse(id string, error GroupwareError, options ...ErrorOpt) ErrorResponse {
	err := ApiError{
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
		Errors: []ApiError{err},
	}
}

func apiError(id string, error GroupwareError, options ...ErrorOpt) ApiError {
	err := ApiError{
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

func apiErrorFromJmap(error jmap.Error) *ApiError {
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

func errorResponses(errors ...ApiError) ErrorResponse {
	return ErrorResponse{Errors: errors}
}
