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

package metadata

import (
	"context"
	"errors"
	"io"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func init() {
	tracer = otel.Tracer("github.com/cs3org/reva/pkg/storage/utils/decomposedfs/metadata")
}

var errUnconfiguredError = errors.New("no metadata backend configured. Bailing out")

type UnlockFunc func() error

type MetadataNode interface {
	GetSpaceID() string
	GetID() string
	InternalPath() string
}

// Backend defines the interface for file attribute backends
type Backend interface {
	Name() string
	IdentifyPath(ctx context.Context, path string) (string, string, string, time.Time, error)

	All(ctx context.Context, n MetadataNode) (map[string][]byte, error)
	AllWithLockedSource(ctx context.Context, n MetadataNode, source io.Reader) (map[string][]byte, error)

	Get(ctx context.Context, n MetadataNode, key string) ([]byte, error)
	GetInt64(ctx context.Context, n MetadataNode, key string) (int64, error)
	Set(ctx context.Context, n MetadataNode, key string, val []byte) error
	SetMultiple(ctx context.Context, n MetadataNode, attribs map[string][]byte, acquireLock bool) error
	Remove(ctx context.Context, n MetadataNode, key string, acquireLock bool) error

	Lock(n MetadataNode) (UnlockFunc, error)
	Purge(ctx context.Context, n MetadataNode) error
	Rename(oldNode, newNode MetadataNode) error
	MetadataPath(n MetadataNode) string
	LockfilePath(n MetadataNode) string

	IsMetaFile(path string) bool
}

// NullBackend is the default stub backend, used to enforce the configuration of a proper backend
type NullBackend struct{}

// Name returns the name of the backend
func (NullBackend) Name() string { return "null" }

// IdentifyPath returns the ids and mtime of a file
func (NullBackend) IdentifyPath(ctx context.Context, path string) (string, string, string, time.Time, error) {
	return "", "", "", time.Time{}, errUnconfiguredError
}

// All reads all extended attributes for a node
func (NullBackend) All(ctx context.Context, n MetadataNode) (map[string][]byte, error) {
	return nil, errUnconfiguredError
}

// Get an extended attribute value for the given key
func (NullBackend) Get(ctx context.Context, n MetadataNode, key string) ([]byte, error) {
	return []byte{}, errUnconfiguredError
}

// GetInt64 reads a string as int64 from the xattrs
func (NullBackend) GetInt64(ctx context.Context, n MetadataNode, key string) (int64, error) {
	return 0, errUnconfiguredError
}

// Set sets one attribute for the given path
func (NullBackend) Set(ctx context.Context, n MetadataNode, key string, val []byte) error {
	return errUnconfiguredError
}

// SetMultiple sets a set of attribute for the given path
func (NullBackend) SetMultiple(ctx context.Context, n MetadataNode, attribs map[string][]byte, acquireLock bool) error {
	return errUnconfiguredError
}

// Remove removes an extended attribute key
func (NullBackend) Remove(ctx context.Context, n MetadataNode, key string, acquireLock bool) error {
	return errUnconfiguredError
}

// Lock locks the metadata for the given path
func (NullBackend) Lock(n MetadataNode) (UnlockFunc, error) {
	return nil, nil
}

// IsMetaFile returns whether the given path represents a meta file
func (NullBackend) IsMetaFile(path string) bool { return false }

// Purge purges the data of a given path from any cache that might hold it
func (NullBackend) Purge(_ context.Context, n MetadataNode) error { return errUnconfiguredError }

// Rename moves the data for a given path to a new path
func (NullBackend) Rename(oldNode, newNode MetadataNode) error { return errUnconfiguredError }

// MetadataPath returns the path of the file holding the metadata for the given path
func (NullBackend) MetadataPath(n MetadataNode) string { return "" }

// LockfilePath returns the path of the lock file
func (NullBackend) LockfilePath(n MetadataNode) string { return "" }

// AllWithLockedSource reads all extended attributes from the given reader
// The path argument is used for storing the data in the cache
func (NullBackend) AllWithLockedSource(ctx context.Context, n MetadataNode, source io.Reader) (map[string][]byte, error) {
	return nil, errUnconfiguredError
}
