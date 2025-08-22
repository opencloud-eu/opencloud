package groupware

import (
	"context"
	"errors"
	"net/http"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
)

// UsernameProvider implementation that uses Reva's enrichment of the Context
// to retrieve the current username.
type revaContextUserProvider struct {
}

var _ UserProvider = revaContextUserProvider{}

func NewRevaContextUsernameProvider() UserProvider {
	return revaContextUserProvider{}
}

// var errUserNotInContext = fmt.Errorf("user not in context")

var (
	errUserNotInRevaContext = errors.New("failed to find user in reva context")
)

func (r revaContextUserProvider) GetUser(req *http.Request, ctx context.Context, logger *log.Logger) (User, error) {
	u, ok := revactx.ContextGetUser(ctx)
	if !ok {
		err := errUserNotInRevaContext
		logger.Error().Err(err).Ctx(ctx).Msgf("could not get user: user not in reva context: %v", ctx)
		return nil, err
	}
	return RevaUser{user: u}, nil
}

type RevaUser struct {
	user *userv1beta1.User
}

func (r RevaUser) GetUsername() string {
	return r.user.GetUsername()
}

func (r RevaUser) GetId() string {
	return r.user.GetId().GetOpaqueId()
}

var _ User = RevaUser{}
