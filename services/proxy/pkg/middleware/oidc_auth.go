package middleware

import (
	"context"
	"encoding/base64"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack/v5"
	"go-micro.dev/v4/store"
	"golang.org/x/crypto/sha3"
	"golang.org/x/oauth2"

	"github.com/opencloud-eu/opencloud/pkg/log"
	ocmiddleware "github.com/opencloud-eu/opencloud/pkg/middleware"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	bcl "github.com/opencloud-eu/opencloud/services/proxy/pkg/staticroutes/backchannellogout"
)

const (
	_headerAuthorization = "Authorization"
	_bearerPrefix        = "Bearer "
)

// NewOIDCAuthenticator returns a ready to use authenticator which can handle OIDC authentication.
func NewOIDCAuthenticator(opts ...Option) *OIDCAuthenticator {
	options := newOptions(opts...)

	return &OIDCAuthenticator{
		Logger:                  options.Logger,
		userInfoCache:           options.UserInfoCache,
		HTTPClient:              options.HTTPClient,
		OIDCIss:                 options.OIDCIss,
		DefaultTokenCacheTTL:    options.DefaultAccessTokenTTL,
		oidcClient:              options.OIDCClient,
		AccessTokenVerifyMethod: options.AccessTokenVerifyMethod,
		skipUserInfo:            options.SkipUserInfo,
		logoutMarkers:           options.LogoutMarkers,
		maxAccessTokenLifetime:  options.MaxAccessTokenLifetime,
		TimeFunc:                time.Now,
	}
}

// OIDCAuthenticator is an authenticator responsible for OIDC authentication.
type OIDCAuthenticator struct {
	Logger                  log.Logger
	HTTPClient              *http.Client
	OIDCIss                 string
	userInfoCache           store.Store
	DefaultTokenCacheTTL    time.Duration
	oidcClient              oidc.OIDCClient
	AccessTokenVerifyMethod string
	skipUserInfo            bool
	logoutMarkers           *bcl.LogoutMarkers
	maxAccessTokenLifetime  time.Duration
	TimeFunc                func() time.Time
}

func (m *OIDCAuthenticator) getClaims(token string, req *http.Request) (map[string]any, bool, error) {
	var claims map[string]any

	// use a 64 bytes long hash to have 256-bit collision resistance.
	hash := make([]byte, 64)
	sha3.ShakeSum256(hash, []byte(token))
	encodedHash := base64.URLEncoding.EncodeToString(hash)

	record, err := m.userInfoCache.Read(encodedHash)
	if err != nil && err != store.ErrNotFound {
		m.Logger.Error().Err(err).Msg("could not read from userinfo cache")
	}
	if len(record) > 0 {
		var cached bcl.TokenRecord
		if err = msgpack.Unmarshal(record[0].Value, &cached); err == nil {
			m.Logger.Debug().Interface("claims", cached.Claims).Msg("cache hit for userinfo")
			if verifyExpiresAt(cached.Claims, m.TimeFunc()) {
				if err := m.verifyNotLoggedOut(cached.Subject, cached.SessionID, cached.IssuedAt); err != nil {
					return nil, false, err
				}
				return cached.Claims, false, nil
			}
			m.Logger.Debug().Msg("cached userinfo claims expired, ignoring cache")
		} else {
			m.Logger.Error().Err(err).Msg("failed to unmarshal cached userinfo, ignoring cache")
		}
	}

	aClaims, claims, err := m.oidcClient.VerifyAccessToken(req.Context(), token)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to verify access token")
	}

	if err := m.verifyLifetime(aClaims); err != nil {
		return nil, false, err
	}

	issuedAt := int64(0)
	if aClaims.IssuedAt != nil {
		issuedAt = aClaims.IssuedAt.Unix()
	}
	if err := m.verifyNotLoggedOut(aClaims.Subject, aClaims.SessionID, issuedAt); err != nil {
		return nil, false, err
	}

	if !m.skipUserInfo {
		oauth2Token := &oauth2.Token{
			AccessToken: token,
		}

		userInfo, err := m.oidcClient.UserInfo(
			context.WithValue(req.Context(), oauth2.HTTPClient, m.HTTPClient),
			oauth2.StaticTokenSource(oauth2Token),
		)
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to get userinfo")
		}
		if err := userInfo.Claims(&claims); err != nil {
			return nil, false, errors.Wrap(err, "failed to unmarshal userinfo claims")
		}
	}

	expiration := m.extractExpiration(aClaims)
	// always set an exp claim
	claims["exp"] = expiration.Unix()

	d, err := msgpack.Marshal(bcl.TokenRecord{
		Claims:    claims,
		Subject:   aClaims.Subject,
		SessionID: aClaims.SessionID,
		IssuedAt:  issuedAt,
	})
	if err != nil {
		m.Logger.Error().Err(err).Msg("failed to marshal claims for userinfo cache")

		return claims, true, nil
	}

	go func() {
		err = m.userInfoCache.Write(&store.Record{
			Key:    encodedHash,
			Value:  d,
			Expiry: time.Until(expiration),
		})
		if err != nil {
			m.Logger.Error().Err(err).Msg("failed to write to userinfo cache")
		}

		// fail if creating the storage key fails,
		// it means there is no subject and no session.
		//
		// ok: {key: ".sessionId"}
		// ok: {key: "subject."}
		// ok: {key: "subject.sessionId"}
		// fail: {key: "."}
		subjectSessionKey, err := bcl.NewKey(aClaims.Subject, aClaims.SessionID)
		switch {
		// fails if the verify method is set to `none`, in that case the oidc client verification returns
		// an empty oidcclient.RegClaimsWithSID but no err.
		//
		// revisit once:
		//   - Authelia OpenID Connect Back-Channel Logout 1.0 is implemented,
		//     e.g. https://www.authelia.com/roadmap/active/openid-connect-1.0-provider/#beta-9
		case m.AccessTokenVerifyMethod == config.AccessTokenVerificationNone && errors.Is(err, bcl.ErrInvalidKey):
			return
		case err != nil:
			m.Logger.Error().Err(err).Msg("failed to build subject.session")
			return
		}

		if err := m.userInfoCache.Write(&store.Record{
			Key:    subjectSessionKey,
			Value:  []byte(encodedHash),
			Expiry: time.Until(expiration),
		}); err != nil {
			m.Logger.Error().Err(err).Msg("failed to write session lookup cache")
		}
	}()

	// If we get here this was a new login (or a renewal of the token)
	// add a flag about that to the claims, to be able to distinguish
	// it in the accountresolver middleware

	m.Logger.Debug().Interface("claims", claims).Msg("extracted claims")
	return claims, true, nil
}

// verifyNotLoggedOut rejects a token that is covered by a backchannel logout.
// It also rejects the token if the markers cannot be read, so that a store
// failure cannot make a logged out token valid again.
func (m *OIDCAuthenticator) verifyNotLoggedOut(subject, session string, issuedAt int64) error {
	if m.logoutMarkers == nil {
		return nil
	}
	return m.logoutMarkers.Verify(subject, session, issuedAt)
}

// verifyLifetime rejects JWT access tokens that could outlive the logout markers.
// Markers are only kept for the maximum lifetime, so a longer-lived token could
// become valid again after its marker has expired.
func (m *OIDCAuthenticator) verifyLifetime(aClaims oidc.RegClaimsWithSID) error {
	if m.maxAccessTokenLifetime <= 0 || m.AccessTokenVerifyMethod != config.AccessTokenVerificationJWT {
		return nil
	}
	if aClaims.ExpiresAt == nil {
		return errors.New("access token has no expiration")
	}

	start := m.TimeFunc()
	if aClaims.IssuedAt != nil {
		start = aClaims.IssuedAt.Time
	}
	if aClaims.ExpiresAt.Sub(start) > m.maxAccessTokenLifetime {
		return errors.Errorf("access token lifetime exceeds the maximum of %s", m.maxAccessTokenLifetime)
	}
	return nil
}

// extractExpiration tries to extract the expriration time from the access token
// If the access token does not have an exp claim it will fallback to the configured
// default expiration
func (m OIDCAuthenticator) extractExpiration(aClaims oidc.RegClaimsWithSID) time.Time {
	defaultExpiration := time.Now().Add(m.DefaultTokenCacheTTL)
	if aClaims.ExpiresAt != nil {
		m.Logger.Debug().Str("exp", aClaims.ExpiresAt.String()).Msg("Expiration Time from access_token")
		return aClaims.ExpiresAt.Time
	}
	return defaultExpiration
}

func verifyExpiresAt(claims map[string]any, cmp time.Time) bool {
	var expiry time.Time
	switch v := claims["exp"].(type) {
	case nil:
		return false
	case int64:
		expiry = time.Unix(v, 0)
	case uint32:
		expiry = time.Unix(int64(v), 0)
	}
	return cmp.Before(expiry)
}

// isExpectedTokenError reports whether the error is caused by a token that
// failed its expiry (or not-before) validation or was issued before a backchannel
// logout. These are expected as part of the normal token renewal cycle and
// should not be logged at error level.
func isExpectedTokenError(err error) bool {
	return errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) || errors.Is(err, bcl.ErrLoggedOut)
}

func (m OIDCAuthenticator) shouldServe(req *http.Request) bool {
	if m.OIDCIss == "" {
		return false
	}

	header := req.Header.Get(_headerAuthorization)
	return strings.HasPrefix(header, _bearerPrefix)
}

// Authenticate implements the authenticator interface to authenticate requests via oidc auth.
func (m *OIDCAuthenticator) Authenticate(r *http.Request) AuthenticationResult {
	// there is no bearer token on the request,
	if !m.shouldServe(r) {
		// The authentication of public path requests is handled by another authenticator.
		// Since we can't guarantee the order of execution of the authenticators, we better
		// implement an early return here for paths we can't authenticate in this authenticator.
		return NotApplicable()
	}
	token := strings.TrimPrefix(r.Header.Get(_headerAuthorization), _bearerPrefix)
	if token == "" {
		return NotApplicable()
	}

	claims, newSession, err := m.getClaims(token, r)
	if err != nil {
		host, port, _ := net.SplitHostPort(r.RemoteAddr)

		// Expired (or not yet valid) tokens are a normal part of the token
		// renewal cycle: the client hits the proxy with a token that is just
		// past its expiry, receives a 401 challenge and refreshes. This is not
		// a failure of the proxy, so we log it at debug level to avoid noise.
		logEvent := m.Logger.Debug()
		if !isExpectedTokenError(err) {
			logEvent = m.Logger.Error()
		}

		logEvent.
			Err(err).
			Str("authenticator", "oidc").
			Str("path", r.URL.Path).
			Str("user_agent", r.UserAgent()).
			Str("client.address", ocmiddleware.ClientAddress(r)).
			Str("network.peer.address", host).
			Str("network.peer.port", port).
			Msg("failed to authenticate the request")
		return FailedWithErr(err)
	}
	m.Logger.Debug().
		Str("authenticator", "oidc").
		Str("path", r.URL.Path).
		Msg("successfully authenticated request")

	ctx := r.Context()
	if newSession {
		ctx = oidc.NewContextSessionFlag(ctx, true)
	}

	return Succeeded(r.WithContext(oidc.NewContext(ctx, claims)))
}
