package config

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// Config combines all available configuration parts.
type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service

	Service Service `yaml:"-"`
	App     App     `yaml:"app"`
	Font    Font    `yaml:"font"`
	Store   Store   `yaml:"store"`
	Events  Events  `yaml:"events"`

	TokenManager *TokenManager `yaml:"token_manager"`

	GRPC GRPC `yaml:"grpc"`
	HTTP HTTP `yaml:"http"`

	Wopi   Wopi   `yaml:"wopi"`
	CS3Api CS3Api `yaml:"cs3api"`

	LogLevel string `yaml:"loglevel" env:"OC_LOG_LEVEL;COLLABORATION_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"1.0.0"`
	Debug    Debug  `yaml:"debug"`

	Context context.Context `yaml:"-"`

	MachineAuthAPIKey string `yaml:"machine_auth_api_key" env:"OC_MACHINE_AUTH_API_KEY;COLLABORATION_MACHINE_AUTH_API_KEY" desc:"The machine auth API key used to validate internal requests necessary to access resources from other services." introductionVersion:"7.3.0"`
}
