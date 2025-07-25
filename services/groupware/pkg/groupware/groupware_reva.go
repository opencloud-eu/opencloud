package groupware

import (
	"context"
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/log"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
)

// UsernameProvider implementation that uses Reva's enrichment of the Context
// to retrieve the current username.
type revaContextUsernameProvider struct {
}

type UsernameProvider interface {
	// Provide the username for JMAP operations.
	GetUsername(req *http.Request, ctx context.Context, logger *log.Logger) (string, bool, error)
}

var _ UsernameProvider = revaContextUsernameProvider{}

func NewRevaContextUsernameProvider() UsernameProvider {
	return revaContextUsernameProvider{}
}

// var errUserNotInContext = fmt.Errorf("user not in context")

func (r revaContextUsernameProvider) GetUsername(req *http.Request, ctx context.Context, logger *log.Logger) (string, bool, error) {
	u, ok := revactx.ContextGetUser(ctx)
	if !ok {
		logger.Error().Ctx(ctx).Msgf("could not get user: user not in reva context: %v", ctx)
		return "", false, nil
	}
	return u.GetUsername(), true, nil
}
