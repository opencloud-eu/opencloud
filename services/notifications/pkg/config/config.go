// Package config provides the service configuration.
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

	WebUIURL string `yaml:"opencloud_url" env:"OC_URL;NOTIFICATIONS_WEB_UI_URL" desc:"The public facing URL of the OpenCloud Web UI, used e.g. when sending notification eMails" introductionVersion:"1.0.0"`

	Notifications  Notifications        `yaml:"notifications"`
	GRPCClientTLS  shared.GRPCClientTLS `yaml:"grpc_client_tls"`
	ServiceAccount ServiceAccount       `yaml:"service_account"`

	Context context.Context `yaml:"-"`

	Store Store `yaml:"store"`
}

// Notifications defines the config options for the notifications service.
type Notifications struct {
	SMTP              SMTP                  `yaml:"SMTP"`
	Events            Events                `yaml:"events"`
	EmailTemplatePath string                `yaml:"email_template_path" env:"OC_EMAIL_TEMPLATE_PATH;NOTIFICATIONS_EMAIL_TEMPLATE_PATH" desc:"Path to Email notification templates overriding embedded ones." introductionVersion:"1.0.0"`
	TranslationPath   string                `yaml:"translation_path" env:"OC_TRANSLATION_PATH;NOTIFICATIONS_TRANSLATION_PATH" desc:"(optional) Set this to a path with custom translations to overwrite the builtin translations. Note that file and folder naming rules apply, see the documentation for more details." introductionVersion:"1.0.0"`
	DefaultLanguage   string                `yaml:"default_language" env:"OC_DEFAULT_LANGUAGE" desc:"The default language used by services and the WebUI. If not defined, English will be used as default. See the documentation for more details." introductionVersion:"1.0.0"`
	RevaGateway       string                `yaml:"reva_gateway" env:"OC_REVA_GATEWAY" desc:"CS3 gateway used to look up user metadata" introductionVersion:"1.0.0"`
	GRPCClientTLS     *shared.GRPCClientTLS `yaml:"grpc_client_tls"`
}

// SMTP combines the smtp configuration options.
type SMTP struct {
	Host           string `yaml:"smtp_host" env:"NOTIFICATIONS_SMTP_HOST" desc:"SMTP host to connect to." introductionVersion:"1.0.0"`
	Port           int    `yaml:"smtp_port" env:"NOTIFICATIONS_SMTP_PORT" desc:"Port of the SMTP host to connect to." introductionVersion:"1.0.0"`
	Sender         string `yaml:"smtp_sender" env:"NOTIFICATIONS_SMTP_SENDER" desc:"Sender address of emails that will be sent (e.g. 'OpenCloud <noreply@example.com>'." introductionVersion:"1.0.0"`
	Username       string `yaml:"smtp_username" env:"NOTIFICATIONS_SMTP_USERNAME" desc:"Username for the SMTP host to connect to." introductionVersion:"1.0.0"`
	Password       string `yaml:"smtp_password" env:"NOTIFICATIONS_SMTP_PASSWORD" desc:"Password for the SMTP host to connect to." introductionVersion:"1.0.0"`
	Insecure       bool   `yaml:"insecure" env:"NOTIFICATIONS_SMTP_INSECURE" desc:"Allow insecure connections to the SMTP server." introductionVersion:"1.0.0"`
	Authentication string `yaml:"smtp_authentication" env:"NOTIFICATIONS_SMTP_AUTHENTICATION" desc:"Authentication method for the SMTP communication. Possible values are 'login', 'plain', 'crammd5', 'none' or 'auto'. If set to 'auto' or unset, the authentication method is automatically negotiated with the server." introductionVersion:"1.0.0"`
	Encryption     string `yaml:"smtp_encryption" env:"NOTIFICATIONS_SMTP_ENCRYPTION" desc:"Encryption method for the SMTP communication. Possible values are 'starttls', 'ssltls' and 'none'." introductionVersion:"1.0.0"`
}

// Events combines the configuration options for the event bus.
type Events struct {
	Endpoint             string `yaml:"endpoint" env:"OC_EVENTS_ENDPOINT;NOTIFICATIONS_EVENTS_ENDPOINT" desc:"The address of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture." introductionVersion:"1.0.0"`
	Cluster              string `yaml:"cluster" env:"OC_EVENTS_CLUSTER;NOTIFICATIONS_EVENTS_CLUSTER" desc:"The clusterID of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture. Mandatory when using NATS as event system." introductionVersion:"1.0.0"`
	TLSInsecure          bool   `yaml:"tls_insecure" env:"OC_INSECURE;OC_EVENTS_TLS_INSECURE;NOTIFICATIONS_EVENTS_TLS_INSECURE" desc:"Whether to verify the server TLS certificates." introductionVersion:"1.0.0"`
	TLSRootCACertificate string `yaml:"tls_root_ca_certificate" env:"OC_EVENTS_TLS_ROOT_CA_CERTIFICATE;NOTIFICATIONS_EVENTS_TLS_ROOT_CA_CERTIFICATE" desc:"The root CA certificate used to validate the server's TLS certificate. If provided NOTIFICATIONS_EVENTS_TLS_INSECURE will be seen as false." introductionVersion:"1.0.0"`
	EnableTLS            bool   `yaml:"enable_tls" env:"OC_EVENTS_ENABLE_TLS;NOTIFICATIONS_EVENTS_ENABLE_TLS" desc:"Enable TLS for the connection to the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
	AuthUsername         string `yaml:"username" env:"OC_EVENTS_AUTH_USERNAME;NOTIFICATIONS_EVENTS_AUTH_USERNAME" desc:"The username to authenticate with the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
	AuthPassword         string `yaml:"password" env:"OC_EVENTS_AUTH_PASSWORD;NOTIFICATIONS_EVENTS_AUTH_PASSWORD" desc:"The password to authenticate with the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
}

// ServiceAccount is the configuration for the used service account
type ServiceAccount struct {
	ServiceAccountID     string `yaml:"service_account_id" env:"OC_SERVICE_ACCOUNT_ID;NOTIFICATIONS_SERVICE_ACCOUNT_ID" desc:"The ID of the service account the service should use. See the 'auth-service' service description for more details." introductionVersion:"1.0.0"`
	ServiceAccountSecret string `yaml:"service_account_secret" env:"OC_SERVICE_ACCOUNT_SECRET;NOTIFICATIONS_SERVICE_ACCOUNT_SECRET" desc:"The service account secret." introductionVersion:"1.0.0"`
}

// Store configures the store to use
type Store struct {
	Store        string        `yaml:"store" env:"OC_PERSISTENT_STORE;NOTIFICATIONS_STORE" desc:"The type of the store. Supported values are: 'memory', 'nats-js-kv', 'redis-sentinel', 'noop'. See the text description for details." introductionVersion:"1.0.0"`
	Nodes        []string      `yaml:"nodes" env:"OC_PERSISTENT_STORE_NODES;NOTIFICATIONS_STORE_NODES" desc:"A list of nodes to access the configured store. This has no effect when 'memory' store is configured. Note that the behaviour how nodes are used is dependent on the library of the configured store. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	Database     string        `yaml:"database" env:"NOTIFICATIONS_STORE_DATABASE" desc:"The database name the configured store should use." introductionVersion:"1.0.0"`
	Table        string        `yaml:"table" env:"NOTIFICATIONS_STORE_TABLE" desc:"The database table the store should use." introductionVersion:"1.0.0"`
	TTL          time.Duration `yaml:"ttl" env:"OC_PERSISTENT_STORE_TTL;NOTIFICATIONS_STORE_TTL" desc:"Time to live for notifications in the store. Defaults to '336h' (2 weeks). See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	AuthUsername string        `yaml:"username" env:"OC_PERSISTENT_STORE_AUTH_USERNAME;NOTIFICATIONS_STORE_AUTH_USERNAME" desc:"The username to authenticate with the store. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"1.0.0"`
	AuthPassword string        `yaml:"password" env:"OC_PERSISTENT_STORE_AUTH_PASSWORD;NOTIFICATIONS_STORE_AUTH_PASSWORD" desc:"The password to authenticate with the store. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"1.0.0"`
}
