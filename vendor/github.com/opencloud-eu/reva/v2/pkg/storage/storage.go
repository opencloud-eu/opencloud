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

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	registry "github.com/cs3org/go-cs3apis/cs3/storage/registry/v1beta1"
	tusd "github.com/tus/tusd/v2/pkg/handler"
)

type MoveResult struct {
	SpaceOwner   *userpb.UserId
	OldReference *provider.Reference
	NewReference *provider.Reference
}

type CreateDirResult struct {
	SpaceOwner *userpb.UserId
	SpaceID    string
	ResourceID *provider.ResourceId
}

type TouchFileResult struct {
	SpaceOwner *userpb.UserId
	SpaceID    string
	ResourceID *provider.ResourceId
}

type RestoreRevisionResult struct {
	SpaceOwner *userpb.UserId
	SpaceID    string
}

type RestoreRecycleItemResult struct {
	SpaceOwner *userpb.UserId
	SpaceID    string
}

type SetLockResult struct {
	SpaceOwner *userpb.UserId
	SpaceID    string
}

type UnlockResult struct {
	SpaceOwner *userpb.UserId
	SpaceID    string
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
	CreateDir(ctx context.Context, ref *provider.Reference) (*CreateDirResult, error)
	// TouchFile sets the mtime of a resource, creating an empty file if it does not exist
	// FIXME(OCISDEV-900) remove markprocessing bool: coordinator calls MarkProcessing(true) explicitly after TouchFile
	// FIXME the mtime should either be a time.Time or a CS3 Timestamp, not a string
	TouchFile(ctx context.Context, ref *provider.Reference, markprocessing bool, mtime string) (*TouchFileResult, error)
	// Delete deletes a resource.
	// If the storage driver supports a recycle bin it should move it to the recycle bin
	// On success, returns a DeleteResult used by the wrapper to publish ItemTrashed.
	Delete(ctx context.Context, ref *provider.Reference) (*DeleteResult, error)
	// Move changes the path of a resource and returns resolved references and the
	// space owner for event publication. Drivers without this metadata may return
	// a nil result on success.
	Move(ctx context.Context, oldRef, newRef *provider.Reference) (*MoveResult, error)
	// InitiateUpload returns a list of protocols with urls that can be used to append bytes to a new upload session
	InitiateUpload(ctx context.Context, ref *provider.Reference, uploadLength int64, metadata map[string]string) (map[string]string, error)
	// Upload creates or updates a resource of type file with a new revision
	Upload(ctx context.Context, req UploadRequest, uploadFunc UploadFinishedFunc) (*provider.ResourceInfo, error)
	// MarkProcessing toggles a processing flag on the resource.
	MarkProcessing(ctx context.Context, ref *provider.Reference, processing bool, sessionID string) error
	// CommitUpload writes the staged bytes from source to the resource at ref.
	// Caller owns source.Body and must close it after CommitUpload returns.
	CommitUpload(ctx context.Context, ref *provider.Reference, sessionID string, source UploadSource) error

	// Revisions

	// ListRevisions lists all revisions for the referenced resource
	ListRevisions(ctx context.Context, ref *provider.Reference) ([]*provider.FileVersion, error)
	// DownloadRevision downloads a revision
	DownloadRevision(ctx context.Context, ref *provider.Reference, key string, openReaderFunc func(md *provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error)
	// RestoreRevision restores a revision
	RestoreRevision(ctx context.Context, ref *provider.Reference, key string) (*RestoreRevisionResult, error)

	// Recyce bin

	// ListRecycle lists the content of the recycle bin
	ListRecycle(ctx context.Context, ref *provider.Reference, key, relativePath string) ([]*provider.RecycleItem, error)
	// RestoreRecycleItem restores an item from the recyle bin
	// if restoreRef is nil the resource should be restored at the original path
	RestoreRecycleItem(ctx context.Context, ref *provider.Reference, key, relativePath string, restoreRef *provider.Reference) (*RestoreRecycleItemResult, error)
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
	AddLabel(ctx context.Context, ref *provider.Reference, userID *userpb.UserId, label string) error
	// RemoveLabel removes a label from a resource
	RemoveLabel(ctx context.Context, ref *provider.Reference, userID *userpb.UserId, label string) error

	// Locks

	// GetLock returns an existing lock on the given reference
	GetLock(ctx context.Context, ref *provider.Reference) (*provider.Lock, error)
	// SetLock puts a lock on the given reference
	SetLock(ctx context.Context, ref *provider.Reference, lock *provider.Lock) (*SetLockResult, error)
	// RefreshLock refreshes an existing lock on the given reference
	RefreshLock(ctx context.Context, ref *provider.Reference, lock *provider.Lock, existingLockID string) error
	// Unlock removes an existing lock from the given reference
	Unlock(ctx context.Context, ref *provider.Reference, lock *provider.Lock) (*UnlockResult, error)

	// Spaces

	// CreateStorageSpace creates a storage space
	CreateStorageSpace(ctx context.Context, req *provider.CreateStorageSpaceRequest) (*provider.CreateStorageSpaceResponse, error)
	// UpdateStorageSpace updates a storage space
	UpdateStorageSpace(ctx context.Context, req *provider.UpdateStorageSpaceRequest) (*provider.UpdateStorageSpaceResponse, error)
	// DeleteStorageSpace deletes a storage space.
	// On a successful delete (purge), returns a DeleteStorageSpaceResult used by
	// the wrapper to publish SpaceDeleted. SpaceDisabled does not need this data.
	DeleteStorageSpace(ctx context.Context, req *provider.DeleteStorageSpaceRequest) (*DeleteStorageSpaceResult, error)

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
}

// DeleteResult is returned by FS.Delete on success. It carries the data the
// storageprovider wrapper needs to publish the ItemTrashed event.
type DeleteResult struct {
	// SpaceOwner is the owner of the space the deleted resource belonged to.
	SpaceOwner *userpb.UserId
	// ResourceId is the stable identifier of the deleted resource, used as ItemTrashed.ID in the published event.
	ResourceId *provider.ResourceId
}

// DeleteStorageSpaceResult is returned by FS.DeleteStorageSpace on a successful
// purge. It carries the data the storageprovider wrapper needs to publish the
// SpaceDeleted event. The SpaceDisabled flow does not consume it.
type DeleteStorageSpaceResult struct {
	// SpaceName is the name of the space at the time of deletion.
	SpaceName string
	// FinalMembers is the grant map of the space at the time of deletion,
	// keyed by user/group opaque id.
	FinalMembers map[string]provider.ResourcePermissions
}

// UnscopeFunc is a function that unscopes a user
type UnscopeFunc func()

// Composable is the interface that a struct needs to implement
// to be composable, so that it can support the TUS methods
type ComposableFS interface {
	UseIn(composer *tusd.StoreComposer)
}

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
