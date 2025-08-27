package jmap

import (
	"context"
	"io"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

type ApiClient interface {
	Command(ctx context.Context, logger *log.Logger, session *Session, request Request) ([]byte, Error)
	io.Closer
}

type SessionClient interface {
	GetSession(username string, logger *log.Logger) (SessionResponse, Error)
}

type BlobClient interface {
	UploadBinary(ctx context.Context, logger *log.Logger, session *Session, uploadUrl string, contentType string, content io.Reader) (UploadedBlob, Error)
	DownloadBinary(ctx context.Context, logger *log.Logger, session *Session, downloadUrl string) (*BlobDownload, Error)
}

const (
	logHttpStatusCode = "status"
	logOperation      = "operation"
	logUsername       = "username"
	logAccountId      = "account-id"
	logMailboxId      = "mailbox-id"
	logFetchBodies    = "fetch-bodies"
	logOffset         = "offset"
	logLimit          = "limit"
	logApiUrl         = "apiurl"
	logDownloadUrl    = "downloadurl"
	logBlobId         = "blobId"
	logUploadUrl      = "downloadurl"
	logSessionState   = "session-state"
	logSince          = "since"

	defaultAccountId = "*"
)
