package config

import "github.com/opencloud-eu/opencloud/pkg/tracing"

// Tracing defines the available tracing configuration.
type Tracing struct {
	Exporter  string `yaml:"exporter" env:"OC_TRACES_EXPORTER;SETTINGS_TRACES_EXPORTER;OTEL_TRACES_EXPORTER" desc:"Tracing exporter to use. Supported values are 'none', 'console' and 'otlp'." introductionVersion:"1.0.0"`
	Enabled   bool   `yaml:"enabled" env:"OC_TRACING_ENABLED;SETTINGS_TRACING_ENABLED" desc:"Deprecated: use exporter 'none' to disable tracing." introductionVersion:"1.0.0" deprecationVersion:"2.0.0" deprecationInfo:"Set OC_TRACES_EXPORTER/SETTINGS_TRACES_EXPORTER (or OTEL_TRACES_EXPORTER) to 'none'."`
	Type      string `yaml:"type" env:"OC_TRACING_TYPE;SETTINGS_TRACING_TYPE" desc:"Deprecated: legacy tracing type." introductionVersion:"1.0.0" deprecationVersion:"2.0.0" deprecationInfo:"Use OC_TRACES_EXPORTER or OTEL_TRACES_EXPORTER instead."`
	Endpoint  string `yaml:"endpoint" env:"OC_TRACING_ENDPOINT;SETTINGS_TRACING_ENDPOINT" desc:"Deprecated: legacy tracing endpoint." introductionVersion:"1.0.0" deprecationVersion:"2.0.0" deprecationInfo:"Configure OTEL_EXPORTER_OTLP_ENDPOINT instead."`
	Collector string `yaml:"collector" env:"OC_TRACING_COLLECTOR;SETTINGS_TRACING_COLLECTOR" desc:"Deprecated: legacy tracing collector endpoint." introductionVersion:"1.0.0" deprecationVersion:"2.0.0" deprecationInfo:"Configure OTEL_EXPORTER_OTLP_ENDPOINT/OTEL_EXPORTER_OTLP_PROTOCOL instead."`
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
