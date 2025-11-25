package tracing

// ConfigConverter is the interface for external configuration.
type ConfigConverter interface {
	Convert() Config
}

// Tracing defines the available tracing configuration.
type Config struct {
	Exporter  string `yaml:"exporter" env:"OC_TRACES_EXPORTER;OC_TRACING_EXPORTER;OTEL_TRACES_EXPORTER" desc:"Tracing exporter to use. Supported values are 'none', 'console' and 'otlp'." introductionVersion:"1.0.0"`
	Enabled   bool   `yaml:"enabled" env:"OC_TRACING_ENABLED" desc:"Deprecated: use exporter 'none' to disable tracing." introductionVersion:"1.0.0" deprecationVersion:"2.0.0" deprecationInfo:"Set OC_TRACES_EXPORTER (or OTEL_TRACES_EXPORTER) to 'none'."`
	Type      string `yaml:"type" env:"OC_TRACING_TYPE" desc:"Deprecated: legacy tracing type. Jaeger is no longer supported." introductionVersion:"1.0.0" deprecationVersion:"2.0.0" deprecationInfo:"Use OC_TRACES_EXPORTER or OTEL_TRACES_EXPORTER instead."`
	Endpoint  string `yaml:"endpoint" env:"OC_TRACING_ENDPOINT" desc:"Deprecated: legacy OTLP/Jaeger endpoint." introductionVersion:"1.0.0" deprecationVersion:"2.0.0" deprecationInfo:"Set OTEL_EXPORTER_OTLP_ENDPOINT (and related OTEL_* variables) instead."`
	Collector string `yaml:"collector" env:"OC_TRACING_COLLECTOR" desc:"Deprecated: legacy Jaeger collector endpoint." introductionVersion:"1.0.0" deprecationVersion:"2.0.0" deprecationInfo:"Set OTEL_EXPORTER_OTLP_ENDPOINT / OTEL_EXPORTER_OTLP_PROTOCOL instead."`
}
