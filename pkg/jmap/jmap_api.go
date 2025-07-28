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
