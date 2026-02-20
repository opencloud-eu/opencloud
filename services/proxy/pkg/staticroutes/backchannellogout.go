package staticroutes

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/render"
	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack/v5"
	microstore "go-micro.dev/v4/store"

	bcl "github.com/opencloud-eu/opencloud/services/proxy/pkg/staticroutes/internal/backchannellogout"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// NewRecordKey converts the subject and session to a base64 encoded key
var NewRecordKey = bcl.NewKey

// backchannelLogout handles backchannel logout requests from the identity provider and invalidates the related sessions in the cache
// spec: https://openid.net/specs/openid-connect-backchannel-1_0.html#BCRequest
//
// known side effects of backchannel logout in keycloak:
//
//   - keyCloak "Sign out all active sessions" does not send a backchannel logout request,
//     as the devs mention, this may lead to thousands of backchannel logout requests,
//     therefore, they recommend a short token lifetime.
//     https://github.com/keycloak/keycloak/issues/27342#issuecomment-2408461913
//
//   - keyCloak user self-service portal, "Sign out all devices" may not send a backchannel
//     logout request for each session, it's not mentionex explicitly,
//     but maybe the reason for that is the same as for "Sign out all active sessions"
//     to prevent a flood of backchannel logout requests.
//
//   - if the keycloak setting "Backchannel logout session required" is disabled (or the token has no session id),
//     we resolve the session by the subject which can lead to multiple session records (subject.*),
//     we then send a logout event (sse) to each connected client and delete our stored cache record (subject.session & claim).
//     all sessions besides the one that triggered the backchannel logout continue to exist in the identity provider,
//     so the user will not be fully logged out until all sessions are logged out or expired.
//     this leads to the situation that web renders the logout view even if the instance is not fully logged out yet.
func (s *StaticRouteHandler) backchannelLogout(w http.ResponseWriter, r *http.Request) {
	logger := s.Logger.SubloggerWithRequestID(r.Context())
	if err := r.ParseForm(); err != nil {
		logger.Warn().Err(err).Msg("ParseForm failed")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: err.Error()})
		return
	}

	logoutToken, err := s.OidcClient.VerifyLogoutToken(r.Context(), r.PostFormValue("logout_token"))
	if err != nil {
		msg := "failed to verify logout token"
		logger.Warn().Err(err).Msg(msg)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: msg})
		return
	}

	lookupKey, err := bcl.NewKey(logoutToken.Subject, logoutToken.SessionId)
	if err != nil {
		msg := "failed to build key from logout token"
		logger.Warn().Err(err).Msg(msg)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: msg})
		return
	}

	requestSubjectAndSession, err := bcl.NewSuSe(lookupKey)
	if err != nil {
		msg := "failed to build subjec.session from lookupKey"
		logger.Error().Err(err).Msg(msg)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: msg})
		return
	}

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
		msg := "failed to read userinfo cache"
		logger.Error().Err(err).Msg(msg)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: msg})
		return
	}

	for _, record := range lookupRecords {
		// the record key is in the format "subject.session" or ".session"
		// the record value is the key of the record that contains the claim in its value
		key, value := record.Key, string(record.Value)

		subjectSession, err := bcl.NewSuSe(key)
		if err != nil {
			// never leak any key-related information
			logger.Warn().Err(err).Msgf("failed to parse key: %s", key)
			continue
		}

		session, err := subjectSession.Session()
		if err != nil {
			logger.Warn().Err(err).Msgf("failed to read session for: %s", key)
			continue
		}

		if err := s.publishBackchannelLogoutEvent(r.Context(), session, value); err != nil {
			s.Logger.Warn().Err(err).Msgf("failed to publish backchannel logout event for: %s", key)
			continue
		}

		err = s.UserInfoCache.Delete(value)
		if err != nil && !errors.Is(err, microstore.ErrNotFound) {
			// we have to return a 400 BadRequest when we fail to delete the session
			// https://openid.net/specs/openid-connect-backchannel-1_0.html#rfc.section.2.8
			msg := "failed to delete record"
			s.Logger.Warn().Err(err).Msgf("%s for: %s", msg, key)
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, jse{Error: "invalid_request", ErrorDescription: msg})
			return
		}

		// we can ignore errors when deleting the lookup record
		err = s.UserInfoCache.Delete(key)
		if err != nil {
			logger.Debug().Err(err).Msgf("failed to delete record for: %s", key)
		}
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, nil)
}

// publishBackchannelLogoutEvent publishes a backchannel logout event when the callback revived from the identity provider
func (s *StaticRouteHandler) publishBackchannelLogoutEvent(ctx context.Context, sessionId, claimKey string) error {
	if s.EventsPublisher == nil {
		return errors.New("events publisher not set")
	}

	claimRecords, err := s.UserInfoCache.Read(claimKey)
	switch {
	case err != nil:
		return fmt.Errorf("failed to read userinfo cache: %w", err)
	case len(claimRecords) == 0:
		return fmt.Errorf("no claim found for key: %s", claimKey)
	}

	var claims map[string]interface{}
	if err = msgpack.Unmarshal(claimRecords[0].Value, &claims); err != nil {
		return fmt.Errorf("failed to unmarshal claims: %w", err)
	}

	oidcClaim, ok := claims[s.Config.UserOIDCClaim].(string)
	if !ok {
		return fmt.Errorf("failed to get claim %w", err)
	}

	user, _, err := s.UserProvider.GetUserByClaims(ctx, s.Config.UserCS3Claim, oidcClaim)
	if err != nil || user.GetId() == nil {
		return fmt.Errorf("failed to get user by claims: %w", err)
	}

	e := events.BackchannelLogout{
		Executant: user.GetId(),
		SessionId: sessionId,
		Timestamp: utils.TSNow(),
	}

	if err := events.Publish(ctx, s.EventsPublisher, e); err != nil {
		return fmt.Errorf("failed to publish user logout event %w", err)
	}
	return nil
}
