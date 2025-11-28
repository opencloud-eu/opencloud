package config

import (
	"context"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// Config combines all available configuration parts.
type Config struct {
	Commons        *shared.Commons       `yaml:"-"` // don't use this directly as configuration for a service
	GRPC           GRPC                  `yaml:"grpc"`
	Service        Service               `yaml:"-"`
	Debug          Debug                 `yaml:"debug"`
	Events         Events                `yaml:"events"`
	GRPCClientTLS  *shared.GRPCClientTLS `yaml:"grpc_client_tls"`
	Context        context.Context       `yaml:"-"`
	Log            *Log                  `yaml:"log"`
	Engine         Engine                `yaml:"engine"`
	Postprocessing Postprocessing        `yaml:"postprocessing"`
}

// Service defines the available service configuration.
type Service struct {
	Name string `yaml:"-"`
}

// GRPC defines the available grpc configuration.
type GRPC struct {
	Addr      string                 `yaml:"addr" env:"POLICIES_GRPC_ADDR" desc:"The bind address of the GRPC service." introductionVersion:"1.0.0"`
	Namespace string                 `yaml:"-"`
	TLS       *shared.GRPCServiceTLS `yaml:"tls"`
}

// Engine configures the policy engine.
type Engine struct {
	Timeout  time.Duration `yaml:"timeout" env:"POLICIES_ENGINE_TIMEOUT" desc:"Sets the timeout the rego expression evaluation can take. Rules default to deny if the timeout was reached. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	Policies []string      `yaml:"policies"`
	// Mimes file path, RFC 4288
	Mimes string `yaml:"mimes" env:"POLICIES_ENGINE_MIMES" desc:"Sets the mimes file path which maps mimetypes to associated file extensions. See the text description for details." introductionVersion:"1.0.0"`
}

// Postprocessing defines the config options for the postprocessing policy handling.
type Postprocessing struct {
	Query string `yaml:"query" env:"POLICIES_POSTPROCESSING_QUERY" desc:"Defines the 'Complete Rules' variable defined in the rego rule set this step uses for its evaluation. Defaults to deny if the variable was not found." introductionVersion:"1.0.0"`
}

// Events combines the configuration options for the event bus.
type Events struct {
	Endpoint             string `yaml:"endpoint" env:"OC_EVENTS_ENDPOINT;POLICIES_EVENTS_ENDPOINT" desc:"The address of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture." introductionVersion:"1.0.0"`
	Cluster              string `yaml:"cluster" env:"OC_EVENTS_CLUSTER;POLICIES_EVENTS_CLUSTER" desc:"The clusterID of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture. Mandatory when using NATS as event system." introductionVersion:"1.0.0"`
	TLSInsecure          bool   `yaml:"tls_insecure" env:"OC_INSECURE;OC_EVENTS_TLS_INSECURE;POLICIES_EVENTS_TLS_INSECURE" desc:"Whether the server should skip the client certificate verification during the TLS handshake." introductionVersion:"1.0.0"`
	TLSRootCACertificate string `yaml:"tls_root_ca_certificate" env:"OC_EVENTS_TLS_ROOT_CA_CERTIFICATE;POLICIES_EVENTS_TLS_ROOT_CA_CERTIFICATE" desc:"The root CA certificate used to validate the server's TLS certificate. If provided POLICIES_EVENTS_TLS_INSECURE will be seen as false." introductionVersion:"1.0.0"`
	EnableTLS            bool   `yaml:"enable_tls" env:"OC_EVENTS_ENABLE_TLS;POLICIES_EVENTS_ENABLE_TLS" desc:"Enable TLS for the connection to the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
	AuthUsername         string `yaml:"username" env:"OC_EVENTS_AUTH_USERNAME;POLICIES_EVENTS_AUTH_USERNAME" desc:"The username to authenticate with the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
	AuthPassword         string `yaml:"password" env:"OC_EVENTS_AUTH_PASSWORD;POLICIES_EVENTS_AUTH_PASSWORD" desc:"The password to authenticate with the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
}

// Log defines the available log configuration.
type Log struct {
	Level  string `mapstructure:"level" env:"OC_LOG_LEVEL;POLICIES_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"1.0.0"`
	Pretty bool   `mapstructure:"pretty" env:"OC_LOG_PRETTY;POLICIES_LOG_PRETTY" desc:"Activates pretty log output." introductionVersion:"1.0.0"`
	Color  bool   `mapstructure:"color" env:"OC_LOG_COLOR;POLICIES_LOG_COLOR" desc:"Activates colorized log output." introductionVersion:"1.0.0"`
	File   string `mapstructure:"file" env:"OC_LOG_FILE;POLICIES_LOG_FILE" desc:"The path to the log file. Activates logging to this file if set." introductionVersion:"1.0.0"`
}

// Debug defines the available debug configuration.
type Debug struct {
	Addr   string `yaml:"addr" env:"POLICIES_DEBUG_ADDR" desc:"Bind address of the debug server, where metrics, health, config and debug endpoints will be exposed." introductionVersion:"1.0.0"`
	Token  string `yaml:"token" env:"POLICIES_DEBUG_TOKEN" desc:"Token to secure the metrics endpoint." introductionVersion:"1.0.0"`
	Pprof  bool   `yaml:"pprof" env:"POLICIES_DEBUG_PPROF" desc:"Enables pprof, which can be used for profiling." introductionVersion:"1.0.0"`
	Zpages bool   `yaml:"zpages" env:"POLICIES_DEBUG_ZPAGES" desc:"Enables zpages, which can be used for collecting and viewing in-memory traces." introductionVersion:"1.0.0"`
}
