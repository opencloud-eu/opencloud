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

package share

import (
	"context"
	"time"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/grants"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"google.golang.org/genproto/protobuf/field_mask"
)

const (
	// NoState can be used to signal the filter matching functions to ignore the share state.
	NoState collaboration.ShareState = -1
)

// Metadata contains Metadata for a share
type Metadata struct {
	ETag  string
	Mtime *types.Timestamp
}

// Manager is the interface that manipulates shares.
type Manager interface {
	// Create a new share in fn with the given acl.
	Share(ctx context.Context, md *provider.ResourceInfo, g *collaboration.ShareGrant) (*collaboration.Share, error)

	// GetShare gets the information for a share by the given ref.
	GetShare(ctx context.Context, ref *collaboration.ShareReference) (*collaboration.Share, error)

	// Unshare deletes the share pointed by ref.
	Unshare(ctx context.Context, ref *collaboration.ShareReference) error

	// UpdateShare updates the mode of the given share.
	UpdateShare(ctx context.Context, ref *collaboration.ShareReference, p *collaboration.SharePermissions, updated *collaboration.Share, fieldMask *field_mask.FieldMask) (*collaboration.Share, error)

	// ListShares returns the shares created by the user. If md is provided is not nil,
	// it returns only shares attached to the given resource.
	ListShares(ctx context.Context, filters []*collaboration.Filter) ([]*collaboration.Share, error)

	// ListReceivedShares returns the list of shares the user has access to. `forUser` parameter for service accounts only
	ListReceivedShares(ctx context.Context, filters []*collaboration.Filter, forUser *userv1beta1.UserId) ([]*collaboration.ReceivedShare, error)

	// GetReceivedShare returns the information for a received share.
	GetReceivedShare(ctx context.Context, ref *collaboration.ShareReference) (*collaboration.ReceivedShare, error)

	// UpdateReceivedShare updates the received share with share state.`forUser` parameter for service accounts only
	UpdateReceivedShare(ctx context.Context, share *collaboration.ReceivedShare, fieldMask *field_mask.FieldMask, forUser *userv1beta1.UserId) (*collaboration.ReceivedShare, error)
}

// ReceivedShareWithUser holds the relevant information for representing a received share of a user
type ReceivedShareWithUser struct {
	UserID        *userv1beta1.UserId
	ReceivedShare *collaboration.ReceivedShare
}

// DumpableManager defines a share manager which supports dumping its contents
type DumpableManager interface {
	Dump(ctx context.Context, shareChan chan<- *collaboration.Share, receivedShareChan chan<- ReceivedShareWithUser) error
}

// LoadableManager defines a share manager which supports loading contents from a dump
type LoadableManager interface {
	Load(ctx context.Context, shareChan <-chan *collaboration.Share, receivedShareChan <-chan ReceivedShareWithUser) error
}

// GroupGranteeFilter is an abstraction for creating filter by grantee type group.
func GroupGranteeFilter() *collaboration.Filter {
	return &collaboration.Filter{
		Type: collaboration.Filter_TYPE_GRANTEE_TYPE,
		Term: &collaboration.Filter_GranteeType{
			GranteeType: provider.GranteeType_GRANTEE_TYPE_GROUP,
		},
	}
}

// UserGranteeFilter is an abstraction for creating filter by grantee type user.
func UserGranteeFilter() *collaboration.Filter {
	return &collaboration.Filter{
		Type: collaboration.Filter_TYPE_GRANTEE_TYPE,
		Term: &collaboration.Filter_GranteeType{
			GranteeType: provider.GranteeType_GRANTEE_TYPE_USER,
		},
	}
}

// ResourceIDFilter is an abstraction for creating filter by resource id.
func ResourceIDFilter(id *provider.ResourceId) *collaboration.Filter {
	return &collaboration.Filter{
		Type: collaboration.Filter_TYPE_RESOURCE_ID,
		Term: &collaboration.Filter_ResourceId{
			ResourceId: id,
		},
	}
}

// SpaceIDFilter is an abstraction for creating filter by space id.
func SpaceIDFilter(id string) *collaboration.Filter {
	return &collaboration.Filter{
		Type: collaboration.Filter_TYPE_SPACE_ID,
		Term: &collaboration.Filter_SpaceId{
			SpaceId: id,
		},
	}
}

// StateFilter is an abstraction for creating filter by share state.
func StateFilter(state collaboration.ShareState) *collaboration.Filter {
	return &collaboration.Filter{
		Type: collaboration.Filter_TYPE_STATE,
		Term: &collaboration.Filter_State{
			State: state,
		},
	}
}

// SpaceRootFilter is an abstraction for filtering shares by whether the shared
// resource is a space root. Pass true to include only space-root shares (space
// membership), false to exclude them (file/folder shares only).
func SpaceRootFilter(spaceRoot bool) *collaboration.Filter {
	return &collaboration.Filter{
		Type: collaboration.Filter_TYPE_SPACE_ROOT,
		Term: &collaboration.Filter_SpaceRoot{
			SpaceRoot: spaceRoot,
		},
	}
}

// IsCreatedByUser checks if the user is the owner or creator of the share.
func IsCreatedByUser(share *collaboration.Share, user *userv1beta1.User) bool {
	return utils.UserEqual(user.Id, share.Owner) || utils.UserEqual(user.Id, share.Creator)
}

// IsGrantedToUser checks if the user is a grantee of the share. Either by a user grant or by a group grant.
func IsGrantedToUser(share *collaboration.Share, user *userv1beta1.User) bool {
	if share.Grantee.Type == provider.GranteeType_GRANTEE_TYPE_USER && utils.UserEqual(user.Id, share.Grantee.GetUserId()) {
		return true
	}
	if share.Grantee.Type == provider.GranteeType_GRANTEE_TYPE_GROUP {
		// check if any of the user's group is the grantee of the share
		for _, g := range user.Groups {
			if g == share.Grantee.GetGroupId().OpaqueId {
				return true
			}
		}
	}
	return false
}

// MatchesFilter tests if the share passes the filter.
func MatchesFilter(share *collaboration.Share, state collaboration.ShareState, filter *collaboration.Filter) bool {
	switch filter.Type {
	case collaboration.Filter_TYPE_RESOURCE_ID:
		return utils.ResourceIDEqual(share.ResourceId, filter.GetResourceId())
	case collaboration.Filter_TYPE_GRANTEE_TYPE:
		return share.Grantee.Type == filter.GetGranteeType()
	case collaboration.Filter_TYPE_EXCLUDE_DENIALS:
		// This filter type is used to filter out "denial shares". These are currently implemented by having the permission "0".
		// I.e. if the permission is 0 we don't want to show it.
		return !grants.PermissionsEqual(share.Permissions.Permissions, &provider.ResourcePermissions{})
	case collaboration.Filter_TYPE_SPACE_ID:
		return share.ResourceId.SpaceId == filter.GetSpaceId()
	case collaboration.Filter_TYPE_STATE:
		return state == filter.GetState()
	case collaboration.Filter_TYPE_SPACE_ROOT:
		isSpaceRoot := share.ResourceId.SpaceId == share.ResourceId.OpaqueId
		return isSpaceRoot == filter.GetSpaceRoot()
	default:
		return false
	}
}

// MatchesAnyFilter checks if the share passes at least one of the given filters.
func MatchesAnyFilter(share *collaboration.Share, state collaboration.ShareState, filters []*collaboration.Filter) bool {
	for _, f := range filters {
		if MatchesFilter(share, state, f) {
			return true
		}
	}
	return false
}

// MatchesFilters checks if the share passes the given filters.
// Filters of the same type form a disjuntion, a logical OR. Filters of separate type form a conjunction, a logical AND.
// Here is an example:
// (resource_id=1 OR resource_id=2) AND (grantee_type=USER OR grantee_type=GROUP)
func MatchesFilters(share *collaboration.Share, filters []*collaboration.Filter) bool {
	if len(filters) == 0 {
		return true
	}
	grouped := GroupFiltersByType(filters)
	for _, f := range grouped {
		if !MatchesAnyFilter(share, NoState, f) {
			return false
		}
	}
	return true
}

// MatchesFiltersWithState checks if the share passes the given filters.
// This can check filter by share state
// Filters of the same type form a disjuntion, a logical OR. Filters of separate type form a conjunction, a logical AND.
// Here is an example:
// (resource_id=1 OR resource_id=2) AND (grantee_type=USER OR grantee_type=GROUP)
func MatchesFiltersWithState(share *collaboration.Share, state collaboration.ShareState, filters []*collaboration.Filter) bool {
	if len(filters) == 0 {
		return true
	}
	grouped := GroupFiltersByType(filters)
	for _, f := range grouped {
		if !MatchesAnyFilter(share, state, f) {
			return false
		}
	}
	return true
}

// GroupFiltersByType groups the given filters and returns a map using the filter type as the key.
func GroupFiltersByType(filters []*collaboration.Filter) map[collaboration.Filter_Type][]*collaboration.Filter {
	grouped := make(map[collaboration.Filter_Type][]*collaboration.Filter)
	for _, f := range filters {
		grouped[f.Type] = append(grouped[f.Type], f)
	}
	return grouped
}

// FilterFiltersByType returns a slice of filters by a given type.
// If no filter with the given type exists within the filters, then an
// empty slice is returned.
func FilterFiltersByType(f []*collaboration.Filter, t collaboration.Filter_Type) []*collaboration.Filter {
	return GroupFiltersByType(f)[t]
}

// IsExpired tests whether a share is expired
func IsExpired(s *collaboration.Share) bool {
	if e := s.GetExpiration(); e != nil {
		expiration := time.Unix(int64(e.Seconds), int64(e.Nanos))
		return expiration.Before(time.Now())
	}
	return false
}
