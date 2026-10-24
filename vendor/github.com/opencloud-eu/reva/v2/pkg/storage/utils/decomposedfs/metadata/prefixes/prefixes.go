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

import "github.com/opencloud-eu/reva/v2/pkg/storage/utils/ace"

// Declare a list of xattr keys
// TODO the below comment is currently copied from the owncloud driver, revisit
// Currently,extended file attributes have four separated
// namespaces (user, trusted, security and system) followed by a dot.
// A non root user can only manipulate the user. namespace, which is what
// we will use to store ownCloud specific metadata. To prevent name
// collisions with other apps We are going to introduce a sub namespace
// "user.ocis." in the xattrs_prefix*.go files.
const (
	TypeAttr      string = OcisPrefix + "type"
	IDAttr        string = OcisPrefix + "id"
	ParentidAttr  string = OcisPrefix + "parentid"
	OwnerIDAttr   string = OcisPrefix + "owner.id"
	OwnerIDPAttr  string = OcisPrefix + "owner.idp"
	OwnerTypeAttr string = OcisPrefix + "owner.type"
	// the base name of the node
	// updated when the file is renamed or moved
	NameAttr string = OcisPrefix + "name"

	BlobIDAttr   string = OcisPrefix + "blobid"
	BlobsizeAttr string = OcisPrefix + "blobsize"

	// statusPrefix is the prefix for the node status
	StatusPrefix string = OcisPrefix + "nodestatus"

	// scanPrefix is the prefix for the virus scan status and date
	ScanStatusPrefix string = OcisPrefix + "scanstatus"
	ScanDatePrefix   string = OcisPrefix + "scandate"

	// grantPrefix is the prefix for sharing related extended attributes
	GrantPrefix         string = OcisPrefix + "grant."
	GrantUserAcePrefix  string = OcisPrefix + "grant." + ace.UserAcePrefix
	GrantGroupAcePrefix string = OcisPrefix + "grant." + ace.GroupAcePrefix
	MetadataPrefix      string = OcisPrefix + "md."

	// favorite flag, per user
	FavPrefix string = OcisPrefix + "fav."

	// a temporary etag for a folder that is removed when the mtime propagation happens
	TmpEtagAttr     string = OcisPrefix + "tmp.etag"
	ReferenceAttr   string = OcisPrefix + "cs3.ref"      // arbitrary metadata
	ChecksumPrefix  string = OcisPrefix + "cs."          // followed by the algorithm, eg. ocis.cs.sha1
	TrashOriginAttr string = OcisPrefix + "trash.origin" // trash origin

	// we use a single attribute to enable or disable propagation of both: synctime and treesize
	// The propagation attribute is set to '1' at the top of the (sub)tree. Propagation will stop at
	// that node.
	PropagationAttr string = OcisPrefix + "propagation"

	// we need mtime to keep mtime in sync with the metadata
	MTimeAttr string = OcisPrefix + "mtime"
	// the tree modification time of the tree below this node,
	// propagated when synctime_accounting is true and
	// user.ocis.propagation=1 is set
	// stored as a readable time.RFC3339Nano
	TreeMTimeAttr string = OcisPrefix + "tmtime"

	// the deletion/disabled time of a space or node
	// used to mark space roots as disabled
	// stored as a readable time.RFC3339Nano
	DTimeAttr string = OcisPrefix + "dtime"

	// the size of the tree below this node,
	// propagated when treesize_accounting is true and
	// user.ocis.propagation=1 is set
	// stored as uint64, little endian
	TreesizeAttr string = OcisPrefix + "treesize"

	// the quota for the storage space / tree, regardless who accesses it
	QuotaAttr string = OcisPrefix + "quota"

	// the name given to a storage space. It should not contain any semantics as its only purpose is to be read.
	SpaceIDAttr          string = OcisPrefix + "space.id"
	SpaceNameAttr        string = OcisPrefix + "space.name"
	SpaceTypeAttr        string = OcisPrefix + "space.type"
	SpaceDescriptionAttr string = OcisPrefix + "space.description"
	SpaceReadmeAttr      string = OcisPrefix + "space.readme"
	SpaceImageAttr       string = OcisPrefix + "space.image"
	SpaceAliasAttr       string = OcisPrefix + "space.alias"
)
