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

package ace

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	grouppb "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/grants"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

const (
	UserAcePrefix  = "u:"
	GroupAcePrefix = "g:"
	MailAcePrefix  = "m:"
)

/*
ACE represents an Access Control Entry, mimicing NFSv4 ACLs
The difference is tht grant ACEs are not propagated down the tree when being set on a dir.
The tradeoff is that every read has to check the permissions of all path segments up to the root,
to determine the permissions. But reads can be scaled better than writes, so here we are.
See https://github.com/cs3org/reva/pull/1170#issuecomment-700526118 for more details.

The following is taken from the nfs4_acl man page,
see https://linux.die.net/man/5/nfs4_acl:
the extended attributes will look like this
"user.oc.grant.<type>:<flags>:<principal>:<permissions>"

*type*: will be limited to A for now

  - A: Allow

    allow *principal* to perform actions requiring *permissions*
    In the future we can use:

  - U: aUdit

    log any attempted access by principal which requires
    permissions.

  - L: aLarm

    generate a system alarm at any attempted access by
    principal which requires permissions

  - D: for Deny is not recommended

*flags*: for now empty or g for group, no inheritance yet

  - d directory-inherit

    newly-created subdirectories will inherit the
    ACE.

  - f file-inherit

    newly-created files will inherit the ACE, minus its
    inheritance flags. Newly-created subdirectories
    will inherit the ACE; if directory-inherit is not
    also specified in the parent ACE, inherit-only will
    be added to the inherited ACE.

  - n no-propagate-inherit

    newly-created subdirectories will inherit
    the ACE, minus its inheritance flags.

  - i inherit-only

    the ACE is not considered in permissions checks,
    but it is heritable; however, the inherit-only
    flag is stripped from inherited ACEs.

*principal* a named user, group or special principal

  - the oidc sub@iss maps nicely to this

  - 'OWNER@', 'GROUP@', and 'EVERYONE@', which are, respectively, analogous to the POSIX user/group/other

*permissions*

  - r read-data (files) / list-directory (directories)

  - w write-data (files) / create-file (directories)

  - a append-data (files) / create-subdirectory (directories)

  - x execute (files) / change-directory (directories)

  - d delete - delete the file/directory. Some servers will allow a delete to occur if either this permission is set in the file/directory or if the delete-child permission is set in its parent directory.

  - D delete-child - remove a file or subdirectory from within the given directory (directories only)

  - t read-attributes - read the attributes of the file/directory.

  - T write-attributes - write the attributes of the file/directory.

  - n read-named-attributes - read the named attributes of the file/directory.

  - N write-named-attributes - write the named attributes of the file/directory.

  - c read-ACL - read the file/directory NFSv4 ACL.

  - C write-ACL - write the file/directory NFSv4 ACL.

  - o write-owner - change ownership of the file/directory.

  - y synchronize - allow clients to use synchronous I/O with the server.

*TODO*

  - implement OWNER@ as "user.oc.grant.A::OWNER@:rwaDxtTnNcCy"

*Limitations*

	attribute names are limited to 255 chars by the linux kernel vfs, values to 64 kb
	ext3 extended attributes must fit inside a single filesystem block ... 4096 bytes
	that leaves us with "user.oc.grant.A::someonewithaslightlylongersubject@whateverissuer:rwaDxtTnNcCy" ~80 chars
	4096/80 = 51 shares ... with luck we might move the actual permissions to the value, saving ~15 chars
	4096/64 = 64 shares ... still meh ... we can do better by using ints instead of strings for principals

	"user.oc.grant.u:100000" is pretty neat, but we can still do better: base64 encode the int
	"user.oc.grant.u:6Jqg" but base64 always has at least 4 chars, maybe hex is better for smaller numbers
	well use 4 chars in addition to the ace: "user.oc.grant.u:////" = 65535 -> 18 chars

	4096/18 = 227 shares
	still ... ext attrs for this are not infinite scale ...
	so .. attach shares via fileid.
	<userhome>/metadata/<fileid>/shares, similar to <userhome>/files
	<userhome>/metadata/<fileid>/shares/u/<issuer>/<subject>/A:fdi:rwaDxtTnNcCy permissions as filename to keep them in the stat cache?

	whatever ... 50 shares is good enough. If more is needed we can delegate to the metadata
	if "user.oc.grant.M" is present look inside the metadata app.

*Notes*

  - if we cannot set an ace we might get an io error.
    in that case convert all shares to metadata and try to set "user.oc.grant.m"

    what about metadata like share creator, share time, expiry?

  - creator is same as owner, but can be set

  - share date, or abbreviated st is a unix timestamp

  - expiry is a unix timestamp

  - can be put inside the value

  - we need to reorder the fields:
    "user.oc.grant.<u|g|o>:<principal>" -> "kv:t=<type>:f=<flags>:p=<permissions>:st=<share time>:c=<creator>:e=<expiry>:pw=<password>:n=<name>"
    "user.oc.grant.<u|g|o>:<principal>" -> "v1:<type>:<flags>:<permissions>:<share time>:<creator>:<expiry>:<password>:<name>"
    or the first byte determines the format
    0x00 = key value
    0x01 = v1 ...
*/
type ACE struct {
	// NFSv4 acls
	_type       string // t
	flags       string // f
	principal   string // im key
	permissions string // p

	// sharing specific
	shareTime int    // s
	creator   string // c
	expires   int64  // e
	password  string // w passWord TODO h = hash
	label     string // l
}

// FromGrant creates an ACE from a CS3 grant
func FromGrant(g *provider.Grant) *ACE {
	t := "A"
	// Currently we only deny the full permission set
	if grants.PermissionsEqual(&provider.ResourcePermissions{}, g.Permissions) {
		t = "D"
	}
	e := &ACE{
		_type:       t,
		permissions: getACEPerm(g.Permissions),
		creator:     userIDToString(g.Creator),
	}
	if g.Grantee.Type == provider.GranteeType_GRANTEE_TYPE_GROUP {
		e.flags = "g"
		e.principal = GroupAcePrefix + g.Grantee.GetGroupId().OpaqueId
	} else {
		e.principal = UserAce(g.Grantee.GetUserId())
	}

	if g.Expiration != nil {
		e.expires = int64(g.Expiration.Seconds)*int64(time.Second) + int64(g.Expiration.Nanos)
	}

	return e
}

func UserAce(id *userpb.UserId) string {
	filename := utils.NewFSSafeUserID(id).SafeFilename()
	switch id.GetType() {
	case userpb.UserType_USER_TYPE_GUEST:
		return MailAcePrefix + filename
	default:
		return UserAcePrefix + filename
	}
}

// Principal returns the principal of the ACE, eg. `u:<userid>` or `g:<groupid>`
func (e *ACE) Principal() string {
	return e.principal
}

// Marshal renders a principal and byte[] that can be used to persist the ACE as an extended attribute
func (e *ACE) Marshal() (string, []byte) {
	// NOTE: first byte will be replaced after converting to byte array
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	w.Comma = ':'
	if err := w.Write([]string{
		fmt.Sprintf("_t=%s", e._type),
		fmt.Sprintf("f=%s", e.flags),
		fmt.Sprintf("p=%s", e.permissions),
		fmt.Sprintf("c=%s", e.creator),
		fmt.Sprintf("e=%d", e.expires),
	}); err != nil {
		return "", nil
	}
	w.Flush()

	bs := b.Bytes()
	bs[0] = 0 // indicate key value
	return e.principal, bs
}

// Unmarshal parses a principal string and byte[] into an ACE
func Unmarshal(principal string, v []byte) (e *ACE, err error) {
	// first byte indicates type of value
	switch v[0] {
	case 0: // = ':' separated key=value pairs
		s := string(v[1:])
		if e, err = unmarshalKV(s); err == nil {
			e.principal = principal
		}
		// check consistency of Flags and principal type
		if strings.Contains(e.flags, "g") {
			if !strings.HasPrefix(principal, GroupAcePrefix) {
				return nil, fmt.Errorf("inconsistent ace: expected group")
			}
		} else {
			if !strings.HasPrefix(principal, UserAcePrefix) && !strings.HasPrefix(principal, MailAcePrefix) {
				return nil, fmt.Errorf("inconsistent ace: expected user")
			}
			if strings.HasPrefix(principal, MailAcePrefix) {
				id := &userpb.UserId{Type: userpb.UserType_USER_TYPE_GUEST}
				if _, err := utils.NewFSSafeUserID(id).Decode(strings.TrimPrefix(principal, MailAcePrefix)); err != nil {
					return nil, fmt.Errorf("invalid guest ace principal: %w", err)
				}
			}
		}
	default:
		return nil, fmt.Errorf("unknown ace encoding")
	}
	return
}

// Grant returns a CS3 grant
func (e *ACE) Grant() *provider.Grant {
	// if type equals "D" we have a full denial which means an empty permission set
	permissions := &provider.ResourcePermissions{}
	if e._type == "A" {
		permissions = e.grantPermissionSet()
	}
	g := &provider.Grant{
		Grantee: &provider.Grantee{
			Type: e.granteeType(),
		},
		Permissions: permissions,
		Creator:     userIDFromString(e.creator),
	}
	id := e.principal[2:]
	if e.granteeType() == provider.GranteeType_GRANTEE_TYPE_GROUP {
		g.Grantee.Id = &provider.Grantee_GroupId{GroupId: &grouppb.GroupId{OpaqueId: id}}
	} else if e.granteeType() == provider.GranteeType_GRANTEE_TYPE_USER {
		if strings.HasPrefix(e.principal, MailAcePrefix) {
			// Guest principals are validated by Unmarshal or encoded by FromGrant,
			// so decoding cannot fail for a valid ACE.
			userID, _ := utils.NewFSSafeUserID(&userpb.UserId{Type: userpb.UserType_USER_TYPE_GUEST}).Decode(id)
			g.Grantee.Id = &provider.Grantee_UserId{UserId: userID}
		} else {
			g.Grantee.Id = &provider.Grantee_UserId{UserId: &userpb.UserId{OpaqueId: id, Type: userpb.UserType_USER_TYPE_PRIMARY}}
		}
	}

	if e.expires != 0 {
		g.Expiration = &typesv1beta1.Timestamp{
			Seconds: uint64(e.expires / int64(time.Second)),
			Nanos:   uint32(e.expires % int64(time.Second)),
		}
	}

	return g
}

// granteeType returns the CS3 grantee type
func (e *ACE) granteeType() provider.GranteeType {
	if strings.Contains(e.flags, "g") {
		return provider.GranteeType_GRANTEE_TYPE_GROUP
	}
	return provider.GranteeType_GRANTEE_TYPE_USER
}

// grantPermissionSet returns the set of CS3 resource permissions representing the ACE
func (e *ACE) grantPermissionSet() *provider.ResourcePermissions {
	p := &provider.ResourcePermissions{}
	// t
	if strings.Contains(e.permissions, "t") {
		p.Stat = true
		p.GetPath = true
	}
	// r
	if strings.Contains(e.permissions, "r") {
		p.Stat = true    // currently assumed
		p.GetPath = true // currently assumed
		p.InitiateFileDownload = true
		p.ListContainer = true
	}
	// w
	if strings.Contains(e.permissions, "w") {
		p.InitiateFileUpload = true
		if p.InitiateFileDownload {
			p.Move = true
		}
	}
	// a
	if strings.Contains(e.permissions, "a") {
		// TODO append data to file permission?
		p.CreateContainer = true
	}
	// x
	if strings.Contains(e.permissions, "x") {
		p.ListContainer = true
	}
	// d
	if strings.Contains(e.permissions, "d") {
		p.Delete = true
	}
	// D ?

	// sharing
	if strings.Contains(e.permissions, "C") {
		p.AddGrant = true
	}
	if strings.Contains(e.permissions, "c") {
		p.ListGrants = true
	}
	if strings.Contains(e.permissions, "o") { // missuse o = write-owner
		p.RemoveGrant = true
		p.UpdateGrant = true
	}
	if strings.Contains(e.permissions, "O") {
		p.DenyGrant = true
	}

	// trash
	if strings.Contains(e.permissions, "u") { // u = undelete
		p.ListRecycle = true
	}
	if strings.Contains(e.permissions, "U") {
		p.RestoreRecycleItem = true
	}
	if strings.Contains(e.permissions, "P") {
		p.PurgeRecycle = true
	}

	// versions
	if strings.Contains(e.permissions, "v") {
		p.ListFileVersions = true
	}
	if strings.Contains(e.permissions, "V") {
		p.RestoreFileVersion = true
	}

	// ?
	if strings.Contains(e.permissions, "q") {
		p.GetQuota = true
	}
	// TODO set quota permission?
	return p
}

func unmarshalKV(s string) (*ACE, error) {
	e := &ACE{}
	r := csv.NewReader(strings.NewReader(s))
	r.Comma = ':'
	r.Comment = 0
	r.FieldsPerRecord = -1
	r.LazyQuotes = false
	r.TrimLeadingSpace = false
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) != 1 {
		return nil, fmt.Errorf("more than one row of ace kvs")
	}
	for i := range records[0] {
		kv := strings.Split(records[0][i], "=")
		switch kv[0] {
		case "t":
			e._type = kv[1]
		case "f":
			e.flags = kv[1]
		case "p":
			e.permissions = kv[1]
		case "s":
			v, err := strconv.Atoi(kv[1])
			if err != nil {
				return nil, err
			}
			e.shareTime = v
		case "c":
			e.creator = kv[1]
		case "e":
			v, err := strconv.ParseInt(kv[1], 10, 64)
			if err != nil {
				return nil, err
			}
			e.expires = v
		case "w":
			e.password = kv[1]
		case "l":
			e.label = kv[1]
			// TODO default ... log unknown keys? or add as opaque? hm we need that for tagged shares ...
		}
	}
	return e, nil
}

// getACEPerm produces an NFSv4.x inspired permission string from a CS3 resource permissions set
func getACEPerm(set *provider.ResourcePermissions) string {
	var b strings.Builder

	if set.Stat || set.GetPath {
		b.WriteString("t")
	}
	if set.ListContainer { // we have no dedicated traversal permission, but to listing a container allows traversing it
		b.WriteString("x")
	}
	if set.InitiateFileDownload {
		b.WriteString("r")
	}
	if set.InitiateFileUpload || set.Move {
		b.WriteString("w")
	}
	if set.CreateContainer {
		b.WriteString("a")
	}
	if set.Delete {
		b.WriteString("d")
	}

	// sharing
	if set.AddGrant {
		b.WriteString("C")
	}
	if set.ListGrants {
		b.WriteString("c")
	}
	if set.RemoveGrant || set.UpdateGrant {
		b.WriteString("o")
	}
	if set.DenyGrant {
		b.WriteString("O")
	}

	// trash
	if set.ListRecycle {
		b.WriteString("u")
	}
	if set.RestoreRecycleItem {
		b.WriteString("U")
	}
	if set.PurgeRecycle {
		b.WriteString("P")
	}

	// versions
	if set.ListFileVersions {
		b.WriteString("v")
	}
	if set.RestoreFileVersion {
		b.WriteString("V")
	}

	// quota
	if set.GetQuota {
		b.WriteString("q")
	}
	// TODO set quota permission?
	// TODO GetPath
	return b.String()
}

func userIDToString(u *userpb.UserId) string {
	return u.GetOpaqueId()
}

func userIDFromString(uid string) *userpb.UserId {
	s := strings.SplitN(uid, "!", 2)
	return &userpb.UserId{
		OpaqueId: s[0],
	}
}
