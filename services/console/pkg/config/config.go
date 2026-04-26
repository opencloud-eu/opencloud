package config

import (
	"context"

	"go-micro.dev/v4/client"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// Config combines all available configuration parts.
type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service

	Service Service `yaml:"-"`

	LogLevel string `yaml:"loglevel" env:"OC_LOG_LEVEL;THUMBNAILS_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"1.0.0"`
	Debug    Debug  `yaml:"debug"`

	GRPC GRPCConfig `yaml:"grpc"`

	GRPCClientTLS *shared.GRPCClientTLS `yaml:"grpc_client_tls"`
	GrpcClient    client.Client         `yaml:"-"`
	Events        Events                `yaml:"events"`

	RemoteConsole *ConsoleRemote `yaml:"remote"`

	Context context.Context `yaml:"-"`
}

// Service defines the available service configuration.
type Service struct {
	Name string `yaml:"-"`
}

// Debug defines the available debug configuration.
type Debug struct {
	Addr   string `yaml:"addr" env:"CONSOLE_DEBUG_ADDR" desc:"Bind address of the debug server, where metrics, health, config and debug endpoints will be exposed." introductionVersion:"%%NEXT%%"`
	Token  string `yaml:"token" env:"CONSOLE_DEBUG_TOKEN" desc:"Token to secure the metrics endpoint." introductionVersion:"%%NEXT%%"`
	Pprof  bool   `yaml:"pprof" env:"CONSOLE_DEBUG_PPROF" desc:"Enables pprof, which can be used for profiling." introductionVersion:"%%NEXT%%"`
	Zpages bool   `yaml:"zpages" env:"CONSOLE_DEBUG_ZPAGES" desc:"Enables zpages, which can be used for collecting and viewing in-memory traces." introductionVersion:"%%NEXT%%"`
}

// GRPCConfig defines the available grpc configuration.
type GRPCConfig struct {
	Disabled  bool                   `yaml:"disabled" env:"CONSOLE_GRPC_DISABLED" desc:"Disables the GRPC service. Set this to true if the service should only handle events." introductionVersion:"%%NEXT%%"`
	Addr      string                 `yaml:"addr" env:"CONSOLE_GRPC_ADDR" desc:"The bind address of the GRPC service." introductionVersion:"%%NEXT%%"`
	Namespace string                 `yaml:"-"`
	TLS       *shared.GRPCServiceTLS `yaml:"tls"`
}

// Tracing defines the available tracing configuration.
type Tracing struct {
	Enabled   bool   `yaml:"enabled" env:"OC_TRACING_ENABLED;CONSOLE_TRACING_ENABLED" desc:"Activates tracing." introductionVersion:"%%NEXT%%"`
	Type      string `yaml:"type" env:"OC_TRACING_TYPE;CONSOLE_TRACING_TYPE" desc:"The type of tracing. Defaults to '', which is the same as 'jaeger'. Allowed tracing types are 'jaeger' and '' as of now." introductionVersion:"%%NEXT%%"`
	Endpoint  string `yaml:"endpoint" env:"OC_TRACING_ENDPOINT;CONSOLE_TRACING_ENDPOINT" desc:"The endpoint of the tracing agent." introductionVersion:"%%NEXT%%"`
	Collector string `yaml:"collector" env:"OC_TRACING_COLLECTOR;CONSOLE_TRACING_COLLECTOR" desc:"The HTTP endpoint for sending spans directly to a collector, i.e. http://jaeger-collector:14268/api/traces. Only used if the tracing endpoint is unset." introductionVersion:"%%NEXT%%"`
}

// Events combines the configuration options for the event bus.
type Events struct {
	Endpoint             string `yaml:"endpoint" env:"OC_EVENTS_ENDPOINT;CONSOLE_EVENTS_ENDPOINT" desc:"The address of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture." introductionVersion:"1.0.0"`
	Cluster              string `yaml:"cluster" env:"OC_EVENTS_CLUSTER;CONSOLE_EVENTS_CLUSTER" desc:"The clusterID of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture. Mandatory when using NATS as event system." introductionVersion:"1.0.0"`
	TLSInsecure          bool   `yaml:"tls_insecure" env:"OC_INSECURE;OC_EVENTS_TLS_INSECURE;CONSOLE_EVENTS_TLS_INSECURE" desc:"Whether to verify the server TLS certificates." introductionVersion:"1.0.0"`
	TLSRootCACertificate string `yaml:"tls_root_ca_certificate" env:"OC_EVENTS_TLS_ROOT_CA_CERTIFICATE;CONSOLE_EVENTS_TLS_ROOT_CA_CERTIFICATE" desc:"The root CA certificate used to validate the server's TLS certificate. If provided NOTIFICATIONS_EVENTS_TLS_INSECURE will be seen as false." introductionVersion:"1.0.0"`
	EnableTLS            bool   `yaml:"enable_tls" env:"OC_EVENTS_ENABLE_TLS;CONSOLE_EVENTS_ENABLE_TLS" desc:"Enable TLS for the connection to the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
	AuthUsername         string `yaml:"username" env:"OC_EVENTS_AUTH_USERNAME;CONSOLE_EVENTS_AUTH_USERNAME" desc:"The username to authenticate with the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
	AuthPassword         string `yaml:"password" env:"OC_EVENTS_AUTH_PASSWORD;CONSOLE_EVENTS_AUTH_PASSWORD" desc:"The password to authenticate with the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
}
