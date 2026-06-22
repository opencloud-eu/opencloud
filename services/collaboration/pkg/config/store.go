package config

import "time"

// Store configures the store to use
type Store struct {
	Store                string        `yaml:"store" env:"OC_PERSISTENT_STORE;COLLABORATION_STORE" desc:"The type of the store. Supported values are: 'memory', 'nats-js-kv', 'redis-sentinel', 'noop'. See the text description for details." introductionVersion:"1.0.0"`
	Nodes                []string      `yaml:"nodes" env:"OC_PERSISTENT_STORE_NODES;COLLABORATION_STORE_NODES" desc:"A list of nodes to access the configured store. This has no effect when 'memory' store is configured. Note that the behaviour how nodes are used is dependent on the library of the configured store. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	Database             string        `yaml:"database" env:"COLLABORATION_STORE_DATABASE" desc:"The database name the configured store should use." introductionVersion:"1.0.0"`
	Table                string        `yaml:"table" env:"COLLABORATION_STORE_TABLE" desc:"The database table the store should use." introductionVersion:"1.0.0"`
	TTL                  time.Duration `yaml:"ttl" env:"OC_PERSISTENT_STORE_TTL;COLLABORATION_STORE_TTL" desc:"Time to live for events in the store. Defaults to '30m' (30 minutes). See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	AuthUsername         string        `yaml:"username" env:"OC_PERSISTENT_STORE_AUTH_USERNAME;COLLABORATION_STORE_AUTH_USERNAME" desc:"The username to authenticate with the store. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"1.0.0"`
	AuthPassword         string        `yaml:"password" env:"OC_PERSISTENT_STORE_AUTH_PASSWORD;COLLABORATION_STORE_AUTH_PASSWORD" desc:"The password to authenticate with the store. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"1.0.0"`
	EnableTLS            bool          `yaml:"enable_tls" env:"OC_PERSISTENT_STORE_ENABLE_TLS;COLLABORATION_STORE_ENABLE_TLS" desc:"Enable TLS for the connection to the store. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"%%NEXT%%"`
	TLSInsecure          bool          `yaml:"tls_insecure" env:"OC_INSECURE;OC_PERSISTENT_STORE_TLS_INSECURE;COLLABORATION_STORE_TLS_INSECURE" desc:"Whether to verify the server TLS certificates." introductionVersion:"%%NEXT%%"`
	TLSRootCACertificate string        `yaml:"tls_root_ca_certificate" env:"OC_PERSISTENT_STORE_TLS_ROOT_CA_CERTIFICATE;COLLABORATION_STORE_TLS_ROOT_CA_CERTIFICATE" desc:"The root CA certificate used to validate the server's TLS certificate. If provided COLLABORATION_STORE_TLS_INSECURE will be seen as false." introductionVersion:"%%NEXT%%"`
}
