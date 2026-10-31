package config

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// Config combines all available configuration parts.
type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service

	Service Service `yaml:"-"`

	Log   *Log  `yaml:"log"`
	Debug Debug `yaml:"debug"`

	HTTP HTTP `yaml:"http"`

	Context context.Context `yaml:"-"`

	TokenManager *TokenManager `yaml:"token_manager"`

	Auth Auth `yaml:"auth"`
}
