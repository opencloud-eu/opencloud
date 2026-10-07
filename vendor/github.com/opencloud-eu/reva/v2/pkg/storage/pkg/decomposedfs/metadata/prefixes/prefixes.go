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

package prefixes

import (
	"fmt"
	"sync"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/ace"
)

// Declare a list of xattr keys

// Currently,extended file attributes have four separated
// namespaces (user, trusted, security and system) followed by a dot.
// A non root user can only manipulate the user. namespace, which is what
// we will use to store decomposedfs specific metadata. To prevent name
// collisions with other apps We are going to introduce a sub namespace
// "user.oc." in the xattrs_prefix*.go files.
var (
	TypeAttr      string = OcPrefix + "type"
	IDAttr        string = OcPrefix + "id"
	ParentidAttr  string = OcPrefix + "parentid"
	OwnerIDAttr   string = OcPrefix + "owner.id"
	OwnerIDPAttr  string = OcPrefix + "owner.idp"
	OwnerTypeAttr string = OcPrefix + "owner.type"
	// the base name of the node
	// updated when the file is renamed or moved
	NameAttr string = OcPrefix + "name"

	BlobIDAttr   string = OcPrefix + "blobid"
	BlobsizeAttr string = OcPrefix + "blobsize"

	// statusPrefix is the prefix for the node status
	StatusPrefix string = OcPrefix + "nodestatus"

	// scanPrefix is the prefix for the virus scan status and date
	ScanStatusPrefix string = OcPrefix + "scanstatus"
	ScanDatePrefix   string = OcPrefix + "scandate"

	// grantPrefix is the prefix for sharing related extended attributes
	GrantPrefix         string = OcPrefix + "grant."
	GrantUserAcePrefix  string = OcPrefix + "grant." + ace.UserAcePrefix
	GrantGroupAcePrefix string = OcPrefix + "grant." + ace.GroupAcePrefix
	GrantMailAcePrefix  string = OcPrefix + "grant." + ace.MailAcePrefix
	MetadataPrefix      string = OcPrefix + "md."

	// favorite flag, per user
	FavPrefix string = OcPrefix + "fav."

	// a temporary etag for a folder that is removed when the mtime propagation happens
	TmpEtagAttr     string = OcPrefix + "tmp.etag"
	ReferenceAttr   string = OcPrefix + "cs3.ref"      // arbitrary metadata
	ChecksumPrefix  string = OcPrefix + "cs."          // followed by the algorithm, eg. oc.cs.sha1
	TrashOriginAttr string = OcPrefix + "trash.origin" // trash origin

	// we use a single attribute to enable or disable propagation of both: synctime and treesize
	// The propagation attribute is set to '1' at the top of the (sub)tree. Propagation will stop at
	// that node.
	PropagationAttr string = OcPrefix + "propagation"

	// we need mtime to keep mtime in sync with the metadata
	MTimeAttr string = OcPrefix + "mtime"
	// the tree modification time of the tree below this node,
	// propagated when synctime_accounting is true and
	// user.oc.propagation=1 is set
	// stored as a readable time.RFC3339Nano
	TreeMTimeAttr string = OcPrefix + "tmtime"

	// the deletion/disabled time of a space or node
	// used to mark space roots as disabled
	// stored as a readable time.RFC3339Nano
	DTimeAttr string = OcPrefix + "dtime"

	// the size of the tree below this node,
	// propagated when treesize_accounting is true and
	// user.oc.propagation=1 is set
	// stored as uint64, little endian
	TreesizeAttr string = OcPrefix + "treesize"

	// the quota for the storage space / tree, regardless who accesses it
	QuotaAttr string = OcPrefix + "quota"

	// the name given to a storage space. It should not contain any semantics as its only purpose is to be read.
	SpaceIDAttr          string = OcPrefix + "space.id"
	SpaceNameAttr        string = OcPrefix + "space.name"
	SpaceTypeAttr        string = OcPrefix + "space.type"
	SpaceDescriptionAttr string = OcPrefix + "space.description"
	SpaceReadmeAttr      string = OcPrefix + "space.readme"
	SpaceImageAttr       string = OcPrefix + "space.image"
	SpaceAliasAttr       string = OcPrefix + "space.alias"
	SpaceTenantIDAttr    string = OcPrefix + "space.tenantid"
	SpaceContentTypeAttr string = OcPrefix + "space.contenttype"
)

func FavoriteKey(uid *userpb.UserId) string {
	// the favorite flag is specific to the user, so we need to incorporate the userid
	return FavPrefix + uid.OpaqueId
}

var (
	defaultOcPrefix = OcPrefix

	mu    sync.Mutex
	fixed bool
)

// SetOcPrefix sets the key prefix for the whole process, empty means the
// default. The first call fixes it, a later call with another prefix fails.
// It has to run before any key is used.
func SetOcPrefix(prefix string) error {
	if prefix == "" {
		prefix = defaultOcPrefix
	}

	mu.Lock()
	defer mu.Unlock()

	if fixed {
		if prefix != OcPrefix {
			return fmt.Errorf("metadata prefix is already set to %q, cannot change it to %q", OcPrefix, prefix)
		}
		return nil
	}

	fixed = true
	if prefix != OcPrefix {
		setKeys(prefix)
	}
	return nil
}

func setKeys(p string) {
	OcPrefix = p
	TypeAttr = p + "type"
	IDAttr = p + "id"
	ParentidAttr = p + "parentid"
	OwnerIDAttr = p + "owner.id"
	OwnerIDPAttr = p + "owner.idp"
	OwnerTypeAttr = p + "owner.type"
	NameAttr = p + "name"
	BlobIDAttr = p + "blobid"
	BlobsizeAttr = p + "blobsize"
	StatusPrefix = p + "nodestatus"
	ScanStatusPrefix = p + "scanstatus"
	ScanDatePrefix = p + "scandate"
	GrantPrefix = p + "grant."
	GrantUserAcePrefix = p + "grant." + ace.UserAcePrefix
	GrantGroupAcePrefix = p + "grant." + ace.GroupAcePrefix
	GrantMailAcePrefix = p + "grant." + ace.MailAcePrefix
	MetadataPrefix = p + "md."
	FavPrefix = p + "fav."
	TmpEtagAttr = p + "tmp.etag"
	ReferenceAttr = p + "cs3.ref"
	ChecksumPrefix = p + "cs."
	TrashOriginAttr = p + "trash.origin"
	PropagationAttr = p + "propagation"
	MTimeAttr = p + "mtime"
	TreeMTimeAttr = p + "tmtime"
	DTimeAttr = p + "dtime"
	TreesizeAttr = p + "treesize"
	QuotaAttr = p + "quota"
	SpaceIDAttr = p + "space.id"
	SpaceNameAttr = p + "space.name"
	SpaceTypeAttr = p + "space.type"
	SpaceDescriptionAttr = p + "space.description"
	SpaceReadmeAttr = p + "space.readme"
	SpaceImageAttr = p + "space.image"
	SpaceAliasAttr = p + "space.alias"
	SpaceTenantIDAttr = p + "space.tenantid"
	SpaceContentTypeAttr = p + "space.contenttype"
}
