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
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
)

// UploadRequest represents an upload request and its options
type UploadRequest struct {
	Path    string
	Content []byte

	IfMatchEtag       string
	IfNoneMatch       []string
	IfUnmodifiedSince time.Time
	MTime             time.Time
}

// UploadResponse represents a upload response
type UploadResponse struct {
	Etag   string
	FileID string // only for cs3 storage
}

// DownloadRequest represents a download request and its options
type DownloadRequest struct {
	Path        string
	IfNoneMatch []string
}

// DownloadResponse represents a download response and its options
type DownloadResponse struct {
	Content []byte

	Etag  string
	Mtime time.Time
}

// Storage is the interface to maintain metadata in a storage
type Storage interface {
	Backend() string

	Init(ctx context.Context, name string) (err error)
	Upload(ctx context.Context, req UploadRequest) (*UploadResponse, error)
	Download(ctx context.Context, req DownloadRequest) (*DownloadResponse, error)
	SimpleUpload(ctx context.Context, uploadpath string, content []byte) error
	SimpleDownload(ctx context.Context, path string) ([]byte, error)
	Delete(ctx context.Context, path string) error
	Stat(ctx context.Context, path string) (*provider.ResourceInfo, error)

	ReadDir(ctx context.Context, path string) ([]string, error)
	ListDir(ctx context.Context, path string) ([]*provider.ResourceInfo, error)

	CreateSymlink(ctx context.Context, oldname, newname string) error
	ResolveSymlink(ctx context.Context, name string) (string, error)

	MakeDirIfNotExist(ctx context.Context, name string) error
}

func calcEtag(mtime time.Time, size int64) (string, error) {
	h := md5.New()
	if err := binary.Write(h, binary.BigEndian, mtime.UnixNano()); err != nil {
		return "", err
	}
	if err := binary.Write(h, binary.BigEndian, size); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
