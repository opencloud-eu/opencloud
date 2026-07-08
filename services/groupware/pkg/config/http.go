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
	Addr                     string                `yaml:"addr" env:"GROUPWARE_HTTP_ADDR" desc:"The bind address of the HTTP service." introductionVersion:"1.0.0"`
	TLS                      shared.HTTPServiceTLS `yaml:"tls"`
	Root                     string                `yaml:"root" env:"GROUPWARE_HTTP_ROOT" desc:"Subdirectory that serves as the root for this HTTP service." introductionVersion:"1.0.0"`
	Namespace                string                `yaml:"-"`
	CORS                     CORS                  `yaml:"cors"`
	OpenCloudPublicURL       string                `yaml:"opencloud_public_url" env:"OC_URL;OC_PUBLIC_URL;GROUPWARE_PUBLIC_URL"`
	SendDurationsResponse    bool                  `yaml:"send_durations_response" env:"GROUPWARE_SEND_DURATIONS_RESPONSE"`
	Insecure                 bool                  `yaml:"tls_insecure" env:"OC_INSECURE;GROUPWARE_TLS_INSECURE" desc:"Whether to verify the server TLS certificates." introductionVersion:"1.0.0"`
	TraceRequests            bool                  `yaml:"trace_requests" env:"GROUPWARE_HTTP_TRACE_REQUESTS"`
	TraceMaxRequestBodySize  int64                 `yaml:"trace_max_request_body_size" env:"GROUPWARE_HTTP_TRACE_MAX_REQUEST_BODY_SIZE"`
	TraceResponses           bool                  `yaml:"trace_responses" env:"GROUPWARE_HTTP_TRACE_RESPONSES"`
	TraceMaxResponseBodySize int64                 `yaml:"trace_max_response_body_size" env:"GROUPWARE_HTTP_TRACE_MAX_RESPONSE_BODY_SIZE"`
}
