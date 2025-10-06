package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

type Response struct {
	body            any
	status          int
	err             *Error
	etag            jmap.State
	sessionState    jmap.SessionState
	contentLanguage jmap.Language
}

func errorResponse(err *Error) Response {
	return Response{
		body:         nil,
		err:          err,
		etag:         "",
		sessionState: "",
	}
}

func errorResponseWithSessionState(err *Error, sessionState jmap.SessionState) Response {
	return Response{
		body:         nil,
		err:          err,
		etag:         "",
		sessionState: sessionState,
	}
}

func response(body any, sessionState jmap.SessionState, contentLanguage jmap.Language) Response {
	return Response{
		body:            body,
		err:             nil,
		etag:            jmap.State(sessionState),
		sessionState:    sessionState,
		contentLanguage: contentLanguage,
	}
}

func etagResponse(body any, sessionState jmap.SessionState, etag jmap.State, contentLanguage jmap.Language) Response {
	return Response{
		body:            body,
		err:             nil,
		etag:            etag,
		sessionState:    sessionState,
		contentLanguage: contentLanguage,
	}
}

func etagOnlyResponse(body any, etag jmap.State, contentLanguage jmap.Language) Response {
	return Response{
		body:            body,
		err:             nil,
		etag:            etag,
		sessionState:    "",
		contentLanguage: contentLanguage,
	}
}

func noContentResponse(sessionState jmap.SessionState) Response {
	return Response{
		body:         nil,
		status:       http.StatusNoContent,
		err:          nil,
		etag:         jmap.State(sessionState),
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

func notFoundResponse(sessionState jmap.SessionState) Response {
	return Response{
		body:         nil,
		status:       http.StatusNotFound,
		err:          nil,
		etag:         jmap.State(sessionState),
		sessionState: sessionState,
	}
}
