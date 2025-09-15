package config

import "github.com/opencloud-eu/opencloud/pkg/shared"

// GRPCConfig defines the available grpc configuration.
type GRPCConfig struct {
	Disabled  bool                   `yaml:"disabled" env:"SEARCH_GRPC_DISABLED" desc:"Disables the GRPC service. Set this to true if the service should only handle events." introductionVersion:"%%NEXT%%"`
	Addr      string                 `yaml:"addr" env:"SEARCH_GRPC_ADDR" desc:"The bind address of the GRPC service." introductionVersion:"1.0.0"`
	Namespace string                 `yaml:"-"`
	TLS       *shared.GRPCServiceTLS `yaml:"tls"`
}
