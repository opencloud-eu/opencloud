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

	SkipUserGroupsInToken bool `yaml:"skip_user_groups_in_token" env:"AUTH_BEARER_SKIP_USER_GROUPS_IN_TOKEN" desc:"Disables the encoding of the user's group memberships in the reva access token. This reduces the token size, especially when users are members of a large number of groups." introductionVersion:"1.0.0"`

	OIDC OIDC `yaml:"oidc"`

	Context context.Context `yaml:"-"`
}

type Log struct {
	Level  string `yaml:"level" env:"OC_LOG_LEVEL;AUTH_BEARER_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"1.0.0"`
	Pretty bool   `yaml:"pretty" env:"OC_LOG_PRETTY;AUTH_BEARER_LOG_PRETTY" desc:"Activates pretty log output." introductionVersion:"1.0.0"`
	Color  bool   `yaml:"color" env:"OC_LOG_COLOR;AUTH_BEARER_LOG_COLOR" desc:"Activates colorized log output." introductionVersion:"1.0.0"`
	File   string `yaml:"file" env:"OC_LOG_FILE;AUTH_BEARER_LOG_FILE" desc:"The path to the log file. Activates logging to this file if set." introductionVersion:"1.0.0"`
}

type Service struct {
	Name string `yaml:"-"`
}

type Debug struct {
	Addr   string `yaml:"addr" env:"AUTH_BEARER_DEBUG_ADDR" desc:"Bind address of the debug server, where metrics, health, config and debug endpoints will be exposed." introductionVersion:"1.0.0"`
	Token  string `yaml:"token" env:"AUTH_BEARER_DEBUG_TOKEN" desc:"Token to secure the metrics endpoint." introductionVersion:"1.0.0"`
	Pprof  bool   `yaml:"pprof" env:"AUTH_BEARER_DEBUG_PPROF" desc:"Enables pprof, which can be used for profiling." introductionVersion:"1.0.0"`
	Zpages bool   `yaml:"zpages" env:"AUTH_BEARER_DEBUG_ZPAGES" desc:"Enables zpages, which can be used for collecting and viewing in-memory traces." introductionVersion:"1.0.0"`
}

type GRPCConfig struct {
	Addr      string                 `yaml:"addr" env:"AUTH_BEARER_GRPC_ADDR" desc:"The bind address of the GRPC service." introductionVersion:"1.0.0"`
	TLS       *shared.GRPCServiceTLS `yaml:"tls"`
	Namespace string                 `yaml:"-"`
	Protocol  string                 `yaml:"protocol" env:"OC_GRPC_PROTOCOL;AUTH_BEARER_GRPC_PROTOCOL" desc:"The transport protocol of the GRPC service." introductionVersion:"1.0.0"`
}

type OIDC struct {
	Issuer   string `yaml:"issuer" env:"OC_URL;OC_OIDC_ISSUER;AUTH_BEARER_OIDC_ISSUER" desc:"URL of the OIDC issuer. It defaults to URL of the builtin IDP." introductionVersion:"1.0.0"`
	Insecure bool   `yaml:"insecure" env:"OC_INSECURE;AUTH_BEARER_OIDC_INSECURE" desc:"Allow insecure connections to the OIDC issuer." introductionVersion:"1.0.0"`
	IDClaim  string `yaml:"id_claim" env:"AUTH_BEARER_OIDC_ID_CLAIM" desc:"Name of the claim, which holds the user identifier." introductionVersion:"1.0.0"`
	UIDClaim string `yaml:"uid_claim" env:"AUTH_BEARER_OIDC_UID_CLAIM" desc:"Name of the claim, which holds the UID." introductionVersion:"1.0.0"`
	GIDClaim string `yaml:"gid_claim" env:"AUTH_BEARER_OIDC_GID_CLAIM" desc:"Name of the claim, which holds the GID." introductionVersion:"1.0.0"`
}
