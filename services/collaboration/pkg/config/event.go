package config

// Events combines the configuration options for the event bus.
type Events struct {
	Endpoint             string `yaml:"endpoint" env:"OC_EVENTS_ENDPOINT;COLLABORATION_EVENTS_ENDPOINT" desc:"The address of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture." introductionVersion:"%%NEXT%%"`
	Cluster              string `yaml:"cluster" env:"OC_EVENTS_CLUSTER;COLLABORATION_EVENTS_CLUSTER" desc:"The clusterID of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture. Mandatory when using NATS as event system." introductionVersion:"%%NEXT%%"`
	TLSInsecure          bool   `yaml:"tls_insecure" env:"OC_INSECURE;OC_EVENTS_TLS_INSECURE;COLLABORATION_EVENTS_TLS_INSECURE" desc:"Whether to verify the server TLS certificates." introductionVersion:"%%NEXT%%"`
	TLSRootCACertificate string `yaml:"tls_root_ca_certificate" env:"OC_EVENTS_TLS_ROOT_CA_CERTIFICATE;COLLABORATION_EVENTS_TLS_ROOT_CA_CERTIFICATE" desc:"The root CA certificate used to validate the server's TLS certificate. If provided COLLABORATION_EVENTS_TLS_INSECURE will be seen as false." introductionVersion:"%%NEXT%%"`
	EnableTLS            bool   `yaml:"enable_tls" env:"OC_EVENTS_ENABLE_TLS;COLLABORATION_EVENTS_ENABLE_TLS" desc:"Enable TLS for the connection to the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"%%NEXT%%"`
	AuthUsername         string `yaml:"username" env:"OC_EVENTS_AUTH_USERNAME;COLLABORATION_EVENTS_AUTH_USERNAME" desc:"The username to authenticate with the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"%%NEXT%%"`
	AuthPassword         string `yaml:"password" env:"OC_EVENTS_AUTH_PASSWORD;COLLABORATION_EVENTS_AUTH_PASSWORD" desc:"The password to authenticate with the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"%%NEXT%%"`
}
