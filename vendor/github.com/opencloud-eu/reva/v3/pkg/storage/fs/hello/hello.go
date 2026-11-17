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
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
	"github.com/opencloud-eu/reva/v3/pkg/events"
	"github.com/opencloud-eu/reva/v3/pkg/storage"
	"github.com/opencloud-eu/reva/v3/pkg/storage/fs/registry"
	"github.com/opencloud-eu/reva/v3/pkg/utils"
)

func init() {
	registry.Register("hello", New)
}

type hellofs struct {
	bootTime time.Time
}

const (
	storageid = "hello-storage-id"
	spaceid   = "hello-space-id"
	rootid    = "hello-root-id"
	fileid    = "hello-file-id"
	filename  = "Hello world.txt"
	content   = "Hello world!"
)

func (fs *hellofs) space(withRoot bool) *provider.StorageSpace {
	s := &provider.StorageSpace{
		Id: &provider.StorageSpaceId{OpaqueId: spaceid},
		Root: &provider.ResourceId{
			StorageId: storageid,
			SpaceId:   spaceid,
			OpaqueId:  rootid,
		},
		Quota: &provider.Quota{
			QuotaMaxBytes: uint64(len(content)),
			QuotaMaxFiles: 1,
		},
		Name:      "Hello Space",
		SpaceType: "project",
		RootInfo:  fs.rootInfo(),
		Mtime:     utils.TimeToTS(fs.bootTime),
	}
	// FIXME move this to the CS3 API
	s.Opaque = utils.AppendPlainToOpaque(s.Opaque, "spaceAlias", "project/hello")

	if withRoot {
		s.RootInfo = fs.rootInfo()
	}
	return s
}

func (fs *hellofs) rootInfo() *provider.ResourceInfo {
	return &provider.ResourceInfo{
		Type: provider.ResourceType_RESOURCE_TYPE_CONTAINER,
		Id: &provider.ResourceId{
			StorageId: storageid,
			SpaceId:   spaceid,
			OpaqueId:  rootid,
		},
		Etag:     calcEtag(fs.bootTime, rootid),
		MimeType: "httpd/unix-directory",
		Mtime:    utils.TimeToTS(fs.bootTime),
		Path:     ".",
		PermissionSet: &provider.ResourcePermissions{
			GetPath:              true,
			GetQuota:             true,
			InitiateFileDownload: true,
			Stat:                 true,
			ListContainer:        true,
		},
		Size: uint64(len(content)),
	}
}

func (fs *hellofs) fileInfo() *provider.ResourceInfo {
	return &provider.ResourceInfo{
		Type: provider.ResourceType_RESOURCE_TYPE_FILE,
		Id: &provider.ResourceId{
			StorageId: storageid,
			SpaceId:   spaceid,
			OpaqueId:  fileid,
		},
		Etag:     calcEtag(fs.bootTime, fileid),
		MimeType: "text/plain",
		Mtime:    utils.TimeToTS(fs.bootTime),
		Path:     ".",
		PermissionSet: &provider.ResourcePermissions{
			GetPath:              true,
			GetQuota:             true,
			InitiateFileDownload: true,
			Stat:                 true,
			ListContainer:        true,
		},
		Size: uint64(len(content)),
		ParentId: &provider.ResourceId{
			StorageId: storageid,
			SpaceId:   spaceid,
			OpaqueId:  rootid,
		},
		Name:  filename,
		Space: fs.space(false),
	}
}

func calcEtag(t time.Time, nodeid string) string {
	h := md5.New()
	_ = binary.Write(h, binary.BigEndian, t.Unix())
	_ = binary.Write(h, binary.BigEndian, int64(t.Nanosecond()))
	_ = binary.Write(h, binary.BigEndian, []byte(nodeid))
	etag := fmt.Sprintf(`"%x"`, h.Sum(nil))
	return fmt.Sprintf("\"%s\"", strings.Trim(etag, "\""))
}

// New returns an implementation to of the storage.FS interface that talks to
// a local filesystem with user homes disabled.
func New(_ map[string]interface{}, _ events.Stream, _ *zerolog.Logger) (storage.FS, error) {
	return &hellofs{
		bootTime: time.Now(),
	}, nil
}

// Shutdown is called when the process is exiting to give the driver a chance to flush and close all open handles
func (fs *hellofs) Shutdown(ctx context.Context) error {
	return nil
}

// ListStorageSpaces lists the spaces in the storage.
func (fs *hellofs) ListStorageSpaces(ctx context.Context, filter []*provider.ListStorageSpacesRequest_Filter, unrestricted bool) ([]*provider.StorageSpace, error) {
	return []*provider.StorageSpace{fs.space(true)}, nil
}

// GetQuota returns the quota on the referenced resource
func (fs *hellofs) GetQuota(ctx context.Context, ref *provider.Reference) (uint64, uint64, uint64, error) {
	return uint64(len(content)), uint64(len(content)), 0, nil
}

func (fs *hellofs) lookup(ctx context.Context, ref *provider.Reference) (*provider.ResourceInfo, error) {
	if ref.GetResourceId().GetStorageId() != storageid || ref.GetResourceId().GetSpaceId() != spaceid {
		return nil, errtypes.NotFound("")
	}

	// switch root or file
	switch ref.GetResourceId().GetOpaqueId() {
	case rootid:
		switch ref.GetPath() {
		case "", ".":
			return fs.rootInfo(), nil
		case filename:
			return fs.fileInfo(), nil
		default:
			return nil, errtypes.NotFound("unknown filename")
		}
	case fileid:
		return fs.fileInfo(), nil
	}

	return nil, errtypes.NotFound("unknown id")
}

// GetPathByID returns the path pointed by the file id
func (fs *hellofs) GetPathByID(ctx context.Context, resID *provider.ResourceId) (string, error) {
	info, err := fs.lookup(ctx, &provider.Reference{ResourceId: resID})
	if err != nil {
		return "", err
	}

	return info.Path, nil
}

// GetMD returns the resuorce info for the referenced resource
func (fs *hellofs) GetMD(ctx context.Context, ref *provider.Reference, mdKeys []string, fieldMask []string) (*provider.ResourceInfo, error) {
	return fs.lookup(ctx, ref)
}

// ListFolder returns the resource infos for all children of the referenced resource
func (fs *hellofs) ListFolder(ctx context.Context, ref *provider.Reference, mdKeys, fieldMask []string) ([]*provider.ResourceInfo, error) {
	info, err := fs.lookup(ctx, ref)
	if err != nil {
		return nil, err
	}

	if info.Type != provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		return nil, errtypes.InternalError("expected a container")
	}
	if info.GetId().GetOpaqueId() != rootid {
		return nil, errtypes.InternalError("unknown folder")
	}

	return []*provider.ResourceInfo{
		fs.fileInfo(),
	}, nil
}

// Download returns a ReadCloser for the content of the referenced resource
func (fs *hellofs) Download(ctx context.Context, ref *provider.Reference, openReaderFunc func(md *provider.ResourceInfo) bool) (*provider.ResourceInfo, io.ReadCloser, error) {
	info, err := fs.lookup(ctx, ref)
	if err != nil {
		return nil, nil, err
	}
	if info.Type != provider.ResourceType_RESOURCE_TYPE_FILE {
		return nil, nil, errtypes.InternalError("expected a file")
	}
	if info.GetId().GetOpaqueId() != fileid {
		return nil, nil, errtypes.InternalError("unknown file")
	}

	if !openReaderFunc(info) {
		return info, nil, nil
	}

	b := &bytes.Buffer{}
	b.WriteString(content)
	return info, io.NopCloser(b), nil
}
