package jmap

import (
	"context"
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/log"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
)

// implements HttpJmapUsernameProvider
type RevaContextHttpJmapUsernameProvider struct {
}

func NewRevaContextHttpJmapUsernameProvider() RevaContextHttpJmapUsernameProvider {
	return RevaContextHttpJmapUsernameProvider{}
}

func (r RevaContextHttpJmapUsernameProvider) GetUsername(ctx context.Context, logger *log.Logger) (string, error) {
	u, ok := revactx.ContextGetUser(ctx)
	if !ok {
		logger.Error().Msg("could not get user: user not in context")
		return "", fmt.Errorf("user not in context")
	}
	return u.GetUsername(), nil
}
