package jmap

import (
	"context"
	"io"
	"net/url"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

type ApiClient interface {
	Command(ctx context.Context, logger *log.Logger, session *Session, request Request, acceptLanguage string) ([]byte, Language, Error)
	io.Closer
}

type WsPushListener interface {
	OnNotification(stateChange StateChange)
}

type WsClient interface {
	DisableNotifications() Error
	io.Closer
}

type WsClientFactory interface {
	EnableNotifications(pushState string, sessionProvider func() (*Session, error), listener WsPushListener) (WsClient, Error)
	io.Closer
}

type SessionClient interface {
	GetSession(baseurl *url.URL, username string, logger *log.Logger) (SessionResponse, Error)
	io.Closer
}

type BlobClient interface {
	UploadBinary(ctx context.Context, logger *log.Logger, session *Session, uploadUrl string, endpoint string, contentType string, acceptLanguage string, content io.Reader) (UploadedBlob, Language, Error)
	DownloadBinary(ctx context.Context, logger *log.Logger, session *Session, downloadUrl string, endpoint string, acceptLanguage string) (*BlobDownload, Language, Error)
	io.Closer
}

const (
	logOperation   = "operation"
	logMailboxId   = "mailbox-id"
	logFetchBodies = "fetch-bodies"
	logOffset      = "offset"
	logLimit       = "limit"
	logDownloadUrl = "download-url"
	logBlobId      = "blob-id"
	logUploadUrl   = "download-url"
	logSinceState  = "since-state"
)
