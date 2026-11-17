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

package eosclient

import (
	"context"
	"io"

	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
	"github.com/opencloud-eu/reva/v3/pkg/storage/utils/acl"
)

// EOSClient is the interface which enables access to EOS instances through various interfaces.
type EOSClient interface {
	AddACL(ctx context.Context, auth, rootAuth Authorization, path string, position uint, a *acl.Entry) error
	RemoveACL(ctx context.Context, auth, rootAuth Authorization, path string, a *acl.Entry) error
	UpdateACL(ctx context.Context, auth, rootAuth Authorization, path string, position uint, a *acl.Entry) error
	GetACL(ctx context.Context, auth Authorization, path, aclType, target string) (*acl.Entry, error)
	ListACLs(ctx context.Context, auth Authorization, path string) ([]*acl.Entry, error)
	GetFileInfoByInode(ctx context.Context, auth Authorization, inode uint64) (*FileInfo, error)
	GetFileInfoByFXID(ctx context.Context, auth Authorization, fxid string) (*FileInfo, error)
	GetFileInfoByPath(ctx context.Context, auth Authorization, path string) (*FileInfo, error)
	SetAttr(ctx context.Context, auth Authorization, attr *Attribute, errorIfExists, recursive bool, path string) error
	UnsetAttr(ctx context.Context, auth Authorization, attr *Attribute, recursive bool, path string) error
	GetAttr(ctx context.Context, auth Authorization, key, path string) (*Attribute, error)
	GetQuota(ctx context.Context, username string, rootAuth Authorization, path string) (*QuotaInfo, error)
	SetQuota(ctx context.Context, rooAuth Authorization, info *SetQuotaInfo) error
	Touch(ctx context.Context, auth Authorization, path string) error
	Chown(ctx context.Context, auth, chownauth Authorization, path string) error
	Chmod(ctx context.Context, auth Authorization, mode, path string) error
	CreateDir(ctx context.Context, auth Authorization, path string) error
	Remove(ctx context.Context, auth Authorization, path string, noRecycle bool) error
	Rename(ctx context.Context, auth Authorization, oldPath, newPath string) error
	List(ctx context.Context, auth Authorization, path string) ([]*FileInfo, error)
	Read(ctx context.Context, auth Authorization, path string) (io.ReadCloser, error)
	Write(ctx context.Context, auth Authorization, path string, stream io.ReadCloser) error
	WriteFile(ctx context.Context, auth Authorization, path, source string) error
	ListDeletedEntries(ctx context.Context, auth Authorization) ([]*DeletedEntry, error)
	RestoreDeletedEntry(ctx context.Context, auth Authorization, key string) error
	PurgeDeletedEntries(ctx context.Context, auth Authorization) error
	ListVersions(ctx context.Context, auth Authorization, p string) ([]*FileInfo, error)
	RollbackToVersion(ctx context.Context, auth Authorization, path, version string) error
	ReadVersion(ctx context.Context, auth Authorization, p, version string) (io.ReadCloser, error)
	GenerateToken(ctx context.Context, auth Authorization, path string, a *acl.Entry) (string, error)
}

// AttrType is the type of extended attribute,
// either system (sys) or user (user).
type AttrType uint32

// Attribute represents an EOS extended attribute.
type Attribute struct {
	Type     AttrType
	Key, Val string
}

// FileInfo represents the metadata information returned by querying the EOS namespace.
type FileInfo struct {
	IsDir      bool
	MTimeNanos uint32
	Inode      uint64            `json:"inode"`
	FID        uint64            `json:"fid"`
	UID        uint64            `json:"uid"`
	GID        uint64            `json:"gid"`
	TreeSize   uint64            `json:"tree_size"`
	MTimeSec   uint64            `json:"mtime_sec"`
	Size       uint64            `json:"size"`
	TreeCount  uint64            `json:"tree_count"`
	File       string            `json:"eos_file"`
	ETag       string            `json:"etag"`
	Instance   string            `json:"instance"`
	XS         *Checksum         `json:"xs"`
	SysACL     *acl.ACLs         `json:"sys_acl"`
	Attrs      map[string]string `json:"attrs"`
}

// DeletedEntry represents an entry from the trashbin.
type DeletedEntry struct {
	RestorePath   string
	RestoreKey    string
	Size          uint64
	DeletionMTime uint64
	IsDir         bool
}

// Checksum represents a cheksum entry for a file returned by EOS.
type Checksum struct {
	XSSum  string
	XSType string
}

// QuotaInfo reports the available bytes and inodes for a particular user.
// eos reports all quota values are unsigned long, see https://github.com/cern-eos/eos/blob/93515df8c0d5a858982853d960bec98f983c1285/mgm/Quota.hh#L135
type QuotaInfo struct {
	AvailableBytes, UsedBytes   uint64
	AvailableInodes, UsedInodes uint64
}

// SetQuotaInfo encapsulates the information needed to
// create a quota space in EOS for a user
type SetQuotaInfo struct {
	Username  string
	UID       string
	GID       string
	QuotaNode string
	MaxBytes  uint64
	MaxFiles  uint64
}

// Constants for ACL position
const (
	EndPosition   uint = 0
	StartPosition uint = 1
)

// Role holds the attributes required to authenticate to EOS via role-based access.
type Role struct {
	UID, GID string
}

// Authorization specifies the mechanisms through which EOS can be accessed.
// One of the data members must be set.
type Authorization struct {
	Role  Role
	Token string
}

// AttrAlreadyExistsError is the error raised when setting
// an already existing attr on a resource
const AttrAlreadyExistsError = errtypes.BadRequest("attr already exists")

// AttrNotExistsError is the error raised when removing
// an attribute that does not exist
const AttrNotExistsError = errtypes.BadRequest("attr not exists")
