package config

import (
	"context"
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// Config combines all available configuration parts.
type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service

	Service Service `yaml:"-"`

	Tracing *Tracing `yaml:"tracing"`
	Log     *Log     `yaml:"log"`
	Debug   Debug    `yaml:"debug"`

	HTTP HTTP `yaml:"http"`

	Reva *shared.Reva `yaml:"reva"`

	MachineAuthAPIKey string `yaml:"machine_auth_api_key" env:"OC_MACHINE_AUTH_API_KEY;IDP_MACHINE_AUTH_API_KEY" desc:"Machine auth API key used to validate internal requests necessary for the access to resources from other services." introductionVersion:"1.0.0"`

	Asset   Asset    `yaml:"asset"`
	IDP     Settings `yaml:"idp"`
	Clients []Client `yaml:"clients"`
	Ldap    Ldap     `yaml:"ldap"`

	Context context.Context `yaml:"-"`
}

// Ldap defines the available LDAP configuration.
type Ldap struct {
	URI       string `yaml:"uri" env:"OC_LDAP_URI;IDP_LDAP_URI" desc:"Url of the LDAP service to use as IDP." introductionVersion:"1.0.0"`
	TLSCACert string `yaml:"cacert" env:"OC_LDAP_CACERT;IDP_LDAP_TLS_CACERT" desc:"Path/File name for the root CA certificate (in PEM format) used to validate TLS server certificates of the LDAP service. If not defined, the root directory derives from $OC_BASE_DATA_PATH/idp." introductionVersion:"1.0.0"`

	BindDN       string `yaml:"bind_dn" env:"OC_LDAP_BIND_DN;IDP_LDAP_BIND_DN" desc:"LDAP DN to use for simple bind authentication with the target LDAP server." introductionVersion:"1.0.0"`
	BindPassword string `yaml:"bind_password" env:"OC_LDAP_BIND_PASSWORD;IDP_LDAP_BIND_PASSWORD" desc:"Password to use for authenticating the 'bind_dn'." introductionVersion:"1.0.0"`

	BaseDN string `yaml:"base_dn" env:"OC_LDAP_USER_BASE_DN;IDP_LDAP_BASE_DN" desc:"Search base DN for looking up LDAP users." introductionVersion:"1.0.0"`
	Scope  string `yaml:"scope" env:"OC_LDAP_USER_SCOPE;IDP_LDAP_SCOPE" desc:"LDAP search scope to use when looking up users. Supported scopes are 'base', 'one' and 'sub'." introductionVersion:"1.0.0"`

	LoginAttribute    string `yaml:"login_attribute" env:"IDP_LDAP_LOGIN_ATTRIBUTE" desc:"LDAP User attribute to use for login like 'uid'." introductionVersion:"1.0.0"`
	EmailAttribute    string `yaml:"email_attribute" env:"OC_LDAP_USER_SCHEMA_MAIL;IDP_LDAP_EMAIL_ATTRIBUTE" desc:"LDAP User email attribute like 'mail'." introductionVersion:"1.0.0"`
	NameAttribute     string `yaml:"name_attribute" env:"OC_LDAP_USER_SCHEMA_USERNAME;IDP_LDAP_NAME_ATTRIBUTE" desc:"LDAP User name attribute like 'displayName'." introductionVersion:"1.0.0"`
	UUIDAttribute     string `yaml:"uuid_attribute" env:"OC_LDAP_USER_SCHEMA_ID;IDP_LDAP_UUID_ATTRIBUTE" desc:"LDAP User UUID attribute like 'uid'." introductionVersion:"1.0.0"`
	UUIDAttributeType string `yaml:"uuid_attribute_type" env:"IDP_LDAP_UUID_ATTRIBUTE_TYPE" desc:"LDAP User uuid attribute type like 'text'." introductionVersion:"1.0.0"`

	UserEnabledAttribute string `yaml:"user_enabled_attribute" env:"OC_LDAP_USER_ENABLED_ATTRIBUTE;IDP_USER_ENABLED_ATTRIBUTE" desc:"LDAP Attribute to use as a flag telling if the user is enabled or disabled." introductionVersion:"1.0.0"`
	Filter               string `yaml:"filter" env:"OC_LDAP_USER_FILTER;IDP_LDAP_FILTER" desc:"LDAP filter to add to the default filters for user search like '(objectclass=openCloudUser)'." introductionVersion:"1.0.0"`
	ObjectClass          string `yaml:"objectclass" env:"OC_LDAP_USER_OBJECTCLASS;IDP_LDAP_OBJECTCLASS" desc:"LDAP User ObjectClass like 'inetOrgPerson'." introductionVersion:"1.0.0"`
}

// Asset defines the available asset configuration.
type Asset struct {
	Path               string `yaml:"asset" env:"IDP_ASSET_PATH" desc:"Serve IDP assets from a path on the filesystem instead of the builtin assets." introductionVersion:"1.0.0"`
	LoginBackgroundUrl string `yaml:"login-background-url" env:"IDP_LOGIN_BACKGROUND_URL" desc:"Configure an alternative URL to the background image for the login page." introductionVersion:"1.0.0"`
}

type Client struct {
	ID                     string   `yaml:"id"`
	Name                   string   `yaml:"name"`
	Trusted                bool     `yaml:"trusted"`
	Secret                 string   `yaml:"secret"`
	RedirectURIs           []string `yaml:"redirect_uris"`
	PostLogoutRedirectURIs []string `yaml:"post_logout_redirect_uris"`
	Origins                []string `yaml:"origins"`
	ApplicationType        string   `yaml:"application_type"`
}

type Settings struct {
	Iss string `yaml:"iss" env:"OC_URL;OC_OIDC_ISSUER;IDP_ISS" desc:"The OIDC issuer URL to use." introductionVersion:"1.0.0"`

	IdentityManager string `yaml:"identity_manager" env:"IDP_IDENTITY_MANAGER" desc:"The identity manager implementation to use. Supported identity managers are 'ldap', 'cs3', 'libregraph' and 'guest'." introductionVersion:"1.0.0"`

	URIBasePath string `yaml:"uri_base_path" env:"IDP_URI_BASE_PATH" desc:"IDP uri base path (defaults to '')." introductionVersion:"1.0.0"`

	SignInURI    string `yaml:"sign_in_uri" env:"IDP_SIGN_IN_URI" desc:"IDP sign-in url." introductionVersion:"1.0.0"`
	SignedOutURI string `yaml:"signed_out_uri" env:"IDP_SIGN_OUT_URI" desc:"IDP sign-out url." introductionVersion:"1.0.0"`

	AuthorizationEndpointURI string `yaml:"authorization_endpoint_uri" env:"IDP_ENDPOINT_URI" desc:"URL of the IDP endpoint." introductionVersion:"1.0.0"`
	EndsessionEndpointURI    string `yaml:"-"` // unused, not supported by lico-idp

	Insecure bool `yaml:"ldap_insecure" env:"OC_LDAP_INSECURE;IDP_INSECURE" desc:"Disable TLS certificate validation for the LDAP connections. Do not set this in production environments." introductionVersion:"1.0.0"`

	TrustedProxy []string `yaml:"trusted_proxy"` //TODO: how to configure this via env?

	AllowScope                     []string `yaml:"allow_scope"` // TODO: is this even needed?
	AllowClientGuests              bool     `yaml:"allow_client_guests" env:"IDP_ALLOW_CLIENT_GUESTS" desc:"Allow guest clients to access OpenCloud." introductionVersion:"1.0.0"`
	AllowDynamicClientRegistration bool     `yaml:"allow_dynamic_client_registration" env:"IDP_ALLOW_DYNAMIC_CLIENT_REGISTRATION" desc:"Allow dynamic client registration." introductionVersion:"1.0.0"`

	EncryptionSecretFile string `yaml:"encrypt_secret_file" env:"IDP_ENCRYPTION_SECRET_FILE" desc:"Path to the encryption secret file, if unset, a new certificate will be autogenerated upon each restart, thus invalidating all existing sessions. If not defined, the root directory derives from $OC_BASE_DATA_PATH/idp." introductionVersion:"1.0.0"`

	Listen string

	IdentifierClientDisabled          bool   `yaml:"-"` // unused
	IdentifierClientPath              string `yaml:"-"`
	IdentifierRegistrationConf        string `yaml:"-"`
	IdentifierScopesConf              string `yaml:"-"` // unused
	IdentifierDefaultBannerLogo       string
	IdentifierDefaultSignInPageText   string `yaml:"default_sign_in_page_text" env:"IDP_DEFAULT_SIGNIN_PAGE_TEXT" desc:"" introductionVersion:"2.0.0"`
	IdentifierDefaultLogoTargetURI    string `yaml:"default_logo_target_uri" env:"IDP_DEFAULT_LOGO_TARGET_URI" desc:"Default logo target URI." introductionVersion:"4.0.0"`
	IdentifierDefaultUsernameHintText string
	IdentifierUILocales               []string

	SigningKid             string   `yaml:"signing_kid" env:"IDP_SIGNING_KID" desc:"Value of the KID (Key ID) field which is used in created tokens to uniquely identify the signing-private-key." introductionVersion:"1.0.0"`
	SigningMethod          string   `yaml:"signing_method" env:"IDP_SIGNING_METHOD" desc:"Signing method of IDP requests like 'PS256'" introductionVersion:"1.0.0"`
	SigningPrivateKeyFiles []string `yaml:"signing_private_key_files" env:"IDP_SIGNING_PRIVATE_KEY_FILES" desc:"A list of private key files for signing IDP requests. If not defined, the root directory derives from $OC_BASE_DATA_PATH/idp. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	ValidationKeysPath     string   `yaml:"validation_keys_path" env:"IDP_VALIDATION_KEYS_PATH" desc:"Path to validation keys for IDP requests." introductionVersion:"1.0.0"`

	CookieBackendURI string
	CookieNames      []string
	CookieSameSite   http.SameSite

	AccessTokenDurationSeconds         uint64 `yaml:"access_token_duration_seconds" env:"IDP_ACCESS_TOKEN_EXPIRATION" desc:"'Access token lifespan in seconds (time before an access token is expired).'" introductionVersion:"1.0.0"`
	IDTokenDurationSeconds             uint64 `yaml:"id_token_duration_seconds" env:"IDP_ID_TOKEN_EXPIRATION" desc:"ID token lifespan in seconds (time before an ID token is expired)." introductionVersion:"1.0.0"`
	RefreshTokenDurationSeconds        uint64 `yaml:"refresh_token_duration_seconds" env:"IDP_REFRESH_TOKEN_EXPIRATION" desc:"Refresh token lifespan in seconds (time before an refresh token is expired). This also limits the duration of an idle offline session." introductionVersion:"1.0.0"`
	DynamicClientSecretDurationSeconds uint64 `yaml:"dynamic_client_secret_duration_seconds" env:"IDP_DYNAMIC_CLIENT_SECRET_DURATION" desc:"Lifespan in seconds of a dynamically registered OIDC client." introductionVersion:"1.0.0"`
}
