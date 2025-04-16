package config

import "github.com/opencloud-eu/opencloud/pkg/shared"

// CORS defines the available cors configuration.
type CORS struct {
	AllowedOrigins   []string `yaml:"allow_origins" env:"OC_CORS_ALLOW_ORIGINS;GROUPWARE_CORS_ALLOW_ORIGINS" desc:"A list of allowed CORS origins. See following chapter for more details: *Access-Control-Allow-Origin* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Origin. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	AllowedMethods   []string `yaml:"allow_methods" env:"OC_CORS_ALLOW_METHODS;GROUPWARE_CORS_ALLOW_METHODS" desc:"A list of allowed CORS methods. See following chapter for more details: *Access-Control-Request-Method* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Request-Method. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	AllowedHeaders   []string `yaml:"allow_headers" env:"OC_CORS_ALLOW_HEADERS;GROUPWARE_CORS_ALLOW_HEADERS" desc:"A list of allowed CORS headers. See following chapter for more details: *Access-Control-Request-Headers* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Request-Headers. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	AllowCredentials bool     `yaml:"allow_credentials" env:"OC_CORS_ALLOW_CREDENTIALS;GROUPWARE_CORS_ALLOW_CREDENTIALS" desc:"Allow credentials for CORS.See following chapter for more details: *Access-Control-Allow-Credentials* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Credentials." introductionVersion:"1.0.0"`
}

// HTTP defines the available http configuration.
type HTTP struct {
	Addr      string                `yaml:"addr" env:"GROUPWARE_HTTP_ADDR" desc:"The bind address of the HTTP service." introductionVersion:"1.0.0"`
	TLS       shared.HTTPServiceTLS `yaml:"tls"`
	Root      string                `yaml:"root" env:"GROUPWARE_HTTP_ROOT" desc:"Subdirectory that serves as the root for this HTTP service." introductionVersion:"1.0.0"`
	Namespace string                `yaml:"-"`
	CORS      CORS                  `yaml:"cors"`
}
