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
	Store   Store   `yaml:"store"`

	TokenManager *TokenManager `yaml:"token_manager"`

	GRPC GRPC `yaml:"grpc"`
	HTTP HTTP `yaml:"http"`

	Wopi   Wopi   `yaml:"wopi"`
	CS3Api CS3Api `yaml:"cs3api"`

	Log   *Log  `yaml:"log"`
	Debug Debug `yaml:"debug"`

	Context context.Context `yaml:"-"`
}
