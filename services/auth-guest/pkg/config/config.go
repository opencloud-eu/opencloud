// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// Config combines all available configuration parts.
type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service

	Service Service `yaml:"-"`

	LogLevel string `yaml:"loglevel" env:"OC_LOG_LEVEL;AUTH_GUEST_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"%%NEXT%%"`

	Debug Debug `yaml:"debug"`

	Events Events `yaml:"events"`

	RevaGateway   string                `yaml:"reva_gateway" env:"OC_REVA_GATEWAY" desc:"CS3 gateway used to look up user metadata" introductionVersion:"%%NEXT%%"`
	GRPCClientTLS *shared.GRPCClientTLS `yaml:"grpc_client_tls"`

	GRPC         GRPCConfig    `yaml:"grpc"`
	HTTP         HTTP          `yaml:"http"`
	Storage      Storage       `yaml:"storage"`
	TokenManager *TokenManager `yaml:"token_manager"`
	JWT          JWT           `yaml:"jwt"`

	ServiceAccount ServiceAccount `yaml:"service_account"`

	NumConsumers int `yaml:"num_consumers" env:"AUTH_GUEST_NUM_CONSUMERS" desc:"The amount of concurrent event consumers to start. Event consumers are used for processing events. Multiple consumers increase parallelisation, but will also increase CPU and memory demands." introductionVersion:"%%NEXT%%"`

	Context context.Context `yaml:"-"`
}

// Events combines the configuration options for the event bus.
type Events struct {
	Disabled             bool   `yaml:"disabled" env:"AUTH_GUEST_EVENTS_DISABLED" desc:"Disables listening for events. Set this to true if the service should only handle HTTP requests." introductionVersion:"%%NEXT%%"`
	Endpoint             string `yaml:"endpoint" env:"OC_EVENTS_ENDPOINT" desc:"The address of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture." introductionVersion:"%%NEXT%%"`
	Cluster              string `yaml:"cluster" env:"OC_EVENTS_CLUSTER" desc:"The clusterID of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture. Mandatory when using NATS as event system." introductionVersion:"%%NEXT%%"`
	TLSInsecure          bool   `yaml:"tls_insecure" env:"OC_INSECURE;OC_EVENTS_TLS_INSECURE" desc:"Whether to verify the server TLS certificates." introductionVersion:"%%NEXT%%"`
	TLSRootCACertificate string `yaml:"tls_root_ca_certificate" env:"OC_EVENTS_TLS_ROOT_CA_CERTIFICATE" desc:"The root CA certificate used to validate the server's TLS certificate. If provided AUTH_GUEST_EVENTS_TLS_INSECURE will be seen as false." introductionVersion:"%%NEXT%%"`
	EnableTLS            bool   `yaml:"enable_tls" env:"OC_EVENTS_ENABLE_TLS" desc:"Enable TLS for the connection to the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"%%NEXT%%"`
	AuthUsername         string `yaml:"username" env:"OC_EVENTS_AUTH_USERNAME" desc:"The username to authenticate with the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"%%NEXT%%"`
	AuthPassword         string `yaml:"password" env:"OC_EVENTS_AUTH_PASSWORD" desc:"The password to authenticate with the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"%%NEXT%%"`
}

// ServiceAccount is the configuration for the used service account
type ServiceAccount struct {
	ServiceAccountID     string `yaml:"service_account_id" env:"OC_SERVICE_ACCOUNT_ID;AUTH_GUEST_SERVICE_ACCOUNT_ID" desc:"The ID of the service account the service should use. See the 'auth-service' service description for more details." introductionVersion:"%%NEXT%%"`
	ServiceAccountSecret string `yaml:"service_account_secret" env:"OC_SERVICE_ACCOUNT_SECRET;AUTH_GUEST_SERVICE_ACCOUNT_SECRET" desc:"The service account secret." introductionVersion:"%%NEXT%%"`
}

// CORS defines the available cors configuration.
type CORS struct {
	AllowedOrigins   []string `yaml:"allow_origins" env:"OC_CORS_ALLOW_ORIGINS;AUTH_GUEST_CORS_ALLOW_ORIGINS" desc:"A list of allowed CORS origins. See following chapter for more details: *Access-Control-Allow-Origin* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Origin. See the Environment Variable Types description for more details." introductionVersion:"%%NEXT%%"`
	AllowedMethods   []string `yaml:"allow_methods" env:"OC_CORS_ALLOW_METHODS;AUTH_GUEST_CORS_ALLOW_METHODS" desc:"A list of allowed CORS methods. See following chapter for more details: *Access-Control-Request-Method* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Request-Method. See the Environment Variable Types description for more details." introductionVersion:"%%NEXT%%"`
	AllowedHeaders   []string `yaml:"allow_headers" env:"OC_CORS_ALLOW_HEADERS;AUTH_GUEST_CORS_ALLOW_HEADERS" desc:"A list of allowed CORS headers. See following chapter for more details: *Access-Control-Request-Headers* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Request-Headers. See the Environment Variable Types description for more details." introductionVersion:"%%NEXT%%"`
	AllowCredentials bool     `yaml:"allow_credentials" env:"OC_CORS_ALLOW_CREDENTIALS;AUTH_GUEST_CORS_ALLOW_CREDENTIALS" desc:"Allow credentials for CORS.See following chapter for more details: *Access-Control-Allow-Credentials* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Credentials." introductionVersion:"%%NEXT%%"`
}

// HTTP defines the available http configuration.
type HTTP struct {
	Disabled  bool                  `yaml:"disabled" env:"AUTH_GUEST_HTTP_DISABLED" desc:"Disables the HTTP service. Set this to true if the service should only handle events." introductionVersion:"%%NEXT%%"`
	Addr      string                `yaml:"addr" env:"AUTH_GUEST_HTTP_ADDR" desc:"The bind address of the HTTP service." introductionVersion:"%%NEXT%%"`
	Namespace string                `yaml:"-"`
	Root      string                `yaml:"root" env:"AUTH_GUEST_HTTP_ROOT" desc:"Subdirectory that serves as the root for this HTTP service." introductionVersion:"%%NEXT%%"`
	CORS      CORS                  `yaml:"cors"`
	TLS       shared.HTTPServiceTLS `yaml:"tls"`
}

// GRPCConfig defines the GRPC configuration
type GRPCConfig struct {
	Addr      string                 `yaml:"addr" env:"GUESTAUTH_GRPC_ADDR" desc:"The bind address of the GRPC service." introductionVersion:"%%NEXT%%"`
	TLS       *shared.GRPCServiceTLS `yaml:"tls"`
	Namespace string                 `yaml:"-"`
	Protocol  string                 `yaml:"protocol" env:"OC_GRPC_PROTOCOL;GUESTAUTH_GRPC_PROTOCOL" desc:"The transport protocol of the GRPC service." introductionVersion:"%%NEXT%%"`
}

// Storage defines the configuration for the token storage.
type Storage struct {
	RootDirectory string `yaml:"root_directory" env:"AUTH_GUEST_TOKENS_STORAGE_ROOT" desc:"The directory where the guest share tokens are stored. If not defined, the root directory derives from $OC_BASE_DATA_PATH/auth-guest." introductionVersion:"%%NEXT%%"`
}

// TokenManager is the config for using the reva token manager
type TokenManager struct {
	JWTSecret string `yaml:"jwt_secret" env:"AUTH_GUEST_JWT_SECRET" desc:"The secret to mint and validate jwt tokens." introductionVersion:"%%NEXT%%"`
}

// JWT defines the configuration for guest session tokens.
type JWT struct {
	CookieName string        `yaml:"cookie_name" env:"AUTH_GUEST_JWT_COOKIE_NAME" desc:"The name of the session cookie set when a guest token is redeemed." introductionVersion:"%%NEXT%%"`
	TTL        time.Duration `yaml:"ttl" env:"AUTH_GUEST_JWT_TTL" desc:"The lifetime of a redeemed guest session token." introductionVersion:"%%NEXT%%"`
}
