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

func errorResponse(accountIds []string, err *Error) Response {
	return Response{
		accountIds:   accountIds,
		body:         nil,
		err:          err,
		etag:         "",
		sessionState: "",
	}
}

func errorResponseWithSessionState(accountIds []string, err *Error, sessionState jmap.SessionState) Response {
	return Response{
		accountIds:   accountIds,
		body:         nil,
		err:          err,
		etag:         "",
		sessionState: sessionState,
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

func etagResponse(accountIds []string, body any, sessionState jmap.SessionState, objectType ResponseObjectType, etag jmap.State, contentLanguage jmap.Language) Response {
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

func notFoundResponse(accountIds []string, sessionState jmap.SessionState) Response {
	return Response{
		accountIds:   accountIds,
		body:         nil,
		status:       http.StatusNotFound,
		err:          nil,
		etag:         "",
		sessionState: sessionState,
	}
}

func etagNotFoundResponse(accountIds []string, sessionState jmap.SessionState, objectType ResponseObjectType, etag jmap.State, contentLanguage jmap.Language) Response {
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

func notImplementedResponse() Response {
	return Response{
		body:   nil,
		status: http.StatusNotImplemented,
		err:    nil,
	}
}
