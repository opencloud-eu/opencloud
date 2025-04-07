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
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/renameio/v2"
	"github.com/opencloud-eu/reva/v2/pkg/storage/cache"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/pkg/xattr"
	"github.com/rogpeppe/go-internal/lockedfile"
	"github.com/shamaton/msgpack/v2"
	"go.opentelemetry.io/otel/codes"
)

// MessagePackBackend persists the attributes in messagepack format inside the file
type MessagePackBackend struct {
	metaCache cache.FileMetadataCache
}

// NewMessagePackBackend returns a new MessagePackBackend instance
func NewMessagePackBackend(o cache.Config) MessagePackBackend {
	return MessagePackBackend{
		metaCache: cache.GetFileMetadataCache(o),
	}
}

// Name returns the name of the backend
func (MessagePackBackend) Name() string { return "messagepack" }

// IdentifyPath returns the id and mtime of a file
func (b MessagePackBackend) IdentifyPath(_ context.Context, path string) (string, string, string, time.Time, error) {
	metaPath := filepath.Clean(path + ".mpk")
	source, err := os.Open(metaPath)
	// // No cached entry found. Read from storage and store in cache
	if err != nil {
		return "", "", "", time.Time{}, err
	}
	msgBytes, err := io.ReadAll(source)
	if err != nil || len(msgBytes) == 0 {
		return "", "", "", time.Time{}, err

	}
	attribs := map[string][]byte{}
	err = msgpack.Unmarshal(msgBytes, &attribs)
	if err != nil {
		return "", "", "", time.Time{}, err
	}

	spaceID := attribs[prefixes.IDAttr]
	id := attribs[prefixes.IDAttr]
	parentID := attribs[prefixes.ParentidAttr]

	mtimeAttr := attribs[prefixes.MTimeAttr]
	mtime, err := time.Parse(time.RFC3339Nano, string(mtimeAttr))
	if err != nil {
		return "", "", "", time.Time{}, err
	}
	return string(spaceID), string(id), string(parentID), mtime, nil
}

// All reads all extended attributes for a node
func (b MessagePackBackend) All(ctx context.Context, n MetadataNode) (map[string][]byte, error) {
	return b.loadAttributes(ctx, n, nil)
}

// Get an extended attribute value for the given key
func (b MessagePackBackend) Get(ctx context.Context, n MetadataNode, key string) ([]byte, error) {
	attribs, err := b.loadAttributes(ctx, n, nil)
	if err != nil {
		return []byte{}, err
	}
	val, ok := attribs[key]
	if !ok {
		return []byte{}, &xattr.Error{Op: "mpk.get", Path: n.InternalPath(), Name: key, Err: xattr.ENOATTR}
	}
	return val, nil
}

// GetInt64 reads a string as int64 from the xattrs
func (b MessagePackBackend) GetInt64(ctx context.Context, n MetadataNode, key string) (int64, error) {
	attribs, err := b.loadAttributes(ctx, n, nil)
	if err != nil {
		return 0, err
	}
	val, ok := attribs[key]
	if !ok {
		return 0, &xattr.Error{Op: "mpk.get", Path: n.InternalPath(), Name: key, Err: xattr.ENOATTR}
	}
	i, err := strconv.ParseInt(string(val), 10, 64)
	if err != nil {
		return 0, err
	}
	return i, nil
}

// Set sets one attribute for the given path
func (b MessagePackBackend) Set(ctx context.Context, n MetadataNode, key string, val []byte) error {
	return b.SetMultiple(ctx, n, map[string][]byte{key: val}, true)
}

// SetMultiple sets a set of attribute for the given path
func (b MessagePackBackend) SetMultiple(ctx context.Context, n MetadataNode, attribs map[string][]byte, acquireLock bool) error {
	return b.saveAttributes(ctx, n, attribs, nil, acquireLock)
}

// Remove an extended attribute key
func (b MessagePackBackend) Remove(ctx context.Context, n MetadataNode, key string, acquireLock bool) error {
	return b.saveAttributes(ctx, n, nil, []string{key}, acquireLock)
}

// AllWithLockedSource reads all extended attributes from the given reader (if possible).
// The path argument is used for storing the data in the cache
func (b MessagePackBackend) AllWithLockedSource(ctx context.Context, n MetadataNode, source io.Reader) (map[string][]byte, error) {
	return b.loadAttributes(ctx, n, source)
}

func (b MessagePackBackend) saveAttributes(ctx context.Context, n MetadataNode, setAttribs map[string][]byte, deleteAttribs []string, acquireLock bool) error {
	var (
		err error
	)
	ctx, span := tracer.Start(ctx, "saveAttributes")
	defer func() {
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}
		span.End()
	}()

	metaPath := b.MetadataPath(n)
	if acquireLock {
		unlock, err := b.Lock(n)
		if err != nil {
			return err
		}
		defer func() { _ = unlock() }()
	}
	// Read current state
	_, subspan := tracer.Start(ctx, "os.ReadFile")
	var msgBytes []byte
	msgBytes, err = os.ReadFile(metaPath)
	subspan.End()
	attribs := map[string][]byte{}
	switch {
	case err != nil:
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	case len(msgBytes) == 0:
		// ugh. an empty file? bail out
		return errors.New("encountered empty metadata file")
	default:
		// only unmarshal if we read data
		err = msgpack.Unmarshal(msgBytes, &attribs)
		if err != nil {
			return err
		}
	}

	// prepare metadata
	for key, val := range setAttribs {
		attribs[key] = val
	}
	for _, key := range deleteAttribs {
		delete(attribs, key)
	}
	var d []byte
	d, err = msgpack.Marshal(attribs)
	if err != nil {
		return err
	}

	// overwrite file atomically
	_, subspan = tracer.Start(ctx, "renameio.Writefile")
	err = renameio.WriteFile(metaPath, d, 0600)
	if err != nil {
		return err
	}
	subspan.End()

	_, subspan = tracer.Start(ctx, "metaCache.PushToCache")
	err = b.metaCache.PushToCache(b.cacheKey(n), attribs)
	subspan.End()
	return err
}

func (b MessagePackBackend) loadAttributes(ctx context.Context, n MetadataNode, source io.Reader) (map[string][]byte, error) {
	ctx, span := tracer.Start(ctx, "loadAttributes")
	defer span.End()
	attribs := map[string][]byte{}
	err := b.metaCache.PullFromCache(b.cacheKey(n), &attribs)
	if err == nil {
		return attribs, err
	}

	metaPath := b.MetadataPath(n)
	var msgBytes []byte

	if source == nil {
		// // No cached entry found. Read from storage and store in cache
		_, subspan := tracer.Start(ctx, "os.OpenFile")
		// source, err = lockedfile.Open(metaPath)
		source, err = os.Open(metaPath)
		subspan.End()
		// // No cached entry found. Read from storage and store in cache
		if err != nil {
			if os.IsNotExist(err) {
				// some of the caller rely on ENOTEXISTS to be returned when the
				// actual file (not the metafile) does not exist in order to
				// determine whether a node exists or not -> stat the actual node
				_, subspan := tracer.Start(ctx, "os.Stat")
				_, err := os.Stat(n.InternalPath())
				subspan.End()
				if err != nil {
					return nil, err
				}
				return attribs, nil // no attributes set yet
			}
		}
		_, subspan = tracer.Start(ctx, "io.ReadAll")
		msgBytes, err = io.ReadAll(source)
		source.(*os.File).Close()
		subspan.End()
	} else {
		_, subspan := tracer.Start(ctx, "io.ReadAll")
		msgBytes, err = io.ReadAll(source)
		subspan.End()
	}

	if err != nil {
		return nil, err
	}
	if len(msgBytes) > 0 {
		err = msgpack.Unmarshal(msgBytes, &attribs)
		if err != nil {
			return nil, err
		}
	}

	_, subspan := tracer.Start(ctx, "metaCache.PushToCache")
	err = b.metaCache.PushToCache(b.cacheKey(n), attribs)
	subspan.End()
	if err != nil {
		return nil, err
	}

	return attribs, nil
}

// IsMetaFile returns whether the given path represents a meta file
func (MessagePackBackend) IsMetaFile(path string) bool {
	return strings.HasSuffix(path, ".mpk") || strings.HasSuffix(path, ".mlock")
}

// Purge purges the data of a given path
func (b MessagePackBackend) Purge(_ context.Context, n MetadataNode) error {
	if err := b.metaCache.RemoveMetadata(b.cacheKey(n)); err != nil {
		return err
	}
	return os.Remove(b.MetadataPath(n))
}

// Rename moves the data for a given path to a new path
func (b MessagePackBackend) Rename(oldNode, newNode MetadataNode) error {
	data := map[string][]byte{}
	err := b.metaCache.PullFromCache(b.cacheKey(oldNode), &data)
	if err == nil {
		err = b.metaCache.PushToCache(b.cacheKey(newNode), data)
		if err != nil {
			return err
		}
	}
	err = b.metaCache.RemoveMetadata(b.cacheKey(oldNode))
	if err != nil {
		return err
	}

	return os.Rename(b.MetadataPath(oldNode), b.MetadataPath(newNode))
}

// MetadataPath returns the path of the file holding the metadata for the given path
func (MessagePackBackend) MetadataPath(n MetadataNode) string { return n.InternalPath() + ".mpk" }

// LockfilePath returns the path of the lock file
func (MessagePackBackend) LockfilePath(n MetadataNode) string { return n.InternalPath() + ".mlock" }

// Lock locks the metadata for the given path
func (b MessagePackBackend) Lock(n MetadataNode) (UnlockFunc, error) {
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

func (b MessagePackBackend) cacheKey(n MetadataNode) string {
	return n.GetSpaceID() + "/" + n.GetID()
}
