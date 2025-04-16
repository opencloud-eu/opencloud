package config

import (
	"context"

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
	Master           MasterAuth `yaml:"master"`
	JmapUrl          string     `yaml:"jmap_url" env:"OC_JMAP_URL;GROUPWARE_JMAP_URL"`
	CS3AllowInsecure bool       `yaml:"cs3_allow_insecure" env:"OC_INSECURE;GROUPWARE_CS3SOURCE_INSECURE" desc:"Ignore untrusted SSL certificates when connecting to the CS3 source." introductionVersion:"1.0.0"`
	RevaGateway      string     `yaml:"reva_gateway" env:"OC_REVA_GATEWAY" desc:"CS3 gateway used to look up user metadata" introductionVersion:"1.0.0"`
}
