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

package upload

import (
	"context"
	"encoding/json"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/google/uuid"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/decomposedfs/aspects"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/decomposedfs/metadata"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/decomposedfs/options"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/decomposedfs/usermapper"
	"github.com/pkg/errors"
	"github.com/rogpeppe/go-internal/lockedfile"
	"github.com/rs/zerolog"
	tusd "github.com/tus/tusd/v2/pkg/handler"
)

var _idRegexp = regexp.MustCompile(".*/([^/]+).info")

// PermissionsChecker defines an interface for checking permissions on a Node
type PermissionsChecker interface {
	AssemblePermissions(ctx context.Context, n *node.Node) (ap provider.ResourcePermissions, err error)
}

// OcisStore manages upload sessions
type OcisStore struct {
	fs                storage.FS
	lu                node.PathLookup
	tp                node.Tree
	um                usermapper.Mapper
	root              string
	pub               events.Publisher
	async             bool
	tknopts           options.TokenOptions
	disableVersioning bool
	log               *zerolog.Logger
}

// NewSessionStore returns a new OcisStore
func NewSessionStore(fs storage.FS, aspects aspects.Aspects, root string, async bool, tknopts options.TokenOptions, log *zerolog.Logger) *OcisStore {
	return &OcisStore{
		fs:                fs,
		lu:                aspects.Lookup,
		tp:                aspects.Tree,
		root:              root,
		pub:               aspects.EventStream,
		async:             async,
		tknopts:           tknopts,
		disableVersioning: aspects.DisableVersioning,
		um:                aspects.UserMapper,
		log:               log,
	}
}

// New returns a new upload session
func (store OcisStore) New(ctx context.Context) *OcisSession {
	return &OcisSession{
		store: store,
		info: tusd.FileInfo{
			ID: uuid.New().String(),
			Storage: map[string]string{
				"Type": "OCISStore",
			},
			MetaData: tusd.MetaData{},
		},
	}
}

// List lists all upload sessions
func (store OcisStore) List(ctx context.Context) ([]*OcisSession, error) {
	uploads := []*OcisSession{}
	infoFiles, err := filepath.Glob(filepath.Join(store.root, "uploads", "*.info"))
	if err != nil {
		return nil, err
	}

	for _, info := range infoFiles {
		id := strings.TrimSuffix(filepath.Base(info), filepath.Ext(info))
		progress, err := store.Get(ctx, id)
		if err != nil {
			appctx.GetLogger(ctx).Error().Interface("path", info).Msg("Decomposedfs: could not getUploadSession")
			continue
		}

		uploads = append(uploads, progress)
	}
	return uploads, nil
}

// Get returns the upload session for the given upload id
func (store OcisStore) Get(ctx context.Context, id string) (*OcisSession, error) {
	sessionPath := sessionPath(store.root, id)
	match := _idRegexp.FindStringSubmatch(sessionPath)
	if len(match) < 2 {
		return nil, fmt.Errorf("invalid upload path")
	}

	session := OcisSession{
		store: store,
		info:  tusd.FileInfo{},
	}
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		// handle stale NFS file handles that can occur when the file is deleted betwenn the ATTR and FOPEN call of os.ReadFile
		if pathErr, ok := err.(*os.PathError); ok && pathErr.Err == syscall.ESTALE {
			appctx.GetLogger(ctx).Info().Str("session", id).Err(err).Msg("treating stale file handle as not found")
			err = tusd.ErrNotFound
		}
		if errors.Is(err, iofs.ErrNotExist) {
			// Interpret os.ErrNotExist as 404 Not Found
			err = tusd.ErrNotFound
		}
		return nil, err
	}

	if err := json.Unmarshal(data, &session.info); err != nil {
		return nil, err
	}

	stat, err := os.Stat(session.binPath())
	if err != nil {
		if os.IsNotExist(err) {
			// Interpret os.ErrNotExist as 404 Not Found
			err = tusd.ErrNotFound
		}
		return nil, err
	}

	session.info.Offset = stat.Size()

	return &session, nil
}

// Session is the interface used by the Cleanup call
type Session interface {
	ID() string
	Node(ctx context.Context) (*node.Node, error)
	Context(ctx context.Context) context.Context
	Cleanup(revertNodeMetadata, cleanBin, cleanInfo bool)
}

// CreateNodeForUpload will create the target node for the Upload
// TODO move this to the node package as NodeFromUpload?
// should we in InitiateUpload create the node first? and then the upload?
func (store OcisStore) CreateNodeForUpload(ctx context.Context, session *OcisSession, initAttrs node.Attributes) (*node.Node, error) {
	ctx, span := tracer.Start(session.Context(ctx), "CreateNodeForUpload")
	defer span.End()
	n := node.New(
		session.SpaceID(),
		session.NodeID(),
		session.NodeParentID(),
		session.Filename(),
		session.Size(),
		session.ID(),
		provider.ResourceType_RESOURCE_TYPE_FILE,
		nil,
		store.lu,
	)
	var err error
	n.SpaceRoot, err = node.ReadNode(ctx, store.lu, session.SpaceID(), session.SpaceID(), false, nil, false)
	if err != nil {
		return nil, err
	}

	// check lock
	if err := n.CheckLock(ctx); err != nil {
		return nil, err
	}

	var unlock metadata.UnlockFunc
	if session.NodeExists() { // TODO this is wrong. The node should be created when the upload starts, the revisions should be created independently of the node
		// we do not need to propagate a change when a node is created, only when the upload is ready.
		// that still creates problems for desktop clients because if another change causes propagation it will detects an empty file
		// so the first upload has to point to the first revision with the expected size. The file cannot be downloaded, but it can be overwritten (which will create a new revision and make the node reflect the latest revision)
		// any finished postprocessing will not affect the node metadata.
		// *thinking* but then initializing an upload will lock the file until the upload has finished. That sucks.
		// so we have to check if the node has been created meanwhile (well, only in case the upload does not know the nodeid ... or the NodeExists array that is checked by session.NodeExists())
		// FIXME look at the disk again to see if the file has been created in between, or just try initializing a new node and do the update existing node as a fallback. <- the latter!

		unlock, err = store.updateExistingNode(ctx, session, n, session.SpaceID(), uint64(session.Size()))
		if err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Msg("failed to update existing node")
		}
	} else {
		if c, ok := store.lu.(node.IDCacher); ok {
			err := c.CacheID(ctx, n.SpaceID, n.ID, filepath.Join(n.ParentPath(), n.Name))
			if err != nil {
				appctx.GetLogger(ctx).Error().Err(err).Msg("failed to cache id")
			}
		}

		unlock, err = store.tp.InitNewNode(ctx, n, uint64(session.Size()))
		if err != nil {
			appctx.GetLogger(ctx).Error().Str("path", n.InternalPath()).Err(err).Msg("failed to init new node")
		}
		session.info.MetaData["sizeDiff"] = strconv.FormatInt(session.Size(), 10)
	}
	defer func() {
		if unlock == nil {
			appctx.GetLogger(ctx).Info().Msg("did not get a unlockfunc, not unlocking")
			return
		}

		if err := unlock(); err != nil {
			appctx.GetLogger(ctx).Error().Err(err).Str("nodeid", n.ID).Str("parentid", n.ParentID).Msg("could not close lock")
		}
	}()
	if err != nil {
		return nil, err
	}

	// overwrite technical information
	initAttrs.SetString(prefixes.IDAttr, n.ID)
	initAttrs.SetInt64(prefixes.TypeAttr, int64(provider.ResourceType_RESOURCE_TYPE_FILE))
	initAttrs.SetString(prefixes.ParentidAttr, n.ParentID)
	initAttrs.SetString(prefixes.NameAttr, n.Name)
	initAttrs.SetString(prefixes.BlobIDAttr, n.BlobID)
	initAttrs.SetInt64(prefixes.BlobsizeAttr, n.Blobsize)
	initAttrs.SetString(prefixes.StatusPrefix, node.ProcessingStatus+session.ID())

	// set mtime on the new node
	mtime := time.Now()
	if !session.MTime().IsZero() {
		// overwrite mtime if requested
		mtime = session.MTime()
	}
	err = store.lu.TimeManager().OverrideMtime(ctx, n, &initAttrs, mtime)
	if err != nil {
		return nil, errors.Wrap(err, "Decomposedfs: failed to set the mtime")
	}

	// update node metadata with new blobid etc
	err = n.SetXattrsWithContext(ctx, initAttrs, false)
	if err != nil {
		return nil, errors.Wrap(err, "Decomposedfs: could not write metadata")
	}

	err = store.um.RunInBaseScope(func() error {
		return session.Persist(ctx)
	})
	if err != nil {
		return nil, err
	}

	return n, nil
}

func (store OcisStore) updateExistingNode(ctx context.Context, session *OcisSession, n *node.Node, spaceID string, fsize uint64) (metadata.UnlockFunc, error) {
	_, span := tracer.Start(ctx, "updateExistingNode")
	defer span.End()
	targetPath := n.InternalPath()

	// write lock existing node before reading any metadata
	f, err := lockedfile.OpenFile(store.lu.MetadataBackend().LockfilePath(targetPath), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}

	unlock := func() error {
		// NOTE: to prevent stale NFS file handles do not remove lock file!
		return f.Close()
	}

	old, _ := node.ReadNode(ctx, store.lu, spaceID, n.ID, false, nil, false)
	if _, err := node.CheckQuota(ctx, n.SpaceRoot, true, uint64(old.Blobsize), fsize); err != nil {
		return unlock, err
	}

	oldNodeMtime, err := old.GetMTime(ctx)
	if err != nil {
		return unlock, err
	}
	oldNodeEtag, err := node.CalculateEtag(old.ID, oldNodeMtime)
	if err != nil {
		return unlock, err
	}

	// When the if-match header was set we need to check if the
	// etag still matches before finishing the upload.
	if session.HeaderIfMatch() != "" && session.HeaderIfMatch() != oldNodeEtag {
		return unlock, errtypes.Aborted("etag mismatch")
	}

	// When the if-none-match header was set we need to check if any of the
	// etags matches before finishing the upload.
	if session.HeaderIfNoneMatch() != "" {
		if session.HeaderIfNoneMatch() == "*" {
			return unlock, errtypes.Aborted("etag mismatch, resource exists")
		}
		for _, ifNoneMatchTag := range strings.Split(session.HeaderIfNoneMatch(), ",") {
			if ifNoneMatchTag == oldNodeEtag {
				return unlock, errtypes.Aborted("etag mismatch")
			}
		}
	}

	// When the if-unmodified-since header was set we need to check if the
	// etag still matches before finishing the upload.
	if session.HeaderIfUnmodifiedSince() != "" {
		ifUnmodifiedSince, err := time.Parse(time.RFC3339Nano, session.HeaderIfUnmodifiedSince())
		if err != nil {
			return unlock, errtypes.InternalError(fmt.Sprintf("failed to parse if-unmodified-since time: %s", err))
		}

		if oldNodeMtime.After(ifUnmodifiedSince) {
			return unlock, errtypes.Aborted("if-unmodified-since mismatch")
		}
	}

	if !store.disableVersioning {
		versionPath := session.store.lu.InternalPath(spaceID, n.ID+node.RevisionIDDelimiter+oldNodeMtime.UTC().Format(time.RFC3339Nano))

		// create version node
		_, err := os.OpenFile(versionPath, os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			if !errors.Is(err, os.ErrExist) {
				return unlock, err
			}

			// a revision with this mtime does already exist.
			// If the blobs are the same we can just delete the old one
			if err := validateChecksums(ctx, old, session, versionPath); err != nil {
				return unlock, err
			}

			// delete old blob
			bID, _, err := session.store.lu.ReadBlobIDAndSizeAttr(ctx, versionPath, nil)
			if err != nil {
				return unlock, err
			}
			if err := session.store.tp.DeleteBlob(&node.Node{BlobID: bID, SpaceID: n.SpaceID}); err != nil {
				return unlock, err
			}

			// clean revision file
			span.AddEvent("os.Create")
			if _, err := os.Create(versionPath); err != nil {
				return unlock, err
			}
		}

		// copy blob metadata to version node
		if err := store.lu.CopyMetadataWithSourceLock(ctx, targetPath, versionPath, func(attributeName string, value []byte) (newValue []byte, copy bool) {
			return value, strings.HasPrefix(attributeName, prefixes.ChecksumPrefix) ||
				attributeName == prefixes.TypeAttr ||
				attributeName == prefixes.BlobIDAttr ||
				attributeName == prefixes.BlobsizeAttr ||
				attributeName == prefixes.MTimeAttr
		}, f, true); err != nil {
			return unlock, err
		}
		session.info.MetaData["versionsPath"] = versionPath
		// keep mtime from previous version
		span.AddEvent("os.Chtimes")
		if err := os.Chtimes(session.info.MetaData["versionsPath"], oldNodeMtime, oldNodeMtime); err != nil {
			return unlock, errtypes.InternalError(fmt.Sprintf("failed to change mtime of version node: %s", err))
		}
	}

	session.info.MetaData["sizeDiff"] = strconv.FormatInt((int64(fsize) - old.Blobsize), 10)

	return unlock, nil
}

func validateChecksums(ctx context.Context, n *node.Node, session *OcisSession, versionPath string) error {
	for _, t := range []string{"md5", "sha1", "adler32"} {
		key := prefixes.ChecksumPrefix + t

		checksum, err := n.Xattr(ctx, key)
		if err != nil {
			return err
		}

		revisionChecksum, err := session.store.lu.MetadataBackend().Get(ctx, versionPath, key)
		if err != nil {
			return err
		}

		if string(checksum) == "" || string(revisionChecksum) == "" {
			return errors.New("checksum not found")
		}

		if string(checksum) != string(revisionChecksum) {
			return errors.New("checksum mismatch")
		}
	}

	return nil
}
