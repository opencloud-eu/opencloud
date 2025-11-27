package config

import (
	"time"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// CS3Api defines the available configuration in order to access to the CS3 gateway.
type CS3Api struct {
	Gateway                 Gateway               `yaml:"gateway"`
	DataGateway             DataGateway           `yaml:"datagateway"`
	GRPCClientTLS           *shared.GRPCClientTLS `yaml:"grpc_client_tls"`
	APPRegistrationInterval time.Duration         `yaml:"app_registration_interval" env:"COLLABORATION_CS3API_APP_REGISTRATION_INTERVAL" desc:"The interval at which the app provider registers itself." introductionVersion:"4.0.0"`
}

// Gateway defines the available configuration for the CS3 API gateway
type Gateway struct {
	Name string `yaml:"name" env:"OC_REVA_GATEWAY" desc:"CS3 gateway used to look up user metadata." introductionVersion:"1.0.0"`
}

// DataGateway defines the available configuration for the CS3 API data gateway
type DataGateway struct {
	Insecure bool `yaml:"insecure" env:"COLLABORATION_CS3API_DATAGATEWAY_INSECURE" desc:"Connect to the CS3API data gateway insecurely." introductionVersion:"1.0.0"`
}
