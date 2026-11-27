package config

import (
	"context"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// Config combines all available configuration parts.
type Config struct {
	Commons  *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service
	Service  Service         `yaml:"-"`
	LogLevel string          `yaml:"loglevel" env:"OC_LOG_LEVEL;AUDIT_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"1.0.0"`
	Debug    Debug           `yaml:"debug"`

	Events Events `yaml:"events"`
	Stream Stream `yaml:"stream"`
	Output Output `yaml:"output"`

	// Auditlog configures the legacy audit log, which runs while Stream.Enabled is false.
	Auditlog Auditlog `yaml:"auditlog"`

	Context context.Context `yaml:"-"`
}

// Events combines the configuration options for the event bus.
type Events struct {
	Endpoint             string `yaml:"endpoint" env:"OC_EVENTS_ENDPOINT;AUDIT_EVENTS_ENDPOINT" desc:"The address of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture." introductionVersion:"1.0.0"`
	Cluster              string `yaml:"cluster" env:"OC_EVENTS_CLUSTER;AUDIT_EVENTS_CLUSTER" desc:"The clusterID of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture. Mandatory when using NATS as event system." introductionVersion:"1.0.0"`
	TLSInsecure          bool   `yaml:"tls_insecure" env:"OC_INSECURE;OC_EVENTS_TLS_INSECURE;AUDIT_EVENTS_TLS_INSECURE" desc:"Whether to verify the server TLS certificates." introductionVersion:"1.0.0"`
	TLSRootCACertificate string `yaml:"tls_root_ca_certificate" env:"OC_EVENTS_TLS_ROOT_CA_CERTIFICATE;AUDIT_EVENTS_TLS_ROOT_CA_CERTIFICATE" desc:"The root CA certificate used to validate the server's TLS certificate. If provided AUDIT_EVENTS_TLS_INSECURE will be seen as false." introductionVersion:"1.0.0"`
	EnableTLS            bool   `yaml:"enable_tls" env:"OC_EVENTS_ENABLE_TLS;AUDIT_EVENTS_ENABLE_TLS" desc:"Enable TLS for the connection to the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
	AuthUsername         string `yaml:"username" env:"OC_EVENTS_AUTH_USERNAME;AUDIT_EVENTS_AUTH_USERNAME" desc:"The username to authenticate with the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
	AuthPassword         string `yaml:"password" env:"OC_EVENTS_AUTH_PASSWORD;AUDIT_EVENTS_AUTH_PASSWORD" desc:"The password to authenticate with the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
}

// Stream configures the JetStream stream "audit", the buffer between the services and the audit service.
type Stream struct {
	// Enabled is read from OC_AUDIT_ENABLED like in the services that emit the audit events, so that
	// one variable switches the whole installation, and existing installations keep the legacy audit
	// log until they turn the new one on.
	Enabled  bool          `yaml:"enabled" env:"OC_AUDIT_ENABLED;AUDIT_STREAM_ENABLED" desc:"Write the audit records of the audit stream. OC_AUDIT_ENABLED also makes the services emit them. If false, the deprecated legacy audit log runs instead, with the AUDIT_LOG_TO_CONSOLE, AUDIT_LOG_TO_FILE, AUDIT_FILEPATH and AUDIT_FORMAT settings." introductionVersion:"%%NEXT%%"`
	Replicas int           `yaml:"replicas" env:"AUDIT_STREAM_REPLICAS" desc:"Number of replicas of the audit stream in a clustered NATS. Use 3 in production clusters." introductionVersion:"%%NEXT%%"`
	MaxAge   time.Duration `yaml:"max_age" env:"AUDIT_STREAM_MAX_AGE" desc:"How long audit events stay in the stream. The stream is a buffer for outages of the audit service, not the archive. See the Environment Variable Types description for more details." introductionVersion:"%%NEXT%%"`
}

// Output configures where the audit service writes the audit records.
type Output struct {
	Stdout bool `yaml:"stdout" env:"AUDIT_OUTPUT_STDOUT" desc:"Write the audit records to stdout, one JSON record per line." introductionVersion:"%%NEXT%%"`
	OTLP   OTLP `yaml:"otlp"`
}

// OTLP configures the output to an OpenTelemetry Collector.
type OTLP struct {
	Endpoint string `yaml:"endpoint" env:"AUDIT_OUTPUT_OTLP_ENDPOINT" desc:"Address of an OpenTelemetry Collector to send the audit records to as OTLP logs over gRPC, e.g. 'otel-collector:4317'. Empty disables the output." introductionVersion:"%%NEXT%%"`
	Insecure bool   `yaml:"insecure" env:"AUDIT_OUTPUT_OTLP_INSECURE" desc:"Connect to the OpenTelemetry Collector without TLS." introductionVersion:"%%NEXT%%"`
}

// Auditlog holds audit log information, only used by the legacy audit log.
type Auditlog struct {
	LogToConsole bool   `yaml:"log_to_console" env:"AUDIT_LOG_TO_CONSOLE" desc:"Logs to stdout if set to 'true'. Independent of the LOG_TO_FILE option. Only used by the legacy audit log." introductionVersion:"1.0.0" deprecationVersion:"%%NEXT%%" removalVersion:"%%NEXT_PRODUCTION_VERSION%%" deprecationInfo:"The legacy audit log is replaced by the audit stream, enable it with OC_AUDIT_ENABLED. Without it, the audit service will not start once the legacy audit log is removed." deprecationReplacement:"AUDIT_OUTPUT_STDOUT"`
	LogToFile    bool   `yaml:"log_to_file" env:"AUDIT_LOG_TO_FILE" desc:"Logs to file if set to 'true'. Independent of the LOG_TO_CONSOLE option. Only used by the legacy audit log." introductionVersion:"1.0.0" deprecationVersion:"%%NEXT%%" removalVersion:"%%NEXT_PRODUCTION_VERSION%%" deprecationInfo:"The legacy audit log is replaced by the audit stream, enable it with OC_AUDIT_ENABLED. Without it, the audit service will not start once the legacy audit log is removed. The audit stream writes to stdout or to an OpenTelemetry Collector, not to a file." deprecationReplacement:"AUDIT_OUTPUT_OTLP_ENDPOINT"`
	FilePath     string `yaml:"filepath" env:"AUDIT_FILEPATH" desc:"Filepath of the logfile. Mandatory if LOG_TO_FILE is set to 'true'. Only used by the legacy audit log." introductionVersion:"1.0.0" deprecationVersion:"%%NEXT%%" removalVersion:"%%NEXT_PRODUCTION_VERSION%%" deprecationInfo:"The legacy audit log is replaced by the audit stream, enable it with OC_AUDIT_ENABLED. Without it, the audit service will not start once the legacy audit log is removed. The audit stream writes to stdout or to an OpenTelemetry Collector, not to a file." deprecationReplacement:"AUDIT_OUTPUT_OTLP_ENDPOINT"`
	Format       string `yaml:"format" env:"AUDIT_FORMAT" desc:"Log format. Supported values are '' (empty) and 'json'. Using 'json' is advised, '' (empty) renders the 'minimal' format. See the text description for more details. Only used by the legacy audit log." introductionVersion:"1.0.0" deprecationVersion:"%%NEXT%%" removalVersion:"%%NEXT_PRODUCTION_VERSION%%" deprecationInfo:"The legacy audit log is replaced by the audit stream, enable it with OC_AUDIT_ENABLED. Without it, the audit service will not start once the legacy audit log is removed. The audit stream always writes JSON." deprecationReplacement:""`
}
