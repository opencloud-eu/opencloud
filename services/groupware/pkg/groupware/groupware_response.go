package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

type Response struct {
	body         any
	status       int
	err          *Error
	etag         jmap.State
	sessionState jmap.SessionState
}

func errorResponse(err *Error) Response {
	return Response{
		body:         nil,
		err:          err,
		etag:         "",
		sessionState: "",
	}
}

func response(body any, sessionState jmap.SessionState) Response {
	return Response{
		body:         body,
		err:          nil,
		etag:         jmap.State(sessionState),
		sessionState: sessionState,
	}
}

func etagResponse(body any, sessionState jmap.SessionState, etag jmap.State) Response {
	return Response{
		body:         body,
		err:          nil,
		etag:         etag,
		sessionState: sessionState,
	}
}

func etagOnlyResponse(body any, etag jmap.State) Response {
	return Response{
		body:         body,
		err:          nil,
		etag:         etag,
		sessionState: "",
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
