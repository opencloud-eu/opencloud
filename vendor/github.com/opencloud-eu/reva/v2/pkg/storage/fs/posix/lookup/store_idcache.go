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

package lookup

import (
	"context"
	"strings"

	microstore "go-micro.dev/v4/store"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/storage/cache"
	"github.com/opencloud-eu/reva/v2/pkg/store"
)

type StoreIDCache struct {
	cache microstore.Store
}

// NewMemoryIDCache returns a new MemoryIDCache
func NewStoreIDCache(c cache.Config) *StoreIDCache {
	return &StoreIDCache{
		cache: store.Create(
			store.Store(c.Store),
			store.Size(c.Size),
			microstore.Nodes(c.Nodes...),
			microstore.Database(c.Database),
			microstore.Table(c.Table),
			store.DisablePersistence(c.DisablePersistence),
			store.Authentication(c.AuthUsername, c.AuthPassword),
		),
	}
}

// Delete removes an entry from the cache
func (c *StoreIDCache) Delete(_ context.Context, spaceID, nodeID string) error {
	var rerr error
	v, err := c.cache.Read(cacheKey(spaceID, nodeID))
	if err == nil {
		rerr = c.cache.Delete(reverseCacheKey(string(v[0].Value)))
	}

	err = c.cache.Delete(cacheKey(spaceID, nodeID))
	if err != nil {
		return err
	}
	return rerr
}

// DeleteByPath removes an entry from the cache
func (c *StoreIDCache) DeleteByPath(ctx context.Context, path string) error {
	spaceID, nodeID, ok := c.GetByPath(ctx, path)
	if !ok {
		appctx.GetLogger(ctx).Error().Str("record", path).Msg("could not get spaceID and nodeID from cache")
	} else {
		err := c.cache.Delete(reverseCacheKey(path))
		if err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("record", path).Str("spaceID", spaceID).Str("nodeID", nodeID).Msg("could not get spaceID and nodeID from cache")
		}

		err = c.cache.Delete(cacheKey(spaceID, nodeID))
		if err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("record", path).Str("spaceID", spaceID).Str("nodeID", nodeID).Msg("could not get spaceID and nodeID from cache")
		}
	}

	list, err := c.cache.List(
		microstore.ListPrefix(reverseCacheKey(path) + "/"),
	)
	if err != nil {
		return err
	}
	for _, record := range list {
		spaceID, nodeID, ok := c.GetByPath(ctx, record)
		if !ok {
			appctx.GetLogger(ctx).Error().Str("record", record).Msg("could not get spaceID and nodeID from cache")
			continue
		}

		err := c.cache.Delete(reverseCacheKey(record))
		if err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("record", record).Str("spaceID", spaceID).Str("nodeID", nodeID).Msg("could not get spaceID and nodeID from cache")
		}

		err = c.cache.Delete(cacheKey(spaceID, nodeID))
		if err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("record", record).Str("spaceID", spaceID).Str("nodeID", nodeID).Msg("could not get spaceID and nodeID from cache")
		}
	}
	return nil
}

// DeletePath removes only the path entry from the cache
func (c *StoreIDCache) DeletePath(ctx context.Context, path string) error {
	return c.cache.Delete(reverseCacheKey(path))
}

// Add adds a new entry to the cache
func (c *StoreIDCache) Set(_ context.Context, spaceID, nodeID, val string) error {
	err := c.cache.Write(&microstore.Record{
		Key:   cacheKey(spaceID, nodeID),
		Value: []byte(val),
	})
	if err != nil {
		return err
	}

	return c.cache.Write(&microstore.Record{
		Key:   reverseCacheKey(val),
		Value: []byte(cacheKey(spaceID, nodeID)),
	})
}

// Get returns the value for a given key
func (c *StoreIDCache) Get(_ context.Context, spaceID, nodeID string) (string, bool) {
	records, err := c.cache.Read(cacheKey(spaceID, nodeID))
	if err != nil || len(records) == 0 {
		return "", false
	}
	return string(records[0].Value), true
}

// GetByPath returns the key for a given value
func (c *StoreIDCache) GetByPath(_ context.Context, val string) (string, string, bool) {
	records, err := c.cache.Read(reverseCacheKey(val))
	if err != nil || len(records) == 0 {
		return "", "", false
	}
	parts := strings.SplitN(string(records[0].Value), "!", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func cacheKey(spaceid, nodeID string) string {
	return spaceid + "!" + nodeID
}

func reverseCacheKey(val string) string {
	return val
}
