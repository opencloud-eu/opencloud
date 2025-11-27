package config

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// Config combines all available configuration parts.
type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service

	Service Service `yaml:"-"`

	Log   *Log  `yaml:"log"`
	Debug Debug `yaml:"debug"`

	HTTP HTTP `yaml:"http"`

	Instances    []Instance `yaml:"instances"`
	Relations    []string   `yaml:"relations" env:"WEBFINGER_RELATIONS" desc:"A list of relation URIs or registered relation types to add to webfinger responses. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	IDP          string     `yaml:"idp" env:"OC_URL;OC_OIDC_ISSUER;WEBFINGER_OIDC_ISSUER" desc:"The identity provider href for the openid-discovery relation." introductionVersion:"1.0.0"`
	OpenCloudURL string     `yaml:"opencloud_url" env:"OC_URL;WEBFINGER_OPENCLOUD_SERVER_INSTANCE_URL" desc:"The URL for the legacy OpenCloud server instance relation (not to be confused with the product OpenCloud Server). It defaults to the OC_URL but can be overridden to support some reverse proxy corner cases. To shard the deployment, multiple instances can be configured in the configuration file." introductionVersion:"1.0.0"`
	Insecure     bool       `yaml:"insecure" env:"OC_INSECURE;WEBFINGER_INSECURE" desc:"Allow insecure connections to the WEBFINGER service." introductionVersion:"1.0.0"`

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
