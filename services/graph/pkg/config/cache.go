package config

import "time"

// Cache defines the available configuration for a cache store
type Cache struct {
	Store                string        `yaml:"store" env:"OC_CACHE_STORE;GRAPH_CACHE_STORE" desc:"The type of the cache store. Supported values are: 'memory', 'redis-sentinel', 'nats-js-kv', 'noop'. See the text description for details." introductionVersion:"1.0.0"`
	Nodes                []string      `yaml:"nodes" env:"OC_CACHE_STORE_NODES;GRAPH_CACHE_STORE_NODES" desc:"A list of nodes to access the configured store. This has no effect when 'memory' store are configured. Note that the behaviour how nodes are used is dependent on the library of the configured store. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	Database             string        `yaml:"database" env:"GRAPH_CACHE_STORE_DATABASE" desc:"The database name the configured store should use." introductionVersion:"1.0.0"`
	Table                string        `yaml:"table" env:"GRAPH_CACHE_STORE_TABLE" desc:"The database table the store should use." introductionVersion:"1.0.0"`
	TTL                  time.Duration `yaml:"ttl" env:"OC_CACHE_TTL;GRAPH_CACHE_TTL" desc:"Time to live for cache records in the graph. Defaults to '336h' (2 weeks). See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	DisablePersistence   bool          `yaml:"disable_persistence" env:"OC_CACHE_DISABLE_PERSISTENCE;GRAPH_CACHE_DISABLE_PERSISTENCE" desc:"Disables persistence of the cache. Only applies when store type 'nats-js-kv' is configured. Defaults to false." introductionVersion:"1.0.0"`
	AuthUsername         string        `yaml:"username" env:"OC_CACHE_AUTH_USERNAME;GRAPH_CACHE_AUTH_USERNAME" desc:"The username to authenticate with the cache. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"1.0.0"`
	AuthPassword         string        `yaml:"password" env:"OC_CACHE_AUTH_PASSWORD;GRAPH_CACHE_AUTH_PASSWORD" desc:"The password to authenticate with the cache. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"1.0.0"`
	EnableTLS            bool          `yaml:"enable_tls" env:"OC_CACHE_ENABLE_TLS;GRAPH_CACHE_ENABLE_TLS" desc:"Enable TLS for the connection to file metadata cache." introductionVersion:"%%NEXT%%"`
	TLSInsecure          bool          `yaml:"tls_insecure" env:"OC_INSECURE;OC_CACHE_TLS_INSECURE;GRAPH_CACHE_TLS_INSECURE" desc:"Whether to verify the server TLS certificates." introductionVersion:"%%NEXT%%"`
	TLSRootCACertificate string        `yaml:"tls_root_ca_certificate" env:"OC_CACHE_TLS_ROOT_CA_CERTIFICATE;GRAPH_CACHE_TLS_ROOT_CA_CERTIFICATE" desc:"The root CA certificate used to validate the server's TLS certificate. If provided GRAPH_CACHE_TLS_INSECURE will be seen as false." introductionVersion:"%%NEXT%%"`
}
