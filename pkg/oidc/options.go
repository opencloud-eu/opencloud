package oidc

import (
	"net/http"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"

	goidc "github.com/coreos/go-oidc/v3/oidc"
)

// Option defines a single option function.
type Option func(o *Options)

// Options defines the available options for this package.
type Options struct {
	// HTTPClient to use for requests
	HTTPClient *http.Client
	// Logger to use for logging, must be set
	Logger log.Logger
	// The OpenID Connect Issuer URL
	OIDCIssuer string
	// JWKSOptions to use when retrieving keys
	JWKSOptions config.JWKS
	// the JWKS keyset to use for verifying signatures of Access- and
	// Logout-Tokens
	// this option is mostly needed for unit test. To avoid fetching the keys
	// from the issuer
	JWKS *keyfunc.JWKS
	// KeySet to use when verifiing signatures of jwt encoded
	// user info responses
	// TODO move userinfo verification to use jwt/keyfunc as well
	KeySet KeySet
	// AccessTokenVerifyMethod to use when verifying access tokens
	// TODO pass a function or interface to verify? an AccessTokenVerifier?
	AccessTokenVerifyMethod string
	// AccessTokenAudiences requires at least one matching audience in access tokens.
	// An empty list disables audience validation.
	AccessTokenAudiences []string
	// Config to use
	Config *goidc.Config

	// ProviderMetadata to use
	ProviderMetadata *ProviderMetadata
}

// newOptions initializes the available default options.
func newOptions(opts ...Option) Options {
	opt := Options{}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

// WithOidcIssuer provides a function to set the openid connect issuer option.
func WithOidcIssuer(val string) Option {
	return func(o *Options) {
		o.OIDCIssuer = val
	}
}

// WithLogger provides a function to set the logger option.
func WithLogger(val log.Logger) Option {
	return func(o *Options) {
		o.Logger = val
	}
}

// WithAccessTokenVerifyMethod provides a function to set the accessTokenVerifyMethod option.
func WithAccessTokenVerifyMethod(val string) Option {
	return func(o *Options) {
		o.AccessTokenVerifyMethod = val
	}
}

// WithAccessTokenAudiences sets the allowed audiences for access tokens only.
// An empty list disables audience validation.
func WithAccessTokenAudiences(val []string) Option {
	return func(o *Options) {
		o.AccessTokenAudiences = append([]string(nil), val...)
	}
}

// WithHTTPClient provides a function to set the httpClient option.
func WithHTTPClient(val *http.Client) Option {
	return func(o *Options) {
		o.HTTPClient = val
	}
}

// WithJWKSOptions provides a function to set the jwksOptions option.
func WithJWKSOptions(val config.JWKS) Option {
	return func(o *Options) {
		o.JWKSOptions = val
	}
}

// WithJWKS provides a function to set the JWKS option (mainly useful for testing).
func WithJWKS(val *keyfunc.JWKS) Option {
	return func(o *Options) {
		o.JWKS = val
	}
}

// WithKeySet provides a function to set the KeySet option.
func WithKeySet(val KeySet) Option {
	return func(o *Options) {
		o.KeySet = val
	}
}

// WithConfig provides a function to set the Config option.
func WithConfig(val *goidc.Config) Option {
	return func(o *Options) {
		o.Config = val
	}
}

// WithProviderMetadata provides a function to set the provider option.
func WithProviderMetadata(val *ProviderMetadata) Option {
	return func(o *Options) {
		o.ProviderMetadata = val
	}
}
