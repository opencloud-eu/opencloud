package audit

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// StreamName is the JetStream stream that buffers audit events.
const StreamName = "audit"

// SubjectPrefix of all audit subjects: audit.<tenant>.<category>.
const SubjectPrefix = "audit"

// noTenant replaces an empty tenant in the subject.
const noTenant = "_"

// Subject returns the subject for a tenant and a category.
func Subject(tenant, category string) string {
	return SubjectPrefix + "." + token(tenant) + "." + token(category)
}

// token makes a value safe as one subject token: no dots, wildcards or spaces.
func token(s string) string {
	if s == "" {
		return noTenant
	}
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, s)
}

// StreamConfig configures the audit stream.
type StreamConfig struct {
	// Replicas in clustered NATS, default 1.
	Replicas int
	// MaxAge is how long events stay in the buffer, default 7 days.
	MaxAge time.Duration
	// DuplicateWindow for Nats-Msg-Id deduplication, default 2 minutes.
	DuplicateWindow time.Duration
}

// EnsureStream creates the audit stream or updates its configuration.
func EnsureStream(ctx context.Context, js jetstream.JetStream, cfg StreamConfig) (jetstream.Stream, error) {
	sc := streamConfig(cfg)
	s, err := js.CreateStream(ctx, sc)
	if errors.Is(err, jetstream.ErrStreamNameAlreadyInUse) {
		return js.UpdateStream(ctx, sc)
	}
	return s, err
}

func streamConfig(cfg StreamConfig) jetstream.StreamConfig {
	if cfg.Replicas <= 0 {
		cfg.Replicas = 1
	}
	if cfg.MaxAge <= 0 {
		cfg.MaxAge = 7 * 24 * time.Hour
	}
	if cfg.DuplicateWindow <= 0 {
		cfg.DuplicateWindow = 2 * time.Minute
	}
	return jetstream.StreamConfig{
		Name:        StreamName,
		Description: "OpenCloud audit events, a buffer between the services and the audit service",
		Subjects:    []string{SubjectPrefix + ".>"},
		Storage:     jetstream.FileStorage,
		Retention:   jetstream.LimitsPolicy,
		Replicas:    cfg.Replicas,
		MaxAge:      cfg.MaxAge,
		Duplicates:  cfg.DuplicateWindow,
	}
}

// JetStreamPublisher publishes to JetStream and waits for the ack.
type JetStreamPublisher struct {
	JS jetstream.JetStream
}

// Publish stores the message in the audit stream.
func (p JetStreamPublisher) Publish(ctx context.Context, msg *nats.Msg) error {
	_, err := p.JS.PublishMsg(ctx, msg)
	return err
}
