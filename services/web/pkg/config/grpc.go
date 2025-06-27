package config

import "github.com/opencloud-eu/opencloud/pkg/shared"

// GRPCConfig defines the available grpc configuration.
type GRPCConfig struct {
	Addr      string                 `yaml:"addr" env:"WEB_GRPC_ADDR" desc:"The bind address of the GRPC service." introductionVersion:"%%NEXT%%"`
	Namespace string                 `yaml:"-"`
	TLS       *shared.GRPCServiceTLS `yaml:"tls"`
	Protocol  string                 `yaml:"protocol" env:"OC_GRPC_PROTOCOL;WEB_GRPC_PROTOCOL" desc:"The transport protocol of the GRPC service." introductionVersion:"%%NEXT%%"`
}
