package nats

import (
	"crypto/tls"

	nserver "github.com/nats-io/nats-server/v2/server"
)

// NatsOption configures the nats server
type NatsOption func(*nserver.Options)

// Host sets the host URL for the nats server
func Host(url string) NatsOption {
	return func(o *nserver.Options) {
		o.Host = url
	}
}

// Port sets the host URL for the nats server
func Port(port int) NatsOption {
	return func(o *nserver.Options) {
		o.Port = port
	}
}

// ClusterID sets the name for the nats cluster
func ClusterID(clusterID string) NatsOption {
	return func(o *nserver.Options) {
		o.Cluster.Name = clusterID
	}
}

// StoreDir sets the folder for persistence
func StoreDir(StoreDir string) NatsOption {
	return func(o *nserver.Options) {
		o.StoreDir = StoreDir
	}
}

// TLSConfig sets the tls config for the nats server
func TLSConfig(c *tls.Config) NatsOption {
	return func(o *nserver.Options) {
		o.TLSConfig = c
	}
}

// AllowNonTLS sets the allow non tls options for the nats server
func AllowNonTLS(v bool) NatsOption {
	return func(o *nserver.Options) {
		o.AllowNonTLS = v
	}
}

// enables NATS server monitoring through its HTTP endpoint
func EnableMonitoringHTTPEndpoint(httpHost string, httpPort int) NatsOption {
	return func(o *nserver.Options) {
		o.HTTPHost = httpHost
		o.HTTPPort = httpPort
	}
}

// enables NATS server monitoring through its HTTPS endpoint
func EnableMonitoringHTTPSEndpoint(httpHost string, httpsPort int) NatsOption {
	return func(o *nserver.Options) {
		o.HTTPHost = httpHost
		o.HTTPSPort = httpsPort
	}
}
