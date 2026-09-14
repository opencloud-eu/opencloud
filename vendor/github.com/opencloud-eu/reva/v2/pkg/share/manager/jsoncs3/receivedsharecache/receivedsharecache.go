// Copyright 2018-2022 CERN
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

package receivedsharecache

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/decomposedfs/mtimesyncedcache"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/metadata"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// name is the Tracer name used to identify this instrumentation library.
const tracerName = "receivedsharecache"

// Cache stores the list of received shares and their states
// It functions as an in-memory cache with a persistence layer
// The storage is sharded by user
type Cache struct {
	lockMap sync.Map

	ReceivedSpaces mtimesyncedcache.Map[string, *Spaces]

	storage metadata.Storage
	ttl     time.Duration
}

// Spaces holds the received shares of one user per space
type Spaces struct {
	Spaces map[string]*Space

	etag string
}

// Space holds the received shares of one user in one space
type Space struct {
	States map[string]*State
}

// State holds the state information of a received share
type State struct {
	State      collaboration.ShareState
	MountPoint *provider.Reference
	Hidden     bool
}

// New returns a new Cache instance
func New(s metadata.Storage, ttl time.Duration) Cache {
	return Cache{
		ReceivedSpaces: mtimesyncedcache.Map[string, *Spaces]{},
		storage:        s,
		ttl:            ttl,
		lockMap:        sync.Map{},
	}
}

func (c *Cache) lockUser(userID utils.FilenameEncoder) func() {
	v, _ := c.lockMap.LoadOrStore(userID.SafeFilename(), &sync.Mutex{})
	lock := v.(*sync.Mutex)

	lock.Lock()
	return func() { lock.Unlock() }
}

// Add adds a new entry to the cache
func (c *Cache) Add(ctx context.Context, userID utils.FilenameEncoder, spaceID string, rs *collaboration.ReceivedShare) error {
	userIDKey := userID.SafeFilename()
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Grab lock")
	unlock := c.lockUser(userID)
	span.End()
	span.SetAttributes(attribute.String("cs3.userid.key", userIDKey))
	defer unlock()

	if _, ok := c.ReceivedSpaces.Load(userIDKey); !ok {
		err := c.syncWithLock(ctx, userID)
		if err != nil {
			return err
		}
	}

	ctx, span = appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Add")
	defer span.End()
	span.SetAttributes(attribute.String("cs3.userid.key", userIDKey), attribute.String("cs3.spaceid", spaceID))

	persistFunc := func() error {
		c.initializeIfNeeded(userIDKey, spaceID)

		rss, _ := c.ReceivedSpaces.Load(userIDKey)
		receivedSpace := rss.Spaces[spaceID]
		if receivedSpace.States == nil {
			receivedSpace.States = map[string]*State{}
		}
		receivedSpace.States[rs.Share.Id.GetOpaqueId()] = &State{
			State:      rs.State,
			MountPoint: rs.MountPoint,
			Hidden:     rs.Hidden,
		}

		return c.persist(ctx, userID)
	}

	log := appctx.GetLogger(ctx).With().
		Str("hostname", os.Getenv("HOSTNAME")).
		Str("userIDKey", userIDKey).
		Str("spaceID", spaceID).Logger()

	var err error
	for retries := 100; retries > 0; retries-- {
		err = persistFunc()
		switch err.(type) {
		case nil:
			span.SetStatus(codes.Ok, "")
			return nil
		case errtypes.Aborted:
			log.Debug().Msg("aborted when persisting added received share: etag changed. retrying...")
			// this is the expected status code from the server when the if-match etag check fails
			// continue with sync below
		case errtypes.PreconditionFailed:
			log.Debug().Msg("precondition failed when persisting added received share: etag changed. retrying...")
			// actually, this is the wrong status code and we treat it like errtypes.Aborted because of inconsistencies on the server side
			// continue with sync below
		case errtypes.AlreadyExists:
			log.Debug().Msg("already exists when persisting added received share. retrying...")
			// CS3 uses an already exists error instead of precondition failed when using an If-None-Match=* header / IfExists flag in the InitiateFileUpload call.
			// Thas happens when the cache thinks there is no file.
			// continue with sync below
		default:
			span.SetStatus(codes.Error, fmt.Sprintf("persisting added received share failed. giving up: %s", err.Error()))
			log.Error().Err(err).Msg("persisting added received share failed")
			return err
		}
		if err := c.syncWithLock(ctx, userID); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			log.Error().Err(err).Msg("persisting added received share failed. giving up.")
			return err
		}
	}
	return err
}

// Get returns one entry from the cache
func (c *Cache) Get(ctx context.Context, userID utils.FilenameEncoder, spaceID, shareID string) (*State, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Grab lock")
	userIDKey := userID.SafeFilename()
	unlock := c.lockUser(userID)
	span.End()
	span.SetAttributes(attribute.String("cs3.userid.key", userIDKey))
	defer unlock()

	err := c.syncWithLock(ctx, userID)
	if err != nil {
		return nil, err
	}
	rss, ok := c.ReceivedSpaces.Load(userIDKey)
	if !ok || rss.Spaces[spaceID] == nil {
		return nil, nil
	}
	return rss.Spaces[spaceID].States[shareID], nil
}

// Remove removes an entry from the cache
func (c *Cache) Remove(ctx context.Context, userID utils.FilenameEncoder, spaceID, shareID string) error {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Grab lock")
	userIDKey := userID.SafeFilename()
	unlock := c.lockUser(userID)
	span.End()
	span.SetAttributes(attribute.String("cs3.userid.key", userIDKey))
	defer unlock()

	ctx, span = appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Add")
	defer span.End()
	span.SetAttributes(attribute.String("cs3.userid.key", userIDKey), attribute.String("cs3.spaceid", spaceID))

	persistFunc := func() error {
		c.initializeIfNeeded(userIDKey, spaceID)

		rss, _ := c.ReceivedSpaces.Load(userIDKey)
		receivedSpace := rss.Spaces[spaceID]
		if receivedSpace.States == nil {
			receivedSpace.States = map[string]*State{}
		}
		delete(receivedSpace.States, shareID)
		if len(receivedSpace.States) == 0 {
			delete(rss.Spaces, spaceID)
		}

		return c.persist(ctx, userID)
	}

	log := appctx.GetLogger(ctx).With().
		Str("hostname", os.Getenv("HOSTNAME")).
		Str("userIDKey", userIDKey).
		Str("spaceID", spaceID).Logger()

	var err error
	for retries := 100; retries > 0; retries-- {
		err = persistFunc()
		switch err.(type) {
		case nil:
			span.SetStatus(codes.Ok, "")
			return nil
		case errtypes.Aborted:
			log.Debug().Msg("aborted when persisting added received share: etag changed. retrying...")
			// this is the expected status code from the server when the if-match etag check fails
			// continue with sync below
		case errtypes.PreconditionFailed:
			log.Debug().Msg("precondition failed when persisting added received share: etag changed. retrying...")
			// actually, this is the wrong status code and we treat it like errtypes.Aborted because of inconsistencies on the server side
			// continue with sync below
		case errtypes.AlreadyExists:
			log.Debug().Msg("already exists when persisting added received share. retrying...")
			// CS3 uses an already exists error instead of precondition failed when using an If-None-Match=* header / IfExists flag in the InitiateFileUpload call.
			// Thas happens when the cache thinks there is no file.
			// continue with sync below
		default:
			span.SetStatus(codes.Error, fmt.Sprintf("persisting added received share failed. giving up: %s", err.Error()))
			log.Error().Err(err).Msg("persisting added received share failed")
			return err
		}
		if err := c.syncWithLock(ctx, userID); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			log.Error().Err(err).Msg("persisting added received share failed. giving up.")
			return err
		}
	}
	return err
}

// List returns a list of received shares for a given user
// The return list is guaranteed to be thread-safe
func (c *Cache) List(ctx context.Context, userID utils.FilenameEncoder) (map[string]*Space, error) {
	userIDKey := userID.SafeFilename()
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Grab lock")
	unlock := c.lockUser(userID)
	span.End()
	span.SetAttributes(attribute.String("cs3.userid.key", userIDKey))
	defer unlock()

	err := c.syncWithLock(ctx, userID)
	if err != nil {
		return nil, err
	}

	spaces := map[string]*Space{}
	rss, _ := c.ReceivedSpaces.Load(userIDKey)
	for spaceID, space := range rss.Spaces {
		spaceCopy := &Space{
			States: map[string]*State{},
		}
		for shareID, state := range space.States {
			spaceCopy.States[shareID] = &State{
				State:      state.State,
				MountPoint: state.MountPoint,
				Hidden:     state.Hidden,
			}
		}
		spaces[spaceID] = spaceCopy
	}
	return spaces, nil
}

func (c *Cache) syncWithLock(ctx context.Context, userID utils.FilenameEncoder) error {
	userIDKey := userID.SafeFilename()
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Sync")
	defer span.End()
	span.SetAttributes(attribute.String("cs3.userid.key", userIDKey))

	log := appctx.GetLogger(ctx).With().Str("userIDKey", userIDKey).Logger()

	c.initializeIfNeeded(userIDKey, "")

	jsonPath := userJSONPath(userIDKey)
	span.AddEvent("updating cache")
	//  - update cached list of created shares for the user in memory if changed
	rss, _ := c.ReceivedSpaces.Load(userIDKey)
	dlres, err := c.storage.Download(ctx, metadata.DownloadRequest{
		Path:        jsonPath,
		IfNoneMatch: []string{rss.etag},
	})
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
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to download the received share: %s", err.Error()))
		log.Error().Err(err).Msg("Failed to download the received share")
		return err
	}

	newSpaces := &Spaces{}
	err = json.Unmarshal(dlres.Content, newSpaces)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("Failed to unmarshal the received share: %s", err.Error()))
		log.Error().Err(err).Msg("Failed to unmarshal the received share")
		return err
	}
	newSpaces.etag = dlres.Etag

	c.ReceivedSpaces.Store(userIDKey, newSpaces)
	span.SetStatus(codes.Ok, "")
	return nil
}

// persist persists the data for one user to the storage
func (c *Cache) persist(ctx context.Context, userID utils.FilenameEncoder) error {
	userIDKey := userID.SafeFilename()
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Persist")
	defer span.End()
	span.SetAttributes(attribute.String("cs3.userid.key", userIDKey))

	rss, ok := c.ReceivedSpaces.Load(userIDKey)
	if !ok {
		span.SetStatus(codes.Ok, "no received shares")
		return nil
	}

	createdBytes, err := json.Marshal(rss)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	jsonPath := userJSONPath(userIDKey)
	if err := c.storage.MakeDirIfNotExist(ctx, path.Dir(jsonPath)); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	ur := metadata.UploadRequest{
		Path:        jsonPath,
		Content:     createdBytes,
		IfMatchEtag: rss.etag,
	}
	// when there is no etag in memory make sure the file has not been created on the server, see https://www.rfc-editor.org/rfc/rfc9110#field.if-match
	// > If the field value is "*", the condition is false if the origin server has a current representation for the target resource.
	if rss.etag == "" {
		ur.IfNoneMatch = []string{"*"}
	}

	res, err := c.storage.Upload(ctx, ur)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	rss.etag = res.Etag

	span.SetStatus(codes.Ok, "")
	return nil
}

func userJSONPath(userID string) string {
	return filepath.Join("/users", userID, "received.json")
}

func (c *Cache) initializeIfNeeded(userID, spaceID string) {
	rss, _ := c.ReceivedSpaces.LoadOrStore(userID, &Spaces{Spaces: map[string]*Space{}})
	if spaceID != "" && rss.Spaces[spaceID] == nil {
		rss.Spaces[spaceID] = &Space{}
		c.ReceivedSpaces.Store(userID, rss)
	}
}
