// Copyright 2018-2021 CERN
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
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package idcache

import (
	"context"
	"encoding/base32"
	"path/filepath"
	"strings"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
)

type IDCache struct {
	kv jetstream.KeyValue
}

// NewStoreIDCache returns a new StoreIDCache
func NewStoreIDCache(kv jetstream.KeyValue) (*IDCache, error) {
	return &IDCache{
		kv: kv,
	}, nil
}

// Delete removes an entry from the cache
func (c *IDCache) Delete(ctx context.Context, spaceID, nodeID string) error {
	var rerr error
	v, err := c.kv.Get(ctx, cacheKey(spaceID, nodeID))
	if err == nil {
		rerr = c.kv.Purge(ctx, reverseCacheKey(string(v.Value())))
	}

	err = c.kv.Purge(ctx, cacheKey(spaceID, nodeID))
	if err != nil {
		return err
	}
	return rerr
}

// DeleteByPath removes an entry from the cache
func (c *IDCache) DeleteByPath(ctx context.Context, path string) error {
	baseKey := reverseCacheKey(path)

	spaceID, nodeID, err := c.GetByPath(ctx, path)
	if err != nil {
		if _, ok := err.(errtypes.NotFound); !ok {
			return err
		}
		appctx.GetLogger(ctx).Error().Err(err).Str("record", path).Msg("could not get spaceID and nodeID from cache")
	} else {
		err := c.kv.Purge(ctx, baseKey)
		if err != nil && err != jetstream.ErrKeyNotFound {
			appctx.GetLogger(ctx).Error().Err(err).Str("record", baseKey).Str("spaceID", spaceID).Str("nodeID", nodeID).Msg("could not purge from cache")
		}

		err = c.kv.Purge(ctx, cacheKey(spaceID, nodeID))
		if err != nil && err != jetstream.ErrKeyNotFound {
			appctx.GetLogger(ctx).Error().Err(err).Str("record", cacheKey(spaceID, nodeID)).Str("spaceID", spaceID).Str("nodeID", nodeID).Msg("could not purge from cache")
		}
	}

	watcher, err := c.kv.Watch(ctx, baseKey+".>")
	if err != nil {
		return err
	}
	defer func() { _ = watcher.Stop() }()

	for update := range watcher.Updates() {
		if update == nil {
			break
		}
		key := update.Key()
		spaceID, nodeID, err := c.getByReverseCacheKey(ctx, key)
		if err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("record", key).Msg("could not get spaceID and nodeID from cache")
			continue
		}

		err = c.kv.Purge(ctx, key)
		if err != nil && err != jetstream.ErrKeyNotFound {
			appctx.GetLogger(ctx).Error().Err(err).Str("record", key).Str("spaceID", spaceID).Str("nodeID", nodeID).Msg("could not purge from cache")
		}

		err = c.kv.Purge(ctx, cacheKey(spaceID, nodeID))
		if err != nil && err != jetstream.ErrKeyNotFound {
			appctx.GetLogger(ctx).Error().Err(err).Str("record", cacheKey(spaceID, nodeID)).Str("spaceID", spaceID).Str("nodeID", nodeID).Msg("could not purge from cache")
		}
	}
	return nil
}

// DeletePath removes only the path entry from the cache
func (c *IDCache) DeletePath(ctx context.Context, path string) error {
	return c.kv.Purge(ctx, reverseCacheKey(path))
}

// Set adds a new entry to the cache
func (c *IDCache) Set(ctx context.Context, spaceID, nodeID, val string) error {
	_, err := c.kv.Put(ctx, cacheKey(spaceID, nodeID), []byte(val))
	if err != nil {
		return err
	}

	_, err = c.kv.Put(ctx, reverseCacheKey(val), []byte(cacheKey(spaceID, nodeID)))
	return err
}

// Get returns the value for a given key
func (c *IDCache) Get(ctx context.Context, spaceID, nodeID string) (string, error) {
	record, err := c.kv.Get(ctx, cacheKey(spaceID, nodeID))
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return "", errtypes.NotFound("record not found in cache")
		}
		return "", err
	}
	return string(record.Value()), nil
}

func (c *IDCache) getByReverseCacheKey(ctx context.Context, reverseKey string) (string, string, error) {
	record, err := c.kv.Get(ctx, reverseKey)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return "", "", errtypes.NotFound("record not found in cache")
		}
		return "", "", err
	}
	decoded, err := base32.StdEncoding.DecodeString(string(record.Value()))
	if err != nil {
		return "", "", err
	}
	parts := strings.SplitN(string(decoded), "!", 2)
	if len(parts) != 2 {
		return "", "", errtypes.InternalError("invalid cache record")
	}
	return parts[0], parts[1], nil
}

// GetByPath returns the key for a given value
func (c *IDCache) GetByPath(ctx context.Context, path string) (string, string, error) {
	return c.getByReverseCacheKey(ctx, reverseCacheKey(path))
}

func cacheKey(spaceid, nodeID string) string {
	return base32.StdEncoding.EncodeToString([]byte(spaceid + "!" + nodeID))
}

func reverseCacheKey(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, string(filepath.Separator)), string(filepath.Separator))
	encoded := make([]string, len(parts))
	for i, p := range parts {
		encoded[i] = base32.StdEncoding.EncodeToString([]byte(p))
	}

	return strings.Join(encoded, ".")
}
