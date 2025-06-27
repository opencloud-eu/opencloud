package config

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
)

// Config combines all available configuration parts.
type Config struct {
	Commons       *shared.Commons       `yaml:"-"`
	Service       Service               `yaml:"-"`
	GRPCClientTLS *shared.GRPCClientTLS `yaml:"grpc_client_tls"`
	Tracing       *Tracing              `yaml:"tracing"`
	Log           *Log                  `yaml:"log"`
	Debug         Debug                 `yaml:"debug"`
	Context       context.Context       `yaml:"-"`
	GRPC          GRPCConfig            `yaml:"grpc"`
	RemoteConsole *ConsoleRemote        `yaml:"remote"`
}

// Service defines the available service configuration.
type Service struct {
	Name string `yaml:"-"`
}

// Log defines the available log configuration.
type Log struct {
	Level  string `mapstructure:"level" env:"OC_LOG_LEVEL;CONSOLE_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"%%NEXT%%"`
	Pretty bool   `mapstructure:"pretty" env:"OC_LOG_PRETTY;CONSOLE_LOG_PRETTY" desc:"Activates pretty log output." introductionVersion:"%%NEXT%%"`
	Color  bool   `mapstructure:"color" env:"OC_LOG_COLOR;CONSOLE_LOG_COLOR" desc:"Activates colorized log output." introductionVersion:"%%NEXT%%"`
	File   string `mapstructure:"file" env:"OC_LOG_FILE;CONSOLE_LOG_FILE" desc:"The path to the log file. Activates logging to this file if set." introductionVersion:"%%NEXT%%"`
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

// Convert Tracing to the tracing package's Config struct.
func (t Tracing) Convert() tracing.Config {
	return tracing.Config{
		Enabled:   t.Enabled,
		Type:      t.Type,
		Endpoint:  t.Endpoint,
		Collector: t.Collector,
	}
}
