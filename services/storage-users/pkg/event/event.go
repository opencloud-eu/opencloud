package event

import (
	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/config"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"
	"go-micro.dev/v4/events"
)

// NewStream prepares the requested nats stream and returns it.
func NewStream(cfg *config.Config) (events.Stream, error) {
	connName := generators.GenerateConnectionName(cfg.Service.Name, generators.NTypeBus)
	return stream.NatsFromConfig(connName, false, stream.NatsConfig{
		Endpoint:             cfg.Events.Addr,
		Cluster:              cfg.Events.ClusterID,
		EnableTLS:            cfg.Events.EnableTLS,
		TLSInsecure:          cfg.Events.TLSInsecure,
		TLSRootCACertificate: cfg.Events.TLSRootCaCertPath,
		AuthUsername:         cfg.Events.AuthUsername,
		AuthPassword:         cfg.Events.AuthPassword,
	})
}
