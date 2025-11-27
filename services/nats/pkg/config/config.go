package config

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// Config combines all available configuration parts.
type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service
	Service Service         `yaml:"-"`
	Log     *Log            `yaml:"log"`
	Debug   Debug           `yaml:"debug"`

	Nats Nats `ociConfig:"nats"`

	Context context.Context `yaml:"-"`
}

// Nats is the nats config
type Nats struct {
	Host                    string `yaml:"host" env:"NATS_NATS_HOST" desc:"Bind address." introductionVersion:"1.0.0"`
	Port                    int    `yaml:"port" env:"NATS_NATS_PORT" desc:"Bind port." introductionVersion:"1.0.0"`
	ClusterID               string `yaml:"clusterid" env:"NATS_NATS_CLUSTER_ID" desc:"ID of the NATS cluster." introductionVersion:"1.0.0"`
	StoreDir                string `yaml:"store_dir" env:"NATS_NATS_STORE_DIR" desc:"The directory where the filesystem storage will store NATS JetStream data. If not defined, the root directory derives from $OC_BASE_DATA_PATH/nats." introductionVersion:"1.0.0"`
	TLSCert                 string `yaml:"tls_cert" env:"NATS_TLS_CERT" desc:"Path/File name of the TLS server certificate (in PEM format) for the NATS listener. If not defined, the root directory derives from $OC_BASE_DATA_PATH/nats." introductionVersion:"1.0.0"`
	TLSKey                  string `yaml:"tls_key" env:"NATS_TLS_KEY" desc:"Path/File name for the TLS certificate key (in PEM format) for the NATS listener. If not defined, the root directory derives from $OC_BASE_DATA_PATH/nats." introductionVersion:"1.0.0"`
	TLSSkipVerifyClientCert bool   `yaml:"tls_skip_verify_client_cert" env:"OC_INSECURE;NATS_TLS_SKIP_VERIFY_CLIENT_CERT" desc:"Whether the NATS server should skip the client certificate verification during the TLS handshake." introductionVersion:"1.0.0"`
	EnableTLS               bool   `yaml:"enable_tls" env:"OC_EVENTS_ENABLE_TLS;NATS_EVENTS_ENABLE_TLS" desc:"Enable TLS for the connection to the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
}
