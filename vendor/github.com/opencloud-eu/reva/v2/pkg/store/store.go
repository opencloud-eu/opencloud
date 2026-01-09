// Copyright 2018-2023 CERN
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

package store

import (
	"context"
	"crypto/tls"
	"strings"
	"time"

	natsjs "github.com/go-micro/plugins/v4/store/nats-js"
	natsjskv "github.com/go-micro/plugins/v4/store/nats-js-kv"
	"github.com/go-micro/plugins/v4/store/redis"
	redisopts "github.com/go-redis/redis/v8"
	"github.com/nats-io/nats.go"
	"github.com/opencloud-eu/reva/v2/pkg/store/etcd"
	"github.com/opencloud-eu/reva/v2/pkg/store/memory"
	"go-micro.dev/v4/logger"
	microstore "go-micro.dev/v4/store"
)

var ocMemStore *microstore.Store

const (
	// TypeMemory represents memory stores
	TypeMemory = "memory"
	// TypeNoop represents noop stores
	TypeNoop = "noop"
	// TypeEtcd represents etcd stores
	TypeEtcd = "etcd"
	// TypeRedis represents redis stores
	TypeRedis = "redis"
	// TypeRedisSentinel represents redis-sentinel stores
	TypeRedisSentinel = "redis-sentinel"
	// TypeOCMem represents ocmem stores
	TypeOCMem = "ocmem"
	// TypeNatsJS represents nats-js stores
	TypeNatsJS = "nats-js"
	// TypeNatsJSKV represents nats-js-kv stores
	TypeNatsJSKV = "nats-js-kv"
)

// Create initializes a new store
func Create(opts ...microstore.Option) microstore.Store {
	options := &microstore.Options{
		Context: context.Background(),
	}
	for _, o := range opts {
		o(options)
	}

	// ensure we have a logger
	if options.Logger == nil {
		options.Logger = logger.DefaultLogger
	}

	storeType, _ := options.Context.Value(typeContextKey{}).(string)

	switch storeType {
	case TypeNoop:
		return microstore.NewNoopStore(opts...)
	case TypeEtcd:
		return etcd.NewStore(opts...)
	case TypeRedis:
		// FIXME redis plugin does not support redis cluster or ring -> needs upstream patch or our implementation
		return redis.NewStore(opts...)
	case TypeRedisSentinel:
		redisMaster := ""
		redisNodes := []string{}
		for _, node := range options.Nodes {
			parts := strings.SplitN(node, "/", 2)
			if len(parts) != 2 {
				return nil
			}
			// the first node is used to retrieve the redis master
			redisNodes = append(redisNodes, parts[0])
			if redisMaster == "" {
				redisMaster = parts[1]
			}
		}
		return redis.NewStore(
			microstore.Database(options.Database),
			microstore.Table(options.Table),
			microstore.Nodes(redisNodes...),
			redis.WithRedisOptions(redisopts.UniversalOptions{
				MasterName: redisMaster,
			}),
		)
	case TypeOCMem:
		if ocMemStore == nil {
			var memStore microstore.Store

			sizeNum, _ := options.Context.Value(sizeContextKey{}).(int)
			if sizeNum <= 0 {
				memStore = memory.NewMultiMemStore()
			} else {
				memStore = memory.NewMultiMemStore(
					microstore.WithContext(
						memory.NewContext(
							context.Background(),
							map[string]interface{}{
								"maxCap": sizeNum,
							},
						)),
				)
			}
			ocMemStore = &memStore
		}
		return *ocMemStore
	case TypeNatsJS:
		opts, ttl, natsOptions := natsConfig(options.Logger, options.Context, opts)
		return natsjs.NewStore(
			append(opts,
				natsjs.NatsOptions(natsOptions), // always pass in properly initialized default nats options
				natsjs.DefaultTTL(ttl))...,      // nats needs a DefaultTTL option as it does not support per Write TTL
		)
	case TypeNatsJSKV:
		opts, ttl, natsOptions := natsConfig(options.Logger, options.Context, opts)
		return natsjskv.NewStore(
			append(opts,
				natsjskv.NatsOptions(natsOptions), // always pass in properly initialized default nats options
				natsjskv.EncodeKeys(),             // nats has restrictions on the key, we cannot use slashes
				natsjskv.DefaultTTL(ttl))...,      // nats needs a DefaultTTL option as it does not support per Write TTL
		)
	case TypeMemory, "mem", "": // allow existing short form and use as default
		return microstore.NewMemoryStore(opts...)
	default:
		options.Logger.Logf(logger.ErrorLevel, "unknown store type: '%s', falling back to memory", storeType)
		return microstore.NewMemoryStore(opts...)
	}
}

func natsConfig(log logger.Logger, ctx context.Context, opts []microstore.Option) ([]microstore.Option, time.Duration, nats.Options) {

	if mem, _ := ctx.Value(disablePersistanceContextKey{}).(bool); mem {
		opts = append(opts, natsjs.DefaultMemory())
	}

	ttl, _ := ctx.Value(ttlContextKey{}).(time.Duration)

	// preparing natsOptions before the switch to reuse the same code
	natsOptions := nats.GetDefaultOptions()
	natsOptions.Name = "TODO" // we can pass in the service name to allow identifying the client, but that requires adding a custom context option
	if auth, ok := ctx.Value(authenticationContextKey{}).([]string); ok && len(auth) == 2 {
		natsOptions.User = auth[0]
		natsOptions.Password = auth[1]
	}
	if enableTLS, ok := ctx.Value(tlsEnabledContextKey{}).(bool); ok && enableTLS {
		if rootca, ok := ctx.Value(tlsRootCAContextKey{}).(string); ok && rootca != "" {
			// when root ca is configured use it. an insecure flag is ignored.
			if err := nats.RootCAs(rootca)(&natsOptions); err != nil {
				log.Log(logger.ErrorLevel, err)
			}
		} else {
			// enable tls with insecure option
			insecure := ctx.Value(tlsInsecureContextKey{}).(bool)
			_ = nats.Secure(&tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: insecure})(&natsOptions)
		}
	}

	return opts, ttl, natsOptions
}
