package jmap

import (
	"context"
	"io"
	"net/url"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

type ApiClient interface {
	Command(ctx context.Context, logger *log.Logger, session *Session, request Request) ([]byte, Error)
	io.Closer
}

type SessionClient interface {
	GetSession(baseurl *url.URL, username string, logger *log.Logger) (SessionResponse, Error)
}

type BlobClient interface {
	UploadBinary(ctx context.Context, logger *log.Logger, session *Session, uploadUrl string, endpoint string, contentType string, content io.Reader) (UploadedBlob, Error)
	DownloadBinary(ctx context.Context, logger *log.Logger, session *Session, downloadUrl string, endpoint string) (*BlobDownload, Error)
}

const (
	logOperation    = "operation"
	logUsername     = "username"
	logMailboxId    = "mailbox-id"
	logFetchBodies  = "fetch-bodies"
	logOffset       = "offset"
	logLimit        = "limit"
	logDownloadUrl  = "download-url"
	logBlobId       = "blob-id"
	logUploadUrl    = "download-url"
	logSessionState = "session-state"
	logSince        = "since"
)
