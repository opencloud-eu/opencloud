package config

import (
	"context"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	"go-micro.dev/v4/client"
)

// Config combines all available configuration parts.
type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service

	Service Service `yaml:"-"`

	Log   *Log  `yaml:"log"`
	Debug Debug `yaml:"debug"`

	HTTP HTTP `yaml:"http"`

	GRPCClientTLS *shared.GRPCClientTLS `yaml:"grpc_client_tls"`
	GrpcClient    client.Client         `yaml:"-"`

	SigningKeys *SigningKeys `yaml:"signing_keys"`

	TokenManager *TokenManager `yaml:"token_manager"`

	Context context.Context `yaml:"-"`
}

// SigningKeys is a store configuration.
type SigningKeys struct {
	Store        string        `yaml:"store" env:"OC_CACHE_STORE;OCS_PRESIGNEDURL_SIGNING_KEYS_STORE" desc:"The type of the signing key store. Supported values are: 'redis-sentinel' and 'nats-js-kv'. See the text description for details." introductionVersion:"1.0.0"`
	Nodes        []string      `yaml:"addresses" env:"OC_CACHE_STORE_NODES;OCS_PRESIGNEDURL_SIGNING_KEYS_STORE_NODES" desc:"A list of nodes to access the configured store. Note that the behaviour how nodes are used is dependent on the library of the configured store. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	TTL          time.Duration `yaml:"ttl" env:"OC_CACHE_TTL;OCS_PRESIGNEDURL_SIGNING_KEYS_STORE_TTL" desc:"Default time to live for signing keys. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	AuthUsername string        `yaml:"username" env:"OC_CACHE_AUTH_USERNAME;OCS_PRESIGNEDURL_SIGNING_KEYS_STORE_AUTH_USERNAME" desc:"The username to authenticate with the store. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"1.0.0"`
	AuthPassword string        `yaml:"password" env:"OC_CACHE_AUTH_PASSWORD;OCS_PRESIGNEDURL_SIGNING_KEYS_STORE_AUTH_PASSWORD" desc:"The password to authenticate with the store. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"1.0.0"`
}
