package config

import "time"

// Events combines the configuration options for the event bus.
type Events struct {
	Disabled         bool   `yaml:"disabled" env:"SEARCH_EVENTS_DISABLED" desc:"Disables listening for events. Set this to true if the service should only handle GRPC requests." introductionVersion:"4.0.0"`
	Endpoint         string `yaml:"endpoint" env:"OC_EVENTS_ENDPOINT;SEARCH_EVENTS_ENDPOINT" desc:"The address of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture." introductionVersion:"1.0.0"`
	Cluster          string `yaml:"cluster" env:"OC_EVENTS_CLUSTER;SEARCH_EVENTS_CLUSTER" desc:"The clusterID of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture. Mandatory when using NATS as event system." introductionVersion:"1.0.0"`
	AsyncUploads     bool   `yaml:"async_uploads" env:"OC_ASYNC_UPLOADS;SEARCH_EVENTS_ASYNC_UPLOADS" desc:"Enable asynchronous file uploads." introductionVersion:"1.0.0"`
	NumConsumers     int    `yaml:"num_consumers" env:"SEARCH_EVENTS_NUM_CONSUMERS" desc:"The amount of concurrent event consumers to start. Event consumers are used for searching files. Multiple consumers increase parallelisation, but will also increase CPU and memory demands." introductionVersion:"1.0.0"`
	DebounceDuration int    `yaml:"debounce_duration" env:"SEARCH_EVENTS_REINDEX_DEBOUNCE_DURATION" desc:"The duration in milliseconds the reindex debouncer waits before triggering a reindex of a space that was modified." introductionVersion:"1.0.0"`

	TLSInsecure          bool   `yaml:"tls_insecure" env:"OC_INSECURE;OC_EVENTS_TLS_INSECURE;SEARCH_EVENTS_TLS_INSECURE" desc:"Whether to verify the server TLS certificates." introductionVersion:"1.0.0"`
	TLSRootCACertificate string `yaml:"tls_root_ca_certificate" env:"OC_EVENTS_TLS_ROOT_CA_CERTIFICATE;SEARCH_EVENTS_TLS_ROOT_CA_CERTIFICATE" desc:"The root CA certificate used to validate the server's TLS certificate. If provided SEARCH_EVENTS_TLS_INSECURE will be seen as false." introductionVersion:"1.0.0"`
	EnableTLS            bool   `yaml:"enable_tls" env:"OC_EVENTS_ENABLE_TLS;SEARCH_EVENTS_ENABLE_TLS" desc:"Enable TLS for the connection to the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
	AuthUsername         string `yaml:"username" env:"OC_EVENTS_AUTH_USERNAME;SEARCH_EVENTS_AUTH_USERNAME" desc:"The username to authenticate with the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
	AuthPassword         string `yaml:"password" env:"OC_EVENTS_AUTH_PASSWORD;SEARCH_EVENTS_AUTH_PASSWORD" desc:"The password to authenticate with the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`

	MaxAckPending int           `yaml:"max_ack_pending" env:"SEARCH_EVENTS_MAX_ACK_PENDING" desc:"The maximum number of unacknowledged messages. This is used to limit the number of messages that can be in flight at the same time." introductionVersion:"4.0.0"`
	AckWait       time.Duration `yaml:"ack_wait" env:"SEARCH_EVENTS_ACK_WAIT" desc:"The time to wait for an ack before the message is redelivered. This is used to ensure that messages are not lost if the consumer crashes." introductionVersion:"4.0.0"`
}
