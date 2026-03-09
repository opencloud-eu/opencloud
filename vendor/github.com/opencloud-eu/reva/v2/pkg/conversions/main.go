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

// Package conversions sits between CS3 type definitions and OCS API Responses
package conversions

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"time"

	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/mime"
	"github.com/opencloud-eu/reva/v2/pkg/publicshare"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/user"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	grouppb "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	ocm "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	publicsharemgr "github.com/opencloud-eu/reva/v2/pkg/publicshare/manager/registry"
	usermgr "github.com/opencloud-eu/reva/v2/pkg/user/manager/registry"
)

const (
	// ShareTypeUser refers to user shares
	ShareTypeUser ShareType = 0

	// ShareTypePublicLink refers to public link shares
	ShareTypePublicLink ShareType = 3

	// ShareTypeGroup represents a group share
	ShareTypeGroup ShareType = 1

	// ShareTypeFederatedCloudShare represents a federated share
	ShareTypeFederatedCloudShare ShareType = 6

	// ShareTypeSpaceMembershipUser represents an action regarding user type space members
	ShareTypeSpaceMembershipUser ShareType = 7

	// ShareTypeSpaceMembershipGroup represents an action regarding group type space members
	ShareTypeSpaceMembershipGroup ShareType = 8

	// ShareWithUserTypeUser represents a normal user
	ShareWithUserTypeUser ShareWithUserType = 0

	// ShareWithUserTypeGuest represents a guest user
	ShareWithUserTypeGuest ShareWithUserType = 1

	// The datetime format of ISO8601
	_iso8601 = "2006-01-02T15:04:05Z0700"
)

// ResourceType indicates the OCS type of the resource
type ResourceType int

func (rt ResourceType) String() (s string) {
	switch rt {
	case 0:
		s = "invalid"
	case 1:
		s = "file"
	case 2:
		s = "folder"
	case 3:
		s = "reference"
	default:
		s = "invalid"
	}
	return
}

// ShareType denotes a type of share
type ShareType int

// ShareWithUserType denotes a type of user
type ShareWithUserType int

// ShareData represents https://doc.owncloud.com/server/developer_manual/core/ocs-share-api.html#response-attributes-1
type ShareData struct {
	// TODO int?
	ID string `json:"id" xml:"id"`
	// The share’s type
	ShareType ShareType `json:"share_type" xml:"share_type"`
	// The username of the owner of the share.
	UIDOwner string `json:"uid_owner" xml:"uid_owner"`
	// The display name of the owner of the share.
	DisplaynameOwner string `json:"displayname_owner" xml:"displayname_owner"`
	// Additional info to identify the share owner, eg. the email or username
	AdditionalInfoOwner string `json:"additional_info_owner" xml:"additional_info_owner"`
	// The permission attribute set on the file.
	// TODO(jfd) change the default to read only
	Permissions Permissions `json:"permissions" xml:"permissions"`
	// The UNIX timestamp when the share was created.
	STime uint64 `json:"stime" xml:"stime"`
	// ?
	Parent string `json:"parent" xml:"parent"`
	// The UNIX timestamp when the share expires.
	Expiration string `json:"expiration" xml:"expiration"`
	// The public link to the item being shared.
	Token string `json:"token" xml:"token"`
	// The unique id of the user that owns the file or folder being shared.
	UIDFileOwner string `json:"uid_file_owner" xml:"uid_file_owner"`
	// The display name of the user that owns the file or folder being shared.
	DisplaynameFileOwner string `json:"displayname_file_owner" xml:"displayname_file_owner"`
	// Additional info to identify the file owner, eg. the email or username
	AdditionalInfoFileOwner string `json:"additional_info_file_owner" xml:"additional_info_file_owner"`
	// share state, 0 = accepted, 1 = pending, 2 = declined
	State int `json:"state" xml:"state"`
	// The path to the shared file or folder.
	Path string `json:"path" xml:"path"`
	// The type of the object being shared. This can be one of 'file' or 'folder'.
	ItemType string `json:"item_type" xml:"item_type"`
	// The RFC2045-compliant mimetype of the file.
	MimeType string `json:"mimetype" xml:"mimetype"`
	// The space ID of the original file location
	SpaceID string `json:"space_id" xml:"space_id"`
	// The space alias of the original file location
	SpaceAlias string `json:"space_alias" xml:"space_alias"`
	StorageID  string `json:"storage_id" xml:"storage_id"`
	Storage    uint64 `json:"storage" xml:"storage"`
	// The unique node id of the item being shared.
	ItemSource string `json:"item_source" xml:"item_source"`
	// The unique node id of the item being shared. For legacy reasons item_source and file_source attributes have the same value.
	FileSource string `json:"file_source" xml:"file_source"`
	// The unique node id of the parent node of the item being shared.
	FileParent string `json:"file_parent" xml:"file_parent"`
	// The basename of the shared file.
	FileTarget string `json:"file_target" xml:"file_target"`
	// The uid of the share recipient. This is either
	// - a GID (group id) if it is being shared with a group or
	// - a UID (user id) if the share is shared with a user.
	// - a password for public links
	ShareWith string `json:"share_with,omitempty" xml:"share_with,omitempty"`
	// The type of user
	// - 0 = normal user
	// - 1 = guest account
	ShareWithUserType ShareWithUserType `json:"share_with_user_type" xml:"share_with_user_type"`
	// The display name of the share recipient
	ShareWithDisplayname string `json:"share_with_displayname,omitempty" xml:"share_with_displayname,omitempty"`
	// Additional info to identify the share recipient, eg. the email or username
	ShareWithAdditionalInfo string `json:"share_with_additional_info" xml:"share_with_additional_info"`
	// Whether the recipient was notified, by mail, about the share being shared with them.
	MailSend int `json:"mail_send" xml:"mail_send"`
	// Name of the public share
	Name string `json:"name" xml:"name"`
	// URL of the public share
	URL string `json:"url,omitempty" xml:"url,omitempty"`
	// Attributes associated
	Attributes string `json:"attributes,omitempty" xml:"attributes,omitempty"`
	// Quicklink indicates if the link is the quicklink
	Quicklink bool `json:"quicklink,omitempty" xml:"quicklink,omitempty"`
	// PasswordProtected represents a public share is password protected
	// PasswordProtected bool `json:"password_protected,omitempty" xml:"password_protected,omitempty"`
	Hidden bool `json:"hidden" xml:"hidden"`
}

// ShareeData holds share recipient search results
type ShareeData struct {
	Exact   *ExactMatchesData `json:"exact" xml:"exact"`
	Users   []*MatchData      `json:"users" xml:"users>element"`
	Groups  []*MatchData      `json:"groups" xml:"groups>element"`
	Remotes []*MatchData      `json:"remotes" xml:"remotes>element"`
}

// TokenInfo holds token information
type TokenInfo struct {
	// for all callers
	Token             string `json:"token" xml:"token"`
	LinkURL           string `json:"link_url" xml:"link_url"`
	PasswordProtected bool   `json:"password_protected" xml:"password_protected"`
	Aliaslink         bool   `json:"alias_link" xml:"alias_link"`

	// if not password protected
	ID        string `json:"id" xml:"id"`
	StorageID string `json:"storage_id" xml:"storage_id"`
	SpaceID   string `json:"space_id" xml:"space_id"`
	OpaqueID  string `json:"opaque_id" xml:"opaque_id"`
	Path      string `json:"path" xml:"path"`

	// if native access
	SpacePath  string `json:"space_path" xml:"space_path"`
	SpaceAlias string `json:"space_alias" xml:"space_alias"`
	SpaceURL   string `json:"space_url" xml:"space_url"`
	SpaceType  string `json:"space_type" xml:"space_type"`
}

// ExactMatchesData hold exact matches
type ExactMatchesData struct {
	Users   []*MatchData `json:"users" xml:"users>element"`
	Groups  []*MatchData `json:"groups" xml:"groups>element"`
	Remotes []*MatchData `json:"remotes" xml:"remotes>element"`
}

// MatchData describes a single match
type MatchData struct {
	Label string          `json:"label" xml:"label,omitempty"`
	Value *MatchValueData `json:"value" xml:"value"`
}

// MatchValueData holds the type and actual value
type MatchValueData struct {
	ShareType               int    `json:"shareType" xml:"shareType"`
	ShareWith               string `json:"shareWith" xml:"shareWith"`
	ShareWithProvider       string `json:"shareWithProvider" xml:"shareWithProvider"`
	ShareWithAdditionalInfo string `json:"shareWithAdditionalInfo" xml:"shareWithAdditionalInfo,omitempty"`
	UserType                int    `json:"userType" xml:"userType"`
}

// CS3Share2ShareData converts a cs3api user share into shareData data model
func CS3Share2ShareData(ctx context.Context, share *collaboration.Share) *ShareData {
	sd := &ShareData{
		// share.permissions are mapped below
		// Displaynames are added later
		UIDOwner:     LocalUserIDToString(share.GetCreator()),
		UIDFileOwner: LocalUserIDToString(share.GetOwner()),
	}

	switch share.Grantee.Type {
	case provider.GranteeType_GRANTEE_TYPE_USER:
		sd.ShareType = ShareTypeUser
		sd.ShareWith = LocalUserIDToString(share.Grantee.GetUserId())
		shareType := share.GetGrantee().GetUserId().GetType()
		if shareType == userpb.UserType_USER_TYPE_LIGHTWEIGHT || shareType == userpb.UserType_USER_TYPE_GUEST {
			sd.ShareWithUserType = ShareWithUserTypeGuest
		} else {
			sd.ShareWithUserType = ShareWithUserTypeUser
		}
	case provider.GranteeType_GRANTEE_TYPE_GROUP:
		sd.ShareType = ShareTypeGroup
		sd.ShareWith = LocalGroupIDToString(share.Grantee.GetGroupId())
	}

	if share.Id != nil {
		sd.ID = share.Id.OpaqueId
	}
	if share.GetPermissions().GetPermissions() != nil {
		sd.Permissions = RoleFromResourcePermissions(share.GetPermissions().GetPermissions(), false).OCSPermissions()
	}
	if share.Ctime != nil {
		sd.STime = share.Ctime.Seconds // TODO CS3 api birth time = btime
	}

	if share.Expiration != nil {
		expiration := time.Unix(int64(share.Expiration.Seconds), int64(share.Expiration.Nanos))
		sd.Expiration = expiration.Format(_iso8601)
	}

	return sd
}

// PublicShare2ShareData converts a cs3api public share into shareData data model
func PublicShare2ShareData(share *link.PublicShare, r *http.Request, publicURL string) *ShareData {
	sd := &ShareData{
		// share.permissions are mapped below
		// Displaynames are added later
		ShareType:    ShareTypePublicLink,
		Token:        share.Token,
		Name:         share.DisplayName,
		MailSend:     0,
		URL:          publicURL + path.Join("/", "s/"+share.Token),
		UIDOwner:     LocalUserIDToString(share.Creator),
		UIDFileOwner: LocalUserIDToString(share.Owner),
		Quicklink:    share.Quicklink,
	}
	if share.Id != nil {
		sd.ID = share.Id.OpaqueId
	}

	if s := share.GetPermissions().GetPermissions(); s != nil {
		sd.Permissions = RoleFromResourcePermissions(share.GetPermissions().GetPermissions(), true).OCSPermissions()
	}

	if share.Expiration != nil {
		sd.Expiration = timestampToExpiration(share.Expiration)
	}
	if share.Ctime != nil {
		sd.STime = share.Ctime.Seconds // TODO CS3 api birth time = btime
	}

	// hide password
	if share.PasswordProtected {
		sd.ShareWith = "***redacted***"
		sd.ShareWithDisplayname = "***redacted***"
	}

	return sd
}

func formatRemoteUser(u *userpb.UserId) string {
	return fmt.Sprintf("%s@%s", u.OpaqueId, u.Idp)
}

func webdavInfo(protocols []*ocm.Protocol) (*ocm.WebDAVProtocol, bool) {
	for _, p := range protocols {
		if opt, ok := p.Term.(*ocm.Protocol_WebdavOptions); ok {
			return opt.WebdavOptions, true
		}
	}
	return nil, false
}

// ReceivedOCMShare2ShareData converts a cs3 ocm received share into a share data model.
func ReceivedOCMShare2ShareData(share *ocm.ReceivedShare, path string) (*ShareData, error) {
	webdav, ok := webdavInfo(share.Protocols)
	if !ok {
		return nil, errtypes.InternalError("webdav endpoint not in share")
	}

	opaqueid := base64.StdEncoding.EncodeToString([]byte("/"))

	shareTarget := filepath.Join("/Shares", share.Name)

	s := &ShareData{
		ID:           share.Id.OpaqueId,
		UIDOwner:     formatRemoteUser(share.Creator),
		UIDFileOwner: formatRemoteUser(share.Owner),
		ShareWith:    share.Grantee.GetUserId().OpaqueId,
		Permissions:  RoleFromResourcePermissions(webdav.GetPermissions().GetPermissions(), false).OCSPermissions(),
		ShareType:    ShareTypeFederatedCloudShare,
		Path:         shareTarget,
		FileTarget:   shareTarget,
		MimeType:     mime.Detect(share.ResourceType == provider.ResourceType_RESOURCE_TYPE_CONTAINER, share.Name), //nolint:staticcheck // we will update our ocm implementation later
		ItemType:     ResourceType(share.ResourceType).String(),                                                    //nolint:staticcheck // we will update our ocm implementation later
		ItemSource: storagespace.FormatResourceID(&provider.ResourceId{
			StorageId: utils.OCMStorageProviderID,
			SpaceId:   share.Id.OpaqueId,
			OpaqueId:  opaqueid,
		}),
		STime:   share.Ctime.Seconds,
		Name:    share.Name,
		SpaceID: storagespace.FormatStorageID(utils.OCMStorageProviderID, share.Id.OpaqueId),
	}

	if share.Expiration != nil {
		s.Expiration = timestampToExpiration(share.Expiration)
	}
	return s, nil
}

func webdavAMInfo(methods []*ocm.AccessMethod) (*ocm.WebDAVAccessMethod, bool) {
	for _, a := range methods {
		if opt, ok := a.Term.(*ocm.AccessMethod_WebdavOptions); ok {
			return opt.WebdavOptions, true
		}
	}
	return nil, false
}

// OCMShare2ShareData converts a cs3 ocm share into a share data model.
func OCMShare2ShareData(share *ocm.Share) (*ShareData, error) {
	webdav, ok := webdavAMInfo(share.AccessMethods)
	if !ok {
		return nil, errtypes.InternalError("webdav endpoint not in share")
	}

	s := &ShareData{
		ID:           share.Id.OpaqueId,
		UIDOwner:     share.Creator.OpaqueId,
		UIDFileOwner: share.Owner.OpaqueId,
		ShareWith:    formatRemoteUser(share.Grantee.GetUserId()),
		Permissions:  RoleFromResourcePermissions(webdav.GetPermissions(), false).OCSPermissions(),
		ShareType:    ShareTypeFederatedCloudShare,
		STime:        share.Ctime.Seconds,
		Name:         share.Name,
	}

	if share.Expiration != nil {
		s.Expiration = timestampToExpiration(share.Expiration)
	}

	return s, nil
}

// LocalUserIDToString transforms a cs3api user id into an ocs data model without domain name
// TODO ocs uses user names ... so an additional lookup is needed. see mapUserIds()
func LocalUserIDToString(userID *userpb.UserId) string {
	if userID == nil || userID.OpaqueId == "" {
		return ""
	}
	return userID.OpaqueId
}

// LocalGroupIDToString transforms a cs3api group id into an ocs data model without domain name
func LocalGroupIDToString(groupID *grouppb.GroupId) string {
	if groupID == nil || groupID.OpaqueId == "" {
		return ""
	}
	return groupID.OpaqueId
}

// GetUserManager returns a connection to a user share manager
func GetUserManager(manager string, m map[string]map[string]interface{}) (user.Manager, error) {
	if f, ok := usermgr.NewFuncs[manager]; ok {
		return f(m[manager])
	}

	return nil, fmt.Errorf("driver %s not found for user manager", manager)
}

// GetPublicShareManager returns a connection to a public share manager
func GetPublicShareManager(manager string, m map[string]map[string]interface{}) (publicshare.Manager, error) {
	if f, ok := publicsharemgr.NewFuncs[manager]; ok {
		return f(m[manager])
	}

	return nil, fmt.Errorf("driver %s not found for public shares manager", manager)
}

// timestamp is assumed to be UTC ... just human readable ...
// FIXME and ambiguous / error prone because there is no time zone ...
func timestampToExpiration(t *types.Timestamp) string {
	return time.Unix(int64(t.Seconds), int64(t.Nanos)).UTC().Format("2006-01-02 15:05:05")
}

// ParseTimestamp tries to parse the ocs expiry into a CS3 Timestamp
func ParseTimestamp(timestampString string) (*types.Timestamp, error) {
	parsedTime, err := time.Parse("2006-01-02T15:04:05Z0700", timestampString)
	if err != nil {
		parsedTime, err = time.Parse("2006-01-02", timestampString)
		if err == nil {
			// the link needs to be valid for the whole day
			parsedTime = parsedTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("datetime format invalid: %v, %s", timestampString, err.Error())
	}
	final := parsedTime.UnixNano()

	return &types.Timestamp{
		Seconds: uint64(final / 1000000000),
		Nanos:   uint32(final % 1000000000),
	}, nil
}

// UserTypeString returns human readable strings for various user types
func UserTypeString(userType userpb.UserType) string {
	switch userType {
	case userpb.UserType_USER_TYPE_PRIMARY:
		return "primary"
	case userpb.UserType_USER_TYPE_SECONDARY:
		return "secondary"
	case userpb.UserType_USER_TYPE_SERVICE:
		return "service"
	case userpb.UserType_USER_TYPE_APPLICATION:
		return "application"
	case userpb.UserType_USER_TYPE_GUEST:
		return "guest"
	case userpb.UserType_USER_TYPE_FEDERATED:
		return "federated"
	case userpb.UserType_USER_TYPE_LIGHTWEIGHT:
		return "lightweight"
	}
	return "invalid"
}
