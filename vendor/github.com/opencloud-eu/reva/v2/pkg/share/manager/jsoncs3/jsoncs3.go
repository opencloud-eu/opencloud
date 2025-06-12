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

package jsoncs3

import (
	"context"
	"strings"
	"sync"
	"time"

	gatewayv1beta1 "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"
	"github.com/opencloud-eu/reva/v2/pkg/logger"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/share"
	"github.com/opencloud-eu/reva/v2/pkg/share/manager/jsoncs3/providercache"
	"github.com/opencloud-eu/reva/v2/pkg/share/manager/jsoncs3/receivedsharecache"
	"github.com/opencloud-eu/reva/v2/pkg/share/manager/jsoncs3/sharecache"
	"github.com/opencloud-eu/reva/v2/pkg/share/manager/jsoncs3/shareid"
	"github.com/opencloud-eu/reva/v2/pkg/share/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/metadata" // nolint:staticcheck // we need the legacy package to convert V1 to V2 messages
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/sync/errgroup"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

/*
  The sharded json driver splits the json file per storage space. Similar to fileids shareids are prefixed with the spaceid for easier lookup.
  In addition to the space json the share manager keeps lists for users and groups to cache their lists of created and received shares
  and to hold the state of received shares.

  FAQ
  Q: Why not split shares by user and have a list per user?
  A: While shares are created by users, they are persisted as grants on a file.
     If we persist shares by their creator/owner they would vanish if a user is deprovisioned: shares
	 in project spaces could not be managed collaboratively.
	 By splitting by space, we are in fact not only splitting by user, but more granular, per space.


  File structure in the jsoncs3 space:

  /storages/{storageid}/{spaceid.json} 	// contains the share information of all shares in that space
  /users/{userid}/created.json			// points to the spaces the user created shares in, including the list of shares
  /users/{userid}/received.json			// holds the accepted/pending state and mount point of received shares for users
  /groups/{groupid}/received.json		// points to the spaces the group has received shares in including the list of shares

  Example:
  	├── groups
  	│	└── group1
  	│		└── received.json
  	├── storages
  	│	└── storageid
  	│		└── spaceid.json
  	└── users
   		├── admin
 		│	└── created.json
 		└── einstein
 			└── received.json

  Whenever a share is created, the share manager has to
  1. update the /storages/{storageid}/{spaceid}.json file,
  2. create /users/{userid}/created.json if it doesn't exist yet and add the space/share
  3. create /users/{userid}/received.json or /groups/{groupid}/received.json if it doesn exist yet and add the space/share

  When updating shares /storages/{storageid}/{spaceid}.json is updated accordingly. The etag is used to invalidate in-memory caches:
  - TODO the upload is tried with an if-unmodified-since header
  - TODO when if fails, the {spaceid}.json file is downloaded, the changes are reapplied and the upload is retried with the new etag

  When updating received shares the mountpoint and state are updated in /users/{userid}/received.json (for both user and group shares).

  When reading the list of received shares the /users/{userid}/received.json file and the /groups/{groupid}/received.json files are statted.
  - if the etag changed we download the file to update the local cache

  When reading the list of created shares the /users/{userid}/created.json file is statted
  - if the etag changed we download the file to update the local cache
*/

// TODO implement a channel based aggregation of sharing requests: every in memory cache should read as many share updates to a space that are available and update them all in one go
// whenever a persist operation fails we check if we can read more shares from the channel

// name is the Tracer name used to identify this instrumentation library.
const tracerName = "jsoncs3"

func init() {
	registry.Register("jsoncs3", NewDefault)
}

var (
	_registeredEvents = []events.Unmarshaller{
		events.SpaceDeleted{},
	}
)

type config struct {
	GatewayAddr       string       `mapstructure:"gateway_addr"`
	MaxConcurrency    int          `mapstructure:"max_concurrency"`
	ProviderAddr      string       `mapstructure:"provider_addr"`
	ServiceUserID     string       `mapstructure:"service_user_id"`
	ServiceUserIdp    string       `mapstructure:"service_user_idp"`
	MachineAuthAPIKey string       `mapstructure:"machine_auth_apikey"`
	CacheTTL          int          `mapstructure:"ttl"`
	Events            EventOptions `mapstructure:"events"`
}

// EventOptions are the configurable options for events
type EventOptions struct {
	Endpoint             string `mapstructure:"natsaddress"`
	Cluster              string `mapstructure:"natsclusterid"`
	TLSInsecure          bool   `mapstructure:"tlsinsecure"`
	TLSRootCACertificate string `mapstructure:"tlsrootcacertificate"`
	EnableTLS            bool   `mapstructure:"enabletls"`
	AuthUsername         string `mapstructure:"authusername"`
	AuthPassword         string `mapstructure:"authpassword"`
}

// Manager implements a share manager using a cs3 storage backend with local caching
type Manager struct {
	sync.RWMutex

	Cache              providercache.Cache      // holds all shares, sharded by provider id and space id
	CreatedCache       sharecache.Cache         // holds the list of shares a user has created, sharded by user id
	GroupReceivedCache sharecache.Cache         // holds the list of shares a group has access to, sharded by group id
	UserReceivedStates receivedsharecache.Cache // holds the state of shares a user has received, sharded by user id

	storage   metadata.Storage
	SpaceRoot *provider.ResourceId

	initialized bool

	MaxConcurrency int

	gatewaySelector pool.Selectable[gatewayv1beta1.GatewayAPIClient]
	eventStream     events.Stream
}

// NewDefault returns a new manager instance with default dependencies
func NewDefault(m map[string]interface{}) (share.Manager, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error creating a new manager")
		return nil, err
	}

	s, err := metadata.NewCS3Storage(c.ProviderAddr, c.ProviderAddr, c.ServiceUserID, c.ServiceUserIdp, c.MachineAuthAPIKey)
	if err != nil {
		return nil, err
	}

	gatewaySelector, err := pool.GatewaySelector(c.GatewayAddr)
	if err != nil {
		return nil, err
	}

	var es events.Stream
	if c.Events.Endpoint != "" {
		es, err = stream.NatsFromConfig("jsoncs3-share-manager", false, stream.NatsConfig(c.Events))
		if err != nil {
			return nil, err
		}
	}

	return New(s, gatewaySelector, c.CacheTTL, es, c.MaxConcurrency)
}

// New returns a new manager instance.
func New(s metadata.Storage, gatewaySelector pool.Selectable[gatewayv1beta1.GatewayAPIClient], ttlSeconds int, es events.Stream, maxconcurrency int) (*Manager, error) {
	ttl := time.Duration(ttlSeconds) * time.Second

	m := &Manager{
		Cache:              providercache.New(s, ttl),
		CreatedCache:       sharecache.New(s, "users", "created.json", ttl),
		UserReceivedStates: receivedsharecache.New(s, ttl),
		GroupReceivedCache: sharecache.New(s, "groups", "received.json", ttl),
		storage:            s,
		gatewaySelector:    gatewaySelector,
		eventStream:        es,
		MaxConcurrency:     maxconcurrency,
	}

	// listen for events
	if m.eventStream != nil {
		ch, err := events.Consume(m.eventStream, "jsoncs3sharemanager", _registeredEvents...)
		if err != nil {
			appctx.GetLogger(context.Background()).Error().Err(err).Msg("error consuming events")
		}
		go m.ProcessEvents(ch)
	}

	return m, nil
}

func (m *Manager) initialize(ctx context.Context) error {
	_, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "initialize")
	defer span.End()
	if m.initialized {
		span.SetStatus(codes.Ok, "already initialized")
		return nil
	}

	m.Lock()
	defer m.Unlock()

	if m.initialized { // check if initialization happened while grabbing the lock
		span.SetStatus(codes.Ok, "initialized while grabbing lock")
		return nil
	}

	ctx = context.Background()
	err := m.storage.Init(ctx, "jsoncs3-share-manager-metadata")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	err = m.storage.MakeDirIfNotExist(ctx, "storages")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	err = m.storage.MakeDirIfNotExist(ctx, "users")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	err = m.storage.MakeDirIfNotExist(ctx, "groups")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	m.initialized = true
	span.SetStatus(codes.Ok, "initialized")
	return nil
}

func (m *Manager) ProcessEvents(ch <-chan events.Event) {
	log := logger.New()
	for event := range ch {
		ctx := context.Background()

		if err := m.initialize(ctx); err != nil {
			log.Error().Err(err).Msg("error initializing manager")
		}

		if ev, ok := event.Event.(events.SpaceDeleted); ok {
			log.Debug().Msgf("space deleted event: %v", ev)
			go func() { m.purgeSpace(ctx, ev.ID) }()
		}
	}
}

// Share creates a new share
func (m *Manager) Share(ctx context.Context, md *provider.ResourceInfo, g *collaboration.ShareGrant) (*collaboration.Share, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Share")
	defer span.End()
	if err := m.initialize(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	user := ctxpkg.ContextMustGetUser(ctx)
	ts := utils.TSNow()

	// do not allow share to myself or the owner if share is for a user
	// TODO: should this not already be caught at the gw level?
	if g.Grantee.Type == provider.GranteeType_GRANTEE_TYPE_USER &&
		(utils.UserEqual(g.Grantee.GetUserId(), user.Id) || utils.UserEqual(g.Grantee.GetUserId(), md.Owner)) {
		err := errtypes.BadRequest("jsoncs3: owner/creator and grantee are the same")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// check if share already exists.
	key := &collaboration.ShareKey{
		// Owner:      md.Owner, owner no longer matters as it belongs to the space
		ResourceId: md.Id,
		Grantee:    g.Grantee,
	}

	_, err := m.getByKey(ctx, key)
	if err == nil {
		// share already exists
		err := errtypes.AlreadyExists(key.String())
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	shareID := shareid.Encode(md.GetId().GetStorageId(), md.GetId().GetSpaceId(), uuid.NewString())
	s := &collaboration.Share{
		Id: &collaboration.ShareId{
			OpaqueId: shareID,
		},
		ResourceId:  md.Id,
		Permissions: g.Permissions,
		Grantee:     g.Grantee,
		Expiration:  g.Expiration,
		Owner:       md.Owner,
		Creator:     user.Id,
		Ctime:       ts,
		Mtime:       ts,
	}

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		err := m.Cache.Add(ctx, md.Id.StorageId, md.Id.SpaceId, shareID, s)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		return err
	})

	eg.Go(func() error {
		err := m.CreatedCache.Add(ctx, s.GetCreator().GetOpaqueId(), shareID)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		return err
	})

	spaceID := md.Id.StorageId + shareid.IDDelimiter + md.Id.SpaceId
	// set flag for grantee to have access to share
	switch g.Grantee.Type {
	case provider.GranteeType_GRANTEE_TYPE_USER:
		eg.Go(func() error {
			userid := g.Grantee.GetUserId().GetOpaqueId()

			rs := &collaboration.ReceivedShare{
				Share: s,
				State: collaboration.ShareState_SHARE_STATE_PENDING,
			}
			err := m.UserReceivedStates.Add(ctx, userid, spaceID, rs)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}

			return err
		})
	case provider.GranteeType_GRANTEE_TYPE_GROUP:
		eg.Go(func() error {
			groupid := g.Grantee.GetGroupId().GetOpaqueId()
			err := m.GroupReceivedCache.Add(ctx, groupid, shareID)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}

			return err
		})
	}

	if err = eg.Wait(); err != nil {
		return nil, err
	}

	span.SetStatus(codes.Ok, "")

	return s, nil
}

// getByID must be called in a lock-controlled block.
func (m *Manager) getByID(ctx context.Context, id *collaboration.ShareId) (*collaboration.Share, error) {
	storageID, spaceID, _ := shareid.Decode(id.OpaqueId)

	share, err := m.Cache.Get(ctx, storageID, spaceID, id.OpaqueId, false)
	if err != nil {
		return nil, err
	}
	if share == nil {
		return nil, errtypes.NotFound(id.String())
	}
	return share, nil
}

// getByKey must be called in a lock-controlled block.
func (m *Manager) getByKey(ctx context.Context, key *collaboration.ShareKey) (*collaboration.Share, error) {
	spaceShares, err := m.Cache.ListSpace(ctx, key.ResourceId.StorageId, key.ResourceId.SpaceId)
	if err != nil {
		return nil, err
	}
	for _, share := range spaceShares.Shares {
		if utils.GranteeEqual(key.Grantee, share.Grantee) && utils.ResourceIDEqual(share.ResourceId, key.ResourceId) {
			return share, nil
		}
	}
	return nil, errtypes.NotFound(key.String())
}

// get must be called in a lock-controlled block.
func (m *Manager) get(ctx context.Context, ref *collaboration.ShareReference) (s *collaboration.Share, err error) {
	switch {
	case ref.GetId() != nil:
		s, err = m.getByID(ctx, ref.GetId())
	case ref.GetKey() != nil:
		s, err = m.getByKey(ctx, ref.GetKey())
	default:
		err = errtypes.NotFound(ref.String())
	}
	return
}

// GetShare gets the information for a share by the given ref.
func (m *Manager) GetShare(ctx context.Context, ref *collaboration.ShareReference) (*collaboration.Share, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "GetShare")
	defer span.End()
	sublog := appctx.GetLogger(ctx).With().Str("id", ref.GetId().GetOpaqueId()).Str("key", ref.GetKey().String()).Str("driver", "jsoncs3").Str("handler", "GetShare").Logger()
	if err := m.initialize(ctx); err != nil {
		return nil, err
	}

	s, err := m.get(ctx, ref)
	if err != nil {
		return nil, err
	}
	if share.IsExpired(s) {
		if err := m.removeShare(ctx, s, false); err != nil {
			sublog.Error().Err(err).
				Msg("failed to unshare expired share")
		}
		if err := events.Publish(ctx, m.eventStream, events.ShareExpired{
			ShareID:        s.GetId(),
			ShareOwner:     s.GetOwner(),
			ItemID:         s.GetResourceId(),
			ExpiredAt:      time.Unix(int64(s.GetExpiration().GetSeconds()), int64(s.GetExpiration().GetNanos())),
			GranteeUserID:  s.GetGrantee().GetUserId(),
			GranteeGroupID: s.GetGrantee().GetGroupId(),
		}); err != nil {
			sublog.Error().Err(err).
				Msg("failed to publish share expired event")
		}
	}
	// check if we are the creator or the grantee
	// TODO allow manager to get shares in a space created by other users
	user := ctxpkg.ContextMustGetUser(ctx)
	if share.IsCreatedByUser(s, user) || share.IsGrantedToUser(s, user) {
		return s, nil
	}

	req := &provider.StatRequest{
		Ref: &provider.Reference{ResourceId: s.ResourceId},
		FieldMask: &fieldmaskpb.FieldMask{
			Paths: []string{"permissions"},
		},
	}
	client, err := m.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	res, err := client.Stat(ctx, req)
	if err == nil &&
		res.Status.Code == rpcv1beta1.Code_CODE_OK &&
		res.Info.PermissionSet.ListGrants {
		return s, nil
	}

	// we return not found to not disclose information
	return nil, errtypes.NotFound(ref.String())
}

// Unshare deletes a share
func (m *Manager) Unshare(ctx context.Context, ref *collaboration.ShareReference) error {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "Unshare")
	defer span.End()

	if err := m.initialize(ctx); err != nil {
		return err
	}

	user := ctxpkg.ContextMustGetUser(ctx)

	s, err := m.get(ctx, ref)
	if err != nil {
		return err
	}
	// TODO allow manager to unshare shares in a space created by other users
	if !share.IsCreatedByUser(s, user) {
		// TODO why not permission denied?
		return errtypes.NotFound(ref.String())
	}

	return m.removeShare(ctx, s, false)
}

// UpdateShare updates the mode of the given share.
func (m *Manager) UpdateShare(ctx context.Context, ref *collaboration.ShareReference, p *collaboration.SharePermissions, updated *collaboration.Share, fieldMask *field_mask.FieldMask) (*collaboration.Share, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "UpdateShare")
	defer span.End()

	if err := m.initialize(ctx); err != nil {
		return nil, err
	}

	var toUpdate *collaboration.Share

	if ref != nil {
		var err error
		toUpdate, err = m.get(ctx, ref)
		if err != nil {
			return nil, err
		}
	} else if updated != nil {
		var err error
		toUpdate, err = m.getByID(ctx, updated.Id)
		if err != nil {
			return nil, err
		}
	}

	if fieldMask != nil {
		for i := range fieldMask.Paths {
			switch fieldMask.Paths[i] {
			case "permissions":
				toUpdate.Permissions = updated.Permissions
			case "expiration":
				toUpdate.Expiration = updated.Expiration
			default:
				return nil, errtypes.NotSupported("updating " + fieldMask.Paths[i] + " is not supported")
			}
		}
	}

	user := ctxpkg.ContextMustGetUser(ctx)
	if !share.IsCreatedByUser(toUpdate, user) {
		req := &provider.StatRequest{
			Ref: &provider.Reference{ResourceId: toUpdate.ResourceId},
			FieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"permissions"},
			},
		}
		client, err := m.gatewaySelector.Next()
		if err != nil {
			return nil, err
		}
		res, err := client.Stat(ctx, req)
		if err != nil ||
			res.Status.Code != rpcv1beta1.Code_CODE_OK ||
			!res.Info.PermissionSet.UpdateGrant {
			return nil, errtypes.NotFound(ref.String())
		}
	}

	if p != nil {
		toUpdate.Permissions = p
	}
	toUpdate.Mtime = utils.TSNow()

	// Update provider cache
	unlock := m.Cache.LockSpace(toUpdate.ResourceId.SpaceId)
	defer unlock()
	err := m.Cache.Persist(ctx, toUpdate.ResourceId.StorageId, toUpdate.ResourceId.SpaceId)
	// when persisting fails
	if _, ok := err.(errtypes.IsPreconditionFailed); ok {
		// reupdate
		toUpdate, err = m.get(ctx, ref) // does an implicit sync
		if err != nil {
			return nil, err
		}
		toUpdate.Permissions = p
		toUpdate.Mtime = utils.TSNow()

		// persist again
		err = m.Cache.Persist(ctx, toUpdate.ResourceId.StorageId, toUpdate.ResourceId.SpaceId)
		// TODO try more often?
	}
	if err != nil {
		return nil, err
	}

	return toUpdate, nil
}

// ListShares returns the shares created by the user
func (m *Manager) ListShares(ctx context.Context, filters []*collaboration.Filter) ([]*collaboration.Share, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "ListShares")
	defer span.End()

	if err := m.initialize(ctx); err != nil {
		return nil, err
	}

	user := ctxpkg.ContextMustGetUser(ctx)

	if len(share.FilterFiltersByType(filters, collaboration.Filter_TYPE_RESOURCE_ID)) > 0 {
		return m.listSharesByIDs(ctx, user, filters)
	}

	return m.listCreatedShares(ctx, user, filters)
}

func (m *Manager) listSharesByIDs(ctx context.Context, user *userv1beta1.User, filters []*collaboration.Filter) ([]*collaboration.Share, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "listSharesByIDs")
	defer span.End()
	sublog := appctx.GetLogger(ctx).With().Str("userid", user.GetId().GetOpaqueId()).Str("useridp", user.GetId().GetIdp()).Str("driver", "jsoncs3").Str("handler", "listSharesByIDs").Logger()

	providerSpaces := make(map[string]map[string]struct{})
	for _, f := range share.FilterFiltersByType(filters, collaboration.Filter_TYPE_RESOURCE_ID) {
		storageID := f.GetResourceId().GetStorageId()
		spaceID := f.GetResourceId().GetSpaceId()
		if providerSpaces[storageID] == nil {
			providerSpaces[storageID] = make(map[string]struct{})
		}
		providerSpaces[storageID][spaceID] = struct{}{}
	}

	statCache := make(map[string]struct{})
	var ss []*collaboration.Share
	for providerID, spaces := range providerSpaces {
		for spaceID := range spaces {
			shares, err := m.Cache.ListSpace(ctx, providerID, spaceID)
			if err != nil {
				return nil, err
			}

			for _, s := range shares.Shares {
				resourceID := s.GetResourceId()
				sublog = sublog.With().Str("storageid", resourceID.GetStorageId()).Str("spaceid", resourceID.GetSpaceId()).Str("opaqueid", resourceID.GetOpaqueId()).Logger()
				if share.IsExpired(s) {
					if err := m.removeShare(ctx, s, false); err != nil {
						sublog.Error().Err(err).
							Msg("failed to unshare expired share")
					}
					if err := events.Publish(ctx, m.eventStream, events.ShareExpired{
						ShareOwner:     s.GetOwner(),
						ItemID:         resourceID,
						ExpiredAt:      time.Unix(int64(s.GetExpiration().GetSeconds()), int64(s.GetExpiration().GetNanos())),
						GranteeUserID:  s.GetGrantee().GetUserId(),
						GranteeGroupID: s.GetGrantee().GetGroupId(),
					}); err != nil {
						sublog.Error().Err(err).
							Msg("failed to publish share expired event")
					}
					continue
				}
				if !share.MatchesFilters(s, filters) {
					continue
				}

				if !(share.IsCreatedByUser(s, user) || share.IsGrantedToUser(s, user)) {
					key := storagespace.FormatResourceID(resourceID)
					if _, hit := statCache[key]; !hit {
						req := &provider.StatRequest{
							Ref: &provider.Reference{ResourceId: resourceID},
							FieldMask: &fieldmaskpb.FieldMask{
								Paths: []string{"permissions"},
							},
						}
						client, err := m.gatewaySelector.Next()
						if err != nil {
							sublog.Error().Err(err).Msg("failed to select next gateway client")
							continue
						}
						res, err := client.Stat(ctx, req)
						if err != nil {
							sublog.Error().Err(err).Msg("failed to make stat call")
							continue
						}
						if res.Status.Code != rpcv1beta1.Code_CODE_OK {
							sublog.Debug().Str("code", res.GetStatus().GetCode().String()).Msg(res.GetStatus().GetMessage())
							continue
						}
						if !res.Info.PermissionSet.ListGrants {
							sublog.Debug().Msg("user has no list grants permission")
							continue
						}
						sublog.Debug().Msg("listing share for non participating user")
						statCache[key] = struct{}{}
					}
				}

				ss = append(ss, s)
			}
		}
	}
	span.SetStatus(codes.Ok, "")
	return ss, nil
}

func (m *Manager) listCreatedShares(ctx context.Context, user *userv1beta1.User, filters []*collaboration.Filter) ([]*collaboration.Share, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "listCreatedShares")
	defer span.End()
	sublog := appctx.GetLogger(ctx).With().Str("userid", user.GetId().GetOpaqueId()).Str("useridp", user.GetId().GetIdp()).Str("driver", "jsoncs3").Str("handler", "listCreatedShares").Logger()

	list, err := m.CreatedCache.List(ctx, user.Id.OpaqueId)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	numWorkers := m.MaxConcurrency
	if numWorkers == 0 || len(list) < numWorkers {
		numWorkers = len(list)
	}

	type w struct {
		ssid string
		ids  sharecache.SpaceShareIDs
	}
	work := make(chan w)
	results := make(chan *collaboration.Share)

	g, ctx := errgroup.WithContext(ctx)

	// Distribute work
	g.Go(func() error {
		defer close(work)
		for ssid, ids := range list {
			select {
			case work <- w{ssid, ids}:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})
	// Spawn workers that'll concurrently work the queue
	for i := 0; i < numWorkers; i++ {
		g.Go(func() error {
			for w := range work {
				storageID, spaceID, _ := shareid.Decode(w.ssid)
				// fetch all shares from space with one request
				_, err := m.Cache.ListSpace(ctx, storageID, spaceID)
				if err != nil {
					sublog.Error().Err(err).
						Str("storageid", storageID).
						Str("spaceid", spaceID).
						Msg("failed to list shares in space")
					continue
				}
				for shareID := range w.ids.IDs {
					s, err := m.Cache.Get(ctx, storageID, spaceID, shareID, true)
					if err != nil || s == nil {
						continue
					}
					if share.IsExpired(s) {
						if err := m.removeShare(ctx, s, false); err != nil {
							sublog.Error().Err(err).
								Msg("failed to unshare expired share")
						}
						if err := events.Publish(ctx, m.eventStream, events.ShareExpired{
							ShareOwner:     s.GetOwner(),
							ItemID:         s.GetResourceId(),
							ExpiredAt:      time.Unix(int64(s.GetExpiration().GetSeconds()), int64(s.GetExpiration().GetNanos())),
							GranteeUserID:  s.GetGrantee().GetUserId(),
							GranteeGroupID: s.GetGrantee().GetGroupId(),
						}); err != nil {
							sublog.Error().Err(err).
								Msg("failed to publish share expired event")
						}
						continue
					}
					if utils.UserEqual(user.GetId(), s.GetCreator()) {
						if share.MatchesFilters(s, filters) {
							select {
							case results <- s:
							case <-ctx.Done():
								return ctx.Err()
							}
						}
					}
				}
			}
			return nil
		})
	}

	// Wait for things to settle down, then close results chan
	go func() {
		_ = g.Wait() // error is checked later
		close(results)
	}()

	ss := []*collaboration.Share{}
	for n := range results {
		ss = append(ss, n)
	}

	if err := g.Wait(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return ss, nil
}

// ListReceivedShares returns the list of shares the user has access to.
func (m *Manager) ListReceivedShares(ctx context.Context, filters []*collaboration.Filter, forUser *userv1beta1.UserId) ([]*collaboration.ReceivedShare, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "ListReceivedShares")
	defer span.End()
	sublog := appctx.GetLogger(ctx).With().Str("driver", "jsoncs3").Str("handler", "ListReceivedShares").Logger()

	if err := m.initialize(ctx); err != nil {
		return nil, err
	}

	user := ctxpkg.ContextMustGetUser(ctx)
	if user.GetId().GetType() == userv1beta1.UserType_USER_TYPE_SERVICE {
		client, err := m.gatewaySelector.Next()
		if err != nil {
			return nil, err
		}
		u, err := utils.GetUser(ctx, forUser, client)
		if err != nil {
			return nil, errtypes.BadRequest("user not found")
		}
		user = u
	}

	ssids := map[string]*receivedsharecache.Space{}

	// first collect all spaceids the user has access to as a group member
	for _, group := range user.Groups {
		list, err := m.GroupReceivedCache.List(ctx, group)
		if err != nil {
			continue // ignore error, cache will be updated on next read
		}
		for ssid, spaceShareIDs := range list {
			// add a pending entry, the state will be updated
			// when reading the received shares below if they have already been accepted or denied
			var rs *receivedsharecache.Space
			var ok bool
			if rs, ok = ssids[ssid]; !ok {
				rs = &receivedsharecache.Space{
					States: make(map[string]*receivedsharecache.State, len(spaceShareIDs.IDs)),
				}
				ssids[ssid] = rs
			}

			for shareid := range spaceShareIDs.IDs {
				rs.States[shareid] = &receivedsharecache.State{
					State: collaboration.ShareState_SHARE_STATE_PENDING,
				}
			}
		}
	}

	// add all spaces the user has receved shares for, this includes mount points and share state for groups
	spaces, err := m.UserReceivedStates.List(ctx, user.Id.OpaqueId)
	if err != nil {
		return nil, err
	}
	for ssid, rspace := range spaces {
		if rs, ok := ssids[ssid]; ok {
			for shareid, state := range rspace.States {
				// overwrite state
				rs.States[shareid] = state
			}
		} else {
			ssids[ssid] = rspace
		}
	}

	numWorkers := m.MaxConcurrency
	if numWorkers == 0 || len(ssids) < numWorkers {
		numWorkers = len(ssids)
	}

	type w struct {
		ssid   string
		rspace *receivedsharecache.Space
	}
	work := make(chan w)
	results := make(chan *collaboration.ReceivedShare)

	g, ctx := errgroup.WithContext(ctx)

	// Distribute work
	g.Go(func() error {
		defer close(work)
		for ssid, rspace := range ssids {
			select {
			case work <- w{ssid, rspace}:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})

	// Spawn workers that'll concurrently work the queue
	for i := 0; i < numWorkers; i++ {
		g.Go(func() error {
			for w := range work {
				storageID, spaceID, _ := shareid.Decode(w.ssid)
				sublogr := sublog.With().Str("storageid", storageID).Str("spaceid", spaceID).Logger()
				// fetch all shares from space with one request
				_, err := m.Cache.ListSpace(ctx, storageID, spaceID)
				if err != nil {
					sublogr.Error().Err(err).
						Msg("failed to list shares in space")
					continue
				}
				for shareID, state := range w.rspace.States {
					s, err := m.Cache.Get(ctx, storageID, spaceID, shareID, true)
					if err != nil {
						sublogr.Error().Err(err).Msg("could not retrieve share")
						continue
					}
					if s == nil {
						sublogr.Warn().Str("shareid", shareID).Msg("share not found. cleaning up")
						_ = m.UserReceivedStates.Remove(ctx, user.Id.OpaqueId, w.ssid, shareID)
						continue
					}
					sublogr = sublogr.With().Str("shareid", shareID).Logger()
					if share.IsExpired(s) {
						if err := m.removeShare(ctx, s, false); err != nil {
							sublogr.Error().Err(err).
								Msg("failed to unshare expired share")
						}
						if err := events.Publish(ctx, m.eventStream, events.ShareExpired{
							ShareOwner:     s.GetOwner(),
							ItemID:         s.GetResourceId(),
							ExpiredAt:      time.Unix(int64(s.GetExpiration().GetSeconds()), int64(s.GetExpiration().GetNanos())),
							GranteeUserID:  s.GetGrantee().GetUserId(),
							GranteeGroupID: s.GetGrantee().GetGroupId(),
						}); err != nil {
							sublogr.Error().Err(err).
								Msg("failed to publish share expired event")
						}
						continue
					}

					if share.IsGrantedToUser(s, user) {
						if share.MatchesFiltersWithState(s, state.State, filters) {
							rs := &collaboration.ReceivedShare{
								Share:      s,
								State:      state.State,
								MountPoint: state.MountPoint,
								Hidden:     state.Hidden,
							}
							select {
							case results <- rs:
							case <-ctx.Done():
								return ctx.Err()
							}
						}
					}
				}
			}
			return nil
		})
	}

	// Wait for things to settle down, then close results chan
	go func() {
		_ = g.Wait() // error is checked later
		close(results)
	}()

	rss := []*collaboration.ReceivedShare{}
	for n := range results {
		rss = append(rss, n)
	}

	if err := g.Wait(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return rss, nil
}

// convert must be called in a lock-controlled block.
func (m *Manager) convert(ctx context.Context, userID string, s *collaboration.Share) *collaboration.ReceivedShare {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "convert")
	defer span.End()

	rs := &collaboration.ReceivedShare{
		Share: s,
		State: collaboration.ShareState_SHARE_STATE_PENDING,
	}

	storageID, spaceID, _ := shareid.Decode(s.Id.OpaqueId)

	state, err := m.UserReceivedStates.Get(ctx, userID, storageID+shareid.IDDelimiter+spaceID, s.Id.GetOpaqueId())
	if err == nil && state != nil {
		rs.State = state.State
		rs.MountPoint = state.MountPoint
		rs.Hidden = state.Hidden
	}
	return rs
}

// GetReceivedShare returns the information for a received share.
func (m *Manager) GetReceivedShare(ctx context.Context, ref *collaboration.ShareReference) (*collaboration.ReceivedShare, error) {
	if err := m.initialize(ctx); err != nil {
		return nil, err
	}

	return m.getReceived(ctx, ref)
}

func (m *Manager) getReceived(ctx context.Context, ref *collaboration.ShareReference) (*collaboration.ReceivedShare, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "getReceived")
	defer span.End()
	sublog := appctx.GetLogger(ctx).With().Str("id", ref.GetId().GetOpaqueId()).Str("key", ref.GetKey().String()).Str("driver", "jsoncs3").Str("handler", "getReceived").Logger()

	s, err := m.get(ctx, ref)
	if err != nil {
		return nil, err
	}
	user := ctxpkg.ContextMustGetUser(ctx)
	if user.GetId().GetType() != userv1beta1.UserType_USER_TYPE_SERVICE && !share.IsGrantedToUser(s, user) {
		return nil, errtypes.NotFound(ref.String())
	}
	if share.IsExpired(s) {
		if err := m.removeShare(ctx, s, false); err != nil {
			sublog.Error().Err(err).
				Msg("failed to unshare expired share")
		}
		if err := events.Publish(ctx, m.eventStream, events.ShareExpired{
			ShareOwner:     s.GetOwner(),
			ItemID:         s.GetResourceId(),
			ExpiredAt:      time.Unix(int64(s.GetExpiration().GetSeconds()), int64(s.GetExpiration().GetNanos())),
			GranteeUserID:  s.GetGrantee().GetUserId(),
			GranteeGroupID: s.GetGrantee().GetGroupId(),
		}); err != nil {
			sublog.Error().Err(err).
				Msg("failed to publish share expired event")
		}
	}
	return m.convert(ctx, user.Id.GetOpaqueId(), s), nil
}

// UpdateReceivedShare updates the received share with share state.
func (m *Manager) UpdateReceivedShare(ctx context.Context, receivedShare *collaboration.ReceivedShare, fieldMask *field_mask.FieldMask, forUser *userv1beta1.UserId) (*collaboration.ReceivedShare, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "UpdateReceivedShare")
	defer span.End()

	if err := m.initialize(ctx); err != nil {
		return nil, err
	}

	rs, err := m.getReceived(ctx, &collaboration.ShareReference{Spec: &collaboration.ShareReference_Id{Id: receivedShare.Share.Id}})
	if err != nil {
		return nil, err
	}

	for i := range fieldMask.Paths {
		switch fieldMask.Paths[i] {
		case "state":
			rs.State = receivedShare.State
		case "mount_point":
			rs.MountPoint = receivedShare.MountPoint
		case "hidden":
			rs.Hidden = receivedShare.Hidden
		default:
			return nil, errtypes.NotSupported("updating " + fieldMask.Paths[i] + " is not supported")
		}
	}

	// write back
	u := ctxpkg.ContextMustGetUser(ctx)
	uid := u.GetId().GetOpaqueId()
	if u.GetId().GetType() == userv1beta1.UserType_USER_TYPE_SERVICE {
		uid = forUser.GetOpaqueId()
	}

	err = m.UserReceivedStates.Add(ctx, uid, rs.Share.ResourceId.StorageId+shareid.IDDelimiter+rs.Share.ResourceId.SpaceId, rs)
	if err != nil {
		return nil, err
	}

	return rs, nil
}

func shareIsRoutable(share *collaboration.Share) bool {
	return strings.Contains(share.Id.OpaqueId, shareid.IDDelimiter)
}

func updateShareID(share *collaboration.Share) {
	share.Id.OpaqueId = shareid.Encode(share.ResourceId.StorageId, share.ResourceId.SpaceId, share.Id.OpaqueId)
}

// Load imports shares and received shares from channels (e.g. during migration)
func (m *Manager) Load(ctx context.Context, shareChan <-chan *collaboration.Share, receivedShareChan <-chan share.ReceivedShareWithUser) error {
	log := appctx.GetLogger(ctx)
	if err := m.initialize(ctx); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		for s := range shareChan {
			if s == nil {
				continue
			}
			if !shareIsRoutable(s) {
				updateShareID(s)
			}
			if err := m.Cache.Add(context.Background(), s.GetResourceId().GetStorageId(), s.GetResourceId().GetSpaceId(), s.Id.OpaqueId, s); err != nil {
				log.Error().Err(err).Interface("share", s).Msg("error persisting share")
			} else {
				log.Debug().Str("storageid", s.GetResourceId().GetStorageId()).Str("spaceid", s.GetResourceId().GetSpaceId()).Str("shareid", s.Id.OpaqueId).Msg("imported share")
			}
			if err := m.CreatedCache.Add(ctx, s.GetCreator().GetOpaqueId(), s.Id.OpaqueId); err != nil {
				log.Error().Err(err).Interface("share", s).Msg("error persisting created cache")
			} else {
				log.Debug().Str("creatorid", s.GetCreator().GetOpaqueId()).Str("shareid", s.Id.OpaqueId).Msg("updated created cache")
			}
		}
		wg.Done()
	}()
	go func() {
		for s := range receivedShareChan {
			if s.ReceivedShare != nil {
				if !shareIsRoutable(s.ReceivedShare.GetShare()) {
					updateShareID(s.ReceivedShare.GetShare())
				}
				switch s.ReceivedShare.Share.Grantee.Type {
				case provider.GranteeType_GRANTEE_TYPE_USER:
					if err := m.UserReceivedStates.Add(context.Background(), s.ReceivedShare.GetShare().GetGrantee().GetUserId().GetOpaqueId(), s.ReceivedShare.GetShare().GetResourceId().GetSpaceId(), s.ReceivedShare); err != nil {
						log.Error().Err(err).Interface("received share", s).Msg("error persisting received share for user")
					} else {
						log.Debug().Str("userid", s.ReceivedShare.GetShare().GetGrantee().GetUserId().GetOpaqueId()).Str("spaceid", s.ReceivedShare.GetShare().GetResourceId().GetSpaceId()).Str("shareid", s.ReceivedShare.GetShare().Id.OpaqueId).Msg("updated received share userdata")
					}
				case provider.GranteeType_GRANTEE_TYPE_GROUP:
					if err := m.GroupReceivedCache.Add(context.Background(), s.ReceivedShare.GetShare().GetGrantee().GetGroupId().GetOpaqueId(), s.ReceivedShare.GetShare().GetId().GetOpaqueId()); err != nil {
						log.Error().Err(err).Interface("received share", s).Msg("error persisting received share to group cache")
					} else {
						log.Debug().Str("groupid", s.ReceivedShare.GetShare().GetGrantee().GetGroupId().GetOpaqueId()).Str("shareid", s.ReceivedShare.GetShare().Id.OpaqueId).Msg("updated received share group cache")
					}
				}
			}
		}
		wg.Done()
	}()
	wg.Wait()

	return nil
}

func (m *Manager) purgeSpace(ctx context.Context, id *provider.StorageSpaceId) {
	log := appctx.GetLogger(ctx)
	storageID, spaceID := storagespace.SplitStorageID(id.OpaqueId)

	shares, err := m.Cache.ListSpace(ctx, storageID, spaceID)
	if err != nil {
		log.Error().Err(err).Msg("error listing shares in space")
		return
	}

	// iterate over all shares in the space and remove them
	for _, share := range shares.Shares {
		err := m.removeShare(ctx, share, true)
		if err != nil {
			log.Error().Err(err).Msg("error removing share")
		}
	}

	// remove all shares in the space
	err = m.Cache.PurgeSpace(ctx, storageID, spaceID)
	if err != nil {
		log.Error().Err(err).Msg("error purging space")
	}
}

func (m *Manager) removeShare(ctx context.Context, s *collaboration.Share, skipSpaceCache bool) error {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "removeShare")
	defer span.End()

	eg, ctx := errgroup.WithContext(ctx)
	if !skipSpaceCache {
		eg.Go(func() error {
			storageID, spaceID, _ := shareid.Decode(s.Id.OpaqueId)
			err := m.Cache.Remove(ctx, storageID, spaceID, s.Id.OpaqueId)

			return err
		})
	}

	eg.Go(func() error {
		// remove from created cache
		return m.CreatedCache.Remove(ctx, s.GetCreator().GetOpaqueId(), s.Id.OpaqueId)
	})

	eg.Go(func() error {
		// remove from user received states
		if s.GetGrantee().Type == provider.GranteeType_GRANTEE_TYPE_USER {
			return m.UserReceivedStates.Remove(ctx, s.GetGrantee().GetUserId().GetOpaqueId(), s.GetResourceId().GetStorageId()+shareid.IDDelimiter+s.GetResourceId().GetSpaceId(), s.Id.OpaqueId)
		} else if s.GetGrantee().Type == provider.GranteeType_GRANTEE_TYPE_GROUP {
			return m.GroupReceivedCache.Remove(ctx, s.GetGrantee().GetGroupId().GetOpaqueId(), s.Id.OpaqueId)
		}
		return nil
	})

	return eg.Wait()
}

func (m *Manager) CleanupStaleShares(ctx context.Context) {
	log := appctx.GetLogger(ctx)

	if err := m.initialize(ctx); err != nil {
		return
	}

	// list all shares
	providers, err := m.Cache.All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error listing all shares")
		return
	}

	client, err := m.gatewaySelector.Next()
	if err != nil {
		log.Error().Err(err).Msg("could not get gateway client")
	}

	providers.Range(func(storage string, spaces *providercache.Spaces) bool {
		log.Info().Str("storage", storage).Interface("spaceCount", spaces.Spaces.Count()).Msg("checking storage")

		spaces.Spaces.Range(func(space string, shares *providercache.Shares) bool {
			log.Info().Str("storage", storage).Str("space", space).Interface("shareCount", len(shares.Shares)).Msg("checking space")

			for _, s := range shares.Shares {
				req := &provider.StatRequest{
					Ref: &provider.Reference{ResourceId: s.ResourceId, Path: "."},
				}
				res, err := client.Stat(ctx, req)
				if err != nil {
					log.Error().Err(err).Str("storage", storage).Str("space", space).Msg("could not stat shared resource")
				}
				if res.Status.Code == rpcv1beta1.Code_CODE_NOT_FOUND {
					log.Info().Str("storage", storage).Str("space", space).Msg("shared resource does not exist anymore. cleaning up shares")
					if err := m.removeShare(ctx, s, false); err != nil {
						log.Error().Err(err).Str("storage", storage).Str("space", space).Msg("could not remove share")
					}
				}
			}

			return true
		})

		return true
	})
}
