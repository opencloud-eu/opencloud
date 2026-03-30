package config

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// Config combines all available configuration parts.
type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service

	Service Service `yaml:"-"`

	LogLevel string `yaml:"loglevel" env:"OC_LOG_LEVEL;WEBFINGER_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"1.0.0"`
	Debug    Debug  `yaml:"debug"`

	HTTP HTTP `yaml:"http"`

	Instances           []Instance `yaml:"instances"`
	Relations           []string   `yaml:"relations" env:"WEBFINGER_RELATIONS" desc:"A list of relation URIs or registered relation types to add to webfinger responses. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	IDP                 string     `yaml:"idp" env:"OC_URL;OC_OIDC_ISSUER;WEBFINGER_OIDC_ISSUER" desc:"The identity provider href for the openid-discovery relation." introductionVersion:"1.0.0"`
	AndroidClientID     string     `yaml:"android_client_id" env:"OC_OIDC_CLIENT_ID;WEBFINGER_ANDROID_OIDC_CLIENT_ID" desc:"The OIDC client ID for Android app." introductionVersion:"6.0.0"`
	AndroidClientScopes []string   `yaml:"android_client_scopes" env:"OC_OIDC_CLIENT_SCOPES;WEBFINGER_ANDROID_OIDC_CLIENT_SCOPES" desc:"The OIDC client scopes the Android app should request." introductionVersion:"6.0.0"`
	DesktopClientID     string     `yaml:"desktop_client_id" env:"OC_OIDC_CLIENT_ID;WEBFINGER_DESKTOP_OIDC_CLIENT_ID" desc:"The OIDC client ID for the OpenCloud desktop application." introductionVersion:"6.0.0"`
	DesktopClientScopes []string   `yaml:"desktop_client_scopes" env:"OC_OIDC_CLIENT_SCOPES;WEBFINGER_DESKTOP_OIDC_CLIENT_SCOPES" desc:"The OIDC client scopes the OpenCloud desktop application should request." introductionVersion:"6.0.0"`
	IOSClientID         string     `yaml:"ios_client_id" env:"OC_OIDC_CLIENT_ID;WEBFINGER_IOS_OIDC_CLIENT_ID" desc:"The OIDC client ID for the IOS app." introductionVersion:"6.0.0"`
	IOSClientScopes     []string   `yaml:"ios_client_scopes" env:"OC_OIDC_CLIENT_SCOPES;WEBFINGER_IOS_OIDC_CLIENT_SCOPES" desc:"The OIDC client scopes the IOS app should request." introductionVersion:"6.0.0"`
	// The WEB_OIDC_CLIENT_ID is kept for backware compatibility with the old settings from the `web` service and can be removed in a future release.
	WebClientID string `yaml:"web_client_id" env:"OC_OIDC_CLIENT_ID;WEB_OIDC_CLIENT_ID;WEBFINGER_WEB_OIDC_CLIENT_ID" desc:"The OIDC client ID for the OpenCloud web client. The 'WEB_OIDC_CLIENT_ID' setting is only here for backwards compatibility and will be remove in a future release." introductionVersion:"6.0.0"`
	// The WEB_OIDC_SCOPE is kept for backware compatibility with the old settings from the `web` service and can be removed in a future release.
	WebClientScopes []string `yaml:"web_client_scopes" env:"OC_OIDC_CLIENT_SCOPES;WEB_OIDC_SCOPE;WEBFINGER_WEB_OIDC_CLIENT_SCOPES" desc:"The OIDC client scopes the OpenCloud web client should request. The 'WEB_OIDC_SCOPE' setting is only here for backwards compatibility and will be remove in a future release." introductionVersion:"6.0.0"`
	OpenCloudURL    string   `yaml:"opencloud_url" env:"OC_URL;WEBFINGER_OPENCLOUD_SERVER_INSTANCE_URL" desc:"The URL for the legacy OpenCloud server instance relation (not to be confused with the product OpenCloud Server). It defaults to the OC_URL but can be overridden to support some reverse proxy corner cases. To shard the deployment, multiple instances can be configured in the configuration file." introductionVersion:"1.0.0"`
	Insecure        bool     `yaml:"insecure" env:"OC_INSECURE;WEBFINGER_INSECURE" desc:"Allow insecure connections to the WEBFINGER service." introductionVersion:"1.0.0"`

	OIDCClientConfigs map[string]OIDCClientConfig `yaml:"-"`

	Context context.Context `yaml:"-"`
}

// Instance to use with a matching rule and titles
type Instance struct {
	Claim  string            `yaml:"claim"`
	Regex  string            `yaml:"regex"`
	Href   string            `yaml:"href"`
	Titles map[string]string `yaml:"titles"`
	Break  bool              `yaml:"break"`
}

type OIDCClientConfig struct {
	ClientID string
	Scopes   []string
}
