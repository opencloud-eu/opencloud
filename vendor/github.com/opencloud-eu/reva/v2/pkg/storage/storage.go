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

package storage

import (
	"context"
	"io"
	"net/url"
	"time"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	registry "github.com/cs3org/go-cs3apis/cs3/storage/registry/v1beta1"
)

// MoveResult carries the resolved references returned by a successful Move.
type MoveResult struct {
	SpaceOwner   *user.UserId
	OldReference *provider.Reference
	NewReference *provider.Reference
}

type UploadChecksums struct {
	SHA1    []byte
	MD5     []byte
	Adler32 []byte
}

type UploadInfo struct {
	NodeExisted       bool // true when the target node existed before the upload started
	Size              int64
	MTime             time.Time
	Checksums         UploadChecksums
	IfMatch           string
	IfNoneMatch       string
	IfUnmodifiedSince time.Time
	// Where a new file goes, for a driver whose PrepareUpload creates it. Empty
	// leaves a missing node NotFound.
	ParentID string
	Name     string
}

type PrepareUploadResult struct {
	VersionCreated bool
	SizeDiff       int64
	SpaceOwner     *user.UserId // the space owner or a manager, for a new node; nil keeps the one already known
}

// RollbackInfo carries what a driver needs to undo PrepareUpload. NodeID and
// ParentID come from the upload session rather than the node, so a rollback can
// still release the quota of a node whose own metadata has become unreadable.
type RollbackInfo struct {
	NodeExisted bool // true when the target node had a prior version; drivers with nothing to undo for new nodes may no-op
	SizeDiff    int64
	NodeID      string
	ParentID    string
	Filename    string
	Size        int64
}

// FS is the interface to implement access to the storage.
type FS interface {
	// Minimal set for a readonly storage driver

	// Shutdown is called when the process is exiting to give the driver a chance to flush and close all open handles
	Shutdown(ctx context.Context) error

	// ListStorageSpaces lists the spaces in the storage.
	// FIXME The unrestricted parameter is an implementation detail of decomposedfs, remove it from the function?
	ListStorageSpaces(ctx context.Context, filter []*provider.ListStorageSpacesRequest_Filter, unrestricted bool) ([]*provider.StorageSpace, error)

	// GetQuota returns the quota on the referenced resource
	GetQuota(ctx context.Context, ref *provider.Reference) ( /*TotalBytes*/ uint64 /*UsedBytes*/, uint64 /*RemainingBytes*/, uint64, error)

	// GetMD returns the resuorce info for the referenced resource
	GetMD(ctx context.Context, ref *provider.Reference, mdKeys, fieldMask []string) (*provider.ResourceInfo, error)
	// ListFolder returns the resource infos for all children of the referenced resource
	ListFolder(ctx context.Context, ref *provider.Reference, mdKeys, fieldMask []string) ([]*provider.ResourceInfo, error)
	// Download returns a ReadCloser for the content of the referenced resource
	Download(ctx context.Context, ref *provider.Reference, openReaderfunc func(*provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error)

	// GetPathByID returns the path for the given resource id relative to the space root
	// It should only reveal the path visible to the current user to not leak the names uf unshared parent resources
	// FIXME should be deprecated in favor of calls to GetMD and the fieldmask 'path'
	GetPathByID(ctx context.Context, id *provider.ResourceId) (string, error)

	// Functions for a writeable storage space

	// CreateReference creates a resource of type reference
	CreateReference(ctx context.Context, path string, targetURI *url.URL) error
	// CreateDir creates a resource of type container
	CreateDir(ctx context.Context, ref *provider.Reference) error
	// TouchFile sets the mtime of a resource, creating an empty file if it does not exist
	// FIXME(OCISDEV-900) remove markprocessing bool: PrepareUpload marks the node the coordinator touched
	// FIXME the mtime should either be a time.Time or a CS3 Timestamp, not a string
	TouchFile(ctx context.Context, ref *provider.Reference, markprocessing bool, mtime string) error
	// Delete deletes a resource.
	// If the storage driver supports a recycle bin it should moves it to the recycle bin
	Delete(ctx context.Context, ref *provider.Reference) error
	// Move changes the path of a resource and returns resolved references and the
	// space owner for event publication. Drivers without this metadata may return
	// a nil result on success.
	Move(ctx context.Context, oldRef, newRef *provider.Reference) (*MoveResult, error)
	// MarkProcessing toggles a processing flag on the resource.
	MarkProcessing(ctx context.Context, ref *provider.Reference, processing bool, sessionID string) error
	// CommitUpload writes the staged bytes from source to the resource at ref.
	// source carries a Body, a Path or both. Caller owns source.Body and must close it
	// after CommitUpload returns.
	CommitUpload(ctx context.Context, ref *provider.Reference, sessionID string, source UploadSource) error
	// PrepareUpload is called after all bytes are received and before postprocessing begins.
	// Implementations may lock the target node, snapshot the previous version, write new metadata,
	// mark the node as processing for sessionID, and propagate size changes. Drivers that do not
	// require any of these steps may return immediately.
	PrepareUpload(ctx context.Context, ref *provider.Reference, sessionID string, info UploadInfo) (*PrepareUploadResult, error)
	// RollbackUpload reverts node state after a failed or aborted postprocessing run.
	// It is the inverse of PrepareUpload: restores previous metadata and reverts the optimistic
	// size propagation. The caller (coordinator) is responsible for unmarking the processing flag
	// and deleting the upload session files. Drivers that performed no work in PrepareUpload may return nil.
	// Callers must keep the session on a returned error: the rollback is retryable and info
	// carries state the driver cannot recover once the session files are gone.
	RollbackUpload(ctx context.Context, ref *provider.Reference, sessionID string, info RollbackInfo) error

	// Revisions

	// ListRevisions lists all revisions for the referenced resource
	ListRevisions(ctx context.Context, ref *provider.Reference) ([]*provider.FileVersion, error)
	// DownloadRevision downloads a revision
	DownloadRevision(ctx context.Context, ref *provider.Reference, key string, openReaderFunc func(md *provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error)
	// RestoreRevision restores a revision
	RestoreRevision(ctx context.Context, ref *provider.Reference, key string) error

	// Recyce bin

	// ListRecycle lists the content of the recycle bin
	ListRecycle(ctx context.Context, ref *provider.Reference, key, relativePath string) ([]*provider.RecycleItem, error)
	// RestoreRecycleItem restores an item from the recyle bin
	// if restoreRef is nil the resource should be restored at the original path
	RestoreRecycleItem(ctx context.Context, ref *provider.Reference, key, relativePath string, restoreRef *provider.Reference) error
	// PurgeRecycleItem removes a resource from the recycle bin
	PurgeRecycleItem(ctx context.Context, ref *provider.Reference, key, relativePath string) error
	// EmptyRecycle removes all resource from the recycle bin
	EmptyRecycle(ctx context.Context, ref *provider.Reference) error

	// Grants

	// AddGrant adds a grant to a resource
	AddGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error
	// DenyGrant marks a resource as denied for a recipient
	// The resource and its children must be completely hidden for the recipient
	DenyGrant(ctx context.Context, ref *provider.Reference, g *provider.Grantee) error
	// RemoveGrant removes a grant from a resource
	RemoveGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error
	// UpdateGrant updates a grant on a resource
	UpdateGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error
	// ListGrants lists all grants on a resource
	ListGrants(ctx context.Context, ref *provider.Reference) ([]*provider.Grant, error)

	// Arbitrary Metadata

	// SetArbitraryMetadata sets arbitraty metadata on a resource
	SetArbitraryMetadata(ctx context.Context, ref *provider.Reference, md *provider.ArbitraryMetadata) error
	// UnsetArbitraryMetadata removes arbitraty metadata from a resource
	UnsetArbitraryMetadata(ctx context.Context, ref *provider.Reference, keys []string) error

	// AddLabel adds a label to a resource
	AddLabel(ctx context.Context, ref *provider.Reference, userID *user.UserId, label string) error
	// RemoveLabel removes a label from a resource
	RemoveLabel(ctx context.Context, ref *provider.Reference, userID *user.UserId, label string) error

	// Locks

	// GetLock returns an existing lock on the given reference
	GetLock(ctx context.Context, ref *provider.Reference) (*provider.Lock, error)
	// SetLock puts a lock on the given reference
	SetLock(ctx context.Context, ref *provider.Reference, lock *provider.Lock) error
	// RefreshLock refreshes an existing lock on the given reference
	RefreshLock(ctx context.Context, ref *provider.Reference, lock *provider.Lock, existingLockID string) error
	// Unlock removes an existing lock from the given reference
	Unlock(ctx context.Context, ref *provider.Reference, lock *provider.Lock) error

	// Spaces

	// CreateStorageSpace creates a storage space
	CreateStorageSpace(ctx context.Context, req *provider.CreateStorageSpaceRequest) (*provider.CreateStorageSpaceResponse, error)
	// UpdateStorageSpace updates a storage space
	UpdateStorageSpace(ctx context.Context, req *provider.UpdateStorageSpaceRequest) (*provider.UpdateStorageSpaceResponse, error)
	// DeleteStorageSpace deletes a storage space
	DeleteStorageSpace(ctx context.Context, req *provider.DeleteStorageSpaceRequest) error

	// CreateHome creates a users home
	//
	// Deprecated: use CreateStorageSpace with type personal
	CreateHome(ctx context.Context) error
	// GetHome returns the path to the users home
	//
	// Deprecated: use ListStorageSpaces with type personal
	GetHome(ctx context.Context) (string, error)
}

// SpaceDisabledListableFS is the interface to implement for storage drivers that
// can list resources in disabled spaces.
type SpaceDisabledListableFS interface {
	WithDisabledSpaces() FS
}

// UploadSource carries the staged bytes for a CommitUpload call.
type UploadSource struct {
	Body   io.ReadCloser
	Length int64
	// Path is the staged file, if the caller has one. Drivers may move it into
	// place instead of copying Body; the file is gone after a successful commit.
	Path string

	// ScanResult is the antivirus verdict: empty means clean.
	ScanResult string
	// ScanDate is zero when the upload was not scanned.
	ScanDate time.Time
}

// UnscopeFunc is a function that unscopes a user
type UnscopeFunc func()

// Registry is the interface that storage registries implement
// for discovering storage providers
type Registry interface {
	// GetProvider returns the Address of the storage provider that should be used for the given space.
	// Use it to determine where to create a new storage space.
	GetProvider(ctx context.Context, space *provider.StorageSpace) (*registry.ProviderInfo, error)
	// ListProviders returns the storage providers that match the given filter
	ListProviders(ctx context.Context, filters map[string]string) ([]*registry.ProviderInfo, error)
}

// PathWrapper is the interface to implement for path transformations
type PathWrapper interface {
	Unwrap(ctx context.Context, rp string) (string, error)
	Wrap(ctx context.Context, rp string) (string, error)
}
