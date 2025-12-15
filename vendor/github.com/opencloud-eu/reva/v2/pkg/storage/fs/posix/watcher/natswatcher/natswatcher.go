package natswatcher

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/options"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/watcher"
	"github.com/rs/zerolog"
	"github.com/vmihailenco/msgpack/v5"
)

// natsEvent represents the event encoded in MessagePack.
// we abbreviate the the properties to save some space
type natsEvent struct {
	Event  string `msgpack:"e"`
	Path   string `msgpack:"p,omitempty"`
	ToPath string `msgpack:"t,omitempty"`
	IsDir  bool   `msgpack:"d,omitempty"`
}

// NatsWatcher consumes filesystem-style events from NATS JetStream.
type NatsWatcher struct {
	ctx       context.Context
	tree      Scannable
	log       *zerolog.Logger
	watchRoot string
	config    options.NatsWatcherConfig
}

type Scannable interface {
	Scan(path string, action watcher.EventAction, isDir bool) error
}

// NewNatsWatcher creates a new NATS watcher.
func New(ctx context.Context, tree Scannable, cfg options.NatsWatcherConfig, watchRoot string, log *zerolog.Logger) (*NatsWatcher, error) {
	return &NatsWatcher{
		ctx:       ctx,
		tree:      tree,
		log:       log,
		watchRoot: watchRoot,
		config:    cfg,
	}, nil
}

// Watch starts consuming events from a NATS JetStream subject
func (w *NatsWatcher) Watch(path string) {
	w.log.Info().Str("stream", w.config.Stream).Msg("starting NATS watcher with auto-reconnect")

	for {
		select {
		case <-w.ctx.Done():
			w.log.Debug().Msg("context cancelled, stopping NATS watcher")
			return
		default:
		}

		// Try to connect with exponential backoff
		nc, js, err := w.connectWithBackoff()
		if err != nil {
			w.log.Error().Err(err).Msg("failed to establish NATS connection after retries")
			time.Sleep(5 * time.Second)
			continue
		}

		if err := w.consume(js); err != nil {
			w.log.Error().Err(err).Msg("NATS consumer exited with error, reconnecting")
		}

		_ = nc.Drain()
		nc.Close()
		time.Sleep(2 * time.Second)
	}
}

// connectWithBackoff repeatedly attempts to connect to NATS JetStream with exponential backoff.
func (w *NatsWatcher) connectWithBackoff() (*nats.Conn, jetstream.JetStream, error) {
	var nc *nats.Conn
	var js jetstream.JetStream

	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 1 * time.Second
	b.MaxInterval = 30 * time.Second
	b.MaxElapsedTime = 0 // never stop

	connect := func() error {
		select {
		case <-w.ctx.Done():
			return backoff.Permanent(w.ctx.Err())
		default:
		}

		var err error
		nc, err = w.connect()
		if err != nil {
			w.log.Warn().Err(err).Msg("failed to connect to NATS, retrying")
			return err
		}

		js, err = jetstream.New(nc)
		if err != nil {
			nc.Close()
			w.log.Warn().Err(err).Msg("failed to create jetstream context, retrying")
			return err
		}

		w.log.Info().Str("endpoint", w.config.Endpoint).Msg("connected to NATS JetStream")
		return nil
	}

	if err := backoff.Retry(connect, backoff.WithContext(b, w.ctx)); err != nil {
		return nil, nil, err
	}
	return nc, js, nil
}

// consume subscribes to JetStream and handles messages.
func (w *NatsWatcher) consume(js jetstream.JetStream) error {
	stream, err := js.Stream(w.ctx, w.config.Stream)
	if err != nil {
		return fmt.Errorf("failed to get stream: %w", err)
	}

	consumer, err := stream.CreateOrUpdateConsumer(w.ctx, jetstream.ConsumerConfig{
		Durable:       w.config.Durable,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxAckPending: w.config.MaxAckPending,
		AckWait:       w.config.AckWait,
	})
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}
	w.log.Info().
		Str("stream", w.config.Stream).
		Msg("started consuming from JetStream")

	_, err = consumer.Consume(func(msg jetstream.Msg) {
		defer func() {
			if ackErr := msg.Ack(); ackErr != nil {
				w.log.Warn().Err(ackErr).Msg("failed to ack message")
			}
		}()

		var ev natsEvent
		if err := msgpack.Unmarshal(msg.Data(), &ev); err != nil {
			w.log.Error().Err(err).Msg("failed to decode MessagePack event")
			return
		}

		w.handleEvent(ev)
	})

	if err != nil {
		return fmt.Errorf("consumer error: %w", err)
	}

	<-w.ctx.Done()
	return w.ctx.Err()
}

// connect establishes a single NATS connection with optional TLS and auth.
func (w *NatsWatcher) connect() (*nats.Conn, error) {
	var tlsConf *tls.Config
	if w.config.EnableTLS {
		var rootCAPool *x509.CertPool
		if w.config.TLSRootCACertificate != "" {
			rootCrtFile, err := os.ReadFile(w.config.TLSRootCACertificate)
			if err != nil {
				return nil, fmt.Errorf("failed to read root CA: %w", err)
			}
			rootCAPool = x509.NewCertPool()
			rootCAPool.AppendCertsFromPEM(rootCrtFile)
			w.config.TLSInsecure = false
		}
		tlsConf = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: w.config.TLSInsecure,
			RootCAs:            rootCAPool,
		}
	}

	opts := []nats.Option{nats.Name("opencloud-posixfs-natswatcher")}
	if tlsConf != nil {
		opts = append(opts, nats.Secure(tlsConf))
	}
	if w.config.AuthUsername != "" && w.config.AuthPassword != "" {
		opts = append(opts, nats.UserInfo(w.config.AuthUsername, w.config.AuthPassword))
	}
	return nats.Connect(w.config.Endpoint, opts...)
}

// handleEvent applies the event to the local tree.
func (w *NatsWatcher) handleEvent(ev natsEvent) {
	var err error

	// Determine the relevant path
	path := filepath.Join(w.watchRoot, ev.Path)

	switch ev.Event {
	case "CREATE":
		err = w.tree.Scan(path, watcher.ActionCreate, ev.IsDir)
	case "MOVED_TO":
		err = w.tree.Scan(path, watcher.ActionMove, ev.IsDir)
	case "MOVE_FROM":
		err = w.tree.Scan(path, watcher.ActionMoveFrom, ev.IsDir)
	case "MOVE": // support event with source and target path
		err = w.tree.Scan(path, watcher.ActionMoveFrom, ev.IsDir)
		if err == nil {
			w.log.Error().Err(err).Interface("event", ev).Msg("error processing event")
		}
		tgt := filepath.Join(w.watchRoot, ev.ToPath)
		if tgt == "" {
			w.log.Warn().Interface("event", ev).Msg("MOVE event missing target path")
		} else {
			err = w.tree.Scan(tgt, watcher.ActionMove, ev.IsDir)
		}
	case "CLOSE_WRITE":
		err = w.tree.Scan(path, watcher.ActionUpdate, ev.IsDir)
	case "DELETE":
		err = w.tree.Scan(path, watcher.ActionDelete, ev.IsDir)
	default:
		w.log.Warn().Str("event", ev.Event).Msg("unhandled event type")
	}

	if err != nil {
		w.log.Error().Err(err).Interface("event", ev).Msg("error processing event")
	}
}
