package staticroutes

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/render"
	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack/v5"
	microstore "go-micro.dev/v4/store"

	bcl "github.com/opencloud-eu/opencloud/services/proxy/pkg/staticroutes/internal/backchannellogout"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// BackchannelLogout handles backchannel logout requests from the identity provider and invalidates the related sessions in the cache
// spec: https://openid.net/specs/openid-connect-backchannel-1_0.html#BCRequest
//
// toDo:
//   - keyCloak "Sign out all active sessions" fails to log out, no incoming request
//   - if the keycloak setting "Backchannel logout session required" is disabled,
//     we resolve the session by the subject which can lead to multiple session records,
//     we then send a logout event to each connected client and delete our stored record (subject.session & claim).
//     but the session still exists in the identity provider.
func (s *StaticRouteHandler) backchannelLogout(w http.ResponseWriter, r *http.Request) {
	// parse the application/x-www-form-urlencoded POST request
	logger := s.Logger.SubloggerWithRequestID(r.Context())
	if err := r.ParseForm(); err != nil {
		logger.Warn().Err(err).Msg("ParseForm failed")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: err.Error()})
		return
	}

	logoutToken, err := s.OidcClient.VerifyLogoutToken(r.Context(), r.PostFormValue("logout_token"))
	if err != nil {
		logger.Warn().Err(err).Msg("VerifyLogoutToken failed")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: err.Error()})
		return
	}

	subject, session := strings.Join(strings.Fields(logoutToken.Subject), ""), strings.Join(strings.Fields(logoutToken.SessionId), "")
	if subject == "" && session == "" {
		jseErr := jse{Error: "invalid_request", ErrorDescription: "invalid logout token: subject and session id are empty"}
		logger.Warn().Msg(jseErr.ErrorDescription)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, jseErr)
		return
	}

	requestSubjectAndSession := bcl.SuSe{Session: session, Subject: subject}
	// find out which mode of backchannel logout we are in
	// by checking if the session or subject is present in the token
	logoutMode := bcl.GetLogoutMode(requestSubjectAndSession)
	lookupRecords, err := bcl.GetLogoutRecords(requestSubjectAndSession, logoutMode, s.UserInfoCache)
	if errors.Is(err, microstore.ErrNotFound) || len(lookupRecords) == 0 {
		render.Status(r, http.StatusOK)
		render.JSON(w, r, nil)
		return
	}
	if err != nil {
		logger.Error().Err(err).Msg("Error reading userinfo cache")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: err.Error()})
		return
	}

	for _, record := range lookupRecords {
		// the record key is in the format "subject.session" or ".session"
		// the record value is the key of the record that contains the claim in its value
		key, value := record.Key, string(record.Value)

		subjectSession, ok := bcl.NewSuSe(key)
		if !ok {
			logger.Warn().Msgf("invalid logout record key: %s", key)
			continue
		}

		err := s.publishBackchannelLogoutEvent(r.Context(), subjectSession.Session, value)
		if err != nil {
			s.Logger.Warn().Err(err).Msg("could not publish backchannel logout event")
		}

		err = s.UserInfoCache.Delete(value)
		if err != nil && !errors.Is(err, microstore.ErrNotFound) {
			// we have to return a 400 BadRequest when we fail to delete the session
			// https://openid.net/specs/openid-connect-backchannel-1_0.html#rfc.section.2.8
			logger.Err(err).Msg("could not delete user info from cache")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: err.Error()})
			return
		}

		// we can ignore errors when deleting the lookup record
		err = s.UserInfoCache.Delete(key)
		if err != nil {
			logger.Debug().Err(err).Msg("Failed to cleanup sessionid lookup entry")
		}
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, nil)
}

// publishBackchannelLogoutEvent publishes a backchannel logout event when the callback revived from the identity provider
func (s *StaticRouteHandler) publishBackchannelLogoutEvent(ctx context.Context, sessionId, claimKey string) error {
	if s.EventsPublisher == nil {
		return fmt.Errorf("the events publisher is not set")
	}
	claimRecords, err := s.UserInfoCache.Read(claimKey)
	if err != nil {
		return fmt.Errorf("reading userinfo cache: %w", err)
	}
	if len(claimRecords) == 0 {
		return fmt.Errorf("userinfo not found")
	}

	var claims map[string]interface{}
	if err = msgpack.Unmarshal(claimRecords[0].Value, &claims); err != nil {
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
		SessionId: sessionId,
		Timestamp: utils.TSNow(),
	}

	if err := events.Publish(ctx, s.EventsPublisher, e); err != nil {
		return fmt.Errorf("could not publish user created event %w", err)
	}
	return nil
}
