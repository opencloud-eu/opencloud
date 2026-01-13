package config

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service

	Service  Service `yaml:"-"`
	LogLevel string  `yaml:"loglevel" env:"OC_LOG_LEVEL;APP_REGISTRY_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"1.0.0"`

	Debug Debug `yaml:"debug"`

	GRPC GRPCConfig `yaml:"grpc"`

	TokenManager *TokenManager `yaml:"token_manager"`
	Reva         *shared.Reva  `yaml:"reva"`

	AppRegistry AppRegistry `yaml:"app_registry"`

	Context context.Context `yaml:"-"`
}

type Service struct {
	Name string `yaml:"-"`
}

type Debug struct {
	Addr   string `yaml:"addr" env:"APP_REGISTRY_DEBUG_ADDR" desc:"Bind address of the debug server, where metrics, health, config and debug endpoints will be exposed." introductionVersion:"1.0.0"`
	Token  string `yaml:"token" env:"APP_REGISTRY_DEBUG_TOKEN" desc:"Token to secure the metrics endpoint." introductionVersion:"1.0.0"`
	Pprof  bool   `yaml:"pprof" env:"APP_REGISTRY_DEBUG_PPROF" desc:"Enables pprof, which can be used for profiling." introductionVersion:"1.0.0"`
	Zpages bool   `yaml:"zpages" env:"APP_REGISTRY_DEBUG_ZPAGES" desc:"Enables zpages, which can be used for collecting and viewing in-memory traces." introductionVersion:"1.0.0"`
}

type GRPCConfig struct {
	Addr      string                 `yaml:"addr" env:"APP_REGISTRY_GRPC_ADDR" desc:"The bind address of the GRPC service." introductionVersion:"1.0.0"`
	TLS       *shared.GRPCServiceTLS `yaml:"tls"`
	Namespace string                 `yaml:"-"`
	Protocol  string                 `yaml:"protocol" env:"OC_GRPC_PROTOCOL;APP_REGISTRY_GRPC_PROTOCOL" desc:"The transport protocol of the GRPC service." introductionVersion:"1.0.0"`
}

type AppRegistry struct {
	MimeTypeConfig []MimeTypeConfig `yaml:"mimetypes"`
}

type MimeTypeConfig struct {
	MimeType      string `yaml:"mime_type" mapstructure:"mime_type"`
	Extension     string `yaml:"extension" mapstructure:"extension"`
	Name          string `yaml:"name" mapstructure:"name"`
	Description   string `yaml:"description" mapstructure:"description"`
	Icon          string `yaml:"icon" mapstructure:"icon"`
	DefaultApp    string `yaml:"default_app" mapstructure:"default_app"`
	AllowCreation bool   `yaml:"allow_creation" mapstructure:"allow_creation"`
}
