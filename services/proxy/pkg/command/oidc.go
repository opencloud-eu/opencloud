package command

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/middleware"
	"go-micro.dev/v4/store"
)

func newOIDCAuthenticator(logger log.Logger, cfg *config.Config, userInfoCache store.Store, httpClient *http.Client) *middleware.OIDCAuthenticator {
	if cfg.OIDC.Issuer != "" && len(cfg.OIDC.Audiences) == 0 {
		logger.Warn().Msg("OIDC access token audience validation is disabled. Configure PROXY_OIDC_AUDIENCES to enable it; this is recommended for production.")
	}

	return middleware.NewOIDCAuthenticator(
		middleware.Logger(logger),
		middleware.UserInfoCache(userInfoCache),
		middleware.DefaultAccessTokenTTL(cfg.OIDC.UserinfoCache.TTL),
		middleware.HTTPClient(httpClient),
		middleware.OIDCIss(cfg.OIDC.Issuer),
		middleware.AccessTokenVerifyMethod(cfg.OIDC.AccessTokenVerifyMethod),
		middleware.ValidateAccessTokenOnCacheHit(len(cfg.OIDC.Audiences) > 0),
		middleware.OIDCClient(oidc.NewOIDCClient(
			oidc.WithAccessTokenVerifyMethod(cfg.OIDC.AccessTokenVerifyMethod),
			oidc.WithAccessTokenAudiences(cfg.OIDC.Audiences),
			oidc.WithLogger(logger),
			oidc.WithHTTPClient(httpClient),
			oidc.WithOidcIssuer(cfg.OIDC.Issuer),
			oidc.WithJWKSOptions(cfg.OIDC.JWKS),
		)),
		middleware.SkipUserInfo(cfg.OIDC.SkipUserInfo),
	)
}
