package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"

	goidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	"golang.org/x/oauth2"
)

// newOidcOptions initializes the available default options.
func newOidcOptions(opts ...Option) Options {
	opt := Options{}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

// OIDCProvider used to mock the oidc provider during tests
type OIDCProvider interface {
	UserInfo(ctx context.Context, ts oauth2.TokenSource) (*goidc.UserInfo, error)
}

// OidcAuth provides a middleware to authenticate a bearer auth with an OpenID Connect identity provider
// It will put all claims provided by the userinfo endpoint in the context
func OidcAuth(opts ...Option) func(http.Handler) http.Handler {
	opt := newOidcOptions(opts...)

	// TODO use a micro store cache option

	providerFunc := func() (OIDCProvider, error) {
		// Initialize a provider by specifying the issuer URL.
		// it will fetch the keys from the issuer using the .well-known
		// endpoint
		return goidc.NewProvider(
			context.WithValue(context.Background(), oauth2.HTTPClient, &opt.HttpClient),
			opt.OidcIssuer,
		)
	}
	var provider OIDCProvider
	initializeProviderLock := sync.Mutex{}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			authHeader := r.Header.Get("Authorization")
			switch {
			case strings.HasPrefix(authHeader, "Bearer "):
				if provider == nil {
					// lazy initialize provider
					initializeProviderLock.Lock()
					var err error
					// ensure no other request initialized the provider
					if provider == nil {
						provider, err = providerFunc()
					}
					initializeProviderLock.Unlock()
					if err != nil {
						opt.Logger.Error().Err(err).Msg("could not initialize OIDC provider")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					opt.Logger.Debug().Msg("initialized OIDC provider")
				}

				oauth2Token := &oauth2.Token{
					AccessToken: strings.TrimPrefix(authHeader, "Bearer "),
				}

				userInfo, err := provider.UserInfo(
					context.WithValue(ctx, oauth2.HTTPClient, &opt.HttpClient),
					oauth2.StaticTokenSource(oauth2Token),
				)
				if err != nil {
					w.Header().Add("WWW-Authenticate", `Bearer`)
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				claims := map[string]any{}
				err = userInfo.Claims(&claims)
				if err != nil {
					break
				}

				ctx = oidc.NewContext(ctx, claims)

			default:
				// do nothing
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
