// Copyright 2018-2023 CERN
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

package utils

import (
	"context"
	"errors"
	"time"

	gatewayv1beta1 "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	grouppb "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/conversions"
)

// DBShare stores information about user and public shares.
type DBShare struct {
	ID                           string
	UIDOwner                     string
	UIDInitiator                 string
	Prefix                       string
	ItemSource                   string
	ItemType                     string
	ShareWith                    string
	Token                        string
	Expiration                   string
	Permissions                  int
	ShareType                    int
	ShareName                    string
	STime                        int
	FileTarget                   string
	State                        int
	Quicklink                    bool
	Description                  string
	NotifyUploads                bool
	NotifyUploadsExtraRecipients string
}

// FormatGrantee formats a CS3API grantee to a string.
func FormatGrantee(g *provider.Grantee) (int, string) {
	var granteeType int
	var formattedID string
	switch g.Type {
	case provider.GranteeType_GRANTEE_TYPE_USER:
		granteeType = 0
		formattedID = FormatUserID(g.GetUserId())
	case provider.GranteeType_GRANTEE_TYPE_GROUP:
		granteeType = 1
		formattedID = FormatGroupID(g.GetGroupId())
	default:
		granteeType = -1
	}
	return granteeType, formattedID
}

// ExtractGrantee retrieves the CS3API grantee from a formatted string.
func ExtractGrantee(ctx context.Context, gateway gatewayv1beta1.GatewayAPIClient, t int, g string) (*provider.Grantee, error) {
	var grantee provider.Grantee
	switch t {
	case 0:
		grantee.Type = provider.GranteeType_GRANTEE_TYPE_USER
		user, err := ExtractUserID(ctx, gateway, g)
		if err != nil {
			return nil, err
		}
		grantee.Id = &provider.Grantee_UserId{UserId: user}
	case 1:
		grantee.Type = provider.GranteeType_GRANTEE_TYPE_GROUP
		group, err := ExtractGroupID(ctx, gateway, g)
		if err != nil {
			return nil, err
		}
		grantee.Id = &provider.Grantee_GroupId{GroupId: group}
	default:
		grantee.Type = provider.GranteeType_GRANTEE_TYPE_INVALID
	}
	return &grantee, nil
}

// ResourceTypeToItem maps a resource type to a string.
func ResourceTypeToItem(r provider.ResourceType) string {
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

// ResourceTypeToItemInt maps a resource type to an integer.
func ResourceTypeToItemInt(r provider.ResourceType) int {
	switch r {
	case provider.ResourceType_RESOURCE_TYPE_CONTAINER:
		return 0
	case provider.ResourceType_RESOURCE_TYPE_FILE:
		return 1
	default:
		return -1
	}
}

// SharePermToInt maps read/write permissions to an integer.
func SharePermToInt(p *provider.ResourcePermissions) int {
	var perm int
	switch {
	case p.InitiateFileUpload && !p.InitiateFileDownload:
		perm = 4
	case p.InitiateFileUpload:
		perm = 15
	case p.InitiateFileDownload:
		perm = 1
	}
	// TODO map denials and resharing; currently, denials are mapped to 0
	return perm
}

// IntTosharePerm retrieves read/write permissions from an integer.
func IntTosharePerm(p int, itemType string) *provider.ResourcePermissions {
	switch p {
	case 1:
		return conversions.NewViewerRole().CS3ResourcePermissions()
	case 15:
		if itemType == "folder" {
			return conversions.NewEditorRole().CS3ResourcePermissions()
		}
		return conversions.NewFileEditorRole().CS3ResourcePermissions()
	case 4:
		return conversions.NewUploaderRole().CS3ResourcePermissions()
	default:
		// TODO we may have other options, for now this is a denial
		return &provider.ResourcePermissions{}
	}
}

// IntToShareState retrieves the received share state from an integer.
func IntToShareState(g int) collaboration.ShareState {
	switch g {
	case 0:
		return collaboration.ShareState_SHARE_STATE_PENDING
	case 1:
		return collaboration.ShareState_SHARE_STATE_ACCEPTED
	case -1:
		return collaboration.ShareState_SHARE_STATE_REJECTED
	default:
		return collaboration.ShareState_SHARE_STATE_INVALID
	}
}

// FormatUserID formats a CS3API user ID to a string.
func FormatUserID(u *userpb.UserId) string {
	return u.OpaqueId
}

// ExtractUserID retrieves a CS3API user ID from a string.
func ExtractUserID(ctx context.Context, gateway gatewayv1beta1.GatewayAPIClient, u string) (*userpb.UserId, error) {
	userRes, err := gateway.GetUser(ctx, &userpb.GetUserRequest{
		UserId: &userpb.UserId{OpaqueId: u},
	})
	if err != nil {
		return nil, err
	}
	if userRes.Status.Code != rpcv1beta1.Code_CODE_OK {
		return nil, errors.New(userRes.Status.Message)
	}

	return userRes.User.Id, nil
}

// FormatGroupID formats a CS3API group ID to a string.
func FormatGroupID(u *grouppb.GroupId) string {
	return u.OpaqueId
}

// ExtractGroupID retrieves a CS3API group ID from a string.
func ExtractGroupID(ctx context.Context, gateway gatewayv1beta1.GatewayAPIClient, u string) (*grouppb.GroupId, error) {
	groupRes, err := gateway.GetGroup(ctx, &grouppb.GetGroupRequest{
		GroupId: &grouppb.GroupId{OpaqueId: u},
	})
	if err != nil {
		return nil, err
	}
	if groupRes.Status.Code != rpcv1beta1.Code_CODE_OK {
		return nil, errors.New(groupRes.Status.Message)
	}
	return groupRes.Group.Id, nil
}

// ConvertToCS3Share converts a DBShare to a CS3API collaboration share.
func ConvertToCS3Share(ctx context.Context, gateway gatewayv1beta1.GatewayAPIClient, s DBShare) (*collaboration.Share, error) {
	ts := &typespb.Timestamp{
		Seconds: uint64(s.STime),
	}
	owner, err := ExtractUserID(ctx, gateway, s.UIDOwner)
	if err != nil {
		return nil, err
	}
	creator, err := ExtractUserID(ctx, gateway, s.UIDInitiator)
	if err != nil {
		return nil, err
	}
	grantee, err := ExtractGrantee(ctx, gateway, s.ShareType, s.ShareWith)
	if err != nil {
		return nil, err
	}

	return &collaboration.Share{
		Id: &collaboration.ShareId{
			OpaqueId: s.ID,
		},
		//ResourceId:  &provider.Reference{StorageId: s.Prefix, NodeId: s.ItemSource},
		ResourceId: &provider.ResourceId{
			StorageId: s.Prefix,
			OpaqueId:  s.ItemSource,
		},
		Permissions: &collaboration.SharePermissions{Permissions: IntTosharePerm(s.Permissions, s.ItemType)},
		Grantee:     grantee,
		Owner:       owner,
		Creator:     creator,
		Ctime:       ts,
		Mtime:       ts,
	}, nil
}

// ConvertToCS3ReceivedShare converts a DBShare to a CS3API collaboration received share.
func ConvertToCS3ReceivedShare(ctx context.Context, gateway gatewayv1beta1.GatewayAPIClient, s DBShare) (*collaboration.ReceivedShare, error) {
	share, err := ConvertToCS3Share(ctx, gateway, s)
	if err != nil {
		return nil, err
	}
	return &collaboration.ReceivedShare{
		Share: share,
		State: IntToShareState(s.State),
	}, nil
}

// ConvertToCS3PublicShare converts a DBShare to a CS3API public share.
func ConvertToCS3PublicShare(ctx context.Context, gateway gatewayv1beta1.GatewayAPIClient, s DBShare) (*link.PublicShare, error) {
	ts := &typespb.Timestamp{
		Seconds: uint64(s.STime),
	}
	pwd := s.ShareWith != ""
	var expires *typespb.Timestamp
	if s.Expiration != "" {
		t, err := time.Parse("2006-01-02 15:04:05", s.Expiration)
		if err == nil {
			expires = &typespb.Timestamp{
				Seconds: uint64(t.Unix()),
			}
		}
	}
	owner, err := ExtractUserID(ctx, gateway, s.UIDOwner)
	if err != nil {
		return nil, err
	}
	creator, err := ExtractUserID(ctx, gateway, s.UIDInitiator)
	if err != nil {
		return nil, err
	}
	return &link.PublicShare{
		Id: &link.PublicShareId{
			OpaqueId: s.ID,
		},
		ResourceId: &provider.ResourceId{
			StorageId: s.Prefix,
			OpaqueId:  s.ItemSource,
		},
		Permissions:                  &link.PublicSharePermissions{Permissions: IntTosharePerm(s.Permissions, s.ItemType)},
		Owner:                        owner,
		Creator:                      creator,
		Token:                        s.Token,
		DisplayName:                  s.ShareName,
		PasswordProtected:            pwd,
		Expiration:                   expires,
		Ctime:                        ts,
		Mtime:                        ts,
		Quicklink:                    s.Quicklink,
		Description:                  s.Description,
		NotifyUploads:                s.NotifyUploads,
		NotifyUploadsExtraRecipients: s.NotifyUploadsExtraRecipients,
	}, nil
}
