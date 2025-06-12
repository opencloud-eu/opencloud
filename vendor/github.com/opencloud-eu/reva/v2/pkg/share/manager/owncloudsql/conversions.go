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

package owncloudsql

import (
	"context"
	"strings"
	"time"

	grouppb "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	userprovider "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/jellydator/ttlcache/v2"
	"github.com/opencloud-eu/reva/v2/pkg/conversions"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// DBShare stores information about user and public shares.
type DBShare struct {
	ID           string
	UIDOwner     string
	UIDInitiator string
	ItemStorage  string
	FileSource   string
	ShareWith    string
	Token        string
	Expiration   string
	Permissions  int
	ShareType    int
	ShareName    string
	STime        int
	FileTarget   string
	RejectedBy   string
	State        int
	Parent       int
}

// UserConverter describes an interface for converting user ids to names and back
type UserConverter interface {
	UserNameToUserID(ctx context.Context, username string) (*userpb.UserId, error)
	UserIDToUserName(ctx context.Context, userid *userpb.UserId) (string, error)
	GetUser(ctx context.Context, userid *userpb.UserId) (*userpb.User, error)
}

// GatewayUserConverter converts usernames and ids using the gateway
type GatewayUserConverter struct {
	gwAddr string

	IDCache   *ttlcache.Cache
	NameCache *ttlcache.Cache
}

// NewGatewayUserConverter returns a instance of GatewayUserConverter
func NewGatewayUserConverter(gwAddr string) *GatewayUserConverter {
	IDCache := ttlcache.NewCache()
	_ = IDCache.SetTTL(30 * time.Second)
	IDCache.SkipTTLExtensionOnHit(true)
	NameCache := ttlcache.NewCache()
	_ = NameCache.SetTTL(30 * time.Second)
	NameCache.SkipTTLExtensionOnHit(true)

	return &GatewayUserConverter{
		gwAddr:    gwAddr,
		IDCache:   IDCache,
		NameCache: NameCache,
	}
}

// UserIDToUserName converts a user ID to an username
func (c *GatewayUserConverter) UserIDToUserName(ctx context.Context, userid *userpb.UserId) (string, error) {
	username, err := c.NameCache.Get(userid.String())
	if err == nil {
		return username.(string), nil
	}

	gwConn, err := pool.GetGatewayServiceClient(c.gwAddr)
	if err != nil {
		return "", err
	}
	getUserResponse, err := gwConn.GetUser(ctx, &userprovider.GetUserRequest{
		UserId:                 userid,
		SkipFetchingUserGroups: true,
	})
	if err != nil {
		return "", err
	}
	if getUserResponse.Status.Code != rpc.Code_CODE_OK {
		return "", status.NewErrorFromCode(getUserResponse.Status.Code, "gateway")
	}
	_ = c.NameCache.Set(userid.String(), getUserResponse.User.Username)
	return getUserResponse.User.Username, nil
}

// UserNameToUserID converts a username to an user ID
func (c *GatewayUserConverter) UserNameToUserID(ctx context.Context, username string) (*userpb.UserId, error) {
	id, err := c.IDCache.Get(username)
	if err == nil {
		return id.(*userpb.UserId), nil
	}

	gwConn, err := pool.GetGatewayServiceClient(c.gwAddr)
	if err != nil {
		return nil, err
	}
	getUserResponse, err := gwConn.GetUserByClaim(ctx, &userpb.GetUserByClaimRequest{
		Claim:                  "username",
		Value:                  username,
		SkipFetchingUserGroups: true,
	})
	if err != nil {
		return nil, err
	}
	if getUserResponse.Status.Code != rpc.Code_CODE_OK {
		return nil, status.NewErrorFromCode(getUserResponse.Status.Code, "gateway")
	}
	_ = c.IDCache.Set(username, getUserResponse.User.Id)
	return getUserResponse.User.Id, nil
}

// GetUser gets the user
func (c *GatewayUserConverter) GetUser(ctx context.Context, userid *userpb.UserId) (*userpb.User, error) {
	gwc, err := pool.GetGatewayServiceClient(c.gwAddr)
	if err != nil {
		return nil, err
	}
	return utils.GetUser(ctx, userid, gwc)
}

func (m *mgr) formatGrantee(ctx context.Context, g *provider.Grantee) (int, string, error) {
	var granteeType int
	var formattedID string
	switch g.Type {
	case provider.GranteeType_GRANTEE_TYPE_USER:
		granteeType = 0
		var err error
		formattedID, err = m.userConverter.UserIDToUserName(ctx, g.GetUserId())
		if err != nil {
			return 0, "", err
		}
	case provider.GranteeType_GRANTEE_TYPE_GROUP:
		granteeType = 1
		formattedID = formatGroupID(g.GetGroupId())
	default:
		granteeType = -1
	}
	return granteeType, formattedID, nil
}

func (m *mgr) extractGrantee(ctx context.Context, t int, g string) (*provider.Grantee, error) {
	var grantee provider.Grantee
	switch t {
	case 0:
		userid, err := m.userConverter.UserNameToUserID(ctx, g)
		if err != nil {
			return nil, err
		}
		grantee.Type = provider.GranteeType_GRANTEE_TYPE_USER
		grantee.Id = &provider.Grantee_UserId{UserId: userid}
	case 1, 2:
		grantee.Type = provider.GranteeType_GRANTEE_TYPE_GROUP
		grantee.Id = &provider.Grantee_GroupId{GroupId: extractGroupID(g)}
	default:
		grantee.Type = provider.GranteeType_GRANTEE_TYPE_INVALID
	}
	return &grantee, nil
}

func resourceTypeToItem(r provider.ResourceType) string {
	switch r {
	case provider.ResourceType_RESOURCE_TYPE_FILE:
		return "file"
	case provider.ResourceType_RESOURCE_TYPE_CONTAINER:
		return "folder"
	case provider.ResourceType_RESOURCE_TYPE_REFERENCE:
		return "reference"
	case provider.ResourceType_RESOURCE_TYPE_SYMLINK:
		return "symlink"
	default:
		return ""
	}
}

func sharePermToInt(p *provider.ResourcePermissions) int {
	return int(conversions.RoleFromResourcePermissions(p, false).OCSPermissions())
}

func intTosharePerm(p int) (*provider.ResourcePermissions, error) {
	perms, err := conversions.NewPermissions(p)
	if err != nil {
		return nil, err
	}

	return conversions.RoleFromOCSPermissions(perms, nil).CS3ResourcePermissions(), nil
}

func intToShareState(g int) collaboration.ShareState {
	switch g {
	case 0:
		return collaboration.ShareState_SHARE_STATE_ACCEPTED
	case 1:
		return collaboration.ShareState_SHARE_STATE_PENDING
	case 2:
		return collaboration.ShareState_SHARE_STATE_REJECTED
	default:
		return collaboration.ShareState_SHARE_STATE_INVALID
	}
}

func formatUserID(u *userpb.UserId) string {
	return u.OpaqueId
}

func formatGroupID(u *grouppb.GroupId) string {
	return u.OpaqueId
}

func extractGroupID(u string) *grouppb.GroupId {
	return &grouppb.GroupId{OpaqueId: u}
}

func (m *mgr) convertToCS3Share(ctx context.Context, s DBShare, storageMountID string) (*collaboration.Share, error) {
	ts := &typespb.Timestamp{
		Seconds: uint64(s.STime),
	}
	permissions, err := intTosharePerm(s.Permissions)
	if err != nil {
		return nil, err
	}
	grantee, err := m.extractGrantee(ctx, s.ShareType, s.ShareWith)
	if err != nil {
		return nil, err
	}
	owner, err := m.userConverter.UserNameToUserID(ctx, s.UIDOwner)
	if err != nil {
		return nil, err
	}
	var creator *userpb.UserId
	if s.UIDOwner == s.UIDInitiator {
		creator = owner
	} else {
		creator, err = m.userConverter.UserNameToUserID(ctx, s.UIDOwner)
		if err != nil {
			return nil, err
		}
	}
	return &collaboration.Share{
		Id: &collaboration.ShareId{
			OpaqueId: s.ID,
		},
		ResourceId: &provider.ResourceId{
			SpaceId:  s.ItemStorage,
			OpaqueId: s.FileSource,
		},
		Permissions: &collaboration.SharePermissions{Permissions: permissions},
		Grantee:     grantee,
		Owner:       owner,
		Creator:     creator,
		Ctime:       ts,
		Mtime:       ts,
	}, nil
}

func (m *mgr) convertToCS3ReceivedShare(ctx context.Context, s DBShare, storageMountID string) (*collaboration.ReceivedShare, error) {
	share, err := m.convertToCS3Share(ctx, s, storageMountID)
	if err != nil {
		return nil, err
	}
	var state collaboration.ShareState
	if s.RejectedBy != "" {
		state = collaboration.ShareState_SHARE_STATE_REJECTED
	} else {
		state = intToShareState(s.State)
	}
	return &collaboration.ReceivedShare{
		Share:      share,
		State:      state,
		MountPoint: &provider.Reference{Path: strings.TrimLeft(s.FileTarget, "/")},
	}, nil
}
