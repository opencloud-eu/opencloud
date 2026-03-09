// Copyright 2018-2024 CERN
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

package middleware

import (
	"context"
	"io"
	"net/url"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	tusd "github.com/tus/tusd/v2/pkg/handler"

	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/decomposedfs/upload"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
)

// UnHook is a function that is called after the actual method is executed.
type UnHook func() error

// Hook is a function that is called before the actual method is executed.
type Hook func(methodName string, ctx context.Context, spaceID string) (context.Context, UnHook, error)

// FS is a storage.FS implementation that wraps another storage.FS and calls hooks before and after each method.
type FS struct {
	next  storage.FS
	hooks []Hook
}

func NewFS(next storage.FS, hooks ...Hook) *FS {
	return &FS{
		next:  next,
		hooks: hooks,
	}
}

// ListUploadSessions returns the upload sessions matching the given filter
func (f *FS) ListUploadSessions(ctx context.Context, filter storage.UploadSessionFilter) ([]storage.UploadSession, error) {
	return f.next.(storage.UploadSessionLister).ListUploadSessions(ctx, filter)
}

// UseIn tells the tus upload middleware which extensions it supports.
func (f *FS) UseIn(composer *tusd.StoreComposer) {
	f.next.(storage.ComposableFS).UseIn(composer)
}

// NewUpload returns a new tus Upload instance
func (f *FS) NewUpload(ctx context.Context, info tusd.FileInfo) (upload tusd.Upload, err error) {
	return f.next.(tusd.DataStore).NewUpload(ctx, info)
}

// NewUpload returns a new tus Upload instance
func (f *FS) GetUpload(ctx context.Context, id string) (upload tusd.Upload, err error) {
	return f.next.(tusd.DataStore).GetUpload(ctx, id)
}

// AsTerminatableUpload returns a TerminatableUpload
// To implement the termination extension as specified in https://tus.io/protocols/resumable-upload.html#termination
// the storage needs to implement AsTerminatableUpload
func (f *FS) AsTerminatableUpload(up tusd.Upload) tusd.TerminatableUpload {
	return up.(*upload.OcisSession)
}

// AsLengthDeclarableUpload returns a LengthDeclarableUpload
// To implement the creation-defer-length extension as specified in https://tus.io/protocols/resumable-upload.html#creation
// the storage needs to implement AsLengthDeclarableUpload
func (f *FS) AsLengthDeclarableUpload(up tusd.Upload) tusd.LengthDeclarableUpload {
	return up.(*upload.OcisSession)
}

// AsConcatableUpload returns a ConcatableUpload
// To implement the concatenation extension as specified in https://tus.io/protocols/resumable-upload.html#concatenation
// the storage needs to implement AsConcatableUpload
func (f *FS) AsConcatableUpload(up tusd.Upload) tusd.ConcatableUpload {
	return up.(*upload.OcisSession)
}

func (f *FS) GetHome(ctx context.Context) (string, error) {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("GetHome", ctx, "")
		if err != nil {
			return "", err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0, res1 := f.next.GetHome(ctx) //nolint:staticcheck // we're just exposing the deprecated method here

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return "", err
		}
	}

	return res0, res1
}

func (f *FS) CreateHome(ctx context.Context) error {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("CreateHome", ctx, "")
		if err != nil {
			return err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0 := f.next.CreateHome(ctx) //nolint:staticcheck // we're just exposing the deprecated method here

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return err
		}
	}

	return res0
}

func (f *FS) CreateDir(ctx context.Context, ref *provider.Reference) error {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("CreateDir", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0 := f.next.CreateDir(ctx, ref)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return err
		}
	}

	return res0
}

func (f *FS) TouchFile(ctx context.Context, ref *provider.Reference, markprocessing bool, mtime string) error {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("TouchFile", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0 := f.next.TouchFile(ctx, ref, markprocessing, mtime)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return err
		}
	}

	return res0
}

func (f *FS) Delete(ctx context.Context, ref *provider.Reference) error {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("Delete", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0 := f.next.Delete(ctx, ref)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return err
		}
	}

	return res0
}

func (f *FS) Move(ctx context.Context, oldRef, newRef *provider.Reference) error {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("Move", ctx, oldRef.GetResourceId().GetSpaceId())
		if err != nil {
			return err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0 := f.next.Move(ctx, oldRef, newRef)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return err
		}
	}

	return res0
}

func (f *FS) GetMD(ctx context.Context, ref *provider.Reference, mdKeys, fieldMask []string) (*provider.ResourceInfo, error) {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("GetMD", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return nil, err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0, res1 := f.next.GetMD(ctx, ref, mdKeys, fieldMask)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return nil, err
		}
	}

	return res0, res1
}

func (f *FS) ListFolder(ctx context.Context, ref *provider.Reference, mdKeys, fieldMask []string) ([]*provider.ResourceInfo, error) {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("ListFolder", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return nil, err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0, res1 := f.next.ListFolder(ctx, ref, mdKeys, fieldMask)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return nil, err
		}
	}

	return res0, res1
}

func (f *FS) InitiateUpload(ctx context.Context, ref *provider.Reference, uploadLength int64, metadata map[string]string) (map[string]string, error) {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("InitiateUpload", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return nil, err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0, res1 := f.next.InitiateUpload(ctx, ref, uploadLength, metadata)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return nil, err
		}
	}

	return res0, res1
}

func (f *FS) Upload(ctx context.Context, req storage.UploadRequest, uploadFunc storage.UploadFinishedFunc) (*provider.ResourceInfo, error) {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("Upload", ctx, req.Ref.GetResourceId().GetSpaceId())
		if err != nil {
			return &provider.ResourceInfo{}, err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0, res1 := f.next.Upload(ctx, req, uploadFunc)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return &provider.ResourceInfo{}, err
		}
	}

	return res0, res1
}

func (f *FS) Download(ctx context.Context, ref *provider.Reference, openReaderFunc func(md *provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error) {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("Download", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return nil, nil, err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0, res1, res2 := f.next.Download(ctx, ref, openReaderFunc)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return nil, nil, err
		}
	}

	return res0, res1, res2
}

func (f *FS) ListRevisions(ctx context.Context, ref *provider.Reference) ([]*provider.FileVersion, error) {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("ListRevisions", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return nil, err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0, res1 := f.next.ListRevisions(ctx, ref)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return nil, err
		}
	}

	return res0, res1
}

func (f *FS) DownloadRevision(ctx context.Context, ref *provider.Reference, key string, openReaderFunc func(md *provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error) {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("DownloadRevision", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return nil, nil, err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0, res1, res2 := f.next.DownloadRevision(ctx, ref, key, openReaderFunc)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return nil, nil, err
		}
	}

	return res0, res1, res2
}

func (f *FS) RestoreRevision(ctx context.Context, ref *provider.Reference, key string) error {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("RestoreRevision", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0 := f.next.RestoreRevision(ctx, ref, key)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return err
		}
	}

	return res0
}

func (f *FS) ListRecycle(ctx context.Context, ref *provider.Reference, key, relativePath string) ([]*provider.RecycleItem, error) {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("ListRecycle", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return nil, err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0, res1 := f.next.ListRecycle(ctx, ref, key, relativePath)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return nil, err
		}
	}

	return res0, res1
}

func (f *FS) RestoreRecycleItem(ctx context.Context, ref *provider.Reference, key, relativePath string, restoreRef *provider.Reference) error {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("RestoreRecycleItem", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0 := f.next.RestoreRecycleItem(ctx, ref, key, relativePath, restoreRef)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return err
		}
	}

	return res0
}

func (f *FS) PurgeRecycleItem(ctx context.Context, ref *provider.Reference, key, relativePath string) error {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("PurgeRecycleItem", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0 := f.next.PurgeRecycleItem(ctx, ref, key, relativePath)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return err
		}
	}

	return res0
}

func (f *FS) EmptyRecycle(ctx context.Context, ref *provider.Reference) error {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("EmptyRecycle", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0 := f.next.EmptyRecycle(ctx, ref)
	for _, unhook := range unhooks {
		_ = unhook()
	}
	return res0
}

func (f *FS) GetPathByID(ctx context.Context, id *provider.ResourceId) (string, error) {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("GetPathByID", ctx, id.GetSpaceId())
		if err != nil {
			return "", err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0, res1 := f.next.GetPathByID(ctx, id)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return "", err
		}
	}

	return res0, res1
}

func (f *FS) AddGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("AddGrant", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0 := f.next.AddGrant(ctx, ref, g)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return err
		}
	}

	return res0
}

func (f *FS) DenyGrant(ctx context.Context, ref *provider.Reference, g *provider.Grantee) error {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("DenyGrant", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0 := f.next.DenyGrant(ctx, ref, g)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return err
		}
	}

	return res0
}

func (f *FS) RemoveGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("RemoveGrant", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0 := f.next.RemoveGrant(ctx, ref, g)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return err
		}
	}

	return res0
}

func (f *FS) UpdateGrant(ctx context.Context, ref *provider.Reference, g *provider.Grant) error {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("UpdateGrant", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0 := f.next.UpdateGrant(ctx, ref, g)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return err
		}
	}

	return res0
}

func (f *FS) ListGrants(ctx context.Context, ref *provider.Reference) ([]*provider.Grant, error) {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("ListGrants", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return nil, err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0, res1 := f.next.ListGrants(ctx, ref)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return nil, err
		}
	}

	return res0, res1
}

func (f *FS) GetQuota(ctx context.Context, ref *provider.Reference) (uint64, uint64, uint64, error) {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("GetQuota", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return 0, 0, 0, err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0, res1, res2, res3 := f.next.GetQuota(ctx, ref)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return 0, 0, 0, err
		}
	}

	return res0, res1, res2, res3
}

func (f *FS) CreateReference(ctx context.Context, path string, targetURI *url.URL) error {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("CreateReference", ctx, "")
		if err != nil {
			return err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0 := f.next.CreateReference(ctx, path, targetURI)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return err
		}
	}

	return res0
}

func (f *FS) Shutdown(ctx context.Context) error {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("Shutdown", ctx, "")
		if err != nil {
			return err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0 := f.next.Shutdown(ctx)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return err
		}
	}

	return res0
}

func (f *FS) SetArbitraryMetadata(ctx context.Context, ref *provider.Reference, md *provider.ArbitraryMetadata) error {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("SetArbitraryMetadata", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0 := f.next.SetArbitraryMetadata(ctx, ref, md)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return err
		}
	}

	return res0
}

func (f *FS) UnsetArbitraryMetadata(ctx context.Context, ref *provider.Reference, keys []string) error {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("UnsetArbitraryMetadata", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0 := f.next.UnsetArbitraryMetadata(ctx, ref, keys)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return err
		}
	}

	return res0
}

func (f *FS) SetLock(ctx context.Context, ref *provider.Reference, lock *provider.Lock) error {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("SetLock", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0 := f.next.SetLock(ctx, ref, lock)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return err
		}
	}

	return res0
}

func (f *FS) GetLock(ctx context.Context, ref *provider.Reference) (*provider.Lock, error) {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("GetLock", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return nil, err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0, res1 := f.next.GetLock(ctx, ref)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return nil, err
		}
	}

	return res0, res1
}

func (f *FS) RefreshLock(ctx context.Context, ref *provider.Reference, lock *provider.Lock, existingLockID string) error {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("RefreshLock", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0 := f.next.RefreshLock(ctx, ref, lock, existingLockID)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return err
		}
	}

	return res0
}

func (f *FS) Unlock(ctx context.Context, ref *provider.Reference, lock *provider.Lock) error {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("Unlock", ctx, ref.GetResourceId().GetSpaceId())
		if err != nil {
			return err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0 := f.next.Unlock(ctx, ref, lock)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return err
		}
	}

	return res0
}

func (f *FS) ListStorageSpaces(ctx context.Context, filter []*provider.ListStorageSpacesRequest_Filter, unrestricted bool) ([]*provider.StorageSpace, error) {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("ListStorageSpaces", ctx, "")
		if err != nil {
			return nil, err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0, res1 := f.next.ListStorageSpaces(ctx, filter, unrestricted)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return nil, err
		}
	}

	return res0, res1
}

func (f *FS) CreateStorageSpace(ctx context.Context, req *provider.CreateStorageSpaceRequest) (*provider.CreateStorageSpaceResponse, error) {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("CreateStorageSpace", ctx, "")
		if err != nil {
			return nil, err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0, res1 := f.next.CreateStorageSpace(ctx, req)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return nil, err
		}
	}

	return res0, res1
}

func (f *FS) UpdateStorageSpace(ctx context.Context, req *provider.UpdateStorageSpaceRequest) (*provider.UpdateStorageSpaceResponse, error) {
	var (
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		id, err := storagespace.ParseID(req.StorageSpace.GetId().GetOpaqueId())
		if err != nil {
			return nil, err
		}
		ctx, unhook, err = hook("UpdateStorageSpace", ctx, id.SpaceId)
		if err != nil {
			return nil, err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0, res1 := f.next.UpdateStorageSpace(ctx, req)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return nil, err
		}
	}

	return res0, res1
}

func (f *FS) DeleteStorageSpace(ctx context.Context, req *provider.DeleteStorageSpaceRequest) error {
	var (
		err     error
		unhook  UnHook
		unhooks []UnHook
	)
	for _, hook := range f.hooks {
		ctx, unhook, err = hook("DeleteStorageSpace", ctx, req.GetId().GetOpaqueId())
		if err != nil {
			return err
		}
		if unhook != nil {
			unhooks = append(unhooks, unhook)
		}
	}

	res0 := f.next.DeleteStorageSpace(ctx, req)

	for _, unhook := range unhooks {
		if err := unhook(); err != nil {
			return err
		}
	}

	return res0
}
