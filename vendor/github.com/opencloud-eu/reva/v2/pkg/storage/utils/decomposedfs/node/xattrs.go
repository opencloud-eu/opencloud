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

package node

import (
	"context"
	"io"
	"io/fs"
	"strconv"
	"time"

	"github.com/pkg/xattr"
)

// Attributes is a map of string keys and byte array values
type Attributes map[string][]byte

// String reads a String value
func (md Attributes) String(key string) string {
	return string(md[key])
}

// SetString sets a string value
func (md Attributes) SetString(key, val string) {
	md[key] = []byte(val)
}

// Int64 reads an int64 value
func (md Attributes) Int64(key string) (int64, error) {
	return strconv.ParseInt(string(md[key]), 10, 64)
}

// SetInt64 sets an int64 value
func (md Attributes) SetInt64(key string, val int64) {
	md[key] = []byte(strconv.FormatInt(val, 10))
}

// UInt64 reads an uint64 value
func (md Attributes) UInt64(key string) (uint64, error) {
	return strconv.ParseUint(string(md[key]), 10, 64)
}

// SetInt64 sets an uint64 value
func (md Attributes) SetUInt64(key string, val uint64) {
	md[key] = []byte(strconv.FormatUint(val, 10))
}

// Time reads a time value
func (md Attributes) Time(key string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, string(md[key]))
}

// SetXattrs sets multiple extended attributes on the write-through cache/node
func (n *Node) SetXattrsWithContext(ctx context.Context, attribs map[string][]byte, acquireLock bool) (err error) {
	_, span := tracer.Start(ctx, "SetXattrsWithContext")
	defer span.End()
	if n.xattrsCache != nil {
		for k, v := range attribs {
			n.xattrsCache[k] = v
		}
	}

	return n.lu.MetadataBackend().SetMultiple(ctx, n.InternalPath(), attribs, acquireLock)
}

// SetXattrs sets multiple extended attributes on the write-through cache/node
func (n *Node) SetXattrs(attribs map[string][]byte, acquireLock bool) (err error) {
	return n.SetXattrsWithContext(context.Background(), attribs, acquireLock)
}

// SetXattr sets an extended attribute on the write-through cache/node
func (n *Node) SetXattr(ctx context.Context, key string, val []byte) (err error) {
	if n.xattrsCache != nil {
		n.xattrsCache[key] = val
	}

	return n.lu.MetadataBackend().Set(ctx, n.InternalPath(), key, val)
}

// SetXattrString sets a string extended attribute on the write-through cache/node
func (n *Node) SetXattrString(ctx context.Context, key, val string) (err error) {
	if n.xattrsCache != nil {
		n.xattrsCache[key] = []byte(val)
	}

	return n.lu.MetadataBackend().Set(ctx, n.InternalPath(), key, []byte(val))
}

// ResetXattrsCache clears the in-process xattr cache so the next read fetches from disk.
func (n *Node) ResetXattrsCache() {
	n.xattrsCache = nil
}

// RemoveXattr removes an extended attribute from the write-through cache/node
func (n *Node) RemoveXattr(ctx context.Context, key string, acquireLock bool) error {
	if n.xattrsCache != nil {
		delete(n.xattrsCache, key)
	}
	return n.lu.MetadataBackend().Remove(ctx, n.InternalPath(), key, acquireLock)
}

// XattrsWithReader returns the extended attributes of the node. If the attributes have already
// been cached they are not read from disk again.
func (n *Node) XattrsWithReader(ctx context.Context, r io.Reader) (Attributes, error) {
	if n.ID == "" {
		// Do not try to read the attribute of an empty node. The InternalPath points to the
		// base nodes directory in this case.
		return Attributes{}, &xattr.Error{Op: "node.XattrsWithReader", Path: n.InternalPath(), Err: xattr.ENOATTR}
	}

	if n.xattrsCache != nil {
		return n.xattrsCache, nil
	}

	var attrs Attributes
	var err error
	if r != nil {
		attrs, err = n.lu.MetadataBackend().AllWithLockedSource(ctx, n.InternalPath(), r)
	} else {
		attrs, err = n.lu.MetadataBackend().All(ctx, n.InternalPath())
	}
	if err != nil {
		return nil, err
	}

	n.xattrsCache = attrs
	return n.xattrsCache, nil
}

// Xattrs returns the extended attributes of the node. If the attributes have already
// been cached they are not read from disk again.
func (n *Node) Xattrs(ctx context.Context) (Attributes, error) {
	return n.XattrsWithReader(ctx, nil)
}

// Xattr returns an extended attribute of the node. If the attributes have already
// been cached it is not read from disk again.
func (n *Node) Xattr(ctx context.Context, key string) ([]byte, error) {
	path := n.InternalPath()

	if path == "" {
		// Do not try to read the attribute of an non-existing node
		return []byte{}, fs.ErrNotExist
	}

	if n.ID == "" {
		// Do not try to read the attribute of an empty node. The InternalPath points to the
		// base nodes directory in this case.
		return []byte{}, &xattr.Error{Op: "node.Xattr", Path: path, Name: key, Err: xattr.ENOATTR}
	}

	if n.xattrsCache == nil {
		attrs, err := n.lu.MetadataBackend().All(ctx, path)
		if err != nil {
			return []byte{}, err
		}
		n.xattrsCache = attrs
	}

	if val, ok := n.xattrsCache[key]; ok {
		return val, nil
	}
	// wrap the error as xattr does
	return []byte{}, &xattr.Error{Op: "node.Xattr", Path: path, Name: key, Err: xattr.ENOATTR}
}

// XattrString returns the string representation of an attribute
func (n *Node) XattrString(ctx context.Context, key string) (string, error) {
	b, err := n.Xattr(ctx, key)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// XattrInt32 returns the int32 representation of an attribute
func (n *Node) XattrInt32(ctx context.Context, key string) (int32, error) {
	b, err := n.XattrString(ctx, key)
	if err != nil {
		return 0, err
	}

	typeInt, err := strconv.ParseInt(b, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(typeInt), nil
}

// XattrInt64 returns the int64 representation of an attribute
func (n *Node) XattrInt64(ctx context.Context, key string) (int64, error) {
	b, err := n.XattrString(ctx, key)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(b, 10, 64)
}

// XattrUint64 returns the uint64 representation of an attribute
func (n *Node) XattrUint64(ctx context.Context, key string) (uint64, error) {
	b, err := n.XattrString(ctx, key)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(b, 10, 64)
}
