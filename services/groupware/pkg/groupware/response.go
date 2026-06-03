package groupware

import (
	"net/http"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

type ResponseObjectType string

const (
	UnspecifiedResponseObjectType      = ResponseObjectType("")
	IndexResponseObjectType            = ResponseObjectType("index")
	AccountResponseObjectType          = ResponseObjectType("account")
	IdentityResponseObjectType         = ResponseObjectType("identity")
	BlobResponseObjectType             = ResponseObjectType("blob")
	CalendarResponseObjectType         = ResponseObjectType("calendar")
	EventResponseObjectType            = ResponseObjectType("event")
	AddressBookResponseObjectType      = ResponseObjectType("addressbook")
	ContactResponseObjectType          = ResponseObjectType("contact")
	EmailResponseObjectType            = ResponseObjectType("email")
	MailboxResponseObjectType          = ResponseObjectType("mailbox")
	QuotaResponseObjectType            = ResponseObjectType("quota")
	TaskListResponseObjectType         = ResponseObjectType("tasklist")
	TaskResponseObjectType             = ResponseObjectType("task")
	VacationResponseResponseObjectType = ResponseObjectType("vacationresponse")
)

var zeroDurations = []time.Duration{time.Duration(0)}

type Response struct {
	body            any
	status          int
	err             *Error
	etag            jmap.State
	objectType      ResponseObjectType
	accountIds      []jmap.AccountId
	sessionState    jmap.SessionState
	contentLanguage jmap.Language
	next            NextToken
	durations       []time.Duration
}

func errorResponse(
	accountIds []jmap.AccountId,
	err *Error,
	sessionState jmap.SessionState,
	contentLanguage jmap.Language,
	durations []time.Duration,
) Response {
	return Response{
		accountIds:      accountIds,
		body:            nil,
		err:             err,
		etag:            "",
		sessionState:    sessionState,
		contentLanguage: contentLanguage,
		next:            NoNextToken,
		durations:       durations,
	}
}

func response(
	accountIds []jmap.AccountId,
	body any,
	sessionState jmap.SessionState,
	contentLanguage jmap.Language,
	durations []time.Duration,
) Response {
	return Response{
		accountIds:      accountIds,
		body:            body,
		err:             nil,
		etag:            jmap.State(sessionState),
		sessionState:    sessionState,
		contentLanguage: contentLanguage,
		next:            NoNextToken,
		durations:       durations,
	}
}

func (r *Request) respondWithoutStatus(accountId jmap.AccountId, body any, durations []time.Duration) Response {
	return response(single(accountId), body, r.session.State, jmap.Language(r.language()), durations)
}

func etaggedResponse(
	accountIds []jmap.AccountId,
	body any,
	sessionState jmap.SessionState,
	objectType ResponseObjectType,
	etag jmap.State,
	contentLanguage jmap.Language,
	durations []time.Duration,
) Response {
	return Response{
		accountIds:      accountIds,
		body:            body,
		err:             nil,
		etag:            etag,
		objectType:      objectType,
		sessionState:    sessionState,
		contentLanguage: contentLanguage,
		next:            NoNextToken,
		durations:       durations,
	}
}

func (r *Request) respond(
	accountId jmap.AccountId,
	body any,
	objectType ResponseObjectType,
	result jmap.ResultMetadata,
) Response {
	return etaggedResponse(single(accountId), body, result.GetSessionState(), objectType, result.GetState(), result.GetLanguage(), result.GetDurations())
}

func (r *Request) respondN(
	accountIds []jmap.AccountId,
	body any,
	objectType ResponseObjectType,
	result jmap.ResultMetadata,
) Response {
	return etaggedResponse(accountIds, body, result.GetSessionState(), objectType, result.GetState(), result.GetLanguage(), result.GetDurations())
}

func etaggedNextResponse(
	accountIds []jmap.AccountId,
	body any,
	sessionState jmap.SessionState,
	objectType ResponseObjectType,
	etag jmap.State,
	contentLanguage jmap.Language,
	next NextToken,
	durations []time.Duration,
) Response {
	return Response{
		accountIds:      accountIds,
		body:            body,
		err:             nil,
		etag:            etag,
		objectType:      objectType,
		sessionState:    sessionState,
		contentLanguage: contentLanguage,
		next:            next,
		durations:       durations,
	}
}

func (r *Request) respondNext(
	accountId jmap.AccountId,
	body any,
	objectType ResponseObjectType,
	result jmap.ResultMetadata,
	next NextToken,
) Response {
	return etaggedNextResponse(single(accountId), body, result.GetSessionState(), objectType, result.GetState(), result.GetLanguage(), next, result.GetDurations())
}

/*
func etagOnlyResponse(body any, etag jmap.State, objectType ResponseObjectType, contentLanguage jmap.Language) Response {
	return Response{
		body:            body,
		err:             nil,
		etag:            etag,
		objectType:      objectType,
		sessionState:    "",
		contentLanguage: contentLanguage,
	}
}
*/

func noContentResponse(accountIds []jmap.AccountId, sessionState jmap.SessionState, durations []time.Duration) Response {
	return Response{
		accountIds:   accountIds,
		body:         nil,
		status:       http.StatusNoContent,
		err:          nil,
		etag:         jmap.State(sessionState),
		sessionState: sessionState,
		next:         NoNextToken,
		durations:    durations,
	}
}

func (r *Request) noop(accountId jmap.AccountId, durations []time.Duration) Response {
	return noContentResponse(single(accountId), r.session.State, durations)
}

func (r *Request) noopV(accountId jmap.AccountId) Response {
	return noContentResponse(single(accountId), r.session.State, zeroDurations)
}

func (r *Request) noopN(accountIds []jmap.AccountId, durations []time.Duration) Response {
	return noContentResponse(accountIds, r.session.State, durations)
}

func noContentResponseWithEtag(
	accountIds []jmap.AccountId,
	sessionState jmap.SessionState,
	objectType ResponseObjectType,
	etag jmap.State,
	durations []time.Duration,
) Response {
	return Response{
		accountIds:   accountIds,
		body:         nil,
		status:       http.StatusNoContent,
		err:          nil,
		etag:         etag,
		objectType:   objectType,
		sessionState: sessionState,
		next:         NoNextToken,
		durations:    durations,
	}
}

func (r *Request) noContent(
	accountId jmap.AccountId,
	objectType ResponseObjectType,
	result jmap.ResultMetadata,
) Response {
	return noContentResponseWithEtag(single(accountId), result.GetSessionState(), objectType, result.GetState(), result.GetDurations())
}

/*
func acceptedResponse(sessionState jmap.SessionState) Response {
	return Response{
		body:         nil,
		status:       http.StatusAccepted,
		err:          nil,
		etag:         jmap.State(sessionState),
		sessionState: sessionState,
	}
}
*/

/*
func timeoutResponse(sessionState jmap.SessionState) Response {
	return Response{
		body:         nil,
		status:       http.StatusRequestTimeout,
		err:          nil,
		etag:         "",
		sessionState: sessionState,
	}
}
*/

func notFoundResponse(
	accountIds []jmap.AccountId,
	sessionState jmap.SessionState,
	objectType ResponseObjectType,
	etag jmap.State,
	durations []time.Duration,
) Response {
	return Response{
		accountIds:   accountIds,
		body:         nil,
		status:       http.StatusNotFound,
		err:          nil,
		objectType:   objectType,
		etag:         etag,
		sessionState: sessionState,
		next:         NoNextToken,
		durations:    durations,
	}
}

func (r *Request) notFound(
	accountId jmap.AccountId,
	objectType ResponseObjectType,
	result jmap.ResultMetadata,
) Response {
	return notFoundResponse(single(accountId), result.GetSessionState(), objectType, result.GetState(), result.GetDurations())
}

func etaggedNotFoundResponse(
	accountIds []jmap.AccountId,
	sessionState jmap.SessionState,
	objectType ResponseObjectType,
	etag jmap.State,
	contentLanguage jmap.Language,
	durations []time.Duration,
) Response {
	return Response{
		accountIds:      accountIds,
		body:            nil,
		status:          http.StatusNotFound,
		err:             nil,
		etag:            etag,
		objectType:      objectType,
		sessionState:    sessionState,
		contentLanguage: contentLanguage,
		next:            NoNextToken,
		durations:       durations,
	}
}

func (r *Request) etaggedNotFound(
	accountId jmap.AccountId,
	sessionState jmap.SessionState,
	objectType ResponseObjectType,
	etag jmap.State,
	durations []time.Duration,
) Response {
	return etaggedNotFoundResponse(single(accountId), sessionState, objectType, etag, jmap.Language(r.language()), durations)
}

func notImplementedResponse(accountIds []jmap.AccountId, sessionState jmap.SessionState, objectType ResponseObjectType) Response {
	return Response{
		accountIds:   accountIds,
		body:         nil,
		status:       http.StatusNotImplemented,
		err:          nil,
		objectType:   objectType,
		sessionState: sessionState,
		next:         NoNextToken,
		durations:    nil,
	}
}

func (r *Request) notImplementedN(accountIds []jmap.AccountId, objectType ResponseObjectType) Response {
	return notImplementedResponse(accountIds, r.session.State, objectType)
}
