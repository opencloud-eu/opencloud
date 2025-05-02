package jmap

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

type JmapApiClient interface {
	Command(ctx context.Context, logger *log.Logger, request map[string]any) ([]byte, error)
}
