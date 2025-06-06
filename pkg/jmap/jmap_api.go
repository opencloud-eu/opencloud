package jmap

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

type ApiClient interface {
	Command(ctx context.Context, logger *log.Logger, session *Session, request Request) ([]byte, error)
}

type WellKnownClient interface {
	GetWellKnown(username string, logger *log.Logger) (WellKnownResponse, error)
}
