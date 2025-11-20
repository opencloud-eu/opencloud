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

package cache

import (
	"context"
	"strings"
	"sync"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"go-micro.dev/v4/store"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func init() {
	tracer = otel.Tracer("github.com/cs3org/reva/pkg/storage/cache")
}

// NewStatCache creates a new StatCache
func NewStatCache(cfg Config) StatCache {
	c := statCache{}
	c.s = getStore(cfg)
	c.database = cfg.Database
	c.table = cfg.Table
	c.ttl = cfg.TTL
	return &c
}

type statCache struct {
	cacheStore
}

func (c statCache) RemoveStatContext(ctx context.Context, userID *userpb.UserId, res *provider.ResourceId) {
	_, span := tracer.Start(ctx, "RemoveStatContext")
	defer span.End()

	span.SetAttributes(semconv.EnduserIDKey.String(userID.GetOpaqueId()))

	uid := "uid:" + userID.GetOpaqueId()
	sid := ""
	oid := ""

	if res != nil {
		span.SetAttributes(
			attribute.String("space.id", res.SpaceId),
			attribute.String("node.id", res.OpaqueId),
		)
		sid = "sid:" + res.SpaceId
		oid = "oid:" + res.OpaqueId
	}

	// TODO currently, invalidating the stat cache is inefficient and should be disabled. Storage providers / drivers can more selectively invalidate stat cache entries.
	// This shotgun invalidation wipes all cache entries for the user, space, and nodeid of a changed resource, which means the stat cache is mostly empty, anyway.
	prefixes := []string{uid, "*" + sid, "*" + oid}

	wg := sync.WaitGroup{}
	for _, prefix := range prefixes {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			keys, _ := c.List(store.ListPrefix(p), store.ListLimit(100))
			for _, key := range keys {
				wg.Add(1)
				go func(k string) {
					defer wg.Done()
					_ = c.Delete(k)
				}(key)
			}
		}(prefix)
	}

	wg.Wait()
}

// RemoveStatContext(ctx,  removes a reference from the stat cache
func (c statCache) RemoveStat(userID *userpb.UserId, res *provider.ResourceId) {
	c.RemoveStatContext(context.Background(), userID, res)
}

// generates a user specific key pointing to ref - used for statcache
// a key looks like: uid:1234-1233!sid:5678-5677!oid:9923-9934!path:/path/to/source
// as you see it adds "uid:"/"sid:"/"oid:" prefixes to the uuids so they can be differentiated
func (c statCache) GetKey(userID *userpb.UserId, ref *provider.Reference, metaDataKeys, fieldMaskPaths []string) string {
	if ref == nil || ref.ResourceId == nil || ref.ResourceId.SpaceId == "" {
		return ""
	}

	key := strings.Builder{}
	key.WriteString("uid:")
	key.WriteString(userID.GetOpaqueId())
	key.WriteString("!sid:")
	key.WriteString(ref.ResourceId.SpaceId)
	key.WriteString("!oid:")
	key.WriteString(ref.ResourceId.OpaqueId)
	key.WriteString("!path:")
	key.WriteString(ref.Path)
	for _, k := range metaDataKeys {
		key.WriteString("!mdk:")
		key.WriteString(k)
	}
	for _, p := range fieldMaskPaths {
		key.WriteString("!fmp:")
		key.WriteString(p)
	}

	return key.String()
}
