package config

import (
	"context"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// Config combines all available configuration parts.
type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service

	Service Service `yaml:"-"`

	Tracing *Tracing `yaml:"tracing"`
	Log     *Log     `yaml:"log"`
	Debug   Debug    `yaml:"debug"`

	HTTP HTTP `yaml:"http"`

	Mail Mail `yaml:"mail"`

	Context context.Context `yaml:"-"`
}

type MasterAuth struct {
	Username string `yaml:"username" env:"OC_JMAP_MASTER_USERNAME;GROUPWARE_JMAP_MASTER_USERNAME"`
	Password string `yaml:"password" env:"OC_JMAP_MASTER_PASSWORD;GROUPWARE_JMAP_MASTER_PASSWORD"`
}

type Mail struct {
	Master  MasterAuth    `yaml:"master"`
	BaseUrl string        `yaml:"base_url" env:"OC_JMAP_BASE_URL;GROUPWARE_BASE_URL"`
	JmapUrl string        `yaml:"jmap_url" env:"OC_JMAP_JMAP_URL;GROUPWARE_JMAP_URL"`
	Timeout time.Duration `yaml:"timeout" env:"OC_JMAP_TIMEOUT"`
}
