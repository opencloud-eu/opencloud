// Copyright 2018-2022 CERN
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
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
)

// Disk represents a disk metadata storage
type Disk struct {
	dataDir string
}

// NewDiskStorage returns a new disk storage instance
func NewDiskStorage(dataDir string) (s Storage, err error) {
	return &Disk{
		dataDir: dataDir,
	}, nil
}

// Init creates the metadata space
func (disk *Disk) Init(_ context.Context, _ string) (err error) {
	return os.MkdirAll(disk.dataDir, 0777)
}

// Backend returns the backend name of the storage
func (disk *Disk) Backend() string {
	return "disk"
}

// Stat returns the metadata for the given path
func (disk *Disk) Stat(ctx context.Context, path string) (*provider.ResourceInfo, error) {
	info, err := os.Stat(disk.targetPath(path))
	if err != nil {
		var pathError *fs.PathError
		if errors.As(err, &pathError) {
			return nil, errtypes.NotFound("path not found: " + path)
		}
		return nil, err
	}
	entry := &provider.ResourceInfo{
		Type:  provider.ResourceType_RESOURCE_TYPE_FILE,
		Path:  "./" + info.Name(),
		Name:  info.Name(),
		Mtime: &typesv1beta1.Timestamp{Seconds: uint64(info.ModTime().Unix()), Nanos: uint32(info.ModTime().Nanosecond())},
	}
	if info.IsDir() {
		entry.Type = provider.ResourceType_RESOURCE_TYPE_CONTAINER
	}
	entry.Etag, err = calcEtag(info.ModTime(), info.Size())
	if err != nil {
		return nil, err
	}
	return entry, nil
}

// SimpleUpload stores a file on disk
func (disk *Disk) SimpleUpload(ctx context.Context, uploadpath string, content []byte) error {
	_, err := disk.Upload(ctx, UploadRequest{
		Path:    uploadpath,
		Content: content,
	})
	return err
}

// Upload stores a file on disk
func (disk *Disk) Upload(_ context.Context, req UploadRequest) (*UploadResponse, error) {
	p := disk.targetPath(req.Path)

	// IfNoneMatch: ["*"] means create the file only if it does not already
	// exist. Use O_EXCL so the check and the create are atomic on the local
	// filesystem.
	for _, tag := range req.IfNoneMatch {
		if tag != "*" {
			continue
		}
		f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			if errors.Is(err, os.ErrExist) {
				return nil, errtypes.AlreadyExists(p)
			}
			return nil, err
		}
		if _, err := f.Write(req.Content); err != nil {
			_ = f.Close()
			return nil, err
		}
		if err := f.Close(); err != nil {
			return nil, err
		}
		info, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		res := &UploadResponse{}
		res.Etag, err = calcEtag(info.ModTime(), info.Size())
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	if req.IfMatchEtag != "" {
		info, err := os.Stat(p)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		} else if err == nil {
			etag, err := calcEtag(info.ModTime(), info.Size())
			if err != nil {
				return nil, err
			}
			if etag != req.IfMatchEtag {
				return nil, errtypes.PreconditionFailed("etag mismatch")
			}
		}
	}
	if req.IfUnmodifiedSince != (time.Time{}) {
		info, err := os.Stat(p)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		} else if err == nil {
			if info.ModTime().After(req.IfUnmodifiedSince) {
				return nil, errtypes.PreconditionFailed(fmt.Sprintf("resource has been modified, mtime: %s > since %s", info.ModTime(), req.IfUnmodifiedSince))
			}
		}
	}
	err := os.WriteFile(p, req.Content, 0644)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(disk.targetPath(req.Path))
	if err != nil {
		return nil, err
	}
	res := &UploadResponse{}
	res.Etag, err = calcEtag(info.ModTime(), info.Size())
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Download reads a file from disk
func (disk *Disk) Download(_ context.Context, req DownloadRequest) (*DownloadResponse, error) {
	var err error

	f, err := os.Open(disk.targetPath(req.Path))
	if err != nil {
		var pathError *fs.PathError
		if errors.As(err, &pathError) {
			return nil, errtypes.NotFound("path not found: " + disk.targetPath(req.Path))
		}
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	res := DownloadResponse{}
	res.Mtime = info.ModTime()
	res.Etag, err = calcEtag(info.ModTime(), info.Size())
	if err != nil {
		return nil, err
	}

	res.Content, err = io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// SimpleDownload reads a file from disk
func (disk *Disk) SimpleDownload(ctx context.Context, downloadpath string) ([]byte, error) {
	res, err := disk.Download(ctx, DownloadRequest{Path: downloadpath})
	if err != nil {
		return nil, err
	}
	return res.Content, nil
}

// Delete deletes a path
func (disk *Disk) Delete(_ context.Context, path string) error {
	return os.Remove(disk.targetPath(path))
}

// ReadDir returns the resource infos in a given directory
func (disk *Disk) ReadDir(_ context.Context, p string) ([]string, error) {
	infos, err := os.ReadDir(disk.targetPath(p))
	if err != nil {
		if _, ok := err.(*fs.PathError); ok {
			return []string{}, nil
		}
		return nil, err
	}

	entries := make([]string, 0, len(infos))
	for _, entry := range infos {
		entries = append(entries, filepath.Join(p, entry.Name()))
	}
	return entries, nil
}

// ListDir returns a list of ResourceInfos for the entries in a given directory
func (disk *Disk) ListDir(ctx context.Context, path string) ([]*provider.ResourceInfo, error) {
	diskEntries, err := os.ReadDir(disk.targetPath(path))
	if err != nil {
		if _, ok := err.(*fs.PathError); ok {
			return []*provider.ResourceInfo{}, nil
		}
		return nil, err
	}

	entries := make([]*provider.ResourceInfo, 0, len(diskEntries))
	for _, diskEntry := range diskEntries {
		info, err := diskEntry.Info()
		if err != nil {
			continue
		}

		entry := &provider.ResourceInfo{
			Type:  provider.ResourceType_RESOURCE_TYPE_FILE,
			Path:  "./" + info.Name(),
			Name:  info.Name(),
			Mtime: &typesv1beta1.Timestamp{Seconds: uint64(info.ModTime().Unix()), Nanos: uint32(info.ModTime().Nanosecond())},
		}
		if info.IsDir() {
			entry.Type = provider.ResourceType_RESOURCE_TYPE_CONTAINER
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// MakeDirIfNotExist will create a root node in the metadata storage. Requires an authenticated context.
func (disk *Disk) MakeDirIfNotExist(_ context.Context, path string) error {
	return os.MkdirAll(disk.targetPath(path), 0777)
}

// CreateSymlink creates a symlink
func (disk *Disk) CreateSymlink(_ context.Context, oldname, newname string) error {
	return os.Symlink(oldname, disk.targetPath(newname))
}

// ResolveSymlink resolves a symlink
func (disk *Disk) ResolveSymlink(_ context.Context, path string) (string, error) {
	return os.Readlink(disk.targetPath(path))
}

func (disk *Disk) targetPath(p string) string {
	return filepath.Join(disk.dataDir, filepath.Join("/", p))
}
