package config

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// Config combines all available configuration parts.
type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service
	Service Service         `yaml:"-"`
	Tracing *Tracing        `yaml:"tracing"`
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

// Tracing is the tracing config
type Tracing struct {
	Exporter  string `yaml:"exporter" env:"OC_TRACES_EXPORTER;NATS_TRACES_EXPORTER;OTEL_TRACES_EXPORTER" desc:"Tracing exporter to use. Supported values are 'none', 'console' and 'otlp'." introductionVersion:"1.0.0"`
	Enabled   bool   `yaml:"enabled" env:"OC_TRACING_ENABLED;NATS_TRACING_ENABLED" desc:"Deprecated: use exporter 'none' to disable tracing." introductionVersion:"1.0.0" deprecationVersion:"2.0.0" deprecationInfo:"Set OC_TRACES_EXPORTER/NATS_TRACES_EXPORTER (or OTEL_TRACES_EXPORTER) to 'none'."`
	Type      string `yaml:"type" env:"OC_TRACING_TYPE;NATS_TRACING_TYPE" desc:"Deprecated: legacy tracing type." introductionVersion:"1.0.0" deprecationVersion:"2.0.0" deprecationInfo:"Use OC_TRACES_EXPORTER or OTEL_TRACES_EXPORTER instead."`
	Endpoint  string `yaml:"endpoint" env:"OC_TRACING_ENDPOINT;NATS_TRACING_ENDPOINT" desc:"Deprecated: legacy tracing endpoint." introductionVersion:"1.0.0" deprecationVersion:"2.0.0" deprecationInfo:"Configure OTEL_EXPORTER_OTLP_ENDPOINT instead."`
	Collector string `yaml:"collector" env:"OC_TRACING_COLLECTOR;NATS_TRACING_COLLECTOR" desc:"Deprecated: legacy tracing collector endpoint." introductionVersion:"1.0.0" deprecationVersion:"2.0.0" deprecationInfo:"Configure OTEL_EXPORTER_OTLP_ENDPOINT/OTEL_EXPORTER_OTLP_PROTOCOL instead."`
}
