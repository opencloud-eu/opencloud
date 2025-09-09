package jmap

import (
	"io"
	"net/url"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/rs/zerolog"
)

type Client struct {
	session               SessionClient
	api                   ApiClient
	blob                  BlobClient
	sessionEventListeners *eventListeners[SessionEventListener]
	io.Closer
}

var _ io.Closer = &Client{}

func (j *Client) Close() error {
	return j.api.Close()
}

func NewClient(session SessionClient, api ApiClient, blob BlobClient) Client {
	return Client{
		session:               session,
		api:                   api,
		blob:                  blob,
		sessionEventListeners: newEventListeners[SessionEventListener](),
	}
}

func (j *Client) AddSessionEventListener(listener SessionEventListener) {
	j.sessionEventListeners.add(listener)
}

func (j *Client) onSessionOutdated(session *Session, newSessionState SessionState) {
	j.sessionEventListeners.signal(func(listener SessionEventListener) {
		listener.OnSessionOutdated(session, newSessionState)
	})
}

// Retrieve JMAP well-known data from the Stalwart server and create a Session from that.
func (j *Client) FetchSession(sessionUrl *url.URL, username string, logger *log.Logger) (Session, Error) {
	wk, err := j.session.GetSession(sessionUrl, username, logger)
	if err != nil {
		return Session{}, err
	}
	return newSession(wk)
}

func (j *Client) logger(operation string, _ *Session, logger *log.Logger) *log.Logger {
	l := logger.With().Str(logOperation, operation)
	return log.From(l)
}

func (j *Client) loggerParams(operation string, _ *Session, logger *log.Logger, params func(zerolog.Context) zerolog.Context) *log.Logger {
	l := logger.With().Str(logOperation, operation)
	if params != nil {
		l = params(l)
	}
	return log.From(l)
}

func (j *Client) maxCallsCheck(calls int, session *Session, logger *log.Logger) Error {
	if calls > session.Capabilities.Core.MaxCallsInRequest {
		logger.Warn().
			Int("max-calls-in-request", session.Capabilities.Core.MaxCallsInRequest).
			Int("calls-in-request", calls).
			Msgf("number of calls in request payload (%d) would exceed the allowed maximum (%d)", session.Capabilities.Core.MaxCallsInRequest, calls)
		return simpleError(errTooManyMethodCalls, JmapErrorTooManyMethodCalls)
	}
	return nil
}

// Construct a Request from the given list of Invocation objects.
//
// If an issue occurs, then it is logged prior to returning it.
func (j *Client) request(session *Session, logger *log.Logger, methodCalls ...Invocation) (Request, Error) {
	err := j.maxCallsCheck(len(methodCalls), session, logger)
	if err != nil {
		return Request{}, err
	}
	return Request{
		Using:       []string{JmapCore, JmapMail},
		MethodCalls: methodCalls,
		CreatedIds:  nil,
	}, nil
}
