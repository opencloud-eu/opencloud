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

package nextcloud

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
)

// Response contains data for the Nextcloud mock server to respond
// and to switch to a new server state
type Response struct {
	code           int
	body           string
	newServerState string
}

const serverStateError = "ERROR"
const serverStateEmpty = "EMPTY"
const serverStateHome = "HOME"
const serverStateSubdir = "SUBDIR"
const serverStateNewdir = "NEWDIR"
const serverStateSubdirNewdir = "SUBDIR-NEWDIR"
const serverStateFileRestored = "FILE-RESTORED"
const serverStateGrantAdded = "GRANT-ADDED"
const serverStateGrantUpdated = "GRANT-UPDATED"
const serverStateGrantRemoved = "GRANT-REMOVED"
const serverStateRecycle = "RECYCLE"
const serverStateReference = "REFERENCE"
const serverStateMetadata = "METADATA"

var serverState = serverStateEmpty

var responses = map[string]Response{
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/AddGrant {"ref":{"path":"/subdir"},"g":{"grantee":{"type":1,"Id":{"UserId":{"opaque_id":"4c510ada-c86b-4815-8820-42cdf82c3d51"}}},"permissions":{"initiate_file_download":true,"move":true,"stat":true}}} EMPTY`: {200, ``, serverStateGrantAdded},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/CreateDir {"path":"/subdir"} EMPTY`:  {200, ``, serverStateSubdir},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/CreateDir {"path":"/subdir"} HOME`:   {200, ``, serverStateSubdir},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/CreateDir {"path":"/subdir"} NEWDIR`: {200, ``, serverStateSubdirNewdir},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/CreateDir {"path":"/newdir"} EMPTY`:  {200, ``, serverStateNewdir},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/CreateDir {"path":"/newdir"} HOME`:   {200, ``, serverStateNewdir},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/CreateDir {"path":"/newdir"} SUBDIR`: {200, ``, serverStateSubdirNewdir},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/CreateHome `:   {200, ``, serverStateHome},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/CreateHome {}`: {200, ``, serverStateHome},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/CreateStorageSpace {"owner":{"id":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"username":"einstein"},"type":"personal","name":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c"}`: {200, `{"status":{"code":1},"storage_space":{"id":{"opaque_id":"some-opaque-storage-space-id"},"owner":{"id":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}},"root":{"storage_id":"some-storage-ud","opaque_id":"some-opaque-root-id"},"name":"My Storage Space","quota":{"quota_max_bytes":456,"quota_max_files":123},"space_type":"home","mtime":{"seconds":1234567890}}}`, serverStateHome},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/CreateStorageSpace {"owner":{"id":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1},"username":"einstein"},"type":"personal"}`:                                               {200, `{"status":{"code":1},"storage_space":{"id":{"opaque_id":"some-opaque-storage-space-id"},"owner":{"id":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}},"root":{"storage_id":"some-storage-ud","opaque_id":"some-opaque-root-id"},"name":"My Storage Space","quota":{"quota_max_bytes":456,"quota_max_files":123},"space_type":"home","mtime":{"seconds":1234567890}}}`, serverStateHome},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/CreateReference {"path":"/Shares/reference","url":"scheme://target"}`: {200, `[]`, serverStateReference},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/Delete {"path":"/subdir"}`: {200, ``, serverStateRecycle},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/EmptyRecycle `: {200, ``, serverStateEmpty},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetQuota `: {200, `{"totalBytes":456,"usedBytes":123}`, serverStateEmpty},

	// the parent of an upload target, stated for its permissions and its id
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/"},"mdKeys":[]} EMPTY`: {200, `{"opaque":{},"type":2,"id":{"opaque_id":"fileid-/"},"checksum":{},"etag":"deadbeef","mime_type":"httpd/unix-directory","mtime":{"seconds":1234567890},"path":"/","permission_set":{"initiate_file_upload":true,"stat":true,"list_container":true},"size":12345,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{}}}`, serverStateEmpty},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/"},"mdKeys":null} EMPTY`: {404, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/"},"mdKeys":null} HOME`:  {200, `{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/some/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/","permission_set":{},"size":12345,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{"da":"ta","some":"arbi","trary":"meta"}}}`, serverStateHome},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/newdir"},"mdKeys":null} EMPTY`:         {404, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/newdir"},"mdKeys":null} HOME`:          {404, ``, serverStateHome},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/newdir"},"mdKeys":null} SUBDIR`:        {404, ``, serverStateSubdir},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/newdir"},"mdKeys":null} NEWDIR`:        {200, `{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/some/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/newdir","permission_set":{},"size":12345,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{"da":"ta","some":"arbi","trary":"meta"}}}`, serverStateNewdir},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/newdir"},"mdKeys":null} SUBDIR-NEWDIR`: {200, `{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/some/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/newdir","permission_set":{},"size":12345,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{"da":"ta","some":"arbi","trary":"meta"}}}`, serverStateSubdirNewdir},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/new_subdir"},"mdKeys":null}`: {200, `{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/some/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/new_subdir","permission_set":{},"size":12345,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{"da":"ta","some":"arbi","trary":"meta"}}}`, serverStateEmpty},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/subdir"},"mdKeys":null} EMPTY`:         {404, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/subdir"},"mdKeys":[]} EMPTY`:           {200, `{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/some/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/subdir","permission_set":{},"size":12345,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{}}}`, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/subdir"},"mdKeys":null} HOME`:          {404, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/subdir"},"mdKeys":[]} HOME`:            {200, `{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/some/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/subdir","permission_set":{},"size":12345,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{}}}`, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/subdir"},"mdKeys":null} NEWDIR`:        {404, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/subdir"},"mdKeys":[]} NEWDIR`:          {200, `{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/some/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/subdir","permission_set":{},"size":12345,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{}}}`, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/subdir"},"mdKeys":null} RECYCLE`:       {404, ``, serverStateRecycle},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/subdir"},"mdKeys":[]} RECYCLE`:         {200, `{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/some/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/subdir","permission_set":{},"size":12345,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{}}}`, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/subdir"},"mdKeys":null} SUBDIR`:        {200, `{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/some/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/subdir","permission_set":{},"size":12345,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{}}}`, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/subdir"},"mdKeys":[]} SUBDIR`:          {200, `{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/some/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/subdir","permission_set":{},"size":12345,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{}}}`, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/subdir"},"mdKeys":null} SUBDIR-NEWDIR`: {200, `{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/some/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/subdir","permission_set":{},"size":12345,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{}}}`, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/subdir"},"mdKeys":[]} SUBDIR-NEWDIR`:   {200, `{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/some/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/subdir","permission_set":{},"size":12345,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{}}}`, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/subdir"},"mdKeys":null} METADATA`:      {200, `{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/some/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/subdir","permission_set":{},"size":12345,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{"foo":"bar"}}}`, serverStateMetadata},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/subdir"},"mdKeys":[]} METADATA`:        {200, `{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/some/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/subdir","permission_set":{},"size":12345,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{"foo":"bar"}}}`, serverStateMetadata},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/subdirRestored"},"mdKeys":null} EMPTY`:         {404, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/subdirRestored"},"mdKeys":null} RECYCLE`:       {404, ``, serverStateRecycle},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/subdirRestored"},"mdKeys":null} SUBDIR`:        {404, ``, serverStateSubdir},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/subdirRestored"},"mdKeys":null} FILE-RESTORED`: {200, `{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/some/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/subdirRestored","permission_set":{},"size":12345,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{"da":"ta","some":"arbi","trary":"meta"}}}`, serverStateFileRestored},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/subdir"},"mdKeys":null} FILE-RESTORED`:         {200, `{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/some/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/subdirRestored","permission_set":{},"size":12345,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{"da":"ta","some":"arbi","trary":"meta"}}}`, serverStateFileRestored},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/versionedFile"},"mdKeys":null} EMPTY`:         {200, `{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/some/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/versionedFile","permission_set":{},"size":2,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{"da":"ta","some":"arbi","trary":"meta"}}}`, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/versionedFile"},"mdKeys":null} FILE-RESTORED`: {200, `{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/some/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/versionedFile","permission_set":{},"size":1,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{"da":"ta","some":"arbi","trary":"meta"}}}`, serverStateFileRestored},

	// The coordinator resolves the upload target itself, so it stats the file with the
	// full reference before every upload. Reporting the file as existing keeps it on the
	// overwrite path, which needs neither a parent stat nor a TouchFile (whose mtime is
	// wall-clock derived and could never match this table's exact-body keys).
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"resource_id":{"storage_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c"},"path":"/versionedFile"},"mdKeys":[]}`: {200, `{"opaque":{},"type":1,"id":{"storage_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","opaque_id":"fileid-/versionedFile"},"parent_id":{"opaque_id":"fileid-/"},"name":"versionedFile","checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/versionedFile","permission_set":{"initiate_file_upload":true,"stat":true,"list_container":true,"list_file_versions":true,"restore_file_version":true},"size":1,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{}}}`, serverStateEmpty},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetPathByID {"storage_id":"00000000-0000-0000-0000-000000000000","opaque_id":"fileid-/some/path"} EMPTY`: {200, "/subdir", serverStateEmpty},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/file"},"mdKeys":null}`: {404, ``, serverStateEmpty},
	// the coordinator resolves the upload target itself: the file does not exist yet,
	// so it stats the parent for the permissions and the parent id
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/file"},"mdKeys":[]}`: {404, ``, serverStateEmpty},
	// a zero-length upload finishes at once, so the node is created and read back here
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/TouchFile {"ref":{"resource_id":{"opaque_id":"fileid-/"},"path":"file"},"markprocessing":false,"mtime":"1234567890"}`:                                {200, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"resource_id":{"opaque_id":"fileid-/"},"path":"file"},"mdKeys":[]}`:                                                                    {200, `{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/file"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/file","permission_set":{},"size":0,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{}}}`, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/InitiateUpload {"ref":{"path":"/file"},"uploadLength":0,"metadata":{"providerID":""}}`:                                                               {200, `{"simple": "yes","tus": "yes"}`, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/InitiateUpload {"ref":{"resource_id":{"storage_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c"},"path":"/versionedFile"},"uploadLength":1,"metadata":{}}`: {200, `{"simple": "yes","tus": "yes"}`, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/InitiateUpload {"ref":{"resource_id":{"storage_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c"},"path":"/versionedFile"},"uploadLength":2,"metadata":{}}`: {200, `{"simple": "yes","tus": "yes"}`, serverStateEmpty},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/GetMD {"ref":{"path":"/yes"},"mdKeys":[]}`: {200, `{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/yes"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/yes","permission_set":{},"size":1,"canonical_metadata":{},"arbitrary_metadata":{}}`, serverStateEmpty},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/ListFolder {"ref":{"path":"/"},"mdKeys":null}`: {200, `[{"opaque":{},"type":2,"id":{"opaque_id":"fileid-/subdir"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/subdir","permission_set":{},"size":12345,"canonical_metadata":{},"owner":{"opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c"},"arbitrary_metadata":{"metadata":{"da":"ta","some":"arbi","trary":"meta"}}}]`, serverStateEmpty},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/ListFolder {"ref":{"path":"/Shares"},"mdKeys":null} EMPTY`:     {404, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/ListFolder {"ref":{"path":"/Shares"},"mdKeys":null} SUBDIR`:    {404, ``, serverStateSubdir},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/ListFolder {"ref":{"path":"/Shares"},"mdKeys":null} REFERENCE`: {200, `[{"opaque":{},"type":2,"id":{"opaque_id":"fileid-/some/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/subdir","permission_set":{},"size":12345,"canonical_metadata":{},"owner":{"opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c"},"arbitrary_metadata":{"metadata":{"da":"ta","some":"arbi","trary":"meta"}}}]`, serverStateReference},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/ListGrants {"path":"/subdir"} SUBDIR`:        {200, `[]`, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/ListGrants {"path":"/subdir"} GRANT-ADDED`:   {200, `[{"grantee":{"type":1,"Id":{"UserId":{"idp":"some-idp","opaque_id":"some-opaque-id","type":1}}},"permissions":{"add_grant":true,"create_container":true,"delete":false,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}}]`, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/ListGrants {"path":"/subdir"} GRANT-UPDATED`: {200, `[{"grantee":{"type":1,"Id":{"UserId":{"idp":"some-idp","opaque_id":"some-opaque-id","type":1}}},"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}}]`, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/ListGrants {"path":"/subdir"} GRANT-REMOVED`: {200, `[]`, serverStateEmpty},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/ListRecycle {"key":"","path":""} EMPTY`:   {200, `[]`, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/ListRecycle {"key":"","path":""} RECYCLE`: {200, `[{"opaque":{},"key":"some-deleted-version","ref":{"resource_id":{},"path":"/subdir"},"size":12345,"deletion_time":{"seconds":1234567890}}]`, serverStateRecycle},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/ListRevisions {"path":"/versionedFile"} EMPTY`:         {200, `[{"opaque":{"map":{"some":{"value":"ZGF0YQ=="}}},"key":"version-12","size":1,"mtime":1234567890,"etag":"deadb00f"}]`, serverStateEmpty},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/ListRevisions {"path":"/versionedFile"} FILE-RESTORED`: {200, `[{"opaque":{"map":{"some":{"value":"ZGF0YQ=="}}},"key":"version-12","size":1,"mtime":1234567890,"etag":"deadb00f"},{"opaque":{"map":{"different":{"value":"c3R1ZmY="}}},"key":"asdf","size":2,"mtime":1234567890,"etag":"deadbeef"}]`, serverStateFileRestored},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/Move {"oldRef":{"path":"/subdir"},"newRef":{"path":"/new_subdir"}}`: {200, ``, serverStateEmpty},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/RemoveGrant {"path":"/subdir"} GRANT-ADDED`: {200, ``, serverStateGrantRemoved},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/RestoreRecycleItem null`:                                                                             {200, ``, serverStateSubdir},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/RestoreRecycleItem {"key":"some-deleted-version","path":"","restoreRef":{"path":"/subdirRestored"}}`: {200, ``, serverStateFileRestored},
	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/RestoreRecycleItem {"key":"some-deleted-version","path":"","restoreRef":null}`:                       {200, ``, serverStateFileRestored},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/RestoreRevision {"ref":{"path":"/versionedFile"},"key":"version-12"}`: {200, ``, serverStateFileRestored},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/SetArbitraryMetadata {"ref":{"path":"/subdir"},"md":{"metadata":{"foo":"bar"}}}`: {200, ``, serverStateMetadata},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/UnsetArbitraryMetadata {"ref":{"path":"/subdir"},"keys":["foo"]}`: {200, ``, serverStateSubdir},

	`POST /apps/sciencemesh/~f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c/api/storage/UpdateGrant {"ref":{"path":"/subdir"},"g":{"grantee":{"type":1,"Id":{"UserId":{"opaque_id":"4c510ada-c86b-4815-8820-42cdf82c3d51"}}},"permissions":{"delete":true,"initiate_file_download":true,"move":true,"stat":true}}}`: {200, ``, serverStateGrantUpdated},

	`POST /apps/sciencemesh/~tester/api/storage/GetHome `:    {200, `yes we are`, serverStateHome},
	`POST /apps/sciencemesh/~tester/api/storage/CreateHome `: {201, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/CreateDir {"resource_id":{"storage_id":"storage-id","opaque_id":"opaque-id"},"path":"/some/path"}`:                                                                                                                        {201, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/Delete {"resource_id":{"storage_id":"storage-id","opaque_id":"opaque-id"},"path":"/some/path"}`:                                                                                                                           {200, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/Move {"oldRef":{"resource_id":{"storage_id":"storage-id-1","opaque_id":"opaque-id-1"},"path":"/some/old/path"},"newRef":{"resource_id":{"storage_id":"storage-id-2","opaque_id":"opaque-id-2"},"path":"/some/new/path"}}`: {200, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/GetMD {"ref":{"resource_id":{"storage_id":"storage-id","opaque_id":"opaque-id"},"path":"/some/path"},"mdKeys":["val1","val2","val3"]}`:                                                                                    {200, `{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/some/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/some/path","permission_set":{},"size":12345,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{"da":"ta","some":"arbi","trary":"meta"}}}`, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/ListFolder {"ref":{"resource_id":{"storage_id":"storage-id","opaque_id":"opaque-id"},"path":"/some"},"mdKeys":["val1","val2","val3"]}`:                                                                                    {200, `[{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/some/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/some/path","permission_set":{},"size":12345,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{"da":"ta","some":"arbi","trary":"meta"}}}]`, serverStateEmpty},
	// `POST /apps/sciencemesh/~tester/api/storage/ListFolder {"ref":{"resource_id":{"storage_id":"storage-id","opaque_id":"opaque-id"},"path":"/some"},"mdKeys":["val1","val2","val3"]}`:                                                                                    {200, `[{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/path","permission_set":{},"size":12345,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{"da":"ta","some":"arbi","trary":"meta"}}}]`, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/InitiateUpload {"ref":{"resource_id":{"storage_id":"storage-id","opaque_id":"opaque-id"},"path":"/some/path"},"uploadLength":12345,"metadata":{"key1":"val1","key2":"val2","key3":"val3"}}`: {200, `{ "not":"sure", "what": "should be", "returned": "here" }`, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/TouchFile {"ref":{"resource_id":{"storage_id":"storage-id","opaque_id":"opaque-id"},"path":"some/file/path.txt"},"markprocessing":false,"mtime":"1234567890"}`:                              {200, ``, serverStateEmpty},
	`PUT /apps/sciencemesh/~tester/api/storage/Upload/some/file/path.txt shiny!`:                                                                                                                                                            {200, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/GetMD {"ref":{"resource_id":{"storage_id":"storage-id","opaque_id":"opaque-id"},"path":"some/file/path.txt"},"mdKeys":[]}`:                                                                  {200, `{"opaque":{},"type":1,"id":{"opaque_id":"fileid-/some/path"},"checksum":{},"etag":"deadbeef","mime_type":"text/plain","mtime":{"seconds":1234567890},"path":"/versionedFile","permission_set":{},"size":12345,"canonical_metadata":{},"arbitrary_metadata":{"metadata":{"key1":"val1","key2":"val2","key3":"val3"}}}`, serverStateEmpty},
	`GET /apps/sciencemesh/~tester/api/storage/Download/some/file/path.txt `:                                                                                                                                                                {200, `the contents of the file`, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/ListRevisions {"resource_id":{"storage_id":"storage-id","opaque_id":"opaque-id"},"path":"/some/path"}`:                                                                                      {200, `[{"opaque":{"map":{"some":{"value":"ZGF0YQ=="}}},"key":"version-12","size":12345,"mtime":1234567890,"etag":"deadb00f"},{"opaque":{"map":{"different":{"value":"c3R1ZmY="}}},"key":"asdf","size":12345,"mtime":1234567890,"etag":"deadbeef"}]`, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/ListRevisions {"resource_id":{"storage_id":"storage-id","opaque_id":"opaque-id"},"path":"some/file/path.txt"}`:                                                                              {200, `[{"key":"some/revision","size":12345,"mtime":1234567890,"etag":"deadb00f"}]`, serverStateEmpty},
	`GET /apps/sciencemesh/~tester/api/storage/DownloadRevision/some%2Frevision/some/file/path.txt `:                                                                                                                                        {200, `the contents of that revision`, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/RestoreRevision {"ref":{"resource_id":{"storage_id":"storage-id","opaque_id":"opaque-id"},"path":"some/file/path.txt"},"key":"asdf"}`:                                                       {200, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/ListRecycle {"key":"asdf","path":"/some/file.txt"}`:                                                                                                                                         {200, `[{"opaque":{},"key":"some-deleted-version","ref":{"resource_id":{},"path":"/some/file.txt"},"size":12345,"deletion_time":{"seconds":1234567890}}]`, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/RestoreRecycleItem {"key":"asdf","path":"original/location/when/deleted.txt","restoreRef":{"resource_id":{"storage_id":"storage-id","opaque_id":"opaque-id"},"path":"some/file/path.txt"}}`: {200, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/PurgeRecycleItem {"key":"asdf","path":"original/location/when/deleted.txt"}`:                                                                                                                {200, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/EmptyRecycle `:                                                                                                                                                                              {200, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/GetPathByID {"storage_id":"storage-id","opaque_id":"opaque-id"}`:                                                                                                                            {200, `the/path/for/that/id.txt`, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/AddGrant {"ref":{"resource_id":{"storage_id":"storage-id","opaque_id":"opaque-id"},"path":"some/file/path.txt"},"g":{"grantee":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}}}`: {200, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/DenyGrant {"ref":{"resource_id":{"storage_id":"storage-id","opaque_id":"opaque-id"},"path":"some/file/path.txt"},"g":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}}}`: {200, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/RemoveGrant {"ref":{"resource_id":{"storage_id":"storage-id","opaque_id":"opaque-id"},"path":"some/file/path.txt"},"g":{"grantee":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}}}`: {200, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/UpdateGrant {"ref":{"resource_id":{"storage_id":"storage-id","opaque_id":"opaque-id"},"path":"some/file/path.txt"},"g":{"grantee":{"Id":{"UserId":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}}}`: {200, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/ListGrants {"resource_id":{"storage_id":"storage-id","opaque_id":"opaque-id"},"path":"some/file/path.txt"}`: {200, `[{"grantee":{"type":1,"Id":{"UserId":{"idp":"some-idp","opaque_id":"some-opaque-id","type":1}}},"permissions":{"add_grant":true,"create_container":true,"delete":true,"get_path":true,"get_quota":true,"initiate_file_download":true,"initiate_file_upload":true,"list_grants":true,"list_container":true,"list_file_versions":true,"list_recycle":true,"move":true,"remove_grant":true,"purge_recycle":true,"restore_file_version":true,"restore_recycle_item":true,"stat":true,"update_grant":true,"deny_grant":true}}]`, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/GetQuota `:                                                                             {200, `{"totalBytes":456,"usedBytes":123}`, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/CreateReference {"path":"some/file/path.txt","url":"http://bing.com/search?q=dotnet"}`: {200, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/Shutdown `:                                                                             {200, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/SetArbitraryMetadata {"ref":{"resource_id":{"storage_id":"storage-id","opaque_id":"opaque-id"},"path":"some/file/path.txt"},"md":{"metadata":{"arbi":"trary","meta":"data"}}}`:                                                                                            {200, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/UnsetArbitraryMetadata {"ref":{"resource_id":{"storage_id":"storage-id","opaque_id":"opaque-id"},"path":"some/file/path.txt"},"keys":["arbi"]}`:                                                                                                                           {200, ``, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/ListStorageSpaces [{"type":3,"Term":{"Owner":{"idp":"0.0.0.0:19000","opaque_id":"f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c","type":1}}},{"type":2,"Term":{"Id":{"opaque_id":"opaque-id"}}},{"type":4,"Term":{"SpaceType":"home"}}]`:                                            {200, `	[{"opaque":{"map":{"bar":{"value":"c2FtYQ=="},"foo":{"value":"c2FtYQ=="}}},"id":{"opaque_id":"some-opaque-storage-space-id"},"owner":{"id":{"idp":"some-idp","opaque_id":"some-opaque-user-id","type":1}},"root":{"storage_id":"some-storage-ud","opaque_id":"some-opaque-root-id"},"name":"My Storage Space","quota":{"quota_max_bytes":456,"quota_max_files":123},"space_type":"home","mtime":{"seconds":1234567890}}]`, serverStateEmpty},
	`POST /apps/sciencemesh/~tester/api/storage/CreateStorageSpace {"opaque":{"map":{"bar":{"value":"c2FtYQ=="},"foo":{"value":"c2FtYQ=="}}},"owner":{"id":{"idp":"some-idp","opaque_id":"some-opaque-user-id","type":1}},"type":"home","name":"My Storage Space","quota":{"quota_max_bytes":456,"quota_max_files":123}}`: {200, `{"storage_space":{"opaque":{"map":{"bar":{"value":"c2FtYQ=="},"foo":{"value":"c2FtYQ=="}}},"id":{"opaque_id":"some-opaque-storage-space-id"},"owner":{"id":{"idp":"some-idp","opaque_id":"some-opaque-user-id","type":1}},"root":{"storage_id":"some-storage-ud","opaque_id":"some-opaque-root-id"},"name":"My Storage Space","quota":{"quota_max_bytes":456,"quota_max_files":123},"space_type":"home","mtime":{"seconds":1234567890}}}`, serverStateEmpty},
}

// GetNextcloudServerMock returns a handler that pretends to be a remote Nextcloud server
func GetNextcloudServerMock(called *[]string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(strings.Builder)
		_, err := io.Copy(buf, r.Body)
		if err != nil {
			panic("Error reading response into buffer")
		}
		var key = fmt.Sprintf("%s %s %s", r.Method, r.URL, buf.String())
		*called = append(*called, key)
		response := responses[key]
		if (response == Response{}) {
			key = fmt.Sprintf("%s %s %s %s", r.Method, r.URL, buf.String(), serverState)
			response = responses[key]
		}
		if (response == Response{}) {
			fmt.Printf("%s %s %s %s\n", r.Method, r.URL, buf.String(), serverState)
			response = Response{500, fmt.Sprintf("response not defined! %s", key), serverStateEmpty}
		}
		serverState = responses[key].newServerState
		if serverState == `` {
			serverState = serverStateError
		}
		w.WriteHeader(response.code)
		// w.Header().Set("Etag", "mocker-etag")
		_, err = w.Write([]byte(responses[key].body))
		if err != nil {
			panic(err)
		}
	})
}

// TestingHTTPClient thanks to https://itnext.io/how-to-stub-requests-to-remote-hosts-with-go-6c2c1db32bf2
// Ideally, this function would live in tests/helpers, but
// if we put it there, it gets excluded by .dockerignore, and the
// Docker build fails (see https://github.com/cs3org/reva/issues/1999)
// So putting it here for now - open to suggestions if someone knows
// a better way to inject this.
func TestingHTTPClient(handler http.Handler) (*http.Client, func()) {
	s := httptest.NewServer(handler)

	cli := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, s.Listener.Addr().String())
			},
		},
	}

	return cli, s.Close
}
