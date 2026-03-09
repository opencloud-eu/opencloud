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
	"time"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/jellydator/ttlcache/v2"
	"github.com/opencloud-eu/reva/v2/pkg/conversions"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
)

// DBShare stores information about user and public shares.
type DBShare struct {
	ID           string
	UIDOwner     string
	UIDInitiator string
	ItemStorage  string
	FileSource   string
	ItemType     string // 'file' or 'folder'
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
	getUserResponse, err := gwConn.GetUser(ctx, &userpb.GetUserRequest{
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
	return int(conversions.RoleFromResourcePermissions(p, true).OCSPermissions())
}

func intTosharePerm(p int) (*provider.ResourcePermissions, error) {
	perms, err := conversions.NewPermissions(p)
	if err != nil {
		return nil, err
	}

	return conversions.RoleFromOCSPermissions(perms, nil).CS3ResourcePermissions(), nil
}

func formatUserID(u *userpb.UserId) string {
	return u.OpaqueId
}

// ConvertToCS3PublicShare converts a DBShare to a CS3API public share
func (m *mgr) ConvertToCS3PublicShare(ctx context.Context, s DBShare) (*link.PublicShare, error) {
	ts := &typespb.Timestamp{
		Seconds: uint64(s.STime),
	}
	permissions, err := intTosharePerm(s.Permissions)
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
	pwd := s.ShareWith != ""
	var expires *typespb.Timestamp
	if s.Expiration != "" {
		t, err := time.Parse("2006-01-02 15:04:05", s.Expiration)
		if err != nil {
			t, err = time.Parse("2006-01-02 15:04:05-07:00", s.Expiration)
		}
		if err == nil {
			expires = &typespb.Timestamp{
				Seconds: uint64(t.Unix()),
			}
		}
	}
	return &link.PublicShare{
		Id: &link.PublicShareId{
			OpaqueId: s.ID,
		},
		ResourceId: &provider.ResourceId{
			SpaceId:  s.ItemStorage,
			OpaqueId: s.FileSource,
		},
		Permissions:       &link.PublicSharePermissions{Permissions: permissions},
		Owner:             owner,
		Creator:           creator,
		Token:             s.Token,
		DisplayName:       s.ShareName,
		PasswordProtected: pwd,
		Expiration:        expires,
		Ctime:             ts,
		Mtime:             ts,
	}, nil
}
