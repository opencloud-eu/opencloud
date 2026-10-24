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

package sharecache

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/exp/maps"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/share/manager/jsoncs3/shareid"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/decomposedfs/mtimesyncedcache"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/metadata"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// name is the Tracer name used to identify this instrumentation library.
const tracerName = "sharecache"

// Cache caches the list of share ids for users/groups
// It functions as an in-memory cache with a persistence layer
// The storage is sharded by user/group
type Cache struct {
	lockMap sync.Map

	UserShares mtimesyncedcache.Map[string, *UserShareCache]

	storage   metadata.Storage
	namespace string
	filename  string
	ttl       time.Duration
}

// UserShareCache holds the space/share map for one user
type UserShareCache struct {
	UserShares map[string]*SpaceShareIDs

	Etag string
}

// SpaceShareIDs holds the unique list of share ids for a space
type SpaceShareIDs struct {
	IDs map[string]struct{}
}

func (c *Cache) lockUser(userID string) func() {
	v, _ := c.lockMap.LoadOrStore(userID, &sync.Mutex{})
	lock := v.(*sync.Mutex)

	lock.Lock()
	return func() { lock.Unlock() }
}

// New returns a new Cache instance
func New(s metadata.Storage, namespace, filename string, ttl time.Duration) Cache {
	return Cache{
		UserShares: mtimesyncedcache.Map[string, *UserShareCache]{},
		storage:    s,
		namespace:  namespace,
		filename:   filename,
		ttl:        ttl,
		lockMap:    sync.Map{},
	}
}

// Add adds a share to the cache
func (c *Cache) Add(ctx context.Context, id utils.FilenameEncoder, shareID string) error {
	key := id.SafeFilename()

	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Grab lock")
	unlock := c.lockUser(key)
	span.End()
	span.SetAttributes(attribute.String("cs3.userid", key))
	defer unlock()

	if _, ok := c.UserShares.Load(key); !ok {
		err := c.syncWithLock(ctx, key)
		if err != nil {
			return err
		}
	}

	ctx, span = appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Add")
	defer span.End()
	span.SetAttributes(attribute.String("cs3.userid", key), attribute.String("cs3.shareid", shareID))

	storageid, spaceid, _ := shareid.Decode(shareID)
	ssid := storageid + shareid.IDDelimiter + spaceid

	persistFunc := func() error {
		c.initializeIfNeeded(key, ssid)

		// add share id
		us, _ := c.UserShares.Load(key)
		us.UserShares[ssid].IDs[shareID] = struct{}{}
		return c.Persist(ctx, key)
	}

	log := appctx.GetLogger(ctx).With().
		Str("hostname", os.Getenv("HOSTNAME")).
		Str("userID", key).
		Str("shareID", shareID).Logger()

	var err error
	for retries := 100; retries > 0; retries-- {
		err = persistFunc()
		switch err.(type) {
		case nil:
			span.SetStatus(codes.Ok, "")
			return nil
		case errtypes.Aborted:
			log.Debug().Msg("aborted when persisting added share: etag changed. retrying...")
			// this is the expected status code from the server when the if-match etag check fails
			// continue with sync below
		case errtypes.PreconditionFailed:
			log.Debug().Msg("precondition failed when persisting added share: etag changed. retrying...")
			// actually, this is the wrong status code and we treat it like errtypes.Aborted because of inconsistencies on the server side
			// continue with sync below
		case errtypes.AlreadyExists:
			log.Debug().Msg("already exists when persisting added share. retrying...")
			// CS3 uses an already exists error instead of precondition failed when using an If-None-Match=* header / IfExists flag in the InitiateFileUpload call.
			// Thas happens when the cache thinks there is no file.
			// continue with sync below
		default:
			span.SetStatus(codes.Error, fmt.Sprintf("persisting added share failed. giving up: %s", err.Error()))
			log.Error().Err(err).Msg("persisting added share failed")
			return err
		}
		if err := c.syncWithLock(ctx, key); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			log.Error().Err(err).Msg("persisting added share failed. giving up.")
			return err
		}
	}
	return err
}

// Remove removes a share for the given user
func (c *Cache) Remove(ctx context.Context, id utils.FilenameEncoder, shareID string) error {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Grab lock")
	key := id.SafeFilename()
	unlock := c.lockUser(key)
	span.End()
	span.SetAttributes(attribute.String("cs3.userid", key))
	defer unlock()

	if _, ok := c.UserShares.Load(key); ok {
		err := c.syncWithLock(ctx, key)
		if err != nil {
			return err
		}
	}

	ctx, span = appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Remove")
	defer span.End()
	span.SetAttributes(attribute.String("cs3.userid", key), attribute.String("cs3.shareid", shareID))

	storageid, spaceid, _ := shareid.Decode(shareID)
	ssid := storageid + shareid.IDDelimiter + spaceid

	persistFunc := func() error {
		us, loaded := c.UserShares.LoadOrStore(key, &UserShareCache{
			UserShares: map[string]*SpaceShareIDs{},
		})

		if loaded {
			// remove share id
			delete(us.UserShares[ssid].IDs, shareID)
		}

		return c.Persist(ctx, key)
	}

	log := appctx.GetLogger(ctx).With().
		Str("hostname", os.Getenv("HOSTNAME")).
		Str("userID", key).
		Str("shareID", shareID).Logger()

	var err error
	for retries := 100; retries > 0; retries-- {
		err = persistFunc()
		switch err.(type) {
		case nil:
			span.SetStatus(codes.Ok, "")
			return nil
		case errtypes.Aborted:
			log.Debug().Msg("aborted when persisting removed share: etag changed. retrying...")
			// this is the expected status code from the server when the if-match etag check fails
			// continue with sync below
		case errtypes.PreconditionFailed:
			log.Debug().Msg("precondition failed when persisting removed share: etag changed. retrying...")
			// actually, this is the wrong status code and we treat it like errtypes.Aborted because of inconsistencies on the server side
			// continue with sync below
		case errtypes.AlreadyExists:
			log.Debug().Msg("file already existed when persisting removed share. retrying...")
			// CS3 uses an already exists error instead of precondition failed when using an If-None-Match=* header / IfExists flag in the InitiateFileUpload call.
			// Thas happens when the cache thinks there is no file.
			// continue with sync below
		default:
			span.SetStatus(codes.Error, fmt.Sprintf("persisting removed share failed. giving up: %s", err.Error()))
			log.Error().Err(err).Msg("persisting removed share failed")
			return err
		}
		if err := c.syncWithLock(ctx, key); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	return err
}

// List return the list of spaces/shares for the given user/group
func (c *Cache) List(ctx context.Context, id utils.FilenameEncoder) (map[string]SpaceShareIDs, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Grab lock")
	key := id.SafeFilename()
	unlock := c.lockUser(key)
	span.End()
	span.SetAttributes(attribute.String("cs3.userid", key))
	defer unlock()
	if err := c.syncWithLock(ctx, key); err != nil {
		return nil, err
	}

	r := map[string]SpaceShareIDs{}
	us, ok := c.UserShares.Load(key)
	if !ok {
		return r, nil
	}

	for ssid, cached := range us.UserShares {
		r[ssid] = SpaceShareIDs{
			IDs: maps.Clone(cached.IDs),
		}
	}
	return r, nil
}

func (c *Cache) syncWithLock(ctx context.Context, userID string) error {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Sync")
	defer span.End()
	span.SetAttributes(attribute.String("cs3.userid", userID))

	log := appctx.GetLogger(ctx).With().Str("userID", userID).Logger()

	c.initializeIfNeeded(userID, "")

	userCreatedPath := c.userCreatedPath(userID)
	span.AddEvent("updating cache")
	//  - update cached list of created shares for the user in memory if changed
	dlreq := metadata.DownloadRequest{
		Path: userCreatedPath,
	}
	if us, ok := c.UserShares.Load(userID); ok && us.Etag != "" {
		dlreq.IfNoneMatch = []string{us.Etag}
	}

	dlres, err := c.storage.Download(ctx, dlreq)
	switch err.(type) {
	case nil:
		span.AddEvent("updating local cache")
	case errtypes.NotFound:
		span.SetStatus(codes.Ok, "")
		return nil
	case errtypes.NotModified:
		span.SetStatus(codes.Ok, "")
		return nil
	default:
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to download the share cache: %s", err.Error()))
		log.Error().Err(err).Msg("Failed to download the share cache")
		return err
	}

	newShareCache := &UserShareCache{}
	err = json.Unmarshal(dlres.Content, newShareCache)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to unmarshal the share cache: %s", err.Error()))
		log.Error().Err(err).Msg("Failed to unmarshal the share cache")
		return err
	}
	newShareCache.Etag = dlres.Etag

	c.UserShares.Store(userID, newShareCache)
	span.SetStatus(codes.Ok, "")
	return nil
}

// Persist persists the data for one user/group to the storage
func (c *Cache) Persist(ctx context.Context, key string) error {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Persist")
	defer span.End()
	span.SetAttributes(attribute.String("cs3.userid", key))

	us, ok := c.UserShares.Load(key)
	if !ok {
		span.SetStatus(codes.Ok, "no user shares")
		return nil
	}
	createdBytes, err := json.Marshal(us)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	jsonPath := c.userCreatedPath(key)
	if err := c.storage.MakeDirIfNotExist(ctx, path.Dir(jsonPath)); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	ur := metadata.UploadRequest{
		Path:        jsonPath,
		Content:     createdBytes,
		IfMatchEtag: us.Etag,
	}
	// when there is no etag in memory make sure the file has not been created on the server, see https://www.rfc-editor.org/rfc/rfc9110#field.if-match
	// > If the field value is "*", the condition is false if the origin server has a current representation for the target resource.
	if us.Etag == "" {
		ur.IfNoneMatch = []string{"*"}
	}

	res, err := c.storage.Upload(ctx, ur)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	us.Etag = res.Etag

	span.SetStatus(codes.Ok, "")
	return nil
}

func (c *Cache) userCreatedPath(key string) string {
	return filepath.Join("/", c.namespace, key, c.filename)
}

func (c *Cache) initializeIfNeeded(userid, ssid string) {
	us, _ := c.UserShares.LoadOrStore(userid, &UserShareCache{
		UserShares: map[string]*SpaceShareIDs{},
	})
	if ssid != "" && us.UserShares[ssid] == nil {
		us.UserShares[ssid] = &SpaceShareIDs{
			IDs: map[string]struct{}{},
		}
	}
}
