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

package hello

import (
	"context"
	"io"
	"net/url"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"

	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
)

// hellofs is readonly so these remain unimplemented

// CreateReference creates a resource of type reference
func (fs *hellofs) CreateReference(ctx context.Context, path string, targetURI *url.URL) error {
	return errtypes.NotSupported("unimplemented")
}

// CreateStorageSpace creates a storage space
func (fs *hellofs) CreateStorageSpace(ctx context.Context, req *provider.CreateStorageSpaceRequest) (*provider.CreateStorageSpaceResponse, error) {
	return nil, errtypes.NotSupported("unimplemented: CreateStorageSpace")
}

// UpdateStorageSpace updates a storage space
func (fs *hellofs) UpdateStorageSpace(ctx context.Context, req *provider.UpdateStorageSpaceRequest) (*provider.UpdateStorageSpaceResponse, error) {
	return nil, errtypes.NotSupported("update storage space")
}

// DeleteStorageSpace deletes a storage space
func (fs *hellofs) DeleteStorageSpace(ctx context.Context, req *provider.DeleteStorageSpaceRequest) error {
	return errtypes.NotSupported("delete storage space")
}

// CreateDir creates a resource of type container
func (fs *hellofs) CreateDir(ctx context.Context, ref *provider.Reference) error {
	return errtypes.NotSupported("unimplemented")
}

// TouchFile sets the mtime of a resource, creating an empty file if it does not exist
// FIXME the markprocessing flag is an implementation detail of decomposedfs, remove it from the function
// FIXME the mtime should either be a time.Time or a CS3 Timestamp, not a string
func (fs *hellofs) TouchFile(ctx context.Context, ref *provider.Reference, _ bool, _ string) error {
	return errtypes.NotSupported("unimplemented")
}

// Delete deletes a resource.
// If the storage driver supports a recycle bin it should moves it to the recycle bin
func (fs *hellofs) Delete(ctx context.Context, ref *provider.Reference) error {
	return errtypes.NotSupported("unimplemented")
}

// Move changes the path of a resource
func (fs *hellofs) Move(ctx context.Context, oldRef, newRef *provider.Reference) error {
	return errtypes.NotSupported("unimplemented")
}

// Upload creates or updates a resource of type file with a new revision
func (fs *hellofs) Upload(ctx context.Context, req storage.UploadRequest, uff storage.UploadFinishedFunc) (*provider.ResourceInfo, error) {
	return nil, errtypes.NotSupported("hellofs: upload not supported")
}

// InitiateUpload returns a list of protocols with urls that can be used to append bytes to a new upload session
func (fs *hellofs) InitiateUpload(ctx context.Context, ref *provider.Reference, uploadLength int64, metadata map[string]string) (map[string]string, error) {
	return nil, errtypes.NotSupported("hellofs: initiate upload not supported")
}

// grants

// DenyGrant marks a resource as denied for a recipient
// The resource and its children must be completely hidden for the recipient
func (fs *hellofs) DenyGrant(ctx context.Context, ref *provider.Reference, g *provider.Grantee) error {
	return errtypes.NotSupported("hellofs: deny grant not supported")
}

// AddGrant adds a grant to a resource
func (fs *hellofs) AddGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error {
	return errtypes.NotSupported("unimplemented")
}

// ListGrants lists all grants on a resource
func (fs *hellofs) ListGrants(ctx context.Context, ref *provider.Reference) ([]*provider.Grant, error) {
	return nil, errtypes.NotSupported("unimplemented")
}

// RemoveGrant removes a grant from a resource
func (fs *hellofs) RemoveGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error {
	return errtypes.NotSupported("unimplemented")
}

// UpdateGrant updates a grant on a resource
func (fs *hellofs) UpdateGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error {
	return errtypes.NotSupported("unimplemented")
}

// arbitrary metadata

// SetArbitraryMetadata sets arbitraty metadata on a resource
func (fs *hellofs) SetArbitraryMetadata(ctx context.Context, ref *provider.Reference, md *provider.ArbitraryMetadata) error {
	return errtypes.NotSupported("unimplemented")
}

// UnsetArbitraryMetadata removes arbitraty metadata from a resource
func (fs *hellofs) UnsetArbitraryMetadata(ctx context.Context, ref *provider.Reference, keys []string) error {
	return errtypes.NotSupported("unimplemented")
}

// locks

// GetLock returns an existing lock on the given reference
func (fs *hellofs) GetLock(ctx context.Context, ref *provider.Reference) (*provider.Lock, error) {
	return nil, errtypes.NotSupported("unimplemented")
}

// SetLock puts a lock on the given reference
func (fs *hellofs) SetLock(ctx context.Context, ref *provider.Reference, lock *provider.Lock) error {
	return errtypes.NotSupported("unimplemented")
}

// RefreshLock refreshes an existing lock on the given reference
func (fs *hellofs) RefreshLock(ctx context.Context, ref *provider.Reference, lock *provider.Lock, existingLockID string) error {
	return errtypes.NotSupported("unimplemented")
}

// Unlock removes an existing lock from the given reference
func (fs *hellofs) Unlock(ctx context.Context, ref *provider.Reference, lock *provider.Lock) error {
	return errtypes.NotSupported("unimplemented")
}

// revisions

// ListRevisions lists all revisions for the referenced resource
func (fs *hellofs) ListRevisions(ctx context.Context, ref *provider.Reference) ([]*provider.FileVersion, error) {
	return nil, errtypes.NotSupported("unimplemented")
}

// DownloadRevision downloads a revision
func (fs *hellofs) DownloadRevision(ctx context.Context, ref *provider.Reference, revisionKey string, openReaderFunc func(md *provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error) {
	return nil, nil, errtypes.NotSupported("unimplemented")
}

// RestoreRevision restores a revision
func (fs *hellofs) RestoreRevision(ctx context.Context, ref *provider.Reference, revisionKey string) error {
	return errtypes.NotSupported("unimplemented")
}

// trash

// PurgeRecycleItem removes a resource from the recycle bin
func (fs *hellofs) PurgeRecycleItem(ctx context.Context, ref *provider.Reference, key, relativePath string) error {
	return errtypes.NotSupported("unimplemented")
}

// EmptyRecycle removes all resource from the recycle bin
func (fs *hellofs) EmptyRecycle(ctx context.Context, ref *provider.Reference) error {
	return errtypes.NotSupported("unimplemented")
}

// ListRecycle lists the content of the recycle bin
func (fs *hellofs) ListRecycle(ctx context.Context, ref *provider.Reference, key, relativePath string) ([]*provider.RecycleItem, error) {
	return nil, errtypes.NotSupported("unimplemented")
}

// RestoreRecycleItem restores an item from the recyle bin
// if restoreRef is nil the resource should be restored at the original path
func (fs *hellofs) RestoreRecycleItem(ctx context.Context, ref *provider.Reference, key, relativePath string, restoreRef *provider.Reference) error {
	return errtypes.NotSupported("unimplemented")
}

// CreateHome creates a users home
//
// Deprecated: use CreateStorageSpace with type personal
func (fs *hellofs) CreateHome(ctx context.Context) error {
	return errtypes.NotSupported("unimplemented")
}

// GetHome returns the path to the users home
//
// Deprecated: use ListStorageSpaces with type personal
func (fs *hellofs) GetHome(ctx context.Context) (string, error) {
	return "", errtypes.NotSupported("unimplemented")
}
