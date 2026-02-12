package staticroutes

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/render"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack/v5"
	microstore "go-micro.dev/v4/store"
)

// handle backchannel logout requests as per https://openid.net/specs/openid-connect-backchannel-1_0.html#BCRequest
func (s *StaticRouteHandler) backchannelLogout(w http.ResponseWriter, r *http.Request) {
	// parse the application/x-www-form-urlencoded POST request
	logger := s.Logger.SubloggerWithRequestID(r.Context())
	if err := r.ParseForm(); err != nil {
		logger.Warn().Err(err).Msg("ParseForm failed")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: err.Error()})
		return
	}

	if r.PostFormValue("logout_token") == "" {
		logger.Warn().Msg("logout_token is missing")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: "logout_token is missing"})
		return
	}

	logoutToken, err := s.OidcClient.VerifyLogoutToken(r.Context(), r.PostFormValue("logout_token"))
	if err != nil {
		logger.Warn().Err(err).Msg("VerifyLogoutToken failed")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: err.Error()})
		return
	}

	var cacheKeys map[string]bool

	if strings.TrimSpace(logoutToken.SessionId) != "" {
		cacheKeys[logoutToken.SessionId] = true
	} else if strings.TrimSpace(logoutToken.Subject) != "" {
		records, err := s.UserInfoCache.Read(fmt.Sprintf("%s.*", logoutToken.Subject))
		if errors.Is(err, microstore.ErrNotFound) || len(records) == 0 {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: err.Error()})
			return
		}
		if err != nil {
			logger.Error().Err(err).Msg("Error reading userinfo cache")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: err.Error()})
			return
		}
		for _, record := range records {
			cacheKeys[string(record.Value)] = true
			cacheKeys[record.Key] = false
		}
	} else {
		logger.Warn().Msg("invalid logout token")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: "invalid logout token"})
		return
	}

	for key, isSID := range cacheKeys {
		if isSID {
			records, err := s.UserInfoCache.Read(key)
			if err != nil && !errors.Is(err, microstore.ErrNotFound) {
				logger.Error().Err(err).Msg("Error reading userinfo cache")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: err.Error()})
				return
			}
			for _, record := range records {
				err = s.UserInfoCache.Delete(string(record.Value))
				if err != nil && !errors.Is(err, microstore.ErrNotFound) {
					logger.Error().Err(err).Msg("Error deleting userinfo cache")
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: err.Error()})
					return
				}
			}
		}

		err = s.UserInfoCache.Delete(key)
		if err != nil && !errors.Is(err, microstore.ErrNotFound) {
			// Spec requires us to return a 400 BadRequest when the session could not be destroyed
			logger.Err(err).Msg(fmt.Errorf("could not delete session from cache (%s)", key).Error())
			// We only return on requests that do only attempt to destroy a single session, not multiple
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: err.Error()})
			return
		}
		if isSID {
			err := s.publishBackchannelLogoutEvent(r.Context(), key, logoutToken)
			if err != nil {
				s.Logger.Warn().Err(err).Msg("could not publish backchannel logout event")
			}
		}
		logger.Debug().Msg("Deleted userinfo from cache")
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, nil)
}

// publishBackchannelLogoutEvent publishes a backchannel logout event when the callback revived from the identity provider
func (s StaticRouteHandler) publishBackchannelLogoutEvent(ctx context.Context, cacheKey string, logoutToken *oidc.LogoutToken) error {
	if s.EventsPublisher == nil {
		return fmt.Errorf("the events publisher is not set")
	}
	urecords, err := s.UserInfoCache.Read(cacheKey)
	if err != nil {
		return fmt.Errorf("reading userinfo cache: %w", err)
	}
	if len(urecords) == 0 {
		return fmt.Errorf("userinfo not found")
	}

	var claims map[string]interface{}
	if err = msgpack.Unmarshal(urecords[0].Value, &claims); err != nil {
		return fmt.Errorf("could not unmarshal userinfo: %w", err)
	}

	oidcClaim, ok := claims[s.Config.UserOIDCClaim].(string)
	if !ok {
		return fmt.Errorf("could not get claim %w", err)
	}

	user, _, err := s.UserProvider.GetUserByClaims(ctx, s.Config.UserCS3Claim, oidcClaim)
	if err != nil || user.GetId() == nil {
		return fmt.Errorf("could not get user by claims: %w", err)
	}

	e := events.BackchannelLogout{
		Executant: user.GetId(),
		SessionId: logoutToken.SessionId,
		Timestamp: utils.TSNow(),
	}

	if err := events.Publish(ctx, s.EventsPublisher, e); err != nil {
		return fmt.Errorf("could not publish user created event %w", err)
	}
	return nil
}
