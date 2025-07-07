package jmap

import (
	"context"
	"io"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

type ApiClient interface {
	Command(ctx context.Context, logger *log.Logger, session *Session, request Request) ([]byte, error)
	io.Closer
}

type WellKnownClient interface {
	GetWellKnown(username string, logger *log.Logger) (WellKnownResponse, error)
}
