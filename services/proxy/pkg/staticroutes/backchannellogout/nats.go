package backchannellogout

import (
	"context"
	"errors"
	"time"

	"github.com/nats-io/nats.go"
	"go-micro.dev/v4/store"
)

type natsCacheStore struct {
	store.Store
	connect func() (*nats.Conn, error)
}

// WithNATSCleanup adds delete-marker maintenance to a NATS KV store. The
// connection factory must use the same endpoints, authentication, and TLS
// settings as the underlying store. Each maintenance connection is closed.
func WithNATSCleanup(underlying store.Store, connect func() (*nats.Conn, error)) store.Store {
	return &natsCacheStore{Store: underlying, connect: connect}
}

// PurgeDeleted removes tombstones from this store's table. They otherwise remain
// forever in the OIDC bucket, which deliberately has no bucket-wide TTL.
func (s *natsCacheStore) PurgeDeleted(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	conn, err := s.connect()
	if err != nil {
		return err
	}
	defer conn.Close()
	js, err := conn.JetStream(nats.Context(ctx))
	if err != nil {
		return err
	}
	return s.purgeDeleted(ctx, js)
}

func (s *natsCacheStore) purgeDeleted(ctx context.Context, js nats.JetStreamContext) error {
	// Use the adapter's decoder so cleanup respects its key encoding and table
	// namespace without duplicating the encoding implementation.
	keys, ok := s.Store.(interface {
		MicroKeyFilter(table, natsKey, prefix, suffix string) (string, bool)
	})
	if !ok {
		return errors.New("NATS cache store does not expose its key decoder")
	}
	opts := s.Options()
	bucket, err := js.KeyValue(opts.Database)
	if err != nil {
		return err
	}
	watcher, err := bucket.WatchAll(nats.MetaOnly(), nats.Context(ctx))
	if err != nil {
		return err
	}
	defer watcher.Stop()
	var markers []nats.KeyValueEntry
snapshot:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case entry, open := <-watcher.Updates():
			if !open {
				return errors.New("NATS cache marker snapshot ended before completion")
			}
			if entry == nil {
				break snapshot
			}
			if entry.Operation() != nats.KeyValueDelete && entry.Operation() != nats.KeyValuePurge {
				continue
			}
			if _, matches := keys.MicroKeyFilter(opts.Table, entry.Key(), "", ""); matches {
				markers = append(markers, entry)
			}
		}
	}
	_ = watcher.Stop()
	for _, marker := range markers {
		if err := ctx.Err(); err != nil {
			return err
		}
		// Bound the purge by the observed revision. A new value written after
		// the snapshot must survive, even if it reuses a deleted token's key.
		if err := js.PurgeStream("KV_"+opts.Database, &nats.StreamPurgeRequest{
			Subject: "$KV." + opts.Database + "." + marker.Key(), Sequence: marker.Revision() + 1,
		}); err != nil {
			return err
		}
	}
	return nil
}
