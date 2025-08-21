package jmap

import (
	"io"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/rs/zerolog"
)

type Client struct {
	wellKnown             SessionClient
	api                   ApiClient
	blob                  BlobClient
	sessionEventListeners *eventListeners[SessionEventListener]
	io.Closer
}

func (j *Client) Close() error {
	return j.api.Close()
}

func NewClient(wellKnown SessionClient, api ApiClient, blob BlobClient) Client {
	return Client{
		wellKnown:             wellKnown,
		api:                   api,
		blob:                  blob,
		sessionEventListeners: newEventListeners[SessionEventListener](),
	}
}

func (j *Client) AddSessionEventListener(listener SessionEventListener) {
	j.sessionEventListeners.add(listener)
}

func (j *Client) onSessionOutdated(session *Session, newSessionState string) {
	j.sessionEventListeners.signal(func(listener SessionEventListener) {
		listener.OnSessionOutdated(session, newSessionState)
	})
}

// Retrieve JMAP well-known data from the Stalwart server and create a Session from that.
func (j *Client) FetchSession(username string, logger *log.Logger) (Session, Error) {
	wk, err := j.wellKnown.GetSession(username, logger)
	if err != nil {
		return Session{}, err
	}
	return newSession(wk)
}

func (j *Client) logger(accountId string, operation string, session *Session, logger *log.Logger) *log.Logger {
	l := logger.With().Str(logOperation, operation).Str(logUsername, session.Username)
	if accountId != "" {
		l = l.Str(logAccountId, accountId)
	}
	return log.From(l)
}

func (j *Client) loggerParams(accountId string, operation string, session *Session, logger *log.Logger, params func(zerolog.Context) zerolog.Context) *log.Logger {
	l := logger.With().Str(logOperation, operation).Str(logUsername, session.Username)
	if accountId != "" {
		l = l.Str(logAccountId, accountId)
	}
	if params != nil {
		l = params(l)
	}
	return log.From(l)
}
