package config

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service
	Service Service         `yaml:"-"`
	Log     *Log            `yaml:"log"`
	Debug   Debug           `yaml:"debug"`

	GRPC GRPCConfig `yaml:"grpc"`

	TokenManager *TokenManager `yaml:"token_manager"`
	Reva         *shared.Reva  `yaml:"reva"`

	// TODO: when using multiple service accounts we need to find a way to configure them
	ServiceAccount ServiceAccount `yaml:"service_account"`

	Context context.Context `yaml:"-"`
}

type Log struct {
	Level  string `yaml:"level" env:"OC_LOG_LEVEL;AUTH_SERVICE_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"1.0.0"`
	Pretty bool   `yaml:"pretty" env:"OC_LOG_PRETTY;AUTH_SERVICE_LOG_PRETTY" desc:"Activates pretty log output." introductionVersion:"1.0.0"`
	Color  bool   `yaml:"color" env:"OC_LOG_COLOR;AUTH_SERVICE_LOG_COLOR" desc:"Activates colorized log output." introductionVersion:"1.0.0"`
	File   string `yaml:"file" env:"OC_LOG_FILE;AUTH_SERVICE_LOG_FILE" desc:"The path to the log file. Activates logging to this file if set." introductionVersion:"1.0.0"`
}

type Service struct {
	Name string `yaml:"-"`
}

type Debug struct {
	Addr   string `yaml:"addr" env:"AUTH_SERVICE_DEBUG_ADDR" desc:"Bind address of the debug server, where metrics, health, config and debug endpoints will be exposed." introductionVersion:"1.0.0"`
	Token  string `yaml:"token" env:"AUTH_SERVICE_DEBUG_TOKEN" desc:"Token to secure the metrics endpoint." introductionVersion:"1.0.0"`
	Pprof  bool   `yaml:"pprof" env:"AUTH_SERVICE_DEBUG_PPROF" desc:"Enables pprof, which can be used for profiling." introductionVersion:"1.0.0"`
	Zpages bool   `yaml:"zpages" env:"AUTH_SERVICE_DEBUG_ZPAGES" desc:"Enables zpages, which can be used for collecting and viewing in-memory traces." introductionVersion:"1.0.0"`
}

type GRPCConfig struct {
	Addr      string                 `yaml:"addr" env:"AUTH_SERVICE_GRPC_ADDR" desc:"The bind address of the GRPC service." introductionVersion:"1.0.0"`
	TLS       *shared.GRPCServiceTLS `yaml:"tls"`
	Namespace string                 `yaml:"-"`
	Protocol  string                 `yaml:"protocol" env:"OC_GRPC_PROTOCOL;AUTH_SERVICE_GRPC_PROTOCOL" desc:"The transport protocol of the GRPC service." introductionVersion:"1.0.0"`
}

// ServiceAccount is the configuration for the used service account
type ServiceAccount struct {
	ServiceAccountID     string `yaml:"service_account_id" env:"OC_SERVICE_ACCOUNT_ID;AUTH_SERVICE_SERVICE_ACCOUNT_ID" desc:"The ID of the service account the service should use. See the 'auth-service' service description for more details." introductionVersion:"1.0.0"`
	ServiceAccountSecret string `yaml:"service_account_secret" env:"OC_SERVICE_ACCOUNT_SECRET;AUTH_SERVICE_SERVICE_ACCOUNT_SECRET" desc:"The service account secret." introductionVersion:"1.0.0"`
}
