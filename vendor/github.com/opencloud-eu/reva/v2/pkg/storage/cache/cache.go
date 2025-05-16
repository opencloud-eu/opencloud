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
	"fmt"
	"strings"
	"sync"
	"time"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/store"
	"github.com/vmihailenco/msgpack/v5"
	microstore "go-micro.dev/v4/store"
)

var (
	// DefaultStatCache is the memory store.
	statCaches                = make(map[string]StatCache)
	providerCaches            = make(map[string]ProviderCache)
	createHomeCaches          = make(map[string]CreateHomeCache)
	createPersonalSpaceCaches = make(map[string]CreatePersonalSpaceCache)
	fileMetadataCaches        = make(map[string]FileMetadataCache)
	mutex                     sync.Mutex
)

// Config contains the configuring for a cache
type Config struct {
	Store              string        `mapstructure:"cache_store"`
	Nodes              []string      `mapstructure:"cache_nodes"`
	Database           string        `mapstructure:"cache_database"`
	Table              string        `mapstructure:"cache_table"`
	TTL                time.Duration `mapstructure:"cache_ttl"`
	Size               int           `mapstructure:"cache_size"`
	DisablePersistence bool          `mapstructure:"cache_disable_persistence"`
	AuthUsername       string        `mapstructure:"cache_auth_username"`
	AuthPassword       string        `mapstructure:"cache_auth_password"`
}

// Cache handles key value operations on caches
// It, and the interfaces derived from it, are currently being used
// for building caches around go-micro stores, encoding the data
// in the messsagepack format.
type Cache interface {
	PullFromCache(key string, dest interface{}) error
	PushToCache(key string, src interface{}) error
	List(opts ...microstore.ListOption) ([]string, error)
	Delete(key string, opts ...microstore.DeleteOption) error
	Close() error
}

// StatCache handles removing keys from a stat cache
type StatCache interface {
	Cache
	RemoveStat(userID *userpb.UserId, res *provider.ResourceId)
	RemoveStatContext(ctx context.Context, userID *userpb.UserId, res *provider.ResourceId)
	GetKey(userID *userpb.UserId, ref *provider.Reference, metaDataKeys, fieldMaskPaths []string) string
}

// ProviderCache handles removing keys from a storage provider cache
type ProviderCache interface {
	Cache
	RemoveListStorageProviders(res *provider.ResourceId)
	GetKey(userID *userpb.UserId, spaceID string) string
}

// CreateHomeCache handles removing keys from a create home cache
type CreateHomeCache interface {
	Cache
	RemoveCreateHome(res *provider.ResourceId)
	GetKey(userID *userpb.UserId) string
}

// CreatePersonalSpaceCache handles removing keys from a create home cache
type CreatePersonalSpaceCache interface {
	Cache
	GetKey(userID *userpb.UserId) string
}

// FileMetadataCache handles file metadata
type FileMetadataCache interface {
	Cache
	RemoveMetadata(path string) error
}

// GetStatCache will return an existing StatCache for the given store, nodes, database and table
// If it does not exist yet it will be created, different TTLs are ignored
func GetStatCache(cfg Config) StatCache {
	mutex.Lock()
	defer mutex.Unlock()

	key := strings.Join(append(append([]string{cfg.Store}, cfg.Nodes...), cfg.Database, cfg.Table), ":")
	if statCaches[key] == nil {
		statCaches[key] = NewStatCache(cfg)
	}
	return statCaches[key]
}

// GetProviderCache will return an existing ProviderCache for the given store, nodes, database and table
// If it does not exist yet it will be created, different TTLs are ignored
func GetProviderCache(cfg Config) ProviderCache {
	mutex.Lock()
	defer mutex.Unlock()

	key := strings.Join(append(append([]string{cfg.Store}, cfg.Nodes...), cfg.Database, cfg.Table), ":")
	if providerCaches[key] == nil {
		providerCaches[key] = NewProviderCache(cfg)
	}
	return providerCaches[key]
}

// GetCreateHomeCache will return an existing CreateHomeCache for the given store, nodes, database and table
// If it does not exist yet it will be created, different TTLs are ignored
func GetCreateHomeCache(cfg Config) CreateHomeCache {
	mutex.Lock()
	defer mutex.Unlock()

	key := strings.Join(append(append([]string{cfg.Store}, cfg.Nodes...), cfg.Database, cfg.Table), ":")
	if createHomeCaches[key] == nil {
		createHomeCaches[key] = NewCreateHomeCache(cfg)
	}
	return createHomeCaches[key]
}

// GetCreatePersonalSpaceCache will return an existing CreatePersonalSpaceCache for the given store, nodes, database and table
// If it does not exist yet it will be created, different TTLs are ignored
func GetCreatePersonalSpaceCache(cfg Config) CreatePersonalSpaceCache {
	mutex.Lock()
	defer mutex.Unlock()

	key := strings.Join(append(append([]string{cfg.Store}, cfg.Nodes...), cfg.Database, cfg.Table), ":")
	if createPersonalSpaceCaches[key] == nil {
		createPersonalSpaceCaches[key] = NewCreatePersonalSpaceCache(cfg)
	}
	return createPersonalSpaceCaches[key]
}

// GetFileMetadataCache will return an existing GetFileMetadataCache for the given store, nodes, database and table
// If it does not exist yet it will be created, different TTLs are ignored
func GetFileMetadataCache(cfg Config) FileMetadataCache {
	mutex.Lock()
	defer mutex.Unlock()

	key := strings.Join(append(append([]string{cfg.Store}, cfg.Nodes...), cfg.Database, cfg.Table), ":")
	if fileMetadataCaches[key] == nil {
		fileMetadataCaches[key] = NewFileMetadataCache(cfg)
	}
	return fileMetadataCaches[key]
}

// CacheStore holds cache store specific configuration
type cacheStore struct {
	s               microstore.Store
	database, table string
	ttl             time.Duration
}

// PullFromCache pulls a value from the configured database and table of the underlying store using the given key
func (cache cacheStore) PullFromCache(key string, dest interface{}) error {
	r, err := cache.s.Read(key, microstore.ReadFrom(cache.database, cache.table), microstore.ReadLimit(1))
	if err != nil {
		return err
	}
	if len(r) == 0 {
		return fmt.Errorf("not found")
	}

	return msgpack.Unmarshal(r[0].Value, &dest)
}

// PushToCache pushes a key and value to the configured database and table of the underlying store
func (cache cacheStore) PushToCache(key string, src interface{}) error {
	b, err := msgpack.Marshal(src)
	if err != nil {
		return err
	}

	record := &microstore.Record{
		Key:    key,
		Value:  b,
		Expiry: cache.ttl,
	}

	return cache.s.Write(
		record,
		microstore.WriteTo(cache.database, cache.table),
		microstore.WriteTTL(cache.ttl),
	)
}

// List lists the keys on the configured database and table of the underlying store
func (cache cacheStore) List(opts ...microstore.ListOption) ([]string, error) {
	o := []microstore.ListOption{
		microstore.ListFrom(cache.database, cache.table),
	}
	o = append(o, opts...)
	keys, err := cache.s.List(o...)
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// Delete deletes the given key on the configured database and table of the underlying store
func (cache cacheStore) Delete(key string, opts ...microstore.DeleteOption) error {
	o := []microstore.DeleteOption{
		microstore.DeleteFrom(cache.database, cache.table),
	}
	o = append(o, opts...)
	return cache.s.Delete(key, o...)
}

// Close closes the underlying store
func (cache cacheStore) Close() error {
	return cache.s.Close()
}

func getStore(cfg Config) microstore.Store {
	return store.Create(
		store.Store(cfg.Store),
		microstore.Nodes(cfg.Nodes...),
		microstore.Database(cfg.Database),
		microstore.Table(cfg.Table),
		store.TTL(cfg.TTL),
		store.Size(cfg.Size),
		store.DisablePersistence(cfg.DisablePersistence),
		store.Authentication(cfg.AuthUsername, cfg.AuthPassword),
	)
}
