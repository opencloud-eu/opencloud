package audit

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/opencloud-eu/reva/v3/pkg/events/raw"
)

// NewNatsEmitter connects to NATS and returns an emitter that publishes to
// the audit stream. It creates the stream with defaults when it is missing,
// so events published before the audit service started are kept. It never
// changes an existing stream, that is the job of the audit service.
func NewNatsEmitter(ctx context.Context, connName string, cfg raw.Config, opts Options) (*NatsEmitter, error) {
	js, err := raw.JetStream(ctx, connName, cfg)
	if err != nil {
		return nil, err
	}
	if err := createStreamIfMissing(ctx, js); err != nil {
		return nil, err
	}
	return NewEmitter(JetStreamPublisher{JS: js}, opts), nil
}

func createStreamIfMissing(ctx context.Context, js jetstream.JetStream) error {
	if _, err := js.Stream(ctx, StreamName); err == nil {
		return nil
	} else if !errors.Is(err, jetstream.ErrStreamNotFound) {
		return err
	}
	// another service may create it at the same time, that is fine
	_, err := js.CreateStream(ctx, streamConfig(StreamConfig{}))
	if errors.Is(err, jetstream.ErrStreamNameAlreadyInUse) {
		return nil
	}
	return err
}

// EmitterConfig is the audit part of a reva service config.
type EmitterConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Mode    string `mapstructure:"mode"`
	Service string `mapstructure:"service"`
	// ServiceVersion is the version of the service, e.g. of OpenCloud.
	ServiceVersion string `mapstructure:"service_version"`
	// NATS connection, same keys as the events config: address, clusterID, ...
	Nats raw.Config `mapstructure:",squash"`
}

// ErrNoEndpoint is returned when audit is enabled without a NATS endpoint.
var ErrNoEndpoint = errors.New("audit is enabled but no events endpoint is configured")

// EmitterFromConfig returns an emitter for the config, one that drops
// everything when audit is off. Audit without an endpoint or with an unknown
// mode is an error, not a silent discard.
func EmitterFromConfig(ctx context.Context, connName string, cfg EmitterConfig, opts Options) (Emitter, error) {
	if !cfg.Enabled {
		return Discard{}, nil
	}
	if cfg.Nats.Endpoint == "" {
		return nil, ErrNoEndpoint
	}
	if m := Mode(cfg.Mode); m != "" && m != ModeBestEffort {
		return nil, fmt.Errorf("%w: %q", ErrUnknownMode, cfg.Mode)
	}
	opts.Service.Name = cfg.Service
	opts.Service.Version = cfg.ServiceVersion
	return NewNatsEmitter(ctx, connName, cfg.Nats, opts)
}
