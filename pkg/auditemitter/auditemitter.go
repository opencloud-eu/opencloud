// Package auditemitter builds the audit emitter of a service.
package auditemitter

import (
	"context"
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/opencloud-eu/reva/v3/pkg/audit"
	"github.com/opencloud-eu/reva/v3/pkg/events/raw"
)

// Settings are the audit settings of a service. The services keep their own
// config struct for the env vars, it converts to Settings.
type Settings struct {
	Enabled bool
	Mode    string
}

// New returns the audit emitter of a service. It connects to NATS with its own
// connection and names the service and the OpenCloud version in the records.
// With audit off it returns an emitter that drops everything. A missing events
// endpoint or an unknown mode is an error, the service must not start without
// the audit log it was configured for. Call Close on shutdown.
func New(ctx context.Context, service string, s Settings, nats raw.Config, logger log.Logger) (audit.Emitter, error) {
	connName := generators.GenerateConnectionName(service, generators.NTypeBus) + "-audit"
	e, err := audit.EmitterFromConfig(ctx, connName, audit.EmitterConfig{
		Enabled:        s.Enabled,
		Mode:           s.Mode,
		Service:        service,
		ServiceVersion: version.GetString(),
		Nats:           nats,
	}, audit.Options{Logger: &logger.Logger})
	if err != nil {
		return nil, fmt.Errorf("could not initialize the audit emitter: %w", err)
	}
	return e, nil
}

// Close flushes the records the emitter still holds. It does nothing for an
// emitter that holds none.
func Close(e audit.Emitter) {
	audit.CloseEmitter(e)
}
