package middleware

import (
	"net/http"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	policiessvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/policies/v0"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/user/backend"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/userroles"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"go-micro.dev/v4/store"
	"go.opentelemetry.io/otel/trace"
)

// Option defines a single option function.
type Option func(o *Options)

// Options defines the available options for this package.
type Options struct {
	// Logger to use for logging, must be set
	Logger log.Logger
	// PolicySelectorConfig for using the policy selector
	PolicySelector config.PolicySelector
	// HTTPClient to use for communication with the oidcAuth provider
	HTTPClient *http.Client
	// UserProvider backend to use for resolving User
	UserProvider backend.UserBackend
	// UserRoleAssigner to user for assign a users default role
	UserRoleAssigner userroles.UserRoleAssigner
	// SettingsRoleService for the roles API in settings
	SettingsRoleService settingssvc.RoleService
	// PoliciesProviderService for policy evaluation
	PoliciesProviderService policiessvc.PoliciesProviderService
	// OIDCClient to fetch user info and verify tokens, must be set for the oidc_auth middleware
	OIDCClient oidc.OIDCClient
	// OIDCIss is the oidcAuth-issuer
	OIDCIss string
	// RevaGatewaySelector to send requests to the reva gateway
	RevaGatewaySelector pool.Selectable[gateway.GatewayAPIClient]
	// PreSignedURLConfig to configure the middleware
	PreSignedURLConfig config.PreSignedURL
	// UserOIDCClaim to read from the oidc claims
	UserOIDCClaim string
	// UserCS3Claim to use when looking up a user in the CS3 API
	UserCS3Claim string
	// AutoprovisionAccounts when an accountResolver does not exist.
	AutoprovisionAccounts bool
	// EnableBasicAuth to allow basic auth
	EnableBasicAuth bool
	// DefaultAccessTokenTTL is used to calculate the expiration when an access token has no expiration set
	DefaultAccessTokenTTL time.Duration
	// UserInfoCache sets the access token cache store
	UserInfoCache store.Store
	// CredentialsByUserAgent sets the auth challenges on a per user-agent basis
	CredentialsByUserAgent map[string]string
	// AccessTokenVerifyMethod configures how access_tokens should be verified but the oidc_auth middleware.
	// Possible values currently: "jwt" and "none"
	AccessTokenVerifyMethod string
	// JWKS sets the options for fetching the JWKS from the IDP
	JWKS config.JWKS
	// RoleQuotas hold userid:quota mappings. These will be used when provisioning new users.
	// The users will get as much quota as is set for their role.
	RoleQuotas map[string]uint64
	// TraceProvider sets the tracing provider.
	TraceProvider trace.TracerProvider
	// SkipUserInfo prevents the oidc middleware from querying the userinfo endpoint and read any claims directly from the access token instead
	SkipUserInfo bool
	// MultiTenantEnabled causes the account resolve middleware to reject users that don't have a tenant id assigned
	MultiTenantEnabled bool
	EventsPublisher    events.Publisher
}

// newOptions initializes the available default options.
func newOptions(opts ...Option) Options {
	opt := Options{}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

// Logger provides a function to set the logger option.
func Logger(l log.Logger) Option {
	return func(o *Options) {
		o.Logger = l
	}
}

// PolicySelectorConfig provides a function to set the policy selector config option.
func PolicySelectorConfig(cfg config.PolicySelector) Option {
	return func(o *Options) {
		o.PolicySelector = cfg
	}
}

// HTTPClient provides a function to set the http client config option.
func HTTPClient(c *http.Client) Option {
	return func(o *Options) {
		o.HTTPClient = c
	}
}

// SettingsRoleService provides a function to set the role service option.
func SettingsRoleService(rc settingssvc.RoleService) Option {
	return func(o *Options) {
		o.SettingsRoleService = rc
	}
}

// PoliciesProviderService provides a function to set the policies provider option.
func PoliciesProviderService(pps policiessvc.PoliciesProviderService) Option {
	return func(o *Options) {
		o.PoliciesProviderService = pps
	}
}

// OIDCClient provides a function to set the oidc client option.
func OIDCClient(val oidc.OIDCClient) Option {
	return func(o *Options) {
		o.OIDCClient = val
	}
}

// OIDCIss sets the oidcAuth issuer url
func OIDCIss(iss string) Option {
	return func(o *Options) {
		o.OIDCIss = iss
	}
}

// CredentialsByUserAgent sets UserAgentChallenges.
func CredentialsByUserAgent(v map[string]string) Option {
	return func(o *Options) {
		o.CredentialsByUserAgent = v
	}
}

// WithRevaGatewaySelector provides a function to set the reva gateway service selector option.
func WithRevaGatewaySelector(val pool.Selectable[gateway.GatewayAPIClient]) Option {
	return func(o *Options) {
		o.RevaGatewaySelector = val
	}
}

// PreSignedURLConfig provides a function to set the PreSignedURL config
func PreSignedURLConfig(cfg config.PreSignedURL) Option {
	return func(o *Options) {
		o.PreSignedURLConfig = cfg
	}
}

// UserOIDCClaim provides a function to set the UserClaim config
func UserOIDCClaim(val string) Option {
	return func(o *Options) {
		o.UserOIDCClaim = val
	}
}

// UserCS3Claim provides a function to set the UserClaimType config
func UserCS3Claim(val string) Option {
	return func(o *Options) {
		o.UserCS3Claim = val
	}
}

// AutoprovisionAccounts provides a function to set the AutoprovisionAccounts config
func AutoprovisionAccounts(val bool) Option {
	return func(o *Options) {
		o.AutoprovisionAccounts = val
	}
}

// EnableBasicAuth provides a function to set the EnableBasicAuth config
func EnableBasicAuth(enableBasicAuth bool) Option {
	return func(o *Options) {
		o.EnableBasicAuth = enableBasicAuth
	}
}

// DefaultAccessTokenTTL provides a function to set the DefaultAccessTokenTTL
func DefaultAccessTokenTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.DefaultAccessTokenTTL = ttl
	}
}

// UserInfoCache provides a function to set the UserInfoCache
func UserInfoCache(val store.Store) Option {
	return func(o *Options) {
		o.UserInfoCache = val
	}
}

// UserProvider sets the accounts user provider
func UserProvider(up backend.UserBackend) Option {
	return func(o *Options) {
		o.UserProvider = up
	}
}

// UserRoleAssigner sets the mechanism for assigning the default user roles
func UserRoleAssigner(ra userroles.UserRoleAssigner) Option {
	return func(o *Options) {
		o.UserRoleAssigner = ra
	}
}

// AccessTokenVerifyMethod set the mechanism for access token verification
func AccessTokenVerifyMethod(method string) Option {
	return func(o *Options) {
		o.AccessTokenVerifyMethod = method
	}
}

// RoleQuotas sets the role quota mapping setting
func RoleQuotas(roleQuotas map[string]uint64) Option {
	return func(o *Options) {
		o.RoleQuotas = roleQuotas
	}
}

// TraceProvider sets the tracing provider.
func TraceProvider(tp trace.TracerProvider) Option {
	return func(o *Options) {
		o.TraceProvider = tp
	}
}

// SkipUserInfo sets the skipUserInfo flag.
func SkipUserInfo(val bool) Option {
	return func(o *Options) {
		o.SkipUserInfo = val
	}
}

// MultiTenantEnabled sets the MultiTenantEnabled flag.
func MultiTenantEnabled(val bool) Option {
	return func(o *Options) {
		o.MultiTenantEnabled = val
	}
}

// EventsPublisher sets the events publisher.
func EventsPublisher(ep events.Publisher) Option {
	return func(o *Options) {
		o.EventsPublisher = ep
	}
}
