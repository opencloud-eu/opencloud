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

package providercache

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/mtimesyncedcache"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/metadata"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/maps"
)

var tracer trace.Tracer

func init() {
	tracer = otel.Tracer("github.com/opencloud-eu/reva/v2/pkg/share/manager/jsoncs3/providercache")
}

// Cache holds share information structured by provider and space
type Cache struct {
	lockMap sync.Map

	Providers mtimesyncedcache.Map[string, *Spaces]

	storage metadata.Storage
	ttl     time.Duration
}

// Spaces holds the share information for provider
type Spaces struct {
	Spaces mtimesyncedcache.Map[string, *Shares]
}

// Shares holds the share information of one space
type Shares struct {
	Shares map[string]*collaboration.Share

	Etag string
}

// UnmarshalJSON overrides the default unmarshaling
// Shares are tricky to unmarshal because they contain an interface (Grantee) which makes the json Unmarshal bail out
// To work around that problem we unmarshal into json.RawMessage in a first step and then try to manually unmarshal
// into the specific types in a second step.
func (s *Shares) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Shares map[string]json.RawMessage
	}{}

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	s.Shares = make(map[string]*collaboration.Share, len(tmp.Shares))
	for id, genericShare := range tmp.Shares {
		userShare := &collaboration.Share{
			Grantee: &provider.Grantee{Id: &provider.Grantee_UserId{}},
		}
		err = json.Unmarshal(genericShare, userShare) // is this a user share?
		if err == nil && userShare.Grantee.Type == provider.GranteeType_GRANTEE_TYPE_USER {
			s.Shares[id] = userShare
			continue
		}

		groupShare := &collaboration.Share{
			Grantee: &provider.Grantee{Id: &provider.Grantee_GroupId{}},
		}
		err = json.Unmarshal(genericShare, groupShare) // is this a group share?
		if err == nil && groupShare.Grantee.Type == provider.GranteeType_GRANTEE_TYPE_GROUP {
			s.Shares[id] = groupShare
			continue
		}

		invalidShare := &collaboration.Share{}
		err = json.Unmarshal(genericShare, invalidShare) // invalid
		if err == nil {
			s.Shares[id] = invalidShare
			continue
		}

		return err
	}

	return nil
}

// LockSpace locks the cache for a given space and returns an unlock function
func (c *Cache) LockSpace(spaceID string) func() {
	v, _ := c.lockMap.LoadOrStore(spaceID, &sync.Mutex{})
	lock := v.(*sync.Mutex)

	lock.Lock()
	return func() { lock.Unlock() }
}

// New returns a new Cache instance
func New(s metadata.Storage, ttl time.Duration) Cache {
	return Cache{
		Providers: mtimesyncedcache.Map[string, *Spaces]{},
		storage:   s,
		ttl:       ttl,
		lockMap:   sync.Map{},
	}
}

func (c *Cache) isSpaceCached(storageID, spaceID string) bool {
	spaces, ok := c.Providers.Load(storageID)
	if !ok {
		return false
	}
	_, ok = spaces.Spaces.Load(spaceID)
	return ok
}

// Add adds a share to the cache
func (c *Cache) Add(ctx context.Context, storageID, spaceID, shareID string, share *collaboration.Share) error {
	ctx, span := tracer.Start(ctx, "Add")
	defer span.End()
	span.SetAttributes(attribute.String("cs3.storageid", storageID), attribute.String("cs3.spaceid", spaceID), attribute.String("cs3.shareid", shareID))

	switch {
	case storageID == "":
		return fmt.Errorf("missing storage id")
	case spaceID == "":
		return fmt.Errorf("missing space id")
	case shareID == "":
		return fmt.Errorf("missing share id")
	}

	unlock := c.LockSpace(spaceID)
	defer unlock()
	span.AddEvent("got lock")

	var err error
	if !c.isSpaceCached(storageID, spaceID) {
		err = c.syncWithLock(ctx, storageID, spaceID)
		if err != nil {
			return err
		}
	}

	log := appctx.GetLogger(ctx).With().
		Str("hostname", os.Getenv("HOSTNAME")).
		Str("storageID", storageID).
		Str("spaceID", spaceID).
		Str("shareID", shareID).Logger()

	persistFunc := func() error {

		spaces, _ := c.Providers.Load(storageID)
		space, _ := spaces.Spaces.Load(spaceID)

		log.Info().Interface("shares", maps.Keys(space.Shares)).Str("New share", shareID).Msg("Adding share to space")
		space.Shares[shareID] = share

		return c.Persist(ctx, storageID, spaceID)
	}

	for retries := 100; retries > 0; retries-- {
		err = persistFunc()
		switch err.(type) {
		case nil:
			span.SetStatus(codes.Ok, "")
			return nil
		case errtypes.Aborted:
			log.Debug().Msg("aborted when persisting added provider share: etag changed. retrying...")
			// this is the expected status code from the server when the if-match etag check fails
			// continue with sync below
		case errtypes.PreconditionFailed:
			log.Debug().Msg("precondition failed when persisting added provider share: etag changed. retrying...")
			// actually, this is the wrong status code and we treat it like errtypes.Aborted because of inconsistencies on the server side
			// continue with sync below
		case errtypes.AlreadyExists:
			log.Debug().Msg("already exists when persisting added provider share. retrying...")
			// CS3 uses an already exists error instead of precondition failed when using an If-None-Match=* header / IfExists flag in the InitiateFileUpload call.
			// Thas happens when the cache thinks there is no file.
			// continue with sync below
		default:
			span.SetStatus(codes.Error, fmt.Sprintf("persisting added provider share failed. giving up: %s", err.Error()))
			log.Error().Err(err).Msg("persisting added provider share failed")
			return err
		}
		if err := c.syncWithLock(ctx, storageID, spaceID); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			log.Error().Err(err).Msg("persisting added provider share failed. giving up.")
			return err
		}
	}

	return err
}

// Remove removes a share from the cache
func (c *Cache) Remove(ctx context.Context, storageID, spaceID, shareID string) error {
	ctx, span := tracer.Start(ctx, "Remove")
	defer span.End()
	span.SetAttributes(attribute.String("cs3.storageid", storageID), attribute.String("cs3.spaceid", spaceID), attribute.String("cs3.shareid", shareID))

	unlock := c.LockSpace(spaceID)
	defer unlock()
	span.AddEvent("got lock")

	if !c.isSpaceCached(storageID, spaceID) {
		err := c.syncWithLock(ctx, storageID, spaceID)
		if err != nil {
			return err
		}
	}

	persistFunc := func() error {
		spaces, ok := c.Providers.Load(storageID)
		if !ok {
			return nil
		}
		space, _ := spaces.Spaces.Load(spaceID)
		if !ok {
			return nil
		}
		delete(space.Shares, shareID)

		return c.Persist(ctx, storageID, spaceID)
	}

	log := appctx.GetLogger(ctx).With().
		Str("hostname", os.Getenv("HOSTNAME")).
		Str("storageID", storageID).
		Str("spaceID", spaceID).
		Str("shareID", shareID).Logger()

	var err error
	for retries := 100; retries > 0; retries-- {
		err = persistFunc()
		switch err.(type) {
		case nil:
			span.SetStatus(codes.Ok, "")
			return nil
		case errtypes.Aborted:
			log.Debug().Msg("aborted when persisting removed provider share: etag changed. retrying...")
			// this is the expected status code from the server when the if-match etag check fails
			// continue with sync below
		case errtypes.PreconditionFailed:
			log.Debug().Msg("precondition failed when persisting removed provider share: etag changed. retrying...")
			// actually, this is the wrong status code and we treat it like errtypes.Aborted because of inconsistencies on the server side
			// continue with sync below
		default:
			span.SetStatus(codes.Error, fmt.Sprintf("persisting removed provider share failed. giving up: %s", err.Error()))
			log.Error().Err(err).Msg("persisting removed provider share failed")
			return err
		}
		if err := c.syncWithLock(ctx, storageID, spaceID); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			log.Error().Err(err).Msg("persisting removed provider share failed. giving up.")
			return err
		}
	}
	return err
}

// Get returns one entry from the cache
func (c *Cache) Get(ctx context.Context, storageID, spaceID, shareID string, skipSync bool) (*collaboration.Share, error) {
	ctx, span := tracer.Start(ctx, "Get")
	defer span.End()
	span.SetAttributes(attribute.String("cs3.storageid", storageID), attribute.String("cs3.spaceid", spaceID), attribute.String("cs3.shareid", shareID))

	unlock := c.LockSpace(spaceID)
	defer unlock()
	span.AddEvent("got lock")

	if !skipSync {
		// sync cache, maybe our data is outdated
		err := c.syncWithLock(ctx, storageID, spaceID)
		if err != nil {
			return nil, err
		}
	}

	spaces, ok := c.Providers.Load(storageID)
	if !ok {
		return nil, nil
	}
	space, ok := spaces.Spaces.Load(spaceID)
	if !ok {
		return nil, nil
	}
	return space.Shares[shareID], nil
}

// All returns all entries in the storage
func (c *Cache) All(ctx context.Context) (*mtimesyncedcache.Map[string, *Spaces], error) {
	ctx, span := tracer.Start(ctx, "All")
	defer span.End()

	providers, err := c.storage.ListDir(ctx, "/storages")
	if err != nil {
		return nil, err
	}
	for _, provider := range providers {
		storageID := provider.Name
		spaces, err := c.storage.ListDir(ctx, path.Join("/storages", storageID))
		if err != nil {
			return nil, err
		}
		for _, space := range spaces {
			spaceID := strings.TrimSuffix(space.Name, ".json")

			unlock := c.LockSpace(spaceID)
			span.AddEvent("got lock for space " + spaceID)
			if err := c.syncWithLock(ctx, storageID, spaceID); err != nil {
				return nil, err
			}
			unlock()
		}
	}

	return &c.Providers, nil
}

// ListSpace returns the list of shares in a given space
func (c *Cache) ListSpace(ctx context.Context, storageID, spaceID string) (*Shares, error) {
	ctx, span := tracer.Start(ctx, "ListSpace")
	defer span.End()
	span.SetAttributes(attribute.String("cs3.storageid", storageID), attribute.String("cs3.spaceid", spaceID))

	unlock := c.LockSpace(spaceID)
	defer unlock()
	span.AddEvent("got lock")

	// sync cache, maybe our data is outdated
	err := c.syncWithLock(ctx, storageID, spaceID)
	if err != nil {
		return nil, err
	}

	spaces, ok := c.Providers.Load(storageID)
	if !ok {
		return &Shares{}, nil
	}

	space, ok := spaces.Spaces.Load(spaceID)
	if !ok {
		return &Shares{}, nil
	}

	shares := &Shares{
		Shares: maps.Clone(space.Shares),
		Etag:   space.Etag,
	}
	return shares, nil
}

// Persist persists the data of one space
func (c *Cache) Persist(ctx context.Context, storageID, spaceID string) error {
	ctx, span := tracer.Start(ctx, "Persist")
	defer span.End()
	span.SetAttributes(attribute.String("cs3.storageid", storageID), attribute.String("cs3.spaceid", spaceID))

	spaces, ok := c.Providers.Load(storageID)
	if !ok {
		span.AddEvent("nothing to persist")
		span.SetStatus(codes.Ok, "")
		return nil
	}
	space, ok := spaces.Spaces.Load(spaceID)
	if !ok {
		span.AddEvent("nothing to persist")
		span.SetStatus(codes.Ok, "")
		return nil
	}
	span.SetAttributes(attribute.String("BeforeEtag", space.Etag))
	log := appctx.GetLogger(ctx).With().Str("storageID", storageID).Str("spaceID", spaceID).Logger()
	log = log.With().Str("BeforeEtag", space.Etag).Logger()

	createdBytes, err := json.Marshal(space)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	jsonPath := spaceJSONPath(storageID, spaceID)
	if err := c.storage.MakeDirIfNotExist(ctx, path.Dir(jsonPath)); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetAttributes(attribute.String("etag", space.Etag))

	ur := metadata.UploadRequest{
		Path:        jsonPath,
		Content:     createdBytes,
		IfMatchEtag: space.Etag,
	}
	// when there is no etag in memory make sure the file has not been created on the server, see https://www.rfc-editor.org/rfc/rfc9110#field.if-match
	// > If the field value is "*", the condition is false if the origin server has a current representation for the target resource.
	if space.Etag == "" {
		ur.IfNoneMatch = []string{"*"}
	}

	res, err := c.storage.Upload(ctx, ur)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.Debug().Err(err).Msg("persisting provider cache failed")
		return err
	}
	space.Etag = res.Etag

	span.SetStatus(codes.Ok, "")
	shares := []string{}
	for _, s := range space.Shares {
		shares = append(shares, s.GetId().GetOpaqueId())
	}
	log.Debug().Str("AfterEtag", space.Etag).Interface("Shares", shares).Msg("persisted provider cache")
	return nil
}

// PurgeSpace removes a space from the cache
func (c *Cache) PurgeSpace(ctx context.Context, storageID, spaceID string) error {
	ctx, span := tracer.Start(ctx, "PurgeSpace")
	defer span.End()

	unlock := c.LockSpace(spaceID)
	defer unlock()
	span.AddEvent("got lock")

	if !c.isSpaceCached(storageID, spaceID) {
		err := c.syncWithLock(ctx, storageID, spaceID)
		if err != nil {
			return err
		}
	}

	spaces, ok := c.Providers.Load(storageID)
	if !ok {
		return nil
	}
	newShares := &Shares{}
	if space, ok := spaces.Spaces.Load(spaceID); ok {
		newShares.Etag = space.Etag // keep the etag to allow overwriting the state on the server
	}
	spaces.Spaces.Store(spaceID, newShares)

	return c.Persist(ctx, storageID, spaceID)
}

func (c *Cache) syncWithLock(ctx context.Context, storageID, spaceID string) error {
	ctx, span := tracer.Start(ctx, "syncWithLock")
	defer span.End()

	c.initializeIfNeeded(storageID, spaceID)

	spaces, _ := c.Providers.Load(storageID)
	space, _ := spaces.Spaces.Load(spaceID)
	span.SetAttributes(attribute.String("cs3.storageid", storageID), attribute.String("cs3.spaceid", spaceID), attribute.String("etag", space.Etag))
	log := appctx.GetLogger(ctx).With().Str("storageID", storageID).Str("spaceID", spaceID).Str("etag", space.Etag).Str("hostname", os.Getenv("HOSTNAME")).Logger()

	dlreq := metadata.DownloadRequest{
		Path: spaceJSONPath(storageID, spaceID),
	}
	// when we know an etag, only download if it changed remotely
	if space.Etag != "" {
		dlreq.IfNoneMatch = []string{space.Etag}
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "downloading provider cache failed")
		return err
	}

	span.AddEvent("updating local cache")
	newShares := &Shares{}
	err = json.Unmarshal(dlres.Content, newShares)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "unmarshaling provider cache failed")
		log.Error().Err(err).Msg("unmarshaling provider cache failed")
		return err
	}
	newShares.Etag = dlres.Etag

	spaces.Spaces.Store(spaceID, newShares)
	span.SetStatus(codes.Ok, "")
	return nil
}

func (c *Cache) initializeIfNeeded(storageID, spaceID string) {
	spaces, _ := c.Providers.LoadOrStore(storageID, &Spaces{
		Spaces: mtimesyncedcache.Map[string, *Shares]{},
	})
	_, _ = spaces.Spaces.LoadOrStore(spaceID, &Shares{
		Shares: map[string]*collaboration.Share{},
	})
}

func spaceJSONPath(storageID, spaceID string) string {
	return filepath.Join("/storages", storageID, spaceID+".json")
}
