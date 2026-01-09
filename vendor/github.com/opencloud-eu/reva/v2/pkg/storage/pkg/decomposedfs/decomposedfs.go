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

package decomposedfs

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/jellydator/ttlcache/v2"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	tusd "github.com/tus/tusd/v2/pkg/handler"
	microstore "go-micro.dev/v4/store"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"

	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/logger"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/datatx/metrics"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/aspects"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/lookup"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/options"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/permissions"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/spaceidindex"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/timemanager"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/trashbin"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/tree"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/upload"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/usermapper"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/chunking"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/filelocks"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/templates"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/store"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

type CtxKey int

const (
	CtxKeySpaceGID CtxKey = iota
)

var (
	tracer trace.Tracer

	_registeredEvents = []events.Unmarshaller{
		events.PostprocessingFinished{},
		events.PostprocessingStepFinished{},
		events.RestartPostprocessing{},
	}
)

func init() {
	tracer = otel.Tracer("github.com/cs3org/reva/pkg/storage/pkg/decomposedfs")
}

// Session is the interface that DecomposedfsSession implements. By combining tus.Upload,
// storage.UploadSession and custom functions we can reuse the same struct throughout
// the whole upload lifecycle.
//
// Some functions that are only used by decomposedfs are not yet part of this interface.
// They might be added after more refactoring.
type Session interface {
	tusd.Upload
	storage.UploadSession
	upload.Session
	LockID() string
}

type IDCachingTree interface {
	WarmupIDCache(root string, assimilate, onlyDirty bool) error
}

type SessionStore interface {
	New(ctx context.Context) *upload.DecomposedFsSession
	List(ctx context.Context) ([]*upload.DecomposedFsSession, error)
	Get(ctx context.Context, id string) (*upload.DecomposedFsSession, error)
	Cleanup(ctx context.Context, session upload.Session, revertNodeMetadata, keepUpload, unmarkPostprocessing bool)
}

// Decomposedfs provides the base for decomposed filesystem implementations
type Decomposedfs struct {
	lu           node.PathLookup
	tp           node.Tree
	trashbin     trashbin.Trashbin
	o            *options.Options
	p            permissions.Permissions
	um           usermapper.Mapper
	chunkHandler *chunking.ChunkHandler
	stream       events.Stream
	sessionStore SessionStore

	UserCache       *ttlcache.Cache
	userSpaceIndex  *spaceidindex.Index
	groupSpaceIndex *spaceidindex.Index
	spaceTypeIndex  *spaceidindex.Index

	log *zerolog.Logger
}

// NewDefault returns an instance with default components
func NewDefault(m map[string]interface{}, bs node.Blobstore, es events.Stream, log *zerolog.Logger) (storage.FS, error) {
	if log == nil {
		log = &zerolog.Logger{}
	}

	o, err := options.New(m)
	if err != nil {
		return nil, err
	}

	var lu *lookup.Lookup
	switch o.MetadataBackend {
	case "xattrs":
		lu = lookup.New(metadata.NewXattrsBackend(o.FileMetadataCache), o, &timemanager.Manager{})
	case "messagepack":
		lu = lookup.New(metadata.NewMessagePackBackend(o.FileMetadataCache), o, &timemanager.Manager{})
	default:
		return nil, fmt.Errorf("unknown metadata backend %s, only 'messagepack' or 'xattrs' (default) supported", o.MetadataBackend)
	}

	permissionsSelector, err := pool.PermissionsSelector(o.PermissionsSVC, pool.WithTLSMode(o.PermTLSMode))
	if err != nil {
		return nil, err
	}
	p := permissions.NewPermissions(node.NewPermissions(lu), permissionsSelector)

	tp := tree.New(lu, bs, o, p, store.Create(
		store.Store(o.IDCache.Store),
		store.TTL(o.IDCache.TTL),
		store.Size(o.IDCache.Size),
		microstore.Nodes(o.IDCache.Nodes...),
		microstore.Database(o.IDCache.Database),
		microstore.Table(o.IDCache.Table),
		store.DisablePersistence(o.IDCache.DisablePersistence),
		store.Authentication(o.IDCache.AuthUsername, o.IDCache.AuthPassword),
		store.TLSEnabled(o.IDCache.TLSEnabled),
		store.TLSInsecure(o.IDCache.TLSInsecure),
		store.TLSRootCA(o.IDCache.TLSRootCACertificate),
	), log)

	aspects := aspects.Aspects{
		Lookup:            lu,
		Tree:              tp,
		Permissions:       p,
		EventStream:       es,
		DisableVersioning: o.DisableVersioning,
		Trashbin:          &DecomposedfsTrashbin{},
	}

	return New(o, aspects, log)
}

// New returns an implementation of the storage.FS interface that talks to
// a local filesystem.
func New(o *options.Options, aspects aspects.Aspects, log *zerolog.Logger) (storage.FS, error) {
	if log == nil {
		log = &zerolog.Logger{}
	}

	err := aspects.Tree.Setup()
	if err != nil {
		log.Error().Err(err).Msg("could not setup tree")
		return nil, errors.Wrap(err, "could not setup tree")
	}

	if o.MaxAcquireLockCycles != 0 {
		filelocks.SetMaxLockCycles(o.MaxAcquireLockCycles)
	}

	if o.LockCycleDurationFactor != 0 {
		filelocks.SetLockCycleDurationFactor(o.LockCycleDurationFactor)
	}
	userSpaceIndex := spaceidindex.New(filepath.Join(o.Root, "indexes"), "by-user-id")
	err = userSpaceIndex.Init()
	if err != nil {
		return nil, err
	}
	groupSpaceIndex := spaceidindex.New(filepath.Join(o.Root, "indexes"), "by-group-id")
	err = groupSpaceIndex.Init()
	if err != nil {
		return nil, err
	}
	spaceTypeIndex := spaceidindex.New(filepath.Join(o.Root, "indexes"), "by-type")
	err = spaceTypeIndex.Init()
	if err != nil {
		return nil, err
	}

	if aspects.Trashbin == nil {
		return nil, errors.New("need trashbin")
	}
	// set a null usermapper if we don't have one
	if aspects.UserMapper == nil {
		aspects.UserMapper = &usermapper.NullMapper{}
	}

	fs := &Decomposedfs{
		tp:              aspects.Tree,
		lu:              aspects.Lookup,
		trashbin:        aspects.Trashbin,
		o:               o,
		p:               aspects.Permissions,
		um:              aspects.UserMapper,
		chunkHandler:    chunking.NewChunkHandler(filepath.Join(o.Root, "uploads")),
		stream:          aspects.EventStream,
		UserCache:       ttlcache.NewCache(),
		userSpaceIndex:  userSpaceIndex,
		groupSpaceIndex: groupSpaceIndex,
		spaceTypeIndex:  spaceTypeIndex,
		log:             log,
	}
	fs.sessionStore = upload.NewSessionStore(fs, aspects, o.Root, o.AsyncFileUploads, o.Tokens, log)
	if err = fs.trashbin.Setup(fs); err != nil {
		return nil, err
	}

	if o.AsyncFileUploads {
		if fs.stream == nil {
			log.Error().Msg("need event stream for async file processing")
			return nil, errors.New("need nats for async file processing")
		}

		ch, err := events.Consume(fs.stream, "dcfs", _registeredEvents...)
		if err != nil {
			return nil, err
		}

		if o.Events.NumConsumers <= 0 {
			o.Events.NumConsumers = 1
		}

		for i := 0; i < o.Events.NumConsumers; i++ {
			go fs.Postprocessing(ch)
		}
	}

	return fs, nil
}

// Postprocessing starts the postprocessing result collector
func (fs *Decomposedfs) Postprocessing(ch <-chan events.Event) {
	ctx := context.TODO() // we should pass the trace id in the event and initialize the trace provider here
	ctx, span := tracer.Start(ctx, "Postprocessing")
	defer span.End()
	log := logger.New()
	for event := range ch {
		switch ev := event.Event.(type) {
		case events.PostprocessingFinished:
			sublog := log.With().Str("event", "PostprocessingFinished").Str("uploadid", ev.UploadID).Logger()
			session, err := fs.sessionStore.Get(ctx, ev.UploadID)
			if err != nil {
				sublog.Error().Err(err).Msg("Failed to get upload")
				continue // NOTE: since we can't get the upload, we can't delete the blob
			}

			ctx = session.Context(ctx)

			n, err := session.Node(ctx)
			if err != nil {
				sublog.Error().Err(err).Msg("could not read node")
				continue
			}
			sublog = log.With().Str("spaceid", session.SpaceID()).Str("nodeid", session.NodeID()).Logger()
			if !n.Exists {
				sublog.Debug().Msg("node no longer exists")
				fs.sessionStore.Cleanup(ctx, session, false, false, false)
				continue
			}

			var (
				failed             bool
				revertNodeMetadata bool
				keepUpload         bool
			)
			unmarkPostprocessing := true

			switch ev.Outcome {
			default:
				sublog.Error().Str("outcome", string(ev.Outcome)).Msg("unknown postprocessing outcome - aborting")
				fallthrough
			case events.PPOutcomeAbort:
				failed = true
				revertNodeMetadata = true
				keepUpload = true
				metrics.UploadSessionsAborted.Inc()
			case events.PPOutcomeContinue:
				if err := session.Finalize(ctx); err != nil {
					sublog.Error().Err(err).Msg("could not finalize upload")
					failed = true
					revertNodeMetadata = false
					keepUpload = true
					// keep postprocessing status so the upload is not deleted during housekeeping
					unmarkPostprocessing = false
				} else {
					metrics.UploadSessionsFinalized.Inc()
				}
			case events.PPOutcomeDelete:
				failed = true
				revertNodeMetadata = true
				metrics.UploadSessionsDeleted.Inc()
			}

			getParent := func() *node.Node {
				p, err := n.Parent(ctx)
				if err != nil {
					sublog.Error().Err(err).Msg("could not read parent")
					return nil
				}
				return p
			}

			now := time.Now()
			var parentId *provider.ResourceId
			if failed {
				// if no other upload session is in progress (processing id != session id) or has finished (processing id == "")
				latestSession, err := n.ProcessingID(ctx)
				if err != nil {
					sublog.Error().Err(err).Msg("reading node for session failed")
				}
				if latestSession == session.ID() {
					// propagate reverted sizeDiff after failed postprocessing
					if err := fs.tp.Propagate(ctx, n, -session.SizeDiff()); err != nil {
						sublog.Error().Err(err).Msg("could not propagate tree size change")
					}
				}
			} else if p := getParent(); p != nil {
				parentId = &provider.ResourceId{
					StorageId: session.ProviderID(),
					SpaceId:   session.SpaceID(),
					OpaqueId:  p.ID,
				}
				// update parent tmtime to propagate etag change after successful postprocessing
				_ = p.SetTMTime(ctx, &now)
				if err := fs.tp.Propagate(ctx, p, 0); err != nil {
					sublog.Error().Err(err).Msg("could not propagate etag change")
				}
			}

			fs.sessionStore.Cleanup(ctx, session, revertNodeMetadata, keepUpload, unmarkPostprocessing)

			var isVersion bool
			if session.NodeExists() {
				info, err := session.GetInfo(ctx)
				if err == nil && info.MetaData["versionID"] != "" {
					isVersion = true
				}
			}

			if err := events.Publish(
				ctx,
				fs.stream,
				events.UploadReady{
					UploadID:      ev.UploadID,
					Failed:        failed,
					ExecutingUser: ev.ExecutingUser,
					Filename:      ev.Filename,
					FileRef: &provider.Reference{
						ResourceId: &provider.ResourceId{
							StorageId: session.ProviderID(),
							SpaceId:   session.SpaceID(),
							OpaqueId:  session.SpaceID(),
						},
						Path: utils.MakeRelativePath(filepath.Join(session.Dir(), session.Filename())),
					},
					ParentID:          parentId,
					Timestamp:         utils.TimeToTS(now),
					SpaceOwner:        n.SpaceOwnerOrManager(ctx),
					IsVersion:         isVersion,
					ImpersonatingUser: ev.ImpersonatingUser,
				},
			); err != nil {
				sublog.Error().Err(err).Msg("Failed to publish UploadReady event")
			}
		case events.RestartPostprocessing:
			sublog := log.With().Str("event", "RestartPostprocessing").Str("uploadid", ev.UploadID).Logger()
			session, err := fs.sessionStore.Get(ctx, ev.UploadID)
			if err != nil {
				sublog.Error().Err(err).Msg("Failed to get upload")
				continue
			}
			n, err := session.Node(ctx)
			if err != nil {
				sublog.Error().Err(err).Msg("could not read node")
				continue
			}
			sublog = log.With().Str("spaceid", session.SpaceID()).Str("nodeid", session.NodeID()).Logger()
			s, err := session.URL(ctx)
			if err != nil {
				sublog.Error().Err(err).Msg("could not create url")
				continue
			}

			metrics.UploadSessionsRestarted.Inc()

			// restart postprocessing
			if err := events.Publish(ctx, fs.stream, events.BytesReceived{
				UploadID:      session.ID(),
				URL:           s,
				SpaceOwner:    n.SpaceOwnerOrManager(ctx),
				ExecutingUser: &user.User{Id: &user.UserId{OpaqueId: "postprocessing-restart"}}, // send nil instead?
				ResourceID:    &provider.ResourceId{SpaceId: n.SpaceID, OpaqueId: n.ID},
				Filename:      session.Filename(),
				Filesize:      uint64(session.Size()),
			}); err != nil {
				sublog.Error().Err(err).Msg("Failed to publish BytesReceived event")
			}
		case events.PostprocessingStepFinished:
			sublog := log.With().Str("event", "PostprocessingStepFinished").Str("uploadid", ev.UploadID).Logger()
			if ev.FinishedStep != events.PPStepAntivirus {
				// atm we are only interested in antivirus results
				continue
			}

			res := ev.Result.(events.VirusscanResult)
			if res.ErrorMsg != "" {
				// scan failed somehow
				// Should we handle this here?
				continue
			}
			sublog = log.With().Str("scan_description", res.Description).Bool("infected", res.Infected).Logger()

			var n *node.Node
			switch ev.UploadID {
			case "":
				// uploadid is empty -> this was an on-demand scan
				/* ON DEMAND SCANNING NOT SUPPORTED ATM
				ctx := ctxpkg.ContextSetUser(context.Background(), ev.ExecutingUser)
				ref := &provider.Reference{ResourceId: ev.ResourceID}

				no, err := fs.lu.NodeFromResource(ctx, ref)
				if err != nil {
					log.Error().Err(err).Interface("resourceID", ev.ResourceID).Msg("Failed to get node after scan")
					continue

				}
				n = no
				if ev.Outcome == events.PPOutcomeDelete {
					// antivir wants us to delete the file. We must obey and need to

					// check if there a previous versions existing
					revs, err := fs.ListRevisions(ctx, ref)
					if len(revs) == 0 {
						if err != nil {
							log.Error().Err(err).Interface("resourceID", ev.ResourceID).Msg("Failed to list revisions. Fallback to delete file")
						}

						// no versions -> trash file
						err := fs.Delete(ctx, ref)
						if err != nil {
							log.Error().Err(err).Interface("resourceID", ev.ResourceID).Msg("Failed to delete infected resource")
							continue
						}

						// now purge it from the recycle bin
						if err := fs.PurgeRecycleItem(ctx, &provider.Reference{ResourceId: &provider.ResourceId{SpaceId: n.SpaceID, OpaqueId: n.SpaceID}}, n.ID, "/"); err != nil {
							log.Error().Err(err).Interface("resourceID", ev.ResourceID).Msg("Failed to purge infected resource from trash")
						}

						// remove cache entry in gateway
						fs.cache.RemoveStatContext(ctx, ev.ExecutingUser.GetId(), &provider.ResourceId{SpaceId: n.SpaceID, OpaqueId: n.ID})
						continue
					}

					// we have versions - find the newest
					versions := make(map[uint64]string) // remember all versions - we need them later
					var nv uint64
					for _, v := range revs {
						versions[v.Mtime] = v.Key
						if v.Mtime > nv {
							nv = v.Mtime
						}
					}

					// restore newest version
					if err := fs.RestoreRevision(ctx, ref, versions[nv]); err != nil {
						log.Error().Err(err).Interface("resourceID", ev.ResourceID).Str("revision", versions[nv]).Msg("Failed to restore revision")
						continue
					}

					// now find infected version
					revs, err = fs.ListRevisions(ctx, ref)
					if err != nil {
						log.Error().Err(err).Interface("resourceID", ev.ResourceID).Msg("Error listing revisions after restore")
					}

					for _, v := range revs {
						// we looking for a version that was previously not there
						if _, ok := versions[v.Mtime]; ok {
							continue
						}

						if err := fs.DeleteRevision(ctx, ref, v.Key); err != nil {
							log.Error().Err(err).Interface("resourceID", ev.ResourceID).Str("revision", v.Key).Msg("Failed to delete revision")
						}
					}

					// remove cache entry in gateway
					fs.cache.RemoveStatContext(ctx, ev.ExecutingUser.GetId(), &provider.ResourceId{SpaceId: n.SpaceID, OpaqueId: n.ID})
					continue
				}
				*/
			default:
				// uploadid is not empty -> this is an async upload
				session, err := fs.sessionStore.Get(ctx, ev.UploadID)
				if err != nil {
					sublog.Error().Err(err).Msg("Failed to get upload")
					continue
				}

				n, err = session.Node(ctx)
				if err != nil {
					sublog.Error().Err(err).Msg("Failed to get node after scan")
					continue
				}
				sublog = log.With().Str("spaceid", session.SpaceID()).Str("nodeid", session.NodeID()).Logger()

				session.SetScanData(res.Description, res.Scandate)
				if err := session.Persist(ctx); err != nil {
					sublog.Error().Err(err).Msg("Failed to persist scan results")
				}
			}

			if err := n.SetScanData(ctx, res.Description, res.Scandate); err != nil {
				sublog.Error().Err(err).Msg("Failed to set scan results")
				continue
			}

			metrics.UploadSessionsScanned.Inc()
		default:
			log.Error().Interface("event", ev).Msg("Unknown event")
		}
	}
}

// Shutdown shuts down the storage
func (fs *Decomposedfs) Shutdown(ctx context.Context) error {
	return nil
}

// GetQuota returns the quota available
// TODO Document in the cs3 should we return quota or free space?
func (fs *Decomposedfs) GetQuota(ctx context.Context, ref *provider.Reference) (total uint64, inUse uint64, remaining uint64, err error) {
	ctx, span := tracer.Start(ctx, "GetQuota")
	defer span.End()
	var n *node.Node
	if ref == nil {
		err = errtypes.BadRequest("no space given")
		return 0, 0, 0, err
	}
	if n, err = fs.lu.NodeFromResource(ctx, ref); err != nil {
		return 0, 0, 0, err
	}

	if !n.Exists {
		err = errtypes.NotFound(filepath.Join(n.ParentID, n.Name))
		return 0, 0, 0, err
	}

	rp, err := fs.p.AssemblePermissions(ctx, n)
	switch {
	case err != nil:
		return 0, 0, 0, err
	case !rp.GetQuota && !fs.p.ListAllSpaces(ctx):
		f, _ := storagespace.FormatReference(ref)
		if rp.Stat {
			return 0, 0, 0, errtypes.PermissionDenied(f)
		}
		return 0, 0, 0, errtypes.NotFound(f)
	}

	// FIXME move treesize & quota to fieldmask
	ri, err := n.AsResourceInfo(ctx, rp, []string{"treesize", "quota"}, []string{}, true)
	if err != nil {
		return 0, 0, 0, err
	}

	quotaStr := node.QuotaUnknown
	if ri.Opaque != nil && ri.Opaque.Map != nil && ri.Opaque.Map["quota"] != nil && ri.Opaque.Map["quota"].Decoder == "plain" {
		quotaStr = string(ri.Opaque.Map["quota"].Value)
	}

	return fs.calculateTotalUsedRemaining(quotaStr, ri.Size)
}

func (fs *Decomposedfs) calculateTotalUsedRemaining(quotaStr string, inUse uint64) (uint64, uint64, uint64, error) {
	var err error
	var total uint64

	remaining := uint64(math.MaxUint64)
	switch quotaStr {
	case node.QuotaUncalculated, node.QuotaUnknown:
		// best we can do is return current total
		// TODO indicate unlimited total? -> in opaque data?
	case node.QuotaUnlimited:
		total = 0
	default:
		total, err = strconv.ParseUint(quotaStr, 10, 64)
		if err != nil {
			return 0, 0, 0, err
		}

		switch {
		case total > inUse:
			remaining = total - inUse
		case total <= inUse:
			remaining = 0
		}

	}
	return total, inUse, remaining, nil
}

// CreateHome creates a new home node for the given user
func (fs *Decomposedfs) CreateHome(ctx context.Context) (err error) {
	ctx, span := tracer.Start(ctx, "CreateHome")
	defer span.End()
	if fs.o.UserLayout == "" {
		return errtypes.NotSupported("Decomposedfs: CreateHome() home supported disabled")
	}

	u := ctxpkg.ContextMustGetUser(ctx)
	res, err := fs.CreateStorageSpace(ctx, &provider.CreateStorageSpaceRequest{
		Type:  _spaceTypePersonal,
		Owner: u,
	})
	if err != nil {
		return err
	}
	if res.Status.Code != rpcv1beta1.Code_CODE_OK {
		return errtypes.NewErrtypeFromStatus(res.Status)
	}
	return nil
}

// GetHome is called to look up the home path for a user
// It is NOT supposed to return the internal path but the external path
func (fs *Decomposedfs) GetHome(ctx context.Context) (string, error) {
	ctx, span := tracer.Start(ctx, "GetHome")
	defer span.End()
	if fs.o.UserLayout == "" {
		return "", errtypes.NotSupported("Decomposedfs: GetHome() home supported disabled")
	}
	u := ctxpkg.ContextMustGetUser(ctx)
	layout := templates.WithUser(u, fs.o.UserLayout)
	return filepath.Join(fs.o.Root, layout), nil // TODO use a namespace?
}

// GetPathByID returns the fn pointed by the file id, without the internal namespace
func (fs *Decomposedfs) GetPathByID(ctx context.Context, id *provider.ResourceId) (string, error) {
	ctx, span := tracer.Start(ctx, "GetPathByID")
	defer span.End()
	n, err := fs.lu.NodeFromID(ctx, id)
	if err != nil {
		return "", err
	}
	rp, err := fs.p.AssemblePermissions(ctx, n)
	switch {
	case err != nil:
		return "", err
	case !rp.GetPath:
		f := storagespace.FormatResourceID(id)
		if rp.Stat {
			return "", errtypes.PermissionDenied(f)
		}
		return "", errtypes.NotFound(f)
	}

	hp := func(n *node.Node) bool {
		perms, err := fs.p.AssemblePermissions(ctx, n)
		if err != nil {
			return false
		}
		return perms.GetPath
	}
	return fs.lu.Path(ctx, n, hp)
}

// CreateDir creates the specified directory
func (fs *Decomposedfs) CreateDir(ctx context.Context, ref *provider.Reference) (err error) {
	ctx, span := tracer.Start(ctx, "CreateDir")
	defer span.End()

	name := path.Base(ref.Path)
	if name == "" || name == "." || name == "/" {
		return errtypes.BadRequest("Invalid path: " + ref.Path)
	}

	parentRef := &provider.Reference{
		ResourceId: ref.ResourceId,
		Path:       path.Dir(ref.Path),
	}

	// verify parent exists
	var n *node.Node
	if n, err = fs.lu.NodeFromResource(ctx, parentRef); err != nil {
		if e, ok := err.(errtypes.NotFound); ok {
			return errtypes.PreconditionFailed(e.Error())
		}
		return
	}
	// TODO check if user has access to root / space
	if !n.Exists {
		return errtypes.PreconditionFailed(parentRef.Path)
	}

	rp, err := fs.p.AssemblePermissions(ctx, n)
	switch {
	case err != nil:
		return err
	case !rp.CreateContainer:
		f, _ := storagespace.FormatReference(ref)
		if rp.Stat {
			return errtypes.PermissionDenied(f)
		}
		return errtypes.NotFound(f)
	}

	// Set space owner in context
	storagespace.ContextSendSpaceOwnerID(ctx, n.SpaceOwnerOrManager(ctx))

	// check lock
	if err := n.CheckLock(ctx); err != nil {
		return err
	}

	// verify child does not exist, yet
	if n, err = n.Child(ctx, name); err != nil {
		return
	}
	if n.Exists {
		return errtypes.AlreadyExists(ref.Path)
	}

	if err = fs.tp.CreateDir(ctx, n); err != nil {
		return
	}

	return
}

// TouchFile as defined in the storage.FS interface
func (fs *Decomposedfs) TouchFile(ctx context.Context, ref *provider.Reference, markprocessing bool, mtime string) error {
	ctx, span := tracer.Start(ctx, "TouchFile")
	defer span.End()
	parentRef := &provider.Reference{
		ResourceId: ref.ResourceId,
		Path:       path.Dir(ref.Path),
	}

	// verify parent exists
	parent, err := fs.lu.NodeFromResource(ctx, parentRef)
	if err != nil {
		return errtypes.InternalError(err.Error())
	}
	if !parent.Exists {
		return errtypes.NotFound(parentRef.Path)
	}

	n, err := fs.lu.NodeFromResource(ctx, ref)
	if err != nil {
		return errtypes.InternalError(err.Error())
	}

	rp, err := fs.p.AssemblePermissions(ctx, n)
	switch {
	case err != nil:
		return err
	case !rp.InitiateFileUpload:
		f, _ := storagespace.FormatReference(ref)
		if rp.Stat {
			return errtypes.PermissionDenied(f)
		}
		return errtypes.NotFound(f)
	}

	// Set space owner in context
	storagespace.ContextSendSpaceOwnerID(ctx, n.SpaceOwnerOrManager(ctx))

	// check lock
	if err := n.CheckLock(ctx); err != nil {
		return err
	}
	return fs.tp.TouchFile(ctx, n, markprocessing, mtime)
}

// CreateReference creates a reference as a node folder with the target stored in extended attributes
// There is no difference between the /Shares folder and normal nodes because the storage is not supposed to be accessible
// without the storage provider. In effect everything is a shadow namespace.
// To mimic the eos and owncloud driver we only allow references as children of the "/Shares" folder
// FIXME: This comment should explain briefly what a reference is in this context.
func (fs *Decomposedfs) CreateReference(ctx context.Context, p string, targetURI *url.URL) (err error) {
	return errtypes.NotSupported("not implemented")
}

// Move moves a resource from one reference to another
func (fs *Decomposedfs) Move(ctx context.Context, oldRef, newRef *provider.Reference) (err error) {
	ctx, span := tracer.Start(ctx, "Move")
	defer span.End()
	var oldNode, newNode *node.Node
	if oldNode, err = fs.lu.NodeFromResource(ctx, oldRef); err != nil {
		return
	}

	if !oldNode.Exists {
		err = errtypes.NotFound(filepath.Join(oldNode.ParentID, oldNode.Name))
		return
	}

	orp, err := fs.p.AssemblePermissions(ctx, oldNode)
	switch {
	case err != nil:
		return err
	case !orp.Move:
		f, _ := storagespace.FormatReference(oldRef)
		if orp.Stat {
			return errtypes.PermissionDenied(f)
		}
		return errtypes.NotFound(f)
	}

	if newNode, err = fs.lu.NodeFromResource(ctx, newRef); err != nil {
		return
	}
	if newNode.Exists {
		err = errtypes.AlreadyExists(filepath.Join(newNode.ParentID, newNode.Name))
		return
	}

	nrp, err := fs.p.AssemblePermissions(ctx, newNode)
	switch {
	case err != nil:
		return err
	case oldNode.IsDir(ctx) && !nrp.CreateContainer:
		f, _ := storagespace.FormatReference(newRef)
		if nrp.Stat {
			return errtypes.PermissionDenied(f)
		}
		return errtypes.NotFound(f)
	case !oldNode.IsDir(ctx) && !nrp.InitiateFileUpload:
		f, _ := storagespace.FormatReference(newRef)
		if nrp.Stat {
			return errtypes.PermissionDenied(f)
		}
		return errtypes.NotFound(f)
	}

	// Set space owner in context
	storagespace.ContextSendSpaceOwnerID(ctx, newNode.SpaceOwnerOrManager(ctx))

	// check lock on source
	if err := oldNode.CheckLock(ctx); err != nil {
		return err
	}

	if err := fs.tp.Move(ctx, oldNode, newNode); err != nil {
		return err
	}

	fs.publishEvent(ctx, fs.moveEvent(ctx, oldRef, newRef, oldNode, newNode, orp, nrp))

	return nil
}

// GetMD returns the metadata for the specified resource
func (fs *Decomposedfs) GetMD(ctx context.Context, ref *provider.Reference, mdKeys []string, fieldMask []string) (ri *provider.ResourceInfo, err error) {
	ctx, span := tracer.Start(ctx, "GetMD")
	defer span.End()
	var node *node.Node
	if node, err = fs.lu.NodeFromResource(ctx, ref); err != nil {
		return
	}

	if !node.Exists {
		err = errtypes.NotFound(filepath.Join(node.ParentID, node.Name))
		return
	}

	rp, err := fs.p.AssemblePermissions(ctx, node)
	switch {
	case err != nil:
		return nil, err
	case !rp.Stat:
		f, _ := storagespace.FormatReference(ref)
		return nil, errtypes.NotFound(f)
	}

	md, err := node.AsResourceInfo(ctx, rp, mdKeys, fieldMask, utils.IsRelativeReference(ref))
	if err != nil {
		return nil, err
	}

	addSpace := len(fieldMask) == 0
	for _, p := range fieldMask {
		if p == "space" || p == "*" {
			addSpace = true
			break
		}
	}
	if addSpace {
		if md.Space, err = fs.StorageSpaceFromNode(ctx, node, true); err != nil {
			return nil, err
		}
	}

	return md, nil
}

// ListFolder returns a list of resources in the specified folder
func (fs *Decomposedfs) ListFolder(ctx context.Context, ref *provider.Reference, mdKeys []string, fieldMask []string) ([]*provider.ResourceInfo, error) {
	ctx, span := tracer.Start(ctx, "ListFolder")
	defer span.End()
	n, err := fs.lu.NodeFromResource(ctx, ref)
	if err != nil {
		return nil, err
	}

	if !n.Exists {
		return nil, errtypes.NotFound(filepath.Join(n.ParentID, n.Name))
	}

	rp, err := fs.p.AssemblePermissions(ctx, n)
	switch {
	case err != nil:
		return nil, err
	case !rp.ListContainer:
		f, _ := storagespace.FormatReference(ref)
		if rp.Stat {
			return nil, errtypes.PermissionDenied(f)
		}
		return nil, errtypes.NotFound(f)
	}

	children, err := fs.tp.ListFolder(ctx, n)
	if err != nil {
		return nil, err
	}

	numWorkers := fs.o.MaxConcurrency
	if len(children) < numWorkers {
		numWorkers = len(children)
	}
	work := make(chan *node.Node, len(children))
	results := make(chan *provider.ResourceInfo, len(children))

	g, ctx := errgroup.WithContext(ctx)

	// Distribute work
	g.Go(func() error {
		defer close(work)
		for _, child := range children {
			select {
			case work <- child:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})

	// Spawn workers that'll concurrently work the queue
	for i := 0; i < numWorkers; i++ {
		g.Go(func() error {
			for child := range work {
				np := rp
				// add this childs permissions
				pset, _ := child.PermissionSet(ctx)
				node.AddPermissions(np, pset)
				ri, err := child.AsResourceInfo(ctx, np, mdKeys, fieldMask, utils.IsRelativeReference(ref))
				if err != nil {
					return errtypes.InternalError(err.Error())
				}
				select {
				case results <- ri:
				case <-ctx.Done():
					return ctx.Err()
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

	finfos := make([]*provider.ResourceInfo, len(children))
	i := 0
	for fi := range results {
		finfos[i] = fi
		i++
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return finfos, nil
}

// Delete deletes the specified resource
func (fs *Decomposedfs) Delete(ctx context.Context, ref *provider.Reference) (err error) {
	ctx, span := tracer.Start(ctx, "Delete")
	defer span.End()
	var node *node.Node
	if node, err = fs.lu.NodeFromResource(ctx, ref); err != nil {
		return
	}
	if !node.Exists {
		return errtypes.NotFound(filepath.Join(node.ParentID, node.Name))
	}

	rp, err := fs.p.AssemblePermissions(ctx, node)
	switch {
	case err != nil:
		return err
	case !rp.Delete:
		f, _ := storagespace.FormatReference(ref)
		if rp.Stat {
			return errtypes.PermissionDenied(f)
		}
		return errtypes.NotFound(f)
	}

	// Set space owner in context
	storagespace.ContextSendSpaceOwnerID(ctx, node.SpaceOwnerOrManager(ctx))

	if err := node.CheckLock(ctx); err != nil {
		return err
	}

	return fs.tp.Delete(ctx, node)
}

// Download returns a reader to the specified resource
func (fs *Decomposedfs) Download(ctx context.Context, ref *provider.Reference, openReaderFunc func(md *provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error) {
	ctx, span := tracer.Start(ctx, "Download")
	defer span.End()
	// check if we are trying to download a revision
	// TODO the CS3 api should allow initiating a revision download
	if ref.ResourceId != nil && strings.Contains(ref.ResourceId.OpaqueId, node.RevisionIDDelimiter) {
		return fs.DownloadRevision(ctx, ref, ref.ResourceId.OpaqueId, openReaderFunc)
	}

	n, err := fs.lu.NodeFromResource(ctx, ref)
	if err != nil {
		return nil, nil, err
	}

	if !n.Exists {
		err = errtypes.NotFound(filepath.Join(n.ParentID, n.Name))
		return nil, nil, err
	}

	rp, err := fs.p.AssemblePermissions(ctx, n)
	switch {
	case err != nil:
		return nil, nil, err
	case !rp.InitiateFileDownload:
		f, _ := storagespace.FormatReference(ref)
		if rp.Stat {
			return nil, nil, errtypes.PermissionDenied(f)
		}
		return nil, nil, errtypes.NotFound(f)
	}

	ri, err := n.AsResourceInfo(ctx, rp, nil, []string{"size", "mimetype", "etag"}, true)
	if err != nil {
		return nil, nil, err
	}
	var reader io.ReadCloser
	if openReaderFunc(ri) {
		reader, err = fs.tp.ReadBlob(n)
		if err != nil {
			return nil, nil, errors.Wrap(err, "Decomposedfs: error download blob '"+n.ID+"'")
		}
	}
	return ri, reader, nil
}

// GetLock returns an existing lock on the given reference
func (fs *Decomposedfs) GetLock(ctx context.Context, ref *provider.Reference) (*provider.Lock, error) {
	ctx, span := tracer.Start(ctx, "GetLock")
	defer span.End()
	node, err := fs.lu.NodeFromResource(ctx, ref)
	if err != nil {
		return nil, errors.Wrap(err, "Decomposedfs: error resolving ref")
	}

	if !node.Exists {
		err = errtypes.NotFound(filepath.Join(node.ParentID, node.Name))
		return nil, err
	}

	rp, err := fs.p.AssemblePermissions(ctx, node)
	switch {
	case err != nil:
		return nil, err
	case !rp.InitiateFileDownload:
		f, _ := storagespace.FormatReference(ref)
		if rp.Stat {
			return nil, errtypes.PermissionDenied(f)
		}
		return nil, errtypes.NotFound(f)
	}

	return node.ReadLock(ctx, false)
}

// SetLock puts a lock on the given reference
func (fs *Decomposedfs) SetLock(ctx context.Context, ref *provider.Reference, lock *provider.Lock) error {
	ctx, span := tracer.Start(ctx, "SetLock")
	defer span.End()
	node, err := fs.lu.NodeFromResource(ctx, ref)
	if err != nil {
		return errors.Wrap(err, "Decomposedfs: error resolving ref")
	}

	if !node.Exists {
		return errtypes.NotFound(filepath.Join(node.ParentID, node.Name))
	}

	rp, err := fs.p.AssemblePermissions(ctx, node)
	switch {
	case err != nil:
		return err
	case !rp.InitiateFileUpload:
		f, _ := storagespace.FormatReference(ref)
		if rp.Stat {
			return errtypes.PermissionDenied(f)
		}
		return errtypes.NotFound(f)
	}

	return node.SetLock(ctx, lock)
}

// RefreshLock refreshes an existing lock on the given reference
func (fs *Decomposedfs) RefreshLock(ctx context.Context, ref *provider.Reference, lock *provider.Lock, existingLockID string) error {
	ctx, span := tracer.Start(ctx, "RefreshLock")
	defer span.End()
	if lock.LockId == "" {
		return errtypes.BadRequest("missing lockid")
	}

	node, err := fs.lu.NodeFromResource(ctx, ref)
	if err != nil {
		return errors.Wrap(err, "Decomposedfs: error resolving ref")
	}

	if !node.Exists {
		return errtypes.NotFound(filepath.Join(node.ParentID, node.Name))
	}

	rp, err := fs.p.AssemblePermissions(ctx, node)
	switch {
	case err != nil:
		return err
	case !rp.InitiateFileUpload:
		f, _ := storagespace.FormatReference(ref)
		if rp.Stat {
			return errtypes.PermissionDenied(f)
		}
		return errtypes.NotFound(f)
	}

	return node.RefreshLock(ctx, lock, existingLockID)
}

// Unlock removes an existing lock from the given reference
func (fs *Decomposedfs) Unlock(ctx context.Context, ref *provider.Reference, lock *provider.Lock) error {
	ctx, span := tracer.Start(ctx, "Unlock")
	defer span.End()
	if lock.LockId == "" {
		return errtypes.BadRequest("missing lockid")
	}

	node, err := fs.lu.NodeFromResource(ctx, ref)
	if err != nil {
		return errors.Wrap(err, "Decomposedfs: error resolving ref")
	}

	if !node.Exists {
		return errtypes.NotFound(filepath.Join(node.ParentID, node.Name))
	}

	rp, err := fs.p.AssemblePermissions(ctx, node)
	switch {
	case err != nil:
		return err
	case !rp.InitiateFileUpload: // TODO do we need a dedicated permission?
		f, _ := storagespace.FormatReference(ref)
		if rp.Stat {
			return errtypes.PermissionDenied(f)
		}
		return errtypes.NotFound(f)
	}

	return node.Unlock(ctx, lock)
}

func (fs *Decomposedfs) ListRecycle(ctx context.Context, space *provider.Reference, key, relativePath string) ([]*provider.RecycleItem, error) {
	_, span := tracer.Start(ctx, "ListRecycle")
	defer span.End()

	spaceID := space.GetResourceId().GetSpaceId()
	if spaceID == "" {
		return nil, errtypes.BadRequest("missing space reference, needs a space id")
	}
	if key == "" && relativePath != "" {
		return nil, errtypes.BadRequest("key is required when navigating with a path")
	}

	// check permissions
	trashnode, err := fs.lu.NodeFromSpaceID(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	rp, err := fs.p.AssembleTrashPermissions(ctx, trashnode)
	switch {
	case err != nil:
		return nil, err
	case !rp.ListRecycle:
		if rp.Stat {
			return nil, errtypes.PermissionDenied(key)
		}
		return nil, errtypes.NotFound(key)
	}

	return fs.trashbin.ListRecycle(ctx, spaceID, key, relativePath)
}

// RestoreRecycleItem restores a recycle item
// The ref is used to determine the space
// The key is used to determine the recycle item. It is the node id of the recycle item, returned by ListRecycle
// The relativePath is the path is the path relative to the recycle item in case a child should be restored
// The restoreRef is the reference where the item should be restored to
func (fs *Decomposedfs) RestoreRecycleItem(ctx context.Context, space *provider.Reference, key, relativePath string, restoreRef *provider.Reference) error {
	_, span := tracer.Start(ctx, "RestoreRecycleItem")
	defer span.End()

	spaceID := space.GetResourceId().GetSpaceId()
	if spaceID == "" {
		return errtypes.BadRequest("missing space reference, needs a space id")
	}
	if key == "" && relativePath != "" {
		return errtypes.BadRequest("key is required when navigating with a path")
	}

	trashItem := &provider.ResourceId{
		SpaceId:  spaceID,
		OpaqueId: key,
	}

	restoreBaseNode, err := fs.lu.NodeFromID(ctx, trashItem)
	if err != nil {
		return err
	}

	// check permissions of deleted node
	rp, err := fs.p.AssembleTrashPermissions(ctx, restoreBaseNode)
	switch {
	case err != nil:
		return err
	case !rp.RestoreRecycleItem:
		if rp.Stat {
			return errtypes.PermissionDenied(key)
		}
		return errtypes.NotFound(key)
	}

	restoredNode, err := fs.trashbin.RestoreRecycleItem(ctx, spaceID, key, relativePath, restoreRef)
	if err != nil {
		return err
	}

	var sizeDiff int64
	if restoredNode.IsDir(ctx) {
		treeSize, err := restoredNode.GetTreeSize(ctx)
		if err != nil {
			return err
		}
		sizeDiff = int64(treeSize)

		// Warmup posix IDCache if restored path is a directory
		if cachingTree, ok := fs.tp.(IDCachingTree); ok {
			_ = cachingTree.WarmupIDCache(restoredNode.InternalPath(), false, false)
		}
	} else {
		sizeDiff = restoredNode.Blobsize
	}

	return fs.tp.Propagate(ctx, restoredNode, sizeDiff)
}

func (fs *Decomposedfs) PurgeRecycleItem(ctx context.Context, space *provider.Reference, key, relativePath string) error {
	_, span := tracer.Start(ctx, "PurgeRecycleItem")
	defer span.End()

	spaceID := space.GetResourceId().GetSpaceId()
	if spaceID == "" {
		return errtypes.BadRequest("missing space reference, needs a space id")
	}
	if key == "" && relativePath != "" {
		return errtypes.BadRequest("key is required when navigating with a path")
	}

	trashItem := &provider.ResourceId{
		SpaceId:  spaceID,
		OpaqueId: key,
	}

	trashBaseNode, err := fs.lu.NodeFromID(ctx, trashItem)
	if err != nil {
		return err
	}

	// check permissions of deleted node
	rp, err := fs.p.AssembleTrashPermissions(ctx, trashBaseNode)
	switch {
	case err != nil:
		return err
	case !rp.PurgeRecycle:
		if rp.Stat {
			return errtypes.PermissionDenied(key)
		}
		return errtypes.NotFound(key)
	}

	return fs.trashbin.PurgeRecycleItem(ctx, spaceID, key, relativePath)
}

func (fs *Decomposedfs) EmptyRecycle(ctx context.Context, space *provider.Reference) error {
	_, span := tracer.Start(ctx, "EmptyRecycle")
	defer span.End()

	spaceID := space.GetResourceId().GetSpaceId()
	if spaceID == "" {
		return errtypes.BadRequest("missing space reference, needs a space id")
	}

	trashItem := &provider.ResourceId{
		SpaceId:  spaceID,
		OpaqueId: spaceID,
	}
	trashBaseNode, err := fs.lu.NodeFromID(ctx, trashItem)
	if err != nil {
		return err
	}
	// check permissions of space
	rp, err := fs.p.AssembleTrashPermissions(ctx, trashBaseNode)
	switch {
	case err != nil:
		return err
	case !rp.ListRecycle && !rp.PurgeRecycle:
		if rp.Stat {
			return errtypes.PermissionDenied(spaceID)
		}
		return errtypes.NotFound(spaceID)
	}
	return fs.trashbin.EmptyRecycle(ctx, spaceID)
}

func (fs *Decomposedfs) getNodePath(ctx context.Context, n *node.Node, perms *provider.ResourcePermissions) (string, error) {
	hp := func(n *node.Node) bool {
		return perms.GetGetPath()
	}
	return fs.lu.Path(ctx, n, hp)
}

func (fs *Decomposedfs) refFromNode(ctx context.Context, n *node.Node, storageId string, perms *provider.ResourcePermissions) (*provider.Reference, error) {
	var err error
	if perms == nil {
		perms, err = fs.p.AssemblePermissions(ctx, n)
		if err != nil {
			return nil, err
		}
	}
	path, err := fs.getNodePath(ctx, n, perms)
	if err != nil {
		return nil, err
	}
	return &provider.Reference{
		ResourceId: &provider.ResourceId{
			StorageId: storageId,
			OpaqueId:  n.SpaceID,
			SpaceId:   n.SpaceID,
		},
		Path: path,
	}, nil
}
