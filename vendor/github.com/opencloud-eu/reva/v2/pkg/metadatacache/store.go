// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package metadatacache provides a generic in-memory write-through cache
// backed by a metadata.Storage.  Each key is protected by its own mutex so
// concurrent updates to different keys never contend.  Persistence uses
// etag-based Compare-And-Swap to detect cross-replica conflicts.
package metadatacache

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/metadata"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const tracerName = "metadatacache"

// entry holds the cached value and the etag last seen from storage.
type entry[V any] struct {
	value V
	etag  string
}

// Options configures a Store.
type Options[K comparable, V any] struct {
	// Storage is the metadata backend (required).
	Storage metadata.Storage

	// Path maps a key to a storage path (required).
	Path func(K) string

	// Retries is the maximum number of attempts in Update before giving up.
	// Defaults to 100 when zero.
	Retries int

	// Init returns a valid zero value for V.  Required when Update is called
	// with createIfNotFound=true (e.g. to produce an initialised map rather
	// than a nil map).
	Init func() V
}

// Store is a generic in-memory write-through cache.  V must be
// JSON-serialisable.  K must be comparable (map key) and must produce a
// human-readable string via fmt.Sprint for log/trace attributes.
type Store[K comparable, V any] struct {
	opts    Options[K, V]
	mu      sync.Map // K → *sync.Mutex  (per-key lock)
	entries sync.Map // K → *entry[V]
}

// New returns a ready-to-use Store.
func New[K comparable, V any](opts Options[K, V]) *Store[K, V] {
	if opts.Retries <= 0 {
		opts.Retries = 100
	}
	return &Store[K, V]{opts: opts}
}

// Lock acquires the per-key mutex and returns an unlock function.  The caller
// must call the returned function exactly once (typically via defer).
func (s *Store[K, V]) Lock(key K) func() {
	v, _ := s.mu.LoadOrStore(key, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// Get syncs from storage and returns the current value for key.
// The caller must hold the per-key lock.
// Returns (zero, false, nil) when the key does not exist in storage.
func (s *Store[K, V]) Get(ctx context.Context, key K) (V, bool, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Get")
	defer span.End()
	span.SetAttributes(attribute.String("key", fmt.Sprint(key)))

	if err := s.Sync(ctx, key); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		var zero V
		return zero, false, err
	}

	if e, ok := s.entries.Load(key); ok {
		span.SetStatus(codes.Ok, "")
		return e.(*entry[V]).value, true, nil
	}

	span.SetStatus(codes.Ok, "not found")
	var zero V
	return zero, false, nil
}

// Set stores value for key in the in-memory cache without persisting to
// storage.  The etag of the existing entry (if any) is preserved.
// The caller must hold the per-key lock.
func (s *Store[K, V]) Set(key K, v V) {
	var etag string
	if e, ok := s.entries.Load(key); ok {
		etag = e.(*entry[V]).etag
	}
	s.entries.Store(key, &entry[V]{value: v, etag: etag})
}

// Sync downloads the current state of key from storage and updates the cache.
// It is a no-op (returns nil) when the stored etag matches the cached etag (NotModified).
// If the key does not exist in storage, the cached entry is removed and nil is returned.
// The caller must hold the per-key lock.
func (s *Store[K, V]) Sync(ctx context.Context, key K) error {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Sync")
	defer span.End()
	span.SetAttributes(attribute.String("key", fmt.Sprint(key)))

	var ifNoneMatch []string
	if e, ok := s.entries.Load(key); ok {
		ifNoneMatch = []string{e.(*entry[V]).etag}
	}

	dlres, err := s.opts.Storage.Download(ctx, metadata.DownloadRequest{
		Path:        s.opts.Path(key),
		IfNoneMatch: ifNoneMatch,
	})
	switch err.(type) {
	case nil:
		// fall through to unmarshal
	case errtypes.NotFound:
		span.SetStatus(codes.Ok, "not found")
		s.entries.Delete(key)
		return nil
	case errtypes.NotModified:
		span.SetStatus(codes.Ok, "not modified")
		return nil
	default:
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	var v V
	if err := json.Unmarshal(dlres.Content, &v); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	s.entries.Store(key, &entry[V]{value: v, etag: dlres.Etag})
	span.SetStatus(codes.Ok, "")
	return nil
}

// Persist writes the cached value for key to storage using etag-based CAS.
// If the path's parent directory is non-trivial (not "." or "/"),
// MakeDirIfNotExist is called first.
// The caller must hold the per-key lock.
func (s *Store[K, V]) Persist(ctx context.Context, key K) error {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Persist")
	defer span.End()
	span.SetAttributes(attribute.String("key", fmt.Sprint(key)))

	e, ok := s.entries.Load(key)
	if !ok {
		span.SetStatus(codes.Ok, "nothing to persist")
		return nil
	}
	ent := e.(*entry[V])

	b, err := json.Marshal(ent.value)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	p := s.opts.Path(key)
	if dir := path.Dir(p); dir != "." && dir != "/" {
		if err := s.opts.Storage.MakeDirIfNotExist(ctx, dir); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	ur := metadata.UploadRequest{
		Path:        p,
		Content:     b,
		IfMatchEtag: ent.etag,
	}
	if ent.etag == "" {
		ur.IfNoneMatch = []string{"*"}
	}

	res, err := s.opts.Storage.Upload(ctx, ur)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// Update the etag so subsequent operations use the latest value.
	var newEtag string
	if res != nil {
		newEtag = res.Etag
	}
	s.entries.Store(key, &entry[V]{value: ent.value, etag: newEtag})
	span.SetStatus(codes.Ok, "")
	return nil
}

// Update atomically reads, transforms, and (conditionally) persists the value
// for key. It synchronizes the value from storage first and acquires the
// per-key lock internally; callers must not hold it.
//
// fn receives the current value and returns (newValue, shouldPersist, error).
// When shouldPersist is false the update is skipped entirely (useful for
// read-modify-skip-write patterns like GetAppPassword's utime throttle).
// When err is non-nil the update is abandoned immediately.
//
// createIfNotFound controls whether a missing key is initialised via
// Options.Init (which must be set) or returned as errtypes.NotFound.
//
// Conflicts (Aborted, PreconditionFailed, AlreadyExists, TooEarly) are
// retried up to Options.Retries times, re-syncing from storage before each
// retry.
func (s *Store[K, V]) Update(ctx context.Context, key K, createIfNotFound bool, fn func(V) (V, bool, error)) error {
	log := appctx.GetLogger(ctx).With().
		Str("hostname", os.Getenv("HOSTNAME")).
		Str("key", fmt.Sprint(key)).Logger()

	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Update")
	defer span.End()
	span.SetAttributes(attribute.String("key", fmt.Sprint(key)))

	unlock := s.Lock(key)
	defer unlock()

	// Always synchronize before updating, so changes from other processes are
	// included. Sync uses the cached etag for a conditional download.
	if err := s.Sync(ctx, key); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	var lastErr error
	for attempt := 0; attempt < s.opts.Retries; attempt++ {
		// Load the current value (or initialise if absent and allowed).
		var v V
		if e, ok := s.entries.Load(key); ok {
			v = e.(*entry[V]).value
		} else if createIfNotFound {
			v = s.opts.Init()
		} else {
			span.SetStatus(codes.Error, "not found")
			return errtypes.NotFound(fmt.Sprint(key))
		}

		newV, shouldPersist, err := fn(v)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}

		if !shouldPersist {
			span.SetStatus(codes.Ok, "persist skipped")
			return nil
		}

		s.Set(key, newV)

		lastErr = s.Persist(ctx, key)
		switch lastErr.(type) {
		case nil:
			span.SetAttributes(attribute.Int("attempts", attempt+1))
			span.SetStatus(codes.Ok, "")
			return nil
		case errtypes.Aborted:
			log.Debug().Int("attempt", attempt).Msg("metadatacache: persist aborted (etag mismatch), retrying")
		case errtypes.PreconditionFailed:
			log.Debug().Int("attempt", attempt).Msg("metadatacache: persist precondition failed, retrying")
		case errtypes.AlreadyExists:
			log.Debug().Int("attempt", attempt).Msg("metadatacache: persist already exists, retrying")
		case errtypes.TooEarly:
			log.Debug().Int("attempt", attempt).Msg("metadatacache: persist too early (processing lock), retrying")
		default:
			span.RecordError(lastErr)
			span.SetStatus(codes.Error, lastErr.Error())
			return lastErr
		}

		// Re-sync before the next attempt (skip after the last attempt).
		if attempt+1 < s.opts.Retries {
			if syncErr := s.Sync(ctx, key); syncErr != nil {
				span.RecordError(syncErr)
				span.SetStatus(codes.Error, syncErr.Error())
				return syncErr
			}
		}
	}

	span.RecordError(lastErr)
	span.SetStatus(codes.Error, fmt.Sprintf("gave up after %d attempts: %s", s.opts.Retries, lastErr))
	return fmt.Errorf("metadatacache: update of %v failed after %d attempts: %w", key, s.opts.Retries, lastErr)
}
