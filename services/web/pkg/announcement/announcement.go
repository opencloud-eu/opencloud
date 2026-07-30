// Package announcement persists and serves the web announcement banner.
package announcement

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/nats-io/nats.go/jetstream"
)

// _storeKey is the single key under which the announcement is persisted.
const _storeKey = "announcement"

// Announcement is a banner message shown above the top bar to all users.
type Announcement struct {
	// Enabled controls whether the announcement is live (injected into config.json).
	Enabled    bool   `json:"enabled"`
	BannerText string `json:"bannerText"`
	InfoText   string `json:"infoText"`
}

// Store persists a single announcement in a NATS JetStream key-value bucket.
type Store struct {
	kv jetstream.KeyValue
}

// NewStore returns a new announcement Store backed by the given key-value bucket.
func NewStore(kv jetstream.KeyValue) *Store {
	return &Store{kv: kv}
}

// Get returns the currently stored announcement. An unset announcement is returned as the zero value.
func (s *Store) Get(ctx context.Context) (Announcement, error) {
	var a Announcement

	entry, err := s.kv.Get(ctx, _storeKey)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			return a, nil
		}
		return a, err
	}

	if err := json.Unmarshal(entry.Value(), &a); err != nil {
		return a, err
	}

	return a, nil
}

// Set persists the given announcement, overwriting any existing one.
func (s *Store) Set(ctx context.Context, a Announcement) error {
	value, err := json.Marshal(a)
	if err != nil {
		return err
	}

	_, err = s.kv.Put(ctx, _storeKey, value)
	return err
}

// Delete removes the stored announcement. Deleting a missing announcement is a no-op.
func (s *Store) Delete(ctx context.Context) error {
	if err := s.kv.Delete(ctx, _storeKey); err != nil && !errors.Is(err, jetstream.ErrKeyNotFound) {
		return err
	}
	return nil
}
