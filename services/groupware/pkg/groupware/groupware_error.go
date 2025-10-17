package groupware

import (
	"context"
	"net/http"
	"strconv"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

type Link struct {
	// A string whose value is a URI-reference [RFC3986 Section 4.1] pointing to the link’s target.
	//
	// [RFC3986 Section 4.1]: https://datatracker.ietf.org/doc/html/rfc3986#section-4.1
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
	//
	// [RFC6901]: https://datatracker.ietf.org/doc/html/rfc6901
	Pointer string `json:"pointer,omitempty"`
	// A string indicating which URI query parameter caused the error.
	Parameter string `json:"parameter,omitempty"`
	// A string indicating the name of a single request header which caused the error.
	Header string `json:"header,omitempty"`
}

// [Error] describes an error.
//
// [Error]: https://jsonapi.org/format/#error-objects
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
	// The [JSON:API] Content Type for errors
	//
	// [JSON:API]: https://jsonapi.org/
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
		return &ErrorInvalidBackendRequest
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
	case jmap.JmapErrorUnspecifiedType, jmap.JmapErrorUnknownMethod, jmap.JmapErrorInvalidArguments, jmap.JmapErrorInvalidResultReference:
		return &ErrorInvalidGroupwareRequest
	case jmap.JmapErrorServerUnavailable:
		return &ErrorServerUnavailable
	case jmap.JmapErrorServerFail:
		return &ErrorServerFailure
	case jmap.JmapErrorForbidden:
		return &ErrorForbiddenOperation
	case jmap.JmapErrorAccountNotFound:
		return &ErrorAccountNotFound
	case jmap.JmapErrorAccountNotSupportedByMethod:
		return &ErrorAccountNotSupportedByMethod
	case jmap.JmapErrorAccountReadOnly:
		return &ErrorAccountReadOnly
	default:
		return &ErrorGeneric
	}
}

const (
	ErrorCodeGeneric                           = "ERRGEN"
	ErrorCodeInvalidAuthentication             = "AUTINV"
	ErrorCodeMissingAuthentication             = "AUTMIS"
	ErrorCodeForbiddenGeneric                  = "AUTFOR"
	ErrorCodeInvalidBackendRequest             = "INVREQ"
	ErrorCodeServerResponse                    = "SRVRSP"
	ErrorCodeStreamingResponse                 = "SRVRST"
	ErrorCodeServerReadingResponse             = "SRVRRE"
	ErrorCodeServerDecodingResponseBody        = "SRVDRB"
	ErrorCodeEncodingRequestBody               = "ENCREQ"
	ErrorCodeCreatingRequest                   = "CREREQ"
	ErrorCodeSendingRequest                    = "SNDREQ"
	ErrorCodeInvalidSessionResponse            = "INVSES"
	ErrorCodeInvalidRequestPayload             = "INVRQP"
	ErrorCodeInvalidResponsePayload            = "INVRSP"
	ErrorCodeInvalidRequestParameter           = "INVPAR"
	ErrorCodeInvalidRequestBody                = "INVBDY"
	ErrorCodeNonExistingAccount                = "INVACC"
	ErrorCodeIndeterminateAccount              = "INDACC"
	ErrorCodeApiInconsistency                  = "APIINC"
	ErrorCodeInvalidUserRequest                = "INVURQ"
	ErrorCodeUsernameEmailDomainNotGreenListed = "UEDGRE"
	ErrorCodeUsernameEmailDomainRedListed      = "UEDRED"
	ErrorCodeInvalidGroupwareRequest           = "GPRERR"
	ErrorCodeServerUnavailable                 = "SRVUNA"
	ErrorCodeServerFailure                     = "SRVFLR"
	ErrorCodeForbiddenOperation                = "FRBOPR"
	ErrorCodeAccountNotFound                   = "ACCNFD"
	ErrorCodeAccountNotSupportedByMethod       = "ACCNSM"
	ErrorCodeAccountReadOnly                   = "ACCRDO"
	ErrorCodeMissingCalendarsSessionCapability = "MSCCAL"
	ErrorCodeMissingCalendarsAccountCapability = "MACCAL"
	ErrorCodeMissingContactsSessionCapability  = "MSCCON"
	ErrorCodeMissingContactsAccountCapability  = "MACCON"
	ErrorCodeMissingTasksSessionCapability     = "MSCTSK"
	ErrorCodeMissingTaskAccountCapability      = "MACTSK"
	ErrorCodeFailedToDeleteEmail               = "DELEML"
	ErrorCodeFailedToDeleteSomeIdentities      = "DELSID"
)

var (
	ErrorGeneric = GroupwareError{
		Status: http.StatusInternalServerError,
		Code:   ErrorCodeGeneric,
		Title:  "Unspecific Error",
		Detail: "Error without a specific description.",
	}
	ErrorInvalidAuthentication = GroupwareError{
		Status: http.StatusUnauthorized,
		Code:   ErrorCodeMissingAuthentication,
		Title:  "Invalid Authentication",
		Detail: "Failed to determine the authentication credentials.",
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
	ErrorInvalidBackendRequest = GroupwareError{
		Status: http.StatusInternalServerError,
		Code:   ErrorCodeInvalidBackendRequest,
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
	ErrorStreamingResponse = GroupwareError{
		Status: http.StatusInternalServerError,
		Code:   ErrorCodeStreamingResponse,
		Title:  "Server Response Body could not be streamed",
		Detail: "The mail server response body could not be streamed.",
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
	ErrorInvalidRequestParameter = GroupwareError{
		Status: http.StatusBadRequest,
		Code:   ErrorCodeInvalidRequestParameter,
		Title:  "Invalid Request Parameter",
		Detail: "At least one of the parameters in the request is invalid.",
	}
	ErrorInvalidRequestBody = GroupwareError{
		Status: http.StatusBadRequest,
		Code:   ErrorCodeInvalidRequestBody,
		Title:  "Invalid Request Body",
		Detail: "The body of the request is invalid.",
	}
	ErrorInvalidUserRequest = GroupwareError{
		Status: http.StatusBadRequest,
		Code:   ErrorCodeInvalidUserRequest,
		Title:  "Invalid Request",
		Detail: "The request is invalid.",
	}
	ErrorIndeterminateAccount = GroupwareError{
		Status: http.StatusBadRequest,
		Code:   ErrorCodeNonExistingAccount,
		Title:  "Invalid Account Parameter",
		Detail: "The account the request is for does not exist.",
	}
	ErrorNonExistingAccount = GroupwareError{
		Status: http.StatusBadRequest,
		Code:   ErrorCodeIndeterminateAccount,
		Title:  "Failed to determine Account",
		Detail: "The account the request is for could not be determined.",
	}
	ErrorApiInconsistency = GroupwareError{
		Status: http.StatusInternalServerError,
		Code:   ErrorCodeApiInconsistency,
		Title:  "API Inconsistency",
		Detail: "Internal APIs returned unexpected data.",
	}
	ErrorUsernameEmailDomainIsNotGreenlisted = GroupwareError{
		Status: http.StatusUnauthorized,
		Code:   ErrorCodeUsernameEmailDomainNotGreenListed,
		Title:  "Domain is not greenlisted",
		Detail: "The username email address domain is not greenlisted.",
	}
	ErrorUsernameEmailDomainIsRedlisted = GroupwareError{
		Status: http.StatusUnauthorized,
		Code:   ErrorCodeUsernameEmailDomainRedListed,
		Title:  "Domain is redlisted",
		Detail: "The username email address domain is redlisted.",
	}
	ErrorInvalidGroupwareRequest = GroupwareError{
		Status: http.StatusInternalServerError,
		Code:   ErrorCodeInvalidGroupwareRequest,
		Title:  "Internal Request Error",
		Detail: "The request constructed by the Groupware is regarded as invalid by the Mail server.",
	}
	ErrorServerUnavailable = GroupwareError{
		Status: http.StatusServiceUnavailable,
		Code:   ErrorCodeServerUnavailable,
		Title:  "Mail Server is unavailable",
		Detail: "The Mail Server is currently unable to process the request.",
	}
	ErrorServerFailure = GroupwareError{
		Status: http.StatusInternalServerError,
		Code:   ErrorCodeServerFailure,
		Title:  "Mail Server is unable to process the Request",
		Detail: "The Mail Server is unable to process the request.",
	}
	ErrorForbiddenOperation = GroupwareError{
		Status: http.StatusForbidden,
		Code:   ErrorCodeForbiddenOperation,
		Title:  "The Operation is forbidden by the Mail Server",
		Detail: "The Mail Server refuses to perform the request.",
	}
	ErrorAccountNotFound = GroupwareError{
		Status: http.StatusNotFound,
		Code:   ErrorCodeAccountNotFound,
		Title:  "The referenced Account does not exist",
		Detail: "The Account that was referenced in the request does not exist.",
	}
	ErrorAccountNotSupportedByMethod = GroupwareError{
		Status: http.StatusForbidden,
		Code:   ErrorCodeAccountNotSupportedByMethod,
		Title:  "The referenced Account does not supported the requested method",
		Detail: "The Account that was referenced in the request does not supported the requested method or data type.",
	}
	ErrorAccountReadOnly = GroupwareError{
		Status: http.StatusForbidden,
		Code:   ErrorCodeAccountReadOnly,
		Title:  "The referenced Account is read-only",
		Detail: "The Account that was referenced in the request only supports read-only operations.",
	}
	ErrorMissingCalendarsSessionCapability = GroupwareError{
		Status: http.StatusExpectationFailed,
		Code:   ErrorCodeMissingCalendarsSessionCapability,
		Title:  "Session is missing the task capability '" + jmap.JmapCalendars + "'",
		Detail: "The JMAP Session of the user does not have the required capability '" + jmap.JmapTasks + "'.",
	}
	ErrorMissingCalendarsAccountCapability = GroupwareError{
		Status: http.StatusExpectationFailed,
		Code:   ErrorCodeMissingCalendarsSessionCapability,
		Title:  "Account is missing the task capability '" + jmap.JmapCalendars + "'",
		Detail: "The JMAP Account of the user does not have the required capability '" + jmap.JmapTasks + "'.",
	}
	ErrorMissingContactsSessionCapability = GroupwareError{
		Status: http.StatusExpectationFailed,
		Code:   ErrorCodeMissingContactsSessionCapability,
		Title:  "Session is missing the task capability '" + jmap.JmapContacts + "'",
		Detail: "The JMAP Session of the user does not have the required capability '" + jmap.JmapContacts + "'.",
	}
	ErrorMissingContactsAccountCapability = GroupwareError{
		Status: http.StatusExpectationFailed,
		Code:   ErrorCodeMissingContactsSessionCapability,
		Title:  "Account is missing the task capability '" + jmap.JmapContacts + "'",
		Detail: "The JMAP Account of the user does not have the required capability '" + jmap.JmapContacts + "'.",
	}
	ErrorMissingTasksSessionCapability = GroupwareError{
		Status: http.StatusExpectationFailed,
		Code:   ErrorCodeMissingTasksSessionCapability,
		Title:  "Session is missing the task capability '" + jmap.JmapTasks + "'",
		Detail: "The JMAP Session of the user does not have the required capability '" + jmap.JmapTasks + "'.",
	}
	ErrorMissingTasksAccountCapability = GroupwareError{
		Status: http.StatusExpectationFailed,
		Code:   ErrorCodeMissingTasksSessionCapability,
		Title:  "Account is missing the task capability '" + jmap.JmapTasks + "'",
		Detail: "The JMAP Account of the user does not have the required capability '" + jmap.JmapTasks + "'.",
	}
	ErrorFailedToDeleteEmail = GroupwareError{
		Status: http.StatusInternalServerError,
		Code:   ErrorCodeFailedToDeleteEmail,
		Title:  "Failed to delete emails",
		Detail: "One or more emails could not be deleted.",
	}
	ErrorFailedToDeleteSomeIdentities = GroupwareError{
		Status: http.StatusInternalServerError,
		Code:   ErrorCodeFailedToDeleteSomeIdentities,
		Title:  "Failed to delete some Identities",
		Detail: "Failed to delete some or all of the identities.",
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

var _ = withLinks // unused for now, but will be
func withLinks(links *ErrorLinks) ErrorLinksOpt {
	return ErrorLinksOpt{
		links: links,
	}
}

type SourceLinksOpt struct {
	source *ErrorSource
}

func (o SourceLinksOpt) apply(error *Error) {
	error.Source = o.source
}

func withSource(source *ErrorSource) SourceLinksOpt {
	return SourceLinksOpt{
		source: source,
	}
}

type MetaLinksOpt struct {
	meta map[string]any
}

func (o MetaLinksOpt) apply(error *Error) {
	error.Meta = o.meta
}

var _ = withMeta // unused for now, but will be
func withMeta(meta map[string]any) MetaLinksOpt {
	return MetaLinksOpt{
		meta: meta,
	}
}

type CodeOpt struct {
	code string
}

func (o CodeOpt) apply(error *Error) {
	error.Code = o.code
}

var _ = withCode // unused for now, but will be
func withCode(code string) CodeOpt {
	return CodeOpt{
		code: code,
	}
}

type TitleOpt struct {
	title  string
	detail string
}

func (o TitleOpt) apply(error *Error) {
	error.Title = o.title
	error.Detail = o.detail
}

var _ = withTitle // unused for now, but will be
func withTitle(title string, detail string) TitleOpt {
	return TitleOpt{
		title:  title,
		detail: detail,
	}
}

type DetailOpt struct {
	detail string
}

func (o DetailOpt) apply(error *Error) {
	error.Detail = o.detail
}

func withDetail(detail string) DetailOpt {
	return DetailOpt{
		detail: detail,
	}
}

/*
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
*/

func errorId(r *http.Request, ctx context.Context) string {
	requestId := chimiddleware.GetReqID(ctx)
	if requestId == "" {
		requestId = r.Header.Get("x-request-id")
	}
	localId := uuid.NewString()
	if requestId != "" {
		return requestId + "." + localId
	} else {
		return localId
	}
}

func (r Request) errorId() string {
	return errorId(r.r, r.ctx)
}

func apiError(id string, gwerr GroupwareError, options ...ErrorOpt) *Error {
	err := &Error{
		Id:        id,
		NumStatus: gwerr.Status,
		Status:    strconv.Itoa(gwerr.Status),
		Code:      gwerr.Code,
		Title:     gwerr.Title,
		Detail:    gwerr.Detail,
	}

	for _, o := range options {
		o.apply(err)
	}

	return err
}

func (r Request) observedParameterError(gwerr GroupwareError, options ...ErrorOpt) *Error {
	return r.observeParameterError(apiError(r.errorId(), gwerr, options...))
}

func (r Request) apiError(err *GroupwareError, options ...ErrorOpt) *Error {
	if err == nil {
		return nil
	}
	errorId := r.errorId()
	return apiError(errorId, *err, options...)
}

func (r Request) apiErrorFromJmap(err jmap.Error) *Error {
	if err == nil {
		return nil
	}
	gwe := groupwareErrorFromJmap(err)
	if gwe == nil {
		return nil
	}

	errorId := r.errorId()
	return apiError(errorId, *gwe)
}

func errorResponses(errors ...Error) ErrorResponse {
	return ErrorResponse{Errors: errors}
}

func (r Request) errorResponseFromJmap(err jmap.Error) Response {
	return errorResponse(r.apiErrorFromJmap(r.observeJmapError(err)))
}
