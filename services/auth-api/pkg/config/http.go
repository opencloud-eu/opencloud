package config

import "github.com/opencloud-eu/opencloud/pkg/shared"

// HTTP defines the available http configuration.
type HTTP struct {
	Addr      string                `yaml:"addr" env:"AUTHAPI_HTTP_ADDR" desc:"The bind address of the HTTP service." introductionVersion:"1.0.0"`
	TLS       shared.HTTPServiceTLS `yaml:"tls"`
	Root      string                `yaml:"root" env:"AUTHAPI_HTTP_ROOT" desc:"Subdirectory that serves as the root for this HTTP service." introductionVersion:"1.0.0"`
	Namespace string                `yaml:"-"`
}
