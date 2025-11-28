package config

import (
	"context"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	"go-micro.dev/v4/client"
)

// Config combines all available configuration parts.
type Config struct {
	Commons *shared.Commons `yaml:"-" mask:"struct"` // don't use this directly as configuration for a service

	Service Service `yaml:"-"`

	Log   *Log  `yaml:"log"`
	Debug Debug `yaml:"debug" mask:"struct"`

	HTTP HTTP `yaml:"http"`

	Reva          *shared.Reva          `yaml:"reva"`
	GRPCClientTLS *shared.GRPCClientTLS `yaml:"grpc_client_tls"`
	GrpcClient    client.Client         `yaml:"-"`

	RoleQuotas                    map[string]uint64   `yaml:"role_quotas"`
	Policies                      []Policy            `yaml:"policies"`
	AdditionalPolicies            []Policy            `yaml:"additional_policies"`
	OIDC                          OIDC                `yaml:"oidc"`
	ServiceAccount                ServiceAccount      `yaml:"service_account"`
	RoleAssignment                RoleAssignment      `yaml:"role_assignment"`
	PolicySelector                *PolicySelector     `yaml:"policy_selector"`
	PreSignedURL                  PreSignedURL        `yaml:"pre_signed_url"`
	AccountBackend                string              `yaml:"account_backend" env:"PROXY_ACCOUNT_BACKEND_TYPE" desc:"Account backend the PROXY service should use. Currently only 'cs3' is possible here." introductionVersion:"1.0.0"`
	UserOIDCClaim                 string              `yaml:"user_oidc_claim" env:"PROXY_USER_OIDC_CLAIM" desc:"The name of an OpenID Connect claim that is used for resolving users with the account backend. The value of the claim must hold a per user unique, stable and non re-assignable identifier. The availability of claims depends on your Identity Provider. There are common claims available for most Identity providers like 'email' or 'preferred_username' but you can also add your own claim." introductionVersion:"1.0.0"`
	UserCS3Claim                  string              `yaml:"user_cs3_claim" env:"PROXY_USER_CS3_CLAIM" desc:"The name of a CS3 user attribute (claim) that should be mapped to the 'user_oidc_claim'. Supported values are 'username', 'mail' and 'userid'." introductionVersion:"1.0.0"`
	MachineAuthAPIKey             string              `yaml:"machine_auth_api_key" env:"OC_MACHINE_AUTH_API_KEY;PROXY_MACHINE_AUTH_API_KEY" desc:"Machine auth API key used to validate internal requests necessary to access resources from other services." introductionVersion:"1.0.0" mask:"password"`
	AutoprovisionAccounts         bool                `yaml:"auto_provision_accounts" env:"PROXY_AUTOPROVISION_ACCOUNTS" desc:"Set this to 'true' to automatically provision users that do not yet exist in the users service on-demand upon first sign-in. To use this a write-enabled libregraph user backend needs to be setup an running." introductionVersion:"1.0.0"`
	AutoProvisionClaims           AutoProvisionClaims `yaml:"auto_provision_claims"`
	EnableBasicAuth               bool                `yaml:"enable_basic_auth" env:"PROXY_ENABLE_BASIC_AUTH" desc:"Set this to true to enable 'basic authentication' (username/password)." introductionVersion:"1.0.0"`
	InsecureBackends              bool                `yaml:"insecure_backends" env:"PROXY_INSECURE_BACKENDS" desc:"Disable TLS certificate validation for all HTTP backend connections." introductionVersion:"1.0.0"`
	BackendHTTPSCACert            string              `yaml:"backend_https_cacert" env:"PROXY_HTTPS_CACERT" desc:"Path/File for the root CA certificate used to validate the server’s TLS certificate for https enabled backend services." introductionVersion:"1.0.0"`
	AuthMiddleware                AuthMiddleware      `yaml:"auth_middleware"`
	PoliciesMiddleware            PoliciesMiddleware  `yaml:"policies_middleware"`
	CSPConfigFileLocation         string              `yaml:"csp_config_file_location" env:"PROXY_CSP_CONFIG_FILE_LOCATION" desc:"The location of the CSP configuration file." introductionVersion:"1.0.0"`
	CSPConfigFileOverrideLocation string              `yaml:"csp_config_file_override_location" env:"PROXY_CSP_CONFIG_FILE_OVERRIDE_LOCATION" desc:"The location of the CSP configuration file override." introductionVersion:"4.0.0"`
	Events                        Events              `yaml:"events"`

	Context context.Context `json:"-" yaml:"-"`
}

// Policy enables us to use multiple directors.
type Policy struct {
	Name   string  `yaml:"name"`
	Routes []Route `yaml:"routes"`
}

// Route defines forwarding routes
type Route struct {
	Type RouteType `yaml:"type,omitempty"`
	// Method optionally limits the route to this HTTP method
	Method   string `yaml:"method,omitempty"`
	Endpoint string `yaml:"endpoint,omitempty"`
	// Backend is a static URL to forward the request to
	Backend string `yaml:"backend,omitempty"`
	// Service name to look up in the registry
	Service           string            `yaml:"service,omitempty"`
	ApacheVHost       bool              `yaml:"apache_vhost,omitempty"`
	Unprotected       bool              `yaml:"unprotected,omitempty"`
	AdditionalHeaders map[string]string `yaml:"additional_headers,omitempty"`
	RemoteUserHeader  string            `yaml:"remote_user_header,omitempty"`
	SkipXAccessToken  bool              `yaml:"skip_x_access_token"`
}

// RouteType defines the type of route
type RouteType string

const (
	// PrefixRoute are routes matched by a prefix
	PrefixRoute RouteType = "prefix"
	// QueryRoute are routes matched by a prefix and query parameters
	QueryRoute RouteType = "query"
	// RegexRoute are routes matched by a pattern
	RegexRoute RouteType = "regex"
	// DefaultRouteType is the PrefixRoute
	DefaultRouteType RouteType = PrefixRoute
)

var (
	// RouteTypes is an array of the available route types
	RouteTypes = []RouteType{QueryRoute, RegexRoute, PrefixRoute}
)

// AuthMiddleware configures the proxy http auth middleware.
type AuthMiddleware struct {
	CredentialsByUserAgent map[string]string `yaml:"credentials_by_user_agent"`
	AllowAppAuth           bool              `yaml:"allow_app_auth" env:"PROXY_ENABLE_APP_AUTH" desc:"Allow app authentication. This can be used to authenticate 3rd party applications. Note that auth-app service must be running for this feature to work." introductionVersion:"1.0.0"`
}

// PoliciesMiddleware configures the proxy's policies middleware.
type PoliciesMiddleware struct {
	Query string `yaml:"query" env:"PROXY_POLICIES_QUERY" desc:"Defines the 'Complete Rules' variable defined in the rego rule set this step uses for its evaluation. Rules default to deny if the variable was not found." introductionVersion:"1.0.0"`
}

const (
	AccessTokenVerificationNone = "none"
	AccessTokenVerificationJWT  = "jwt"
	// tdb:
	// AccessTokenVerificationIntrospect = "introspect"
)

// OIDC is the config for the OpenID-Connect middleware. If set the proxy will try to authenticate every request
// with the configured oidc-provider
type OIDC struct {
	Issuer                  string `yaml:"issuer" env:"OC_URL;OC_OIDC_ISSUER;PROXY_OIDC_ISSUER" desc:"URL of the OIDC issuer. It defaults to URL of the builtin IDP." introductionVersion:"1.0.0"`
	Insecure                bool   `yaml:"insecure" env:"OC_INSECURE;PROXY_OIDC_INSECURE" desc:"Disable TLS certificate validation for connections to the IDP. Note that this is not recommended for production environments." introductionVersion:"1.0.0"`
	AccessTokenVerifyMethod string `yaml:"access_token_verify_method" env:"PROXY_OIDC_ACCESS_TOKEN_VERIFY_METHOD" desc:"Sets how OIDC access tokens should be verified. Possible values are 'none' and 'jwt'. When using 'none', no special validation apart from using it for accessing the IDP's userinfo endpoint will be done. When using 'jwt', it tries to parse the access token as a jwt token and verifies the signature using the keys published on the IDP's 'jwks_uri'." introductionVersion:"1.0.0"`
	SkipUserInfo            bool   `yaml:"skip_user_info" env:"PROXY_OIDC_SKIP_USER_INFO" desc:"Do not look up user claims at the userinfo endpoint and directly read them from the access token. Incompatible with 'PROXY_OIDC_ACCESS_TOKEN_VERIFY_METHOD=none'." introductionVersion:"1.0.0"`
	UserinfoCache           *Cache `yaml:"user_info_cache"`
	JWKS                    JWKS   `yaml:"jwks"`
	RewriteWellKnown        bool   `yaml:"rewrite_well_known" env:"PROXY_OIDC_REWRITE_WELLKNOWN" desc:"Enables rewriting the /.well-known/openid-configuration to the configured OIDC issuer. Needed by the Desktop Client, Android Client and iOS Client to discover the OIDC provider." introductionVersion:"1.0.0"`
}

type JWKS struct {
	RefreshInterval   uint64 `yaml:"refresh_interval" env:"PROXY_OIDC_JWKS_REFRESH_INTERVAL" desc:"The interval for refreshing the JWKS (JSON Web Key Set) in minutes in the background via a new HTTP request to the IDP." introductionVersion:"1.0.0"`
	RefreshTimeout    uint64 `yaml:"refresh_timeout" env:"PROXY_OIDC_JWKS_REFRESH_TIMEOUT" desc:"The timeout in seconds for an outgoing JWKS request." introductionVersion:"1.0.0"`
	RefreshRateLimit  uint64 `yaml:"refresh_limit" env:"PROXY_OIDC_JWKS_REFRESH_RATE_LIMIT" desc:"Limits the rate in seconds at which refresh requests are performed for unknown keys. This is used to prevent malicious clients from imposing high network load on the IDP via OpenCloud." introductionVersion:"1.0.0"`
	RefreshUnknownKID bool   `yaml:"refresh_unknown_kid" env:"PROXY_OIDC_JWKS_REFRESH_UNKNOWN_KID" desc:"If set to 'true', the JWKS refresh request will occur every time an unknown KEY ID (KID) is seen. Always set a 'refresh_limit' when enabling this." introductionVersion:"1.0.0"`
}

// Cache is a TTL cache configuration.
type Cache struct {
	Store              string        `yaml:"store" env:"OC_CACHE_STORE;PROXY_OIDC_USERINFO_CACHE_STORE" desc:"The type of the cache store. Supported values are: 'memory', 'redis-sentinel', 'nats-js-kv', 'noop'. See the text description for details." introductionVersion:"1.0.0"`
	Nodes              []string      `yaml:"addresses" env:"OC_CACHE_STORE_NODES;PROXY_OIDC_USERINFO_CACHE_STORE_NODES" desc:"A list of nodes to access the configured store. This has no effect when 'memory' store is configured. Note that the behaviour how nodes are used is dependent on the library of the configured store. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	Database           string        `yaml:"database" env:"OC_CACHE_DATABASE" desc:"The database name the configured store should use." introductionVersion:"1.0.0"`
	Table              string        `yaml:"table" env:"PROXY_OIDC_USERINFO_CACHE_TABLE" desc:"The database table the store should use." introductionVersion:"1.0.0"`
	TTL                time.Duration `yaml:"ttl" env:"OC_CACHE_TTL;PROXY_OIDC_USERINFO_CACHE_TTL" desc:"Default time to live for user info in the user info cache. Only applied when access tokens has no expiration. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	DisablePersistence bool          `yaml:"disable_persistence" env:"OC_CACHE_DISABLE_PERSISTENCE;PROXY_OIDC_USERINFO_CACHE_DISABLE_PERSISTENCE" desc:"Disables persistence of the cache. Only applies when store type 'nats-js-kv' is configured. Defaults to false." introductionVersion:"1.0.0"`
	AuthUsername       string        `yaml:"username" env:"OC_CACHE_AUTH_USERNAME;PROXY_OIDC_USERINFO_CACHE_AUTH_USERNAME" desc:"The username to authenticate with the cache. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"1.0.0"`
	AuthPassword       string        `yaml:"password" env:"OC_CACHE_AUTH_PASSWORD;PROXY_OIDC_USERINFO_CACHE_AUTH_PASSWORD" desc:"The password to authenticate with the cache. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"1.0.0"`
}

// RoleAssignment contains the configuration for how to assign roles to users during login
type RoleAssignment struct {
	Driver         string         `yaml:"driver" env:"PROXY_ROLE_ASSIGNMENT_DRIVER" desc:"The mechanism that should be used to assign roles to user upon login. Supported values: 'default' or 'oidc'. 'default' will assign the role 'user' to users which don't have a role assigned at the time they login. 'oidc' will assign the role based on the value of a claim (configured via PROXY_ROLE_ASSIGNMENT_OIDC_CLAIM) from the users OIDC claims." introductionVersion:"1.0.0"`
	OIDCRoleMapper OIDCRoleMapper `yaml:"oidc_role_mapper"`
}

// OIDCRoleMapper contains the configuration for the "oidc" role assignment driver
type OIDCRoleMapper struct {
	RoleClaim string        `yaml:"role_claim" env:"PROXY_ROLE_ASSIGNMENT_OIDC_CLAIM" desc:"The OIDC claim used to create the users role assignment." introductionVersion:"1.0.0"`
	RolesMap  []RoleMapping `yaml:"role_mapping" desc:"A list of mappings of OpenCloud role names to PROXY_ROLE_ASSIGNMENT_OIDC_CLAIM claim values. This setting can only be configured in the configuration file and not via environment variables."`
}

// RoleMapping defines which OpenCloud role matches a specific claim value
type RoleMapping struct {
	RoleName   string `yaml:"role_name" desc:"The name of an OpenCloud role that this mapping should apply for."`
	ClaimValue string `yaml:"claim_value" desc:"The value of the 'PROXY_ROLE_ASSIGNMENT_OIDC_CLAIM' that matches the role defined in 'role_name'."`
}

// AutoProvisionClaims defines which claims from the OIDC userinfo response should be used for auto-provisioning user accounts
type AutoProvisionClaims struct {
	Username    string `yaml:"username" env:"PROXY_AUTOPROVISION_CLAIM_USERNAME" desc:"The name of the OIDC claim that holds the username." introductionVersion:"1.0.0"`
	Email       string `yaml:"email" env:"PROXY_AUTOPROVISION_CLAIM_EMAIL" desc:"The name of the OIDC claim that holds the email." introductionVersion:"1.0.0"`
	DisplayName string `yaml:"display_name" env:"PROXY_AUTOPROVISION_CLAIM_DISPLAYNAME" desc:"The name of the OIDC claim that holds the display name." introductionVersion:"1.0.0"`
	Groups      string `yaml:"groups" env:"PROXY_AUTOPROVISION_CLAIM_GROUPS" desc:"The name of the OIDC claim that holds the groups." introductionVersion:"1.0.0"`
}

// PolicySelector is the toplevel-configuration for different selectors
type PolicySelector struct {
	Static *StaticSelectorConf `yaml:"static"`
	Claims *ClaimsSelectorConf `yaml:"claims"`
	Regex  *RegexSelectorConf  `yaml:"regex"`
}

// StaticSelectorConf is the config for the static-policy-selector
type StaticSelectorConf struct {
	Policy string `yaml:"policy"`
}

// PreSignedURL is the config for the pre-signed url middleware
type PreSignedURL struct {
	AllowedHTTPMethods []string     `yaml:"allowed_http_methods"`
	Enabled            bool         `yaml:"enabled" env:"PROXY_ENABLE_PRESIGNEDURLS" desc:"Allow OCS to get a signing key to sign requests." introductionVersion:"1.0.0"`
	SigningKeys        *SigningKeys `yaml:"signing_keys"`
}

// SigningKeys is a store configuration.
type SigningKeys struct {
	Store              string        `yaml:"store" env:"OC_CACHE_STORE;PROXY_PRESIGNEDURL_SIGNING_KEYS_STORE" desc:"The type of the signing key store. Supported values are: 'redis-sentinel', 'nats-js-kv' and 'opencloudstoreservice' (deprecated). See the text description for details." introductionVersion:"1.0.0"`
	Nodes              []string      `yaml:"addresses" env:"OC_CACHE_STORE_NODES;PROXY_PRESIGNEDURL_SIGNING_KEYS_STORE_NODES" desc:"A list of nodes to access the configured store. Note that the behaviour how nodes are used is dependent on the library of the configured store. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	TTL                time.Duration `yaml:"ttl" env:"OC_CACHE_TTL;PROXY_PRESIGNEDURL_SIGNING_KEYS_STORE_TTL" desc:"Default time to live for signing keys. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	DisablePersistence bool          `yaml:"disable_persistence" env:"OC_CACHE_DISABLE_PERSISTENCE;PROXY_PRESIGNEDURL_SIGNING_KEYS_STORE_DISABLE_PERSISTENCE" desc:"Disables persistence of the store. Only applies when store type 'nats-js-kv' is configured. Defaults to true." introductionVersion:"1.0.0"`
	AuthUsername       string        `yaml:"username" env:"OC_CACHE_AUTH_USERNAME;PROXY_PRESIGNEDURL_SIGNING_KEYS_STORE_AUTH_USERNAME" desc:"The username to authenticate with the store. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"1.0.0"`
	AuthPassword       string        `yaml:"password" env:"OC_CACHE_AUTH_PASSWORD;PROXY_PRESIGNEDURL_SIGNING_KEYS_STORE_AUTH_PASSWORD" desc:"The password to authenticate with the store. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"1.0.0"`
}

// ClaimsSelectorConf is the config for the claims-selector
type ClaimsSelectorConf struct {
	DefaultPolicy         string `yaml:"default_policy"`
	UnauthenticatedPolicy string `yaml:"unauthenticated_policy"`
	SelectorCookieName    string `yaml:"selector_cookie_name"`
}

// RegexSelectorConf is the config for the regex-selector
type RegexSelectorConf struct {
	DefaultPolicy         string          `yaml:"default_policy"`
	MatchesPolicies       []RegexRuleConf `yaml:"matches_policies"`
	UnauthenticatedPolicy string          `yaml:"unauthenticated_policy"`
	SelectorCookieName    string          `yaml:"selector_cookie_name"`
}

type RegexRuleConf struct {
	Priority int    `yaml:"priority"`
	Property string `yaml:"property"`
	Match    string `yaml:"match"`
	Policy   string `yaml:"policy"`
}

// ServiceAccount is the configuration for the used service account
type ServiceAccount struct {
	ServiceAccountID     string `yaml:"service_account_id" env:"OC_SERVICE_ACCOUNT_ID;PROXY_SERVICE_ACCOUNT_ID" desc:"The ID of the service account the service should use. See the 'auth-service' service description for more details." introductionVersion:"1.0.0"`
	ServiceAccountSecret string `yaml:"service_account_secret" env:"OC_SERVICE_ACCOUNT_SECRET;PROXY_SERVICE_ACCOUNT_SECRET" desc:"The service account secret." introductionVersion:"1.0.0"`
}

// Events combines the configuration options for the event bus.
type Events struct {
	Endpoint             string `yaml:"endpoint" env:"OC_EVENTS_ENDPOINT;PROXY_EVENTS_ENDPOINT" desc:"The address of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture. Set to a empty string to disable emitting events." introductionVersion:"1.0.0"`
	Cluster              string `yaml:"cluster" env:"OC_EVENTS_CLUSTER;PROXY_EVENTS_CLUSTER" desc:"The clusterID of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture." introductionVersion:"1.0.0"`
	TLSInsecure          bool   `yaml:"tls_insecure" env:"OC_INSECURE;OC_EVENTS_TLS_INSECURE;PROXY_EVENTS_TLS_INSECURE" desc:"Whether to verify the server TLS certificates." introductionVersion:"1.0.0"`
	TLSRootCACertificate string `yaml:"tls_root_ca_certificate" env:"OC_EVENTS_TLS_ROOT_CA_CERTIFICATE;PROXY_EVENTS_TLS_ROOT_CA_CERTIFICATE" desc:"The root CA certificate used to validate the server's TLS certificate. If provided PROXY_EVENTS_TLS_INSECURE will be seen as false." introductionVersion:"1.0.0"`
	EnableTLS            bool   `yaml:"enable_tls" env:"OC_EVENTS_ENABLE_TLS;PROXY_EVENTS_ENABLE_TLS" desc:"Enable TLS for the connection to the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
	AuthUsername         string `yaml:"username" env:"OC_EVENTS_AUTH_USERNAME;PROXY_EVENTS_AUTH_USERNAME" desc:"The username to authenticate with the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
	AuthPassword         string `yaml:"password" env:"OC_EVENTS_AUTH_PASSWORD;PROXY_EVENTS_AUTH_PASSWORD" desc:"The password to authenticate with the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
}
