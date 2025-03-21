// Copyright 2018-2021 CERN
// Copyright 2025 OpenCloud GmbH <mail@opencloud.eu>
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

package tree

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/pkg/errors"
	"github.com/rogpeppe/go-internal/lockedfile"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// Revision entries are stored inside the revisions directory in .oc-nodes, sharded by the node ids.
// The `.REV.` indicates it is a revision and what follows is a timestamp, so multiple versions
// can be kept in the same location.

// CreateRevision creates a new version of the node
func (tp *Tree) CreateRevision(ctx context.Context, n *node.Node, version string, f *lockedfile.File) (string, error) {
	revNode := node.NewBaseNode(n.SpaceID, n.ID+node.RevisionIDDelimiter+version, tp.lookup)
	versionPath := revNode.InternalPath()

	err := os.MkdirAll(filepath.Dir(versionPath), 0700)
	if err != nil {
		return "", err
	}

	// copy file content to version node
	sf, err := os.OpenFile(n.InternalPath(), os.O_RDONLY, 0)
	if err != nil {
		return "", err
	}
	defer sf.Close()
	vf, err := os.OpenFile(versionPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		if os.IsExist(err) {
			dir := filepath.Dir(versionPath)
			base := filepath.Base(versionPath)
			files, err := os.ReadDir(dir)
			if err != nil {
				return "", err
			}

			// find revision with highest number
			highest := 0
			for _, file := range files {
				if file.IsDir() {
					continue
				}
				name := file.Name()
				if !strings.HasPrefix(name, base) {
					continue
				}
				ext := strings.TrimPrefix(name, base+".")
				if ext == "" || ext == base {
					continue
				}
				num, err := strconv.Atoi(ext)
				if err != nil {
					continue
				}
				if num > highest {
					highest = num
				}
			}

			// rename existing revision
			oldNode := node.NewBaseNode(n.SpaceID, n.ID+node.RevisionIDDelimiter+version+"."+strconv.Itoa(highest+1), tp.lookup)
			err = tp.lookup.MetadataBackend().Rename(revNode, oldNode)
			if err != nil {
				return "", err
			}
			newPath := versionPath + "." + strconv.Itoa(highest+1)
			err = os.Rename(versionPath, newPath)
			if err != nil {
				return "", err
			}
			vf, err = os.OpenFile(versionPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}
	defer vf.Close()
	if _, err := io.Copy(vf, sf); err != nil {
		return "", err
	}

	// copy blob metadata to version node
	if err := tp.lookup.CopyMetadataWithSourceLock(ctx, n, revNode, func(attributeName string, value []byte) (newValue []byte, copy bool) {
		return value, strings.HasPrefix(attributeName, prefixes.ChecksumPrefix) ||
			attributeName == prefixes.TypeAttr ||
			attributeName == prefixes.BlobIDAttr ||
			attributeName == prefixes.BlobsizeAttr ||
			attributeName == prefixes.MTimeAttr
	}, f, true); err != nil {
		return "", err
	}

	return versionPath, nil
}

func (tp *Tree) ListRevisions(ctx context.Context, ref *provider.Reference) (revisions []*provider.FileVersion, err error) {
	_, span := tracer.Start(ctx, "ListRevisions")
	defer span.End()
	var n *node.Node
	if n, err = tp.lookup.NodeFromResource(ctx, ref); err != nil {
		return
	}
	if !n.Exists {
		err = errtypes.NotFound(filepath.Join(n.ParentID, n.Name))
		return
	}

	rp, err := tp.permissions.AssemblePermissions(ctx, n)
	switch {
	case err != nil:
		return nil, err
	case !rp.ListFileVersions:
		f, _ := storagespace.FormatReference(ref)
		if rp.Stat {
			return nil, errtypes.PermissionDenied(f)
		}
		return nil, errtypes.NotFound(f)
	}

	revisions = []*provider.FileVersion{}
	versionGlob := tp.lookup.VersionPath(n.SpaceID, n.ID, "*")
	if items, err := filepath.Glob(versionGlob); err == nil {
		for i := range items {
			if tp.lookup.MetadataBackend().IsMetaFile(items[i]) || strings.HasSuffix(items[i], ".mlock") {
				continue
			}

			if fi, err := os.Stat(items[i]); err == nil {
				parts := strings.SplitN(fi.Name(), node.RevisionIDDelimiter, 2)
				if len(parts) != 2 {
					appctx.GetLogger(ctx).Error().Err(err).Str("name", fi.Name()).Msg("invalid revision name, skipping")
					continue
				}
				mtime := fi.ModTime()
				rev := &provider.FileVersion{
					Key:   n.ID + node.RevisionIDDelimiter + parts[1],
					Mtime: uint64(mtime.Unix()),
				}
				_, blobSize, err := tp.lookup.ReadBlobIDAndSizeAttr(ctx, node.NewBaseNode(n.SpaceID, rev.Key, tp.lookup), nil)
				if err != nil {
					appctx.GetLogger(ctx).Error().Err(err).Str("name", fi.Name()).Msg("error reading blobsize xattr, using 0")
				}
				rev.Size = uint64(blobSize)
				etag, err := node.CalculateEtag(n.ID, mtime)
				if err != nil {
					return nil, errors.Wrapf(err, "error calculating etag")
				}
				rev.Etag = etag
				revisions = append(revisions, rev)
			}
		}
	}
	// maybe we need to sort the list by key
	/*
		sort.Slice(revisions, func(i, j int) bool {
			return revisions[i].Key > revisions[j].Key
		})
	*/

	return
}

// DownloadRevision returns a reader for the specified revision
// FIXME the CS3 api should explicitly allow initiating revision and trash download, a related issue is https://github.com/cs3org/reva/issues/1813
func (tp *Tree) DownloadRevision(ctx context.Context, ref *provider.Reference, revisionKey string, openReaderFunc func(md *provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error) {
	_, span := tracer.Start(ctx, "DownloadRevision")
	defer span.End()
	log := appctx.GetLogger(ctx)

	// verify revision key format
	kp := strings.SplitN(revisionKey, node.RevisionIDDelimiter, 2)
	if len(kp) != 2 {
		log.Error().Str("revisionKey", revisionKey).Msg("malformed revisionKey")
		return nil, nil, errtypes.NotFound(revisionKey)
	}
	log.Debug().Str("revisionKey", revisionKey).Msg("DownloadRevision")

	spaceID := ref.ResourceId.SpaceId
	// check if the node is available and has not been deleted
	n, err := node.ReadNode(ctx, tp.lookup, spaceID, kp[0], false, nil, false)
	if err != nil {
		return nil, nil, err
	}
	if !n.Exists {
		err = errtypes.NotFound(filepath.Join(n.ParentID, n.Name))
		return nil, nil, err
	}

	rp, err := tp.permissions.AssemblePermissions(ctx, n)
	switch {
	case err != nil:
		return nil, nil, err
	case !rp.ListFileVersions || !rp.InitiateFileDownload: // TODO add explicit permission in the CS3 api?
		f, _ := storagespace.FormatReference(ref)
		if rp.Stat {
			return nil, nil, errtypes.PermissionDenied(f)
		}
		return nil, nil, errtypes.NotFound(f)
	}

	revNode := node.NewBaseNode(spaceID, revisionKey, tp.lookup)
	_, blobsize, err := tp.lookup.ReadBlobIDAndSizeAttr(ctx, revNode, nil)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Decomposedfs: could not read blob id and size for revision '%s' of node '%s'", kp[1], n.ID)
	}

	revisionNode := node.New(spaceID, revisionKey, n.ParentID, n.Name, blobsize, "", provider.ResourceType_RESOURCE_TYPE_FILE, n.Owner(), tp.lookup)

	ri, err := n.AsResourceInfo(ctx, rp, nil, []string{"size", "mimetype", "etag"}, true)
	if err != nil {
		return nil, nil, err
	}

	// update resource info with revision data
	mtime, err := time.Parse(time.RFC3339Nano, kp[1])
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Decomposedfs: could not parse mtime for revision '%s' of node '%s'", kp[1], n.ID)
	}
	ri.Size = uint64(blobsize)
	ri.Mtime = utils.TimeToTS(mtime)
	ri.Etag, err = node.CalculateEtag(n.ID, mtime)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "error calculating etag for revision '%s' of node '%s'", kp[1], n.ID)
	}

	var reader io.ReadCloser
	if openReaderFunc(ri) {
		reader, err = tp.ReadBlob(revisionNode)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "Decomposedfs: could not download blob of revision '%s' for node '%s'", n.ID, revisionKey)
		}
	}
	return ri, reader, nil
}

func (tp *Tree) RestoreRevision(ctx context.Context, srcNode, targetNode metadata.MetadataNode) error {
	source := srcNode.InternalPath()
	target := targetNode.InternalPath()
	rf, err := os.Open(source)
	if err != nil {
		return err
	}
	defer rf.Close()

	wf, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer wf.Close()
	err = wf.Truncate(0)
	if err != nil {
		return err
	}

	if _, err := io.Copy(wf, rf); err != nil {
		return err
	}

	err = tp.lookup.CopyMetadata(ctx, srcNode, targetNode, func(attributeName string, value []byte) (newValue []byte, copy bool) {
		return value, strings.HasPrefix(attributeName, prefixes.ChecksumPrefix) ||
			attributeName == prefixes.TypeAttr ||
			attributeName == prefixes.BlobIDAttr ||
			attributeName == prefixes.BlobsizeAttr
	}, false)
	if err != nil {
		return errtypes.InternalError("failed to copy blob xattrs to old revision to node: " + err.Error())
	}

	// always set the node mtime to the current time
	mtime := time.Now()
	err = os.Chtimes(target, mtime, mtime)
	if err != nil {
		return errtypes.InternalError("failed to update times:" + err.Error())
	}

	err = tp.lookup.MetadataBackend().SetMultiple(ctx, targetNode,
		map[string][]byte{
			prefixes.MTimeAttr: []byte(mtime.UTC().Format(time.RFC3339Nano)),
		},
		false)
	if err != nil {
		return errtypes.InternalError("failed to set mtime attribute on node: " + err.Error())
	}

	// update "current" revision
	if tp.options.EnableFSRevisions {
		currenNode := node.NewBaseNode(targetNode.GetSpaceID(), targetNode.GetID()+node.CurrentIDDelimiter, tp.lookup)
		currentPath := currenNode.InternalPath()
		w, err := os.OpenFile(currentPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			tp.log.Error().Err(err).Str("currentPath", currentPath).Str("source", source).Msg("could not open current path for writing")
			return err
		}
		defer w.Close()
		r, err := os.OpenFile(source, os.O_RDONLY, 0600)
		if err != nil {
			tp.log.Error().Err(err).Str("currentPath", currentPath).Str("source", source).Msg("could not open file for reading")
			return err
		}
		defer r.Close()

		_, err = io.Copy(w, r)
		if err != nil {
			tp.log.Error().Err(err).Str("currentPath", currentPath).Str("source", source).Msg("could not copy new version to current version")
			return err
		}
		err = tp.lookup.CopyMetadata(ctx, srcNode, currenNode, func(attributeName string, value []byte) (newValue []byte, copy bool) {
			return value, strings.HasPrefix(attributeName, prefixes.ChecksumPrefix) ||
				attributeName == prefixes.TypeAttr ||
				attributeName == prefixes.BlobIDAttr ||
				attributeName == prefixes.BlobsizeAttr
		}, false)
		if err != nil {
			return errtypes.InternalError("failed to copy xattrs to 'current' file: " + err.Error())
		}
	}

	return nil
}
