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

	Log   *Log  `yaml:"log"`
	Debug Debug `yaml:"debug"`

	HTTP          HTTP                  `yaml:"http"`
	GRPCClientTLS *shared.GRPCClientTLS `yaml:"grpc_client_tls"`

	TokenManager *TokenManager `yaml:"token_manager"`

	RevaGateway     string      `yaml:"reva_gateway" env:"OC_REVA_GATEWAY" desc:"CS3 gateway used to look up user metadata" introductionVersion:"1.0.0"`
	TranslationPath string      `yaml:"translation_path" env:"OC_TRANSLATION_PATH;USERLOG_TRANSLATION_PATH" desc:"(optional) Set this to a path with custom translations to overwrite the builtin translations. Note that file and folder naming rules apply, see the documentation for more details." introductionVersion:"1.0.0"`
	DefaultLanguage string      `yaml:"default_language" env:"OC_DEFAULT_LANGUAGE" desc:"The default language used by services and the WebUI. If not defined, English will be used as default. See the documentation for more details." introductionVersion:"1.0.0"`
	Events          Events      `yaml:"events"`
	MaxConcurrency  int         `yaml:"max_concurrency" env:"OC_MAX_CONCURRENCY;USERLOG_MAX_CONCURRENCY" desc:"Maximum number of concurrent go-routines. Higher values can potentially get work done faster but will also cause more load on the system. Values of 0 or below will be ignored and the default value will be used." introductionVersion:"1.0.0"`
	Persistence     Persistence `yaml:"persistence"`

	DisableSSE bool `yaml:"disable_sse" env:"OC_DISABLE_SSE,USERLOG_DISABLE_SSE" desc:"Disables server-sent events (sse). When disabled, clients will no longer receive sse notifications." introductionVersion:"1.0.0"`

	GlobalNotificationsSecret string `yaml:"global_notifications_secret" env:"USERLOG_GLOBAL_NOTIFICATIONS_SECRET" desc:"The secret to secure the global notifications endpoint. Only system admins and users knowing that secret can call the global notifications POST/DELETE endpoints." introductionVersion:"1.0.0"`

	ServiceAccount ServiceAccount `yaml:"service_account"`

	Context context.Context `yaml:"-"`
}

// Persistence configures the store to use
type Persistence struct {
	Store        string        `yaml:"store" env:"OC_PERSISTENT_STORE;USERLOG_STORE" desc:"The type of the store. Supported values are: 'memory', 'nats-js-kv', 'redis-sentinel', 'noop'. See the text description for details." introductionVersion:"1.0.0"`
	Nodes        []string      `yaml:"nodes" env:"OC_PERSISTENT_STORE_NODES;USERLOG_STORE_NODES" desc:"A list of nodes to access the configured store. This has no effect when 'memory' store is configured. Note that the behaviour how nodes are used is dependent on the library of the configured store. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	Database     string        `yaml:"database" env:"USERLOG_STORE_DATABASE" desc:"The database name the configured store should use." introductionVersion:"1.0.0"`
	Table        string        `yaml:"table" env:"USERLOG_STORE_TABLE" desc:"The database table the store should use." introductionVersion:"1.0.0"`
	TTL          time.Duration `yaml:"ttl" env:"OC_PERSISTENT_STORE_TTL;USERLOG_STORE_TTL" desc:"Time to live for events in the store. Defaults to '336h' (2 weeks). See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	AuthUsername string        `yaml:"username" env:"OC_PERSISTENT_STORE_AUTH_USERNAME;USERLOG_STORE_AUTH_USERNAME" desc:"The username to authenticate with the store. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"1.0.0"`
	AuthPassword string        `yaml:"password" env:"OC_PERSISTENT_STORE_AUTH_PASSWORD;USERLOG_STORE_AUTH_PASSWORD" desc:"The password to authenticate with the store. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"1.0.0"`
}

// Events combines the configuration options for the event bus.
type Events struct {
	Endpoint             string `yaml:"endpoint" env:"OC_EVENTS_ENDPOINT;USERLOG_EVENTS_ENDPOINT" desc:"The address of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture." introductionVersion:"1.0.0"`
	Cluster              string `yaml:"cluster" env:"OC_EVENTS_CLUSTER;USERLOG_EVENTS_CLUSTER" desc:"The clusterID of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture. Mandatory when using NATS as event system." introductionVersion:"1.0.0"`
	TLSInsecure          bool   `yaml:"tls_insecure" env:"OC_INSECURE;OC_EVENTS_TLS_INSECURE;USERLOG_EVENTS_TLS_INSECURE" desc:"Whether to verify the server TLS certificates." introductionVersion:"1.0.0"`
	TLSRootCACertificate string `yaml:"tls_root_ca_certificate" env:"OC_EVENTS_TLS_ROOT_CA_CERTIFICATE;USERLOG_EVENTS_TLS_ROOT_CA_CERTIFICATE" desc:"The root CA certificate used to validate the server's TLS certificate. If provided NOTIFICATIONS_EVENTS_TLS_INSECURE will be seen as false." introductionVersion:"1.0.0"`
	EnableTLS            bool   `yaml:"enable_tls" env:"OC_EVENTS_ENABLE_TLS;USERLOG_EVENTS_ENABLE_TLS" desc:"Enable TLS for the connection to the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
	AuthUsername         string `yaml:"username" env:"OC_EVENTS_AUTH_USERNAME;USERLOG_EVENTS_AUTH_USERNAME" desc:"The username to authenticate with the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
	AuthPassword         string `yaml:"password" env:"OC_EVENTS_AUTH_PASSWORD;USERLOG_EVENTS_AUTH_PASSWORD" desc:"The password to authenticate with the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
}

// CORS defines the available cors configuration.
type CORS struct {
	AllowedOrigins   []string `yaml:"allow_origins" env:"OC_CORS_ALLOW_ORIGINS;USERLOG_CORS_ALLOW_ORIGINS" desc:"A list of allowed CORS origins. See following chapter for more details: *Access-Control-Allow-Origin* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Origin. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	AllowedMethods   []string `yaml:"allow_methods" env:"OC_CORS_ALLOW_METHODS;USERLOG_CORS_ALLOW_METHODS" desc:"A list of allowed CORS methods. See following chapter for more details: *Access-Control-Request-Method* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Request-Method. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	AllowedHeaders   []string `yaml:"allow_headers" env:"OC_CORS_ALLOW_HEADERS;USERLOG_CORS_ALLOW_HEADERS" desc:"A list of allowed CORS headers. See following chapter for more details: *Access-Control-Request-Headers* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Request-Headers. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	AllowCredentials bool     `yaml:"allow_credentials" env:"OC_CORS_ALLOW_CREDENTIALS;USERLOG_CORS_ALLOW_CREDENTIALS" desc:"Allow credentials for CORS.See following chapter for more details: *Access-Control-Allow-Credentials* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Credentials." introductionVersion:"1.0.0"`
}

// HTTP defines the available http configuration.
type HTTP struct {
	Addr      string                `yaml:"addr" env:"USERLOG_HTTP_ADDR" desc:"The bind address of the HTTP service." introductionVersion:"1.0.0"`
	Namespace string                `yaml:"-"`
	Root      string                `yaml:"root" env:"USERLOG_HTTP_ROOT" desc:"Subdirectory that serves as the root for this HTTP service." introductionVersion:"1.0.0"`
	CORS      CORS                  `yaml:"cors"`
	TLS       shared.HTTPServiceTLS `yaml:"tls"`
}

// TokenManager is the config for using the reva token manager
type TokenManager struct {
	JWTSecret string `yaml:"jwt_secret" env:"OC_JWT_SECRET;USERLOG_JWT_SECRET" desc:"The secret to mint and validate jwt tokens." introductionVersion:"1.0.0"`
}

// ServiceAccount is the configuration for the used service account
type ServiceAccount struct {
	ServiceAccountID     string `yaml:"service_account_id" env:"OC_SERVICE_ACCOUNT_ID;USERLOG_SERVICE_ACCOUNT_ID" desc:"The ID of the service account the service should use. See the 'auth-service' service description for more details." introductionVersion:"1.0.0"`
	ServiceAccountSecret string `yaml:"service_account_secret" env:"OC_SERVICE_ACCOUNT_SECRET;USERLOG_SERVICE_ACCOUNT_SECRET" desc:"The service account secret." introductionVersion:"1.0.0"`
}
