package config

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
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

	Tracing *Tracing `yaml:"tracing"`
	Log     *Log     `yaml:"log"`
	Debug   Debug    `yaml:"debug"`

	Context context.Context `yaml:"-"`
}

// Tracing defines the available tracing configuration. Not used at the moment
type Tracing struct {
	Exporter  string `yaml:"exporter" env:"OC_TRACES_EXPORTER;COLLABORATION_TRACES_EXPORTER;OTEL_TRACES_EXPORTER" desc:"Tracing exporter to use. Supported values are 'none', 'console' and 'otlp'." introductionVersion:"1.0.0"`
	Enabled   bool   `yaml:"enabled" env:"OC_TRACING_ENABLED;COLLABORATION_TRACING_ENABLED" desc:"Deprecated: use exporter 'none' to disable tracing." introductionVersion:"1.0.0" deprecationVersion:"2.0.0" deprecationInfo:"Set OC_TRACES_EXPORTER/COLLABORATION_TRACES_EXPORTER (or OTEL_TRACES_EXPORTER) to 'none'."`
	Type      string `yaml:"type" env:"OC_TRACING_TYPE;COLLABORATION_TRACING_TYPE" desc:"Deprecated: legacy tracing type." introductionVersion:"1.0.0" deprecationVersion:"2.0.0" deprecationInfo:"Use OC_TRACES_EXPORTER or OTEL_TRACES_EXPORTER instead."`
	Endpoint  string `yaml:"endpoint" env:"OC_TRACING_ENDPOINT;COLLABORATION_TRACING_ENDPOINT" desc:"Deprecated: legacy tracing endpoint." introductionVersion:"1.0.0" deprecationVersion:"2.0.0" deprecationInfo:"Configure OTEL_EXPORTER_OTLP_ENDPOINT instead."`
	Collector string `yaml:"collector" env:"OC_TRACING_COLLECTOR;COLLABORATION_TRACING_COLLECTOR" desc:"Deprecated: legacy tracing collector endpoint." introductionVersion:"1.0.0" deprecationVersion:"2.0.0" deprecationInfo:"Configure OTEL_EXPORTER_OTLP_ENDPOINT/OTEL_EXPORTER_OTLP_PROTOCOL instead."`
}

// Convert Tracing to the tracing package's Config struct.
func (t Tracing) Convert() tracing.Config {
	return tracing.Config{
		Exporter:  t.Exporter,
		Enabled:   t.Enabled,
		Type:      t.Type,
		Endpoint:  t.Endpoint,
		Collector: t.Collector,
	}
}
