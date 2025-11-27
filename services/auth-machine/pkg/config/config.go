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

	SkipUserGroupsInToken bool `yaml:"skip_user_groups_in_token" env:"AUTH_MACHINE_SKIP_USER_GROUPS_IN_TOKEN" desc:"Disables the encoding of the user's group memberships in the reva access token. This reduces the token size, especially when users are members of a large number of groups." introductionVersion:"1.0.0"`

	MachineAuthAPIKey string `yaml:"machine_auth_api_key" env:"OC_MACHINE_AUTH_API_KEY;AUTH_MACHINE_API_KEY" desc:"Machine auth API key used to validate internal requests necessary for the access to resources from other services." introductionVersion:"1.0.0"`

	Context context.Context `yaml:"-"`
}

type Log struct {
	Level  string `yaml:"level" env:"OC_LOG_LEVEL;AUTH_MACHINE_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"1.0.0"`
	Pretty bool   `yaml:"pretty" env:"OC_LOG_PRETTY;AUTH_MACHINE_LOG_PRETTY" desc:"Activates pretty log output." introductionVersion:"1.0.0"`
	Color  bool   `yaml:"color" env:"OC_LOG_COLOR;AUTH_MACHINE_LOG_COLOR" desc:"Activates colorized log output." introductionVersion:"1.0.0"`
	File   string `yaml:"file" env:"OC_LOG_FILE;AUTH_MACHINE_LOG_FILE" desc:"The path to the log file. Activates logging to this file if set." introductionVersion:"1.0.0"`
}

type Service struct {
	Name string `yaml:"-"`
}

type Debug struct {
	Addr   string `yaml:"addr" env:"AUTH_MACHINE_DEBUG_ADDR" desc:"Bind address of the debug server, where metrics, health, config and debug endpoints will be exposed." introductionVersion:"1.0.0"`
	Token  string `yaml:"token" env:"AUTH_MACHINE_DEBUG_TOKEN" desc:"Token to secure the metrics endpoint." introductionVersion:"1.0.0"`
	Pprof  bool   `yaml:"pprof" env:"AUTH_MACHINE_DEBUG_PPROF" desc:"Enables pprof, which can be used for profiling." introductionVersion:"1.0.0"`
	Zpages bool   `yaml:"zpages" env:"AUTH_MACHINE_DEBUG_ZPAGES" desc:"Enables zpages, which can be used for collecting and viewing in-memory traces." introductionVersion:"1.0.0"`
}

type GRPCConfig struct {
	Addr      string                 `yaml:"addr" env:"AUTH_MACHINE_GRPC_ADDR" desc:"The bind address of the GRPC service." introductionVersion:"1.0.0"`
	TLS       *shared.GRPCServiceTLS `yaml:"tls"`
	Namespace string                 `yaml:"-"`
	Protocol  string                 `yaml:"protocol" env:"OC_GRPC_PROTOCOL;AUTH_MACHINE_GRPC_PROTOCOL" desc:"The transport protocol of the GRPC service." introductionVersion:"1.0.0"`
}
