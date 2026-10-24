package search

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
)

const (
	SkippedSpacesBucket  = "search-skipped-spaces"
	skippedSpacesTimeout = 10 * time.Second
)

// SkippedSpaces keeps track of spaces that were skipped during (re)indexing
// because they were disabled. When such a space gets enabled again it can be
// looked up here and reindexed.
type SkippedSpaces struct {
	kv jetstream.KeyValue
}

// skippedSpace is the value stored in the KV bucket for a skipped space.
type skippedSpace struct {
	SpaceID      string `json:"spaceId"`
	ForceReindex bool   `json:"forceReindex"`
}

// NewSkippedSpaces creates a new SkippedSpaces tracker backed by the given
// NATS JS KV bucket. A nil bucket turns all operations into no-ops, which is
// handy for tests and for deployments that run without an event bus.
func NewSkippedSpaces(kv jetstream.KeyValue) *SkippedSpaces {
	return &SkippedSpaces{kv: kv}
}

// key returns the canonical, KV-safe key for the given space id. NATS KV keys
// may only contain a limited set of characters, so the (possibly `$`/`!`
// separated) space id is normalized and base64url encoded.
func (s *SkippedSpaces) key(spaceID *provider.StorageSpaceId) (string, error) {
	rid, err := storagespace.ParseID(spaceID.GetOpaqueId())
	if err != nil {
		return "", err
	}
	canonical := storagespace.FormatResourceID(&provider.ResourceId{
		StorageId: rid.GetStorageId(),
		SpaceId:   rid.GetSpaceId(),
	})
	return base64.RawURLEncoding.EncodeToString([]byte(canonical)), nil
}

// Mark records that the given space was skipped while it was disabled.
// forceReindex indicates whether the skipped scan was a forced reindex. An
// existing forced mark is never downgraded to a shallow one.
func (s *SkippedSpaces) Mark(spaceID *provider.StorageSpaceId, forceReindex bool) error {
	if s == nil || s.kv == nil {
		return nil
	}
	key, err := s.key(spaceID)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), skippedSpacesTimeout)
	defer cancel()

	if !forceReindex {
		entry, err := s.kv.Get(ctx, key)
		switch {
		case err == nil:
			if decodeSkippedSpace(entry.Value()).ForceReindex {
				return nil
			}
		case !errors.Is(err, jetstream.ErrKeyNotFound):
			return err
		}
	}

	value, err := json.Marshal(skippedSpace{SpaceID: spaceID.GetOpaqueId(), ForceReindex: forceReindex})
	if err != nil {
		return err
	}
	_, err = s.kv.Put(ctx, key, value)
	return err
}

// decodeSkippedSpace decodes a KV value. Values that can not be decoded are
// treated as shallow (non-forced) marks.
func decodeSkippedSpace(value []byte) skippedSpace {
	var ss skippedSpace
	_ = json.Unmarshal(value, &ss)
	return ss
}

// Unmark removes any record for the given space. It is a no-op if the space
// was not tracked.
func (s *SkippedSpaces) Unmark(spaceID *provider.StorageSpaceId) error {
	if s == nil || s.kv == nil {
		return nil
	}
	key, err := s.key(spaceID)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), skippedSpacesTimeout)
	defer cancel()
	if err := s.kv.Delete(ctx, key); err != nil && !errors.Is(err, jetstream.ErrKeyNotFound) {
		return err
	}
	return nil
}

// IsMarked reports whether the given space was previously skipped while it was
// disabled, and whether the skipped scan was a forced reindex.
func (s *SkippedSpaces) IsMarked(spaceID *provider.StorageSpaceId) (marked bool, forceReindex bool, err error) {
	if s == nil || s.kv == nil {
		return false, false, nil
	}
	key, err := s.key(spaceID)
	if err != nil {
		return false, false, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), skippedSpacesTimeout)
	defer cancel()
	entry, err := s.kv.Get(ctx, key)
	switch {
	case err == nil:
		return true, decodeSkippedSpace(entry.Value()).ForceReindex, nil
	case errors.Is(err, jetstream.ErrKeyNotFound):
		return false, false, nil
	default:
		return false, false, err
	}
}
