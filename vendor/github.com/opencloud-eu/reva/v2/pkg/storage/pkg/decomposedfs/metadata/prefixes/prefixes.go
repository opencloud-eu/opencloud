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

// Declare a list of xattr keys

// Currently,extended file attributes have four separated
// namespaces (user, trusted, security and system) followed by a dot.
// A non root user can only manipulate the user. namespace, which is what
// we will use to store decomposedfs specific metadata. To prevent name
// collisions with other apps We are going to introduce a sub namespace
// "user.oc." in the xattrs_prefix*.go files.
const (
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
	GrantUserAcePrefix  string = OcPrefix + "grant." + UserAcePrefix
	GrantGroupAcePrefix string = OcPrefix + "grant." + GroupAcePrefix
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

	UserAcePrefix  string = "u:"
	GroupAcePrefix string = "g:"
)
