package config

import "github.com/opencloud-eu/opencloud/pkg/shared"

type HTTPMetrics struct {
	Disabled bool `yaml:"disabled" env:"GRAPH_HTTP_METRICS_DISABLE" desc:"Disables the metrics for the HTTP service." introductionVersion:"%%NEXT%%"`
}

// HTTP defines the available http configuration.
type HTTP struct {
	Disabled  bool                  `yaml:"disabled" env:"GRAPH_HTTP_DISABLE" desc:"Disables the HTTP service. Set this to true if the service should only consume events." introductionVersion:"%%NEXT%%"`
	Addr      string                `yaml:"addr" env:"GRAPH_HTTP_ADDR" desc:"The bind address of the HTTP service." introductionVersion:"1.0.0"`
	Namespace string                `yaml:"-"`
	Root      string                `yaml:"root" env:"GRAPH_HTTP_ROOT" desc:"Subdirectory that serves as the root for this HTTP service." introductionVersion:"1.0.0"`
	TLS       shared.HTTPServiceTLS `yaml:"tls"`
	APIToken  string                `yaml:"apitoken" env:"GRAPH_HTTP_API_TOKEN" desc:"An optional API bearer token" introductionVersion:"1.0.0"`
	CORS      CORS                  `yaml:"cors"`
	Metrics   HTTPMetrics           `yaml:"metrics"`
}
