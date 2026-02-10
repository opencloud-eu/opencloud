package groupware

import (
	"context"
	"errors"
	"net/http"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
)

// UsernameProvider implementation that uses Reva's enrichment of the Context
// to retrieve the current username.
type revaContextUsernameProvider struct {
}

var _ userProvider = revaContextUsernameProvider{}

func newRevaContextUsernameProvider() userProvider {
	return revaContextUsernameProvider{}
}

var (
	errUserNotInRevaContext = errors.New("failed to find user in reva context")
)

func (r revaContextUsernameProvider) GetUser(req *http.Request, ctx context.Context, logger *log.Logger) (user, error) {
	u, ok := revactx.ContextGetUser(ctx)
	if !ok {
		err := errUserNotInRevaContext
		logger.Error().Err(err).Ctx(ctx).Msgf("could not get user: user not in reva context: %v", ctx)
		return nil, err
	}
	return revaUser{user: u}, nil
}

type revaUser struct {
	user *userv1beta1.User
}

func (r revaUser) GetUsername() string {
	return r.user.GetUsername()
}

func (r revaUser) GetId() string {
	return r.user.GetId().GetOpaqueId()
}

var _ user = revaUser{}

type RevaBearerHttpJmapClientAuthenticator struct {
}

func newRevaBearerHttpJmapClientAuthenticator() jmap.HttpJmapClientAuthenticator {
	return &RevaBearerHttpJmapClientAuthenticator{}
}

var _ jmap.HttpJmapClientAuthenticator = &RevaBearerHttpJmapClientAuthenticator{}

type RevaError struct {
	code int
	err  error
}

var _ jmap.Error = &RevaError{}

func (e RevaError) Code() int {
	return e.code
}
func (e RevaError) Unwrap() error {
	return e.err
}
func (e RevaError) Error() string {
	if e.err != nil {
		return e.err.Error()
	} else {
		return ""
	}
}

const (
	revaErrorTokenMissingInRevaContext = iota + 10000
)

var tokenMissingInRevaContext = RevaError{
	code: revaErrorTokenMissingInRevaContext,
	err:  errors.New("Token is missing from Reva context"),
}

func (h *RevaBearerHttpJmapClientAuthenticator) Authenticate(ctx context.Context, username string, logger *log.Logger, req *http.Request) jmap.Error {
	token, ok := revactx.ContextGetToken(ctx)
	if !ok {
		err := tokenMissingInRevaContext
		logger.Error().Err(err).Ctx(ctx).Msgf("could not get token: token not in reva context: %v", ctx)
		return err
	} else {
		req.Header.Add("Authorization", "Bearer "+token)
		return nil
	}
}

func (h *RevaBearerHttpJmapClientAuthenticator) AuthenticateWS(ctx context.Context, username string, logger *log.Logger, headers http.Header) jmap.Error {
	token, ok := revactx.ContextGetToken(ctx)
	if !ok {
		err := tokenMissingInRevaContext
		logger.Error().Err(err).Ctx(ctx).Msgf("could not get token: token not in reva context: %v", ctx)
		return err
	} else {
		headers.Add("Authorization", "Bearer "+token)
		return nil
	}
}
