package jmap

import (
	"context"
	"fmt"
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/log"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
)

// HttpJmapUsernameProvider implementation that uses Reva's enrichment of the Context
// to retrieve the current username.
type revaContextHttpJmapUsernameProvider struct {
}

var _ HttpJmapUsernameProvider = revaContextHttpJmapUsernameProvider{}

func NewRevaContextHttpJmapUsernameProvider() HttpJmapUsernameProvider {
	return revaContextHttpJmapUsernameProvider{}
}

var errUserNotInContext = fmt.Errorf("user not in context")

func (r revaContextHttpJmapUsernameProvider) GetUsername(req *http.Request, ctx context.Context, logger *log.Logger) (string, error) {
	u, ok := revactx.ContextGetUser(ctx)
	if !ok {
		logger.Error().Msg("could not get user: user not in reva context")
		return "", errUserNotInContext
	}
	return u.GetUsername(), nil
}
