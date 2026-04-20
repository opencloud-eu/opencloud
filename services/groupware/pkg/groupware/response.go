package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

type ResponseObjectType string

const (
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

type Response struct {
	body            any
	status          int
	err             *Error
	etag            jmap.State
	objectType      ResponseObjectType
	accountIds      []string
	sessionState    jmap.SessionState
	contentLanguage jmap.Language
}

func errorResponse(accountIds []string, err *Error, sessionState jmap.SessionState, contentLanguage jmap.Language) Response {
	return Response{
		accountIds:      accountIds,
		body:            nil,
		err:             err,
		etag:            "",
		sessionState:    sessionState,
		contentLanguage: contentLanguage,
	}
}

func response(accountIds []string, body any, sessionState jmap.SessionState, contentLanguage jmap.Language) Response {
	return Response{
		accountIds:      accountIds,
		body:            body,
		err:             nil,
		etag:            jmap.State(sessionState),
		sessionState:    sessionState,
		contentLanguage: contentLanguage,
	}
}

func (r *Request) respondWithoutStatus(accountId string, body any) Response {
	return response(single(accountId), body, r.session.State, jmap.Language(r.language()))
}

func etaggedResponse(accountIds []string, body any, sessionState jmap.SessionState, objectType ResponseObjectType, etag jmap.State, contentLanguage jmap.Language) Response {
	return Response{
		accountIds:      accountIds,
		body:            body,
		err:             nil,
		etag:            etag,
		objectType:      objectType,
		sessionState:    sessionState,
		contentLanguage: contentLanguage,
	}
}

func (r *Request) respond(accountId string, body any, sessionState jmap.SessionState, objectType ResponseObjectType, etag jmap.State, lang jmap.Language) Response {
	return etaggedResponse(single(accountId), body, sessionState, objectType, etag, lang)
}

func (r *Request) respondN(accountIds []string, body any, sessionState jmap.SessionState, objectType ResponseObjectType, etag jmap.State, lang jmap.Language) Response {
	return etaggedResponse(accountIds, body, sessionState, objectType, etag, lang)
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

func noContentResponse(accountIds []string, sessionState jmap.SessionState) Response {
	return Response{
		accountIds:   accountIds,
		body:         nil,
		status:       http.StatusNoContent,
		err:          nil,
		etag:         jmap.State(sessionState),
		sessionState: sessionState,
	}
}

func (r *Request) noop(accountId string) Response {
	return noContentResponse(single(accountId), r.session.State)
}

func (r *Request) noopN(accountIds []string) Response {
	return noContentResponse(accountIds, r.session.State)
}

func noContentResponseWithEtag(accountIds []string, sessionState jmap.SessionState, objectType ResponseObjectType, etag jmap.State) Response {
	return Response{
		accountIds:   accountIds,
		body:         nil,
		status:       http.StatusNoContent,
		err:          nil,
		etag:         etag,
		objectType:   objectType,
		sessionState: sessionState,
	}
}

func (r *Request) noContent(accountId string, sessionState jmap.SessionState, objectType ResponseObjectType, etag jmap.State) Response {
	return noContentResponseWithEtag(single(accountId), sessionState, objectType, etag)
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

func notFoundResponse(accountIds []string, sessionState jmap.SessionState, objectType ResponseObjectType, etag jmap.State) Response {
	return Response{
		accountIds:   accountIds,
		body:         nil,
		status:       http.StatusNotFound,
		err:          nil,
		objectType:   objectType,
		etag:         etag,
		sessionState: sessionState,
	}
}

func (r *Request) notFound(accountId string, sessionState jmap.SessionState, objectType ResponseObjectType, etag jmap.State) Response {
	return notFoundResponse(single(accountId), sessionState, objectType, etag)
}

func (r *Request) notFoundN(accountIds []string, sessionState jmap.SessionState, objectType ResponseObjectType, etag jmap.State) Response {
	return notFoundResponse(accountIds, sessionState, objectType, etag)
}

func etaggedNotFoundResponse(accountIds []string, sessionState jmap.SessionState, objectType ResponseObjectType, etag jmap.State, contentLanguage jmap.Language) Response {
	return Response{
		accountIds:      accountIds,
		body:            nil,
		status:          http.StatusNotFound,
		err:             nil,
		etag:            etag,
		objectType:      objectType,
		sessionState:    sessionState,
		contentLanguage: contentLanguage,
	}
}

func (r *Request) etaggedNotFound(accountId string, sessionState jmap.SessionState, objectType ResponseObjectType, etag jmap.State) Response {
	return etaggedNotFoundResponse(single(accountId), sessionState, objectType, etag, jmap.Language(r.language()))
}

func notImplementedResponse(accountIds []string, sessionState jmap.SessionState, objectType ResponseObjectType) Response {
	return Response{
		accountIds:   accountIds,
		body:         nil,
		status:       http.StatusNotImplemented,
		err:          nil,
		objectType:   objectType,
		sessionState: sessionState,
	}
}

func (r *Request) notImplementedN(accountIds []string, objectType ResponseObjectType) Response {
	return notImplementedResponse(accountIds, r.session.State, objectType)
}
