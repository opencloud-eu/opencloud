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
	"path/filepath"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/appctx"
	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
	"github.com/opencloud-eu/reva/v3/pkg/storage"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/metadata"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v3/pkg/storagespace"
	"github.com/pkg/errors"
)

// SetArbitraryMetadata sets the metadata on a resource
func (fs *Decomposedfs) SetArbitraryMetadata(ctx context.Context, ref *provider.Reference, md *provider.ArbitraryMetadata) (*storage.MetadataResult, error) {
	_, span := tracer.Start(ctx, "SetArbitraryMetadata")
	defer span.End()
	n, err := fs.lu.NodeFromResource(ctx, ref)
	if err != nil {
		return nil, errors.Wrap(err, "Decomposedfs: error resolving ref")
	}
	if !n.Exists {
		err = errtypes.NotFound(filepath.Join(n.ParentID, n.Name))
		return nil, err
	}

	rp, err := fs.p.AssemblePermissions(ctx, n)
	switch {
	case err != nil:
		return nil, err
	case !rp.InitiateFileUpload: // TODO add explicit SetArbitraryMetadata grant to CS3 api, tracked in https://github.com/cs3org/cs3apis/issues/91
		f, _ := storagespace.FormatReference(ref)
		if rp.Stat {
			return nil, errtypes.PermissionDenied(f)
		}
		return nil, errtypes.NotFound(f)
	}

	// check lock
	if err := n.CheckLock(ctx); err != nil {
		return nil, err
	}

	existingAttribs, err := n.Xattrs(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Decomposedfs: could not read metadata attributes")
	}

	errs := []error{}
	// TODO should we really continue updating when an error occurs?
	result := &storage.MetadataResult{SpaceOwner: n.SpaceOwnerOrManager(ctx)}
	if md.Metadata != nil {
		if val, ok := md.Metadata["mtime"]; ok {
			delete(md.Metadata, "mtime")
			if err := n.SetMtimeString(ctx, val); err != nil {
				errs = append(errs, errors.Wrap(err, "could not set mtime"))
			} else {
				result.ChangedKeys = append(result.ChangedKeys, "mtime")
			}
		}
		// TODO(jfd) special handling for atime?
		// TODO(jfd) allow setting birth time (btime)?
		// TODO(jfd) any other metadata that is interesting? fileid?
		// TODO unset when file is updated
		// TODO unset when folder is updated or add timestamp to etag?
		if val, ok := md.Metadata["etag"]; ok {
			delete(md.Metadata, "etag")
			if err := n.SetEtag(ctx, val); err != nil {
				errs = append(errs, errors.Wrap(err, "could not set etag"))
			} else {
				result.ChangedKeys = append(result.ChangedKeys, "etag")
			}
		}
	}

	// one write for the whole set: a per key write publishes every intermediate
	// state to the unlocked, cache first readers
	attribs := make(map[string][]byte, len(md.Metadata))
	for k, v := range md.Metadata {
		if existingVal, ok := existingAttribs[prefixes.MetadataPrefix+k]; ok && string(existingVal) == v {
			continue // no change needed
		}
		result.ChangedKeys = append(result.ChangedKeys, k)
		attribs[prefixes.MetadataPrefix+k] = []byte(v)
	}
	if len(attribs) > 0 {
		if err = n.SetXattrsWithContext(ctx, attribs); err != nil {
			errs = append(errs, errors.Wrap(err, "Decomposedfs: could not set metadata attributes"))
		}
	}

	if len(result.ChangedKeys) > 0 {
		if err = fs.tp.Propagate(ctx, n, 0); err != nil {
			errs = append(errs, errors.Wrap(err, "Decomposedfs: could not propagate metadata changes"))
		}
	}

	switch len(errs) {
	case 0:
		return result, nil
	case 1:
		// TODO Propagate if anything changed
		return nil, errs[0]
	default:
		// TODO Propagate if anything changed
		// TODO how to return multiple errors?
		return nil, errors.New("multiple errors occurred, see log for details")
	}
}

// UnsetArbitraryMetadata unsets the metadata on the given resource
func (fs *Decomposedfs) UnsetArbitraryMetadata(ctx context.Context, ref *provider.Reference, keys []string) (*storage.MetadataResult, error) {
	_, span := tracer.Start(ctx, "UnsetArbitraryMetadata")
	defer span.End()
	n, err := fs.lu.NodeFromResource(ctx, ref)
	if err != nil {
		return nil, errors.Wrap(err, "Decomposedfs: error resolving ref")
	}
	sublog := appctx.GetLogger(ctx).With().Str("spaceid", n.SpaceID).Str("nodeid", n.ID).Logger()

	if !n.Exists {
		err = errtypes.NotFound(filepath.Join(n.ParentID, n.Name))
		return nil, err
	}

	rp, err := fs.p.AssemblePermissions(ctx, n)
	switch {
	case err != nil:
		return nil, err
	case !rp.InitiateFileUpload: // TODO use SetArbitraryMetadata grant to CS3 api, tracked in https://github.com/cs3org/cs3apis/issues/91
		f, _ := storagespace.FormatReference(ref)
		if rp.Stat {
			return nil, errtypes.PermissionDenied(f)
		}
		return nil, errtypes.NotFound(f)
	}

	// check lock
	if err := n.CheckLock(ctx); err != nil {
		return nil, err
	}

	errs := []error{}
	result := &storage.MetadataResult{SpaceOwner: n.SpaceOwnerOrManager(ctx)}
	existingAttribs, err := n.Xattrs(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Decomposedfs: could not read metadata attributes")
	}
	for _, k := range keys {
		if _, ok := existingAttribs[prefixes.MetadataPrefix+k]; !ok {
			continue // already gone, ignore
		}
		if err = n.RemoveXattr(ctx, prefixes.MetadataPrefix+k); err != nil {
			if metadata.IsAttrUnset(err) {
				continue // already gone, ignore
			}
			sublog.Error().Err(err).
				Str("key", k).
				Msg("could not unset metadata")
			errs = append(errs, errors.Wrap(err, "could not unset metadata"))
			continue
		}
		result.RemovedKeys = append(result.RemovedKeys, k)
	}
	switch len(errs) {
	case 0:
		if len(result.RemovedKeys) > 0 {
			if err := fs.tp.Propagate(ctx, n, 0); err != nil {
				return nil, err
			}
		}
		return result, nil
	case 1:
		// TODO Propagate if anything changed
		return nil, errs[0]
	default:
		// TODO Propagate if anything changed
		// TODO how to return multiple errors?
		return nil, errors.New("multiple errors occurred, see log for details")
	}
}
