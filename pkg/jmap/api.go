package jmap

import (
	"context"
	"io"
	"net/url"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

type Context struct {
	Session        *Session
	Context        context.Context
	Logger         *log.Logger
	AcceptLanguage string
}

func (c Context) WithLogger(newLogger *log.Logger) Context {
	return Context{Session: c.Session, Context: c.Context, AcceptLanguage: c.AcceptLanguage, Logger: newLogger}
}

func (c Context) WithContext(newContext context.Context) Context {
	return Context{Session: c.Session, Context: newContext, AcceptLanguage: c.AcceptLanguage, Logger: c.Logger}
}

type ApiClient interface {
	Command(request Request, ctx Context) ([]byte, Language, Error)
	io.Closer
}

type WsPushListener interface {
	OnNotification(username string, stateChange StateChange)
}

type WsClient interface {
	DisableNotifications() Error
	io.Closer
}

type WsClientFactory interface {
	EnableNotifications(ctx context.Context, pushState State, sessionProvider func() (*Session, error), listener WsPushListener) (WsClient, Error)
	io.Closer
}

type SessionClient interface {
	GetSession(ctx context.Context, baseurl *url.URL, username string, logger *log.Logger) (SessionResponse, Error)
	io.Closer
}

type BlobClient interface {
	UploadBinary(uploadUrl string, endpoint string, contentType string, content io.Reader, ctx Context) (UploadedBlob, Language, Error)
	DownloadBinary(downloadUrl string, endpoint string, ctx Context) (*BlobDownload, Language, Error)
	io.Closer
}

const (
	logOperation   = "operation"
	logFetchBodies = "fetch-bodies"
	logPosition    = "position"
	logLimit       = "limit"
	logDownloadUrl = "download-url"
	logBlobId      = "blob-id"
	logSinceState  = "since-state"
)

type ResultMetadata interface {
	GetSessionState() SessionState
	GetState() State
	GetLanguage() Language
}

type Result[T any] struct {
	Payload      T
	SessionState SessionState
	State        State
	Language     Language
}

func RefineResult[A, B any](a Result[A], refiner func(A) B) Result[B] {
	return newResult(
		refiner(a.Payload),
		a.SessionState,
		a.State,
		a.Language,
	)
}

func (r Result[T]) GetSessionState() SessionState {
	return r.SessionState
}

func (r Result[T]) GetState() State {
	return r.State
}

func (r Result[T]) GetLanguage() Language {
	return r.Language
}

func newResult[T any](result T, sessionState SessionState, state State, language Language) Result[T] {
	return Result[T]{
		Payload:      result,
		SessionState: sessionState,
		State:        state,
		Language:     language,
	}
}

func newPartialResult[T any](sessionState SessionState, language Language) Result[T] {
	return Result[T]{
		SessionState: sessionState,
		Language:     language,
	}
}

func ZeroResult[T any]() Result[T] {
	return Result[T]{}
}
