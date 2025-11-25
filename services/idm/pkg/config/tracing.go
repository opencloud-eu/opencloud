package config

// Tracing defines the available tracing configuration.
type Tracing struct {
	Enabled   bool   `yaml:"enabled" env:"OC_TRACING_ENABLED;IDM_TRACING_ENABLED" desc:"Activates tracing." introductionVersion:"1.0.0"`
	Type      string `yaml:"type" env:"OC_TRACING_TYPE;IDM_TRACING_TYPE" desc:"The tracing exporter type. Defaults to '' which auto-detects the OTLP transport. Allowed values are '', 'otlp', 'otlp_grpc', and 'otlp_http'." introductionVersion:"1.0.0"`
	Endpoint  string `yaml:"endpoint" env:"OC_TRACING_ENDPOINT;IDM_TRACING_ENDPOINT" desc:"The OTLP gRPC endpoint (host:port or URL). Used for types 'otlp' and 'otlp_grpc', or when auto-detect selects gRPC." introductionVersion:"1.0.0"`
	Collector string `yaml:"collector" env:"OC_TRACING_COLLECTOR;IDM_TRACING_COLLECTOR" desc:"The OTLP HTTP collector endpoint (URL). Used for type 'otlp_http', or when auto-detect selects HTTP." introductionVersion:"1.0.0"`
}
