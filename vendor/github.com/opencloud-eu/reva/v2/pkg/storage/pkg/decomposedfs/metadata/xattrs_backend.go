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

package metadata

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/opencloud-eu/reva/v2/pkg/storage/cache"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/filelocks"
	"github.com/pkg/errors"
	"github.com/pkg/xattr"
	"github.com/rogpeppe/go-internal/lockedfile"
)

// XattrsBackend stores the file attributes in extended attributes
type XattrsBackend struct {
	metaCache cache.FileMetadataCache
}

// NewMessageBackend returns a new XattrsBackend instance
func NewXattrsBackend(o cache.Config) XattrsBackend {
	return XattrsBackend{
		metaCache: cache.GetFileMetadataCache(o),
	}
}

// Name returns the name of the backend
func (XattrsBackend) Name() string { return "xattrs" }

// IdentifyPath returns the space id, node id and mtime of a file
func (b XattrsBackend) IdentifyPath(_ context.Context, path string) (string, string, string, time.Time, error) {
	spaceID, _ := xattr.Get(path, prefixes.SpaceIDAttr)
	id, _ := xattr.Get(path, prefixes.IDAttr)
	parentID, _ := xattr.Get(path, prefixes.ParentidAttr)

	mtimeAttr, _ := xattr.Get(path, prefixes.MTimeAttr)
	mtime, _ := time.Parse(time.RFC3339Nano, string(mtimeAttr))
	return string(spaceID), string(id), string(parentID), mtime, nil
}

// Get an extended attribute value for the given key
// No file locking is involved here as reading a single xattr is
// considered to be atomic.
func (b XattrsBackend) Get(ctx context.Context, n MetadataNode, key string) ([]byte, error) {
	attribs := map[string][]byte{}
	err := b.metaCache.PullFromCache(b.cacheKey(n), &attribs)
	if err == nil && len(attribs[key]) > 0 {
		return attribs[key], err
	}

	return xattr.Get(n.InternalPath(), key)
}

// GetInt64 reads a string as int64 from the xattrs
func (b XattrsBackend) GetInt64(ctx context.Context, n MetadataNode, key string) (int64, error) {
	attr, err := b.Get(ctx, n, key)
	if err != nil {
		return 0, err
	}
	v, err := strconv.ParseInt(string(attr), 10, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func (b XattrsBackend) list(ctx context.Context, n MetadataNode, acquireLock bool) (attribs []string, err error) {
	filePath := n.InternalPath()
	attrs, err := xattr.List(filePath)
	if err == nil {
		return attrs, nil
	}

	// listing xattrs failed, try again, either with lock or without
	if acquireLock {
		f, err := lockedfile.OpenFile(filePath+filelocks.LockFileSuffix, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return nil, err
		}
		defer cleanupLockfile(ctx, f)

	}
	return xattr.List(filePath)
}

// All reads all extended attributes for a node, protected by a
// shared file lock
func (b XattrsBackend) All(ctx context.Context, n MetadataNode) (map[string][]byte, error) {
	return b.getAll(ctx, n, false, true)
}

func (b XattrsBackend) getAll(ctx context.Context, n MetadataNode, skipCache, acquireLock bool) (map[string][]byte, error) {
	attribs := map[string][]byte{}

	if !skipCache {
		err := b.metaCache.PullFromCache(b.cacheKey(n), &attribs)
		if err == nil {
			return attribs, err
		}
	}

	attrNames, err := b.list(ctx, n, acquireLock)
	if err != nil {
		return nil, err
	}

	if len(attrNames) == 0 {
		return attribs, nil
	}

	var (
		xerrs = 0
		xerr  error
	)
	// error handling: Count if there are errors while reading all attribs.
	// if there were any, return an error.
	attribs = make(map[string][]byte, len(attrNames))
	path := n.InternalPath()
	for _, name := range attrNames {
		var val []byte
		if val, xerr = xattr.Get(path, name); xerr != nil && !IsAttrUnset(xerr) {
			xerrs++
		} else {
			attribs[name] = val
		}
	}

	if xerrs > 0 {
		return nil, errors.Wrap(xerr, "Failed to read all xattrs")
	}

	err = b.metaCache.PushToCache(b.cacheKey(n), attribs)
	if err != nil {
		return nil, err
	}

	return attribs, nil
}

// Set sets one attribute for the given path
func (b XattrsBackend) Set(ctx context.Context, n MetadataNode, key string, val []byte) (err error) {
	return b.SetMultiple(ctx, n, map[string][]byte{key: val}, true)
}

// SetMultiple sets a set of attribute for the given path
func (b XattrsBackend) SetMultiple(ctx context.Context, n MetadataNode, attribs map[string][]byte, acquireLock bool) (err error) {
	path := n.InternalPath()
	if acquireLock {
		err := os.MkdirAll(filepath.Dir(path), 0700)
		if err != nil {
			return err
		}
		lockedFile, err := lockedfile.OpenFile(b.LockfilePath(n), os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer cleanupLockfile(ctx, lockedFile)
	}

	// error handling: Count if there are errors while setting the attribs.
	// if there were any, return an error.
	var (
		xerrs = 0
		xerr  error
	)
	for key, val := range attribs {
		if xerr = xattr.Set(path, key, val); xerr != nil {
			// log
			xerrs++
		}
	}
	if xerrs > 0 {
		return errors.Wrap(xerr, "Failed to set all xattrs")
	}

	attribs, err = b.getAll(ctx, n, true, false)
	if err != nil {
		return err
	}
	return b.metaCache.PushToCache(b.cacheKey(n), attribs)
}

// Remove an extended attribute key
func (b XattrsBackend) Remove(ctx context.Context, n MetadataNode, key string, acquireLock bool) error {
	path := n.InternalPath()
	if acquireLock {
		lockedFile, err := lockedfile.OpenFile(path+filelocks.LockFileSuffix, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer cleanupLockfile(ctx, lockedFile)
	}

	err := xattr.Remove(path, key)
	if err != nil {
		return err
	}
	attribs, err := b.getAll(ctx, n, true, false)
	if err != nil {
		return err
	}
	return b.metaCache.PushToCache(b.cacheKey(n), attribs)
}

// IsMetaFile returns whether the given path represents a meta file
func (XattrsBackend) IsMetaFile(path string) bool { return strings.HasSuffix(path, ".meta.lock") }

// Purge purges the data of a given path
func (b XattrsBackend) Purge(ctx context.Context, n MetadataNode) error {
	path := n.InternalPath()
	_, err := os.Stat(path)
	if err == nil {
		attribs, err := b.getAll(ctx, n, true, true)
		if err != nil {
			return err
		}

		for attr := range attribs {
			if strings.HasPrefix(attr, prefixes.OcPrefix) {
				err := xattr.Remove(path, attr)
				if err != nil {
					return err
				}
			}
		}
	}

	return b.metaCache.RemoveMetadata(b.cacheKey(n))
}

// Rename moves the data for a given path to a new path
func (b XattrsBackend) Rename(oldNode, newNode MetadataNode) error {
	data := map[string][]byte{}
	err := b.metaCache.PullFromCache(b.cacheKey(oldNode), &data)
	if err == nil {
		err = b.metaCache.PushToCache(b.cacheKey(newNode), data)
		if err != nil {
			return err
		}
	}
	return b.metaCache.RemoveMetadata(b.cacheKey(oldNode))
}

// MetadataPath returns the path of the file holding the metadata for the given path
func (XattrsBackend) MetadataPath(n MetadataNode) string { return n.InternalPath() }

// LockfilePath returns the path of the lock file
func (XattrsBackend) LockfilePath(n MetadataNode) string { return n.InternalPath() + ".mlock" }

// Lock locks the metadata for the given path
func (b XattrsBackend) Lock(n MetadataNode) (UnlockFunc, error) {
	metaLockPath := b.LockfilePath(n)
	mlock, err := lockedfile.OpenFile(metaLockPath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	return func() error {
		err := mlock.Close()
		if err != nil {
			return err
		}
		return os.Remove(metaLockPath)
	}, nil
}

func cleanupLockfile(_ context.Context, f *lockedfile.File) {
	_ = f.Close()
	_ = os.Remove(f.Name())
}

// AllWithLockedSource reads all extended attributes from the given reader.
// The path argument is used for storing the data in the cache
func (b XattrsBackend) AllWithLockedSource(ctx context.Context, n MetadataNode, _ io.Reader) (map[string][]byte, error) {
	return b.All(ctx, n)
}

func (b XattrsBackend) cacheKey(n MetadataNode) string {
	// rootPath is guaranteed to have no trailing slash
	// the cache key shouldn't begin with a slash as some stores drop it which can cause
	// confusion
	return n.GetSpaceID() + "/" + n.GetID()
}
