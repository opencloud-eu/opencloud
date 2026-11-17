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

package share

import (
	"context"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	ocm "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
	"google.golang.org/genproto/protobuf/field_mask"
)

// Repository is the interface that manipulates the OCM shares repository.
type Repository interface {
	// StoreShare stores a share.
	StoreShare(ctx context.Context, share *ocm.Share) (*ocm.Share, error)

	// GetShare gets the information for a share by the given ref.
	GetShare(ctx context.Context, user *userpb.User, ref *ocm.ShareReference) (*ocm.Share, error)

	// DeleteShare deletes the share pointed by ref.
	DeleteShare(ctx context.Context, user *userpb.User, ref *ocm.ShareReference) error

	// UpdateShare updates the mode of the given share.
	UpdateShare(ctx context.Context, user *userpb.User, ref *ocm.ShareReference, f ...*ocm.UpdateOCMShareRequest_UpdateField) (*ocm.Share, error)

	// ListShares returns the shares created by the user. If md is provided is not nil,
	// it returns only shares attached to the given resource.
	ListShares(ctx context.Context, user *userpb.User, filters []*ocm.ListOCMSharesRequest_Filter) ([]*ocm.Share, error)

	// StoreReceivedShare stores a received share.
	StoreReceivedShare(ctx context.Context, share *ocm.ReceivedShare) (*ocm.ReceivedShare, error)

	// ListReceivedShares returns the list of shares the user has access.
	ListReceivedShares(ctx context.Context, user *userpb.User) ([]*ocm.ReceivedShare, error)

	// GetReceivedShare returns the information for a received share the user has access.
	GetReceivedShare(ctx context.Context, user *userpb.User, ref *ocm.ShareReference) (*ocm.ReceivedShare, error)

	// UpdateReceivedShare updates the received share with share state.
	UpdateReceivedShare(ctx context.Context, user *userpb.User, share *ocm.ReceivedShare, fieldMask *field_mask.FieldMask) (*ocm.ReceivedShare, error)
}

// ResourceIDFilter is an abstraction for creating filter by resource id.
func ResourceIDFilter(id *provider.ResourceId) *ocm.ListOCMSharesRequest_Filter {
	return &ocm.ListOCMSharesRequest_Filter{
		Type: ocm.ListOCMSharesRequest_Filter_TYPE_RESOURCE_ID,
		Term: &ocm.ListOCMSharesRequest_Filter_ResourceId{
			ResourceId: id,
		},
	}
}

// ErrShareAlreadyExisting is the error returned when the share already exists
// for the 3-tuple consisting of (owner, resource, grantee).
var ErrShareAlreadyExisting = errtypes.AlreadyExists("share already exists")

// ErrShareNotFound is the error returned where the share does not exist.
var ErrShareNotFound = errtypes.NotFound("share not found")
