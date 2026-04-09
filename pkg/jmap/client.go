package jmap

import (
	"context"
	"errors"
	"io"
	"net/url"
	"slices"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/rs/zerolog"
)

type Client struct {
	session               SessionClient
	api                   ApiClient
	blob                  BlobClient
	ws                    WsClientFactory
	sessionEventListeners *eventListeners[SessionEventListener]
	wsPushListeners       *eventListeners[WsPushListener]
	io.Closer
	WsPushListener
}

type ApiSupplier interface {
	Api() ApiClient
}

type Hooks interface {
	OnSessionOutdated(session *Session, newState SessionState)
}

var _ io.Closer = &Client{}
var _ WsPushListener = &Client{}
var _ ApiSupplier = &Client{}
var _ Hooks = &Client{}

func (j *Client) Close() error {
	return errors.Join(j.api.Close(), j.session.Close(), j.blob.Close(), j.ws.Close())
}

func (j *Client) Api() ApiClient {
	return j.api
}

func NewClient(session SessionClient, api ApiClient, blob BlobClient, ws WsClientFactory) Client {
	return Client{
		session:               session,
		api:                   api,
		blob:                  blob,
		ws:                    ws,
		sessionEventListeners: newEventListeners[SessionEventListener](),
		wsPushListeners:       newEventListeners[WsPushListener](),
	}
}

func (j *Client) AddSessionEventListener(listener SessionEventListener) {
	j.sessionEventListeners.add(listener)
}

func (j *Client) OnSessionOutdated(session *Session, newSessionState SessionState) {
	j.sessionEventListeners.signal(func(listener SessionEventListener) {
		listener.OnSessionOutdated(session, newSessionState)
	})
}

func (j *Client) OnNotification(username string, stateChange StateChange) {
	j.wsPushListeners.signal(func(listener WsPushListener) {
		listener.OnNotification(username, stateChange)
	})
}

// Retrieve JMAP well-known data from the Stalwart server and create a Session from that.
func (j *Client) FetchSession(ctx context.Context, sessionUrl *url.URL, username string, logger *log.Logger) (Session, Error) {
	wk, err := j.session.GetSession(ctx, sessionUrl, username, logger)
	if err != nil {
		return Session{}, err
	}
	return newSession(wk)
}

func (j *Client) logger(operation string, ctx Context) *log.Logger {
	l := ctx.Logger.With().Str(logOperation, operation)
	return log.From(l)
}

func (j *Client) loggerParams(operation string, ctx Context, params func(zerolog.Context) zerolog.Context) *log.Logger {
	l := ctx.Logger.With().Str(logOperation, operation)
	if params != nil {
		l = params(l)
	}
	return log.From(l)
}

func (j *Client) maxCallsCheck(calls int, ctx Context) Error {
	if calls > ctx.Session.Capabilities.Core.MaxCallsInRequest {
		ctx.Logger.Error().
			Int("max-calls-in-request", ctx.Session.Capabilities.Core.MaxCallsInRequest).
			Int("calls-in-request", calls).
			Msgf("number of calls in request payload (%d) exceeds the allowed maximum (%d)", ctx.Session.Capabilities.Core.MaxCallsInRequest, calls)
		return jmapError(errTooManyMethodCalls, JmapErrorTooManyMethodCalls)
	}
	return nil
}

// Construct a Request from the given list of Invocation objects.
//
// If an issue occurs, then it is logged prior to returning it.
func (j *Client) request(ctx Context, using []JmapNamespace, methodCalls ...Invocation) (Request, Error) {
	err := j.maxCallsCheck(len(methodCalls), ctx)
	if err != nil {
		return Request{}, err
	}

	if using == nil {
		using = JmapNamespaces
	}
	if !slices.Contains(using, JmapCore) {
		using = slices.Insert(using, 0, JmapCore)
	}
	return Request{
		Using:       using,
		MethodCalls: methodCalls,
		CreatedIds:  nil,
	}, nil
}
