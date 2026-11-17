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

package invite

import (
	"context"
	"errors"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	invitepb "github.com/cs3org/go-cs3apis/cs3/ocm/invite/v1beta1"
)

// Repository is the interfaces used to store the tokens and the invited users.
type Repository interface {
	// AddToken stores the token in the repository.
	AddToken(ctx context.Context, token *invitepb.InviteToken) error

	// GetToken gets the token from the repository.
	GetToken(ctx context.Context, token string) (*invitepb.InviteToken, error)

	// ListTokens gets the valid tokens from the repository (i.e. not expired).
	ListTokens(ctx context.Context, initiator *userpb.UserId) ([]*invitepb.InviteToken, error)

	// AddRemoteUser stores the remote user.
	AddRemoteUser(ctx context.Context, initiator *userpb.UserId, remoteUser *userpb.User) error

	// GetRemoteUser retrieves details about a remote user who has accepted an invite to share.
	GetRemoteUser(ctx context.Context, initiator *userpb.UserId, remoteUserID *userpb.UserId) (*userpb.User, error)

	// FindRemoteUsers finds remote users who have accepted invites based on their attributes.
	FindRemoteUsers(ctx context.Context, initiator *userpb.UserId, query string) ([]*userpb.User, error)

	// DeleteRemoteUser removes from the remote user from the initiator's list.
	DeleteRemoteUser(ctx context.Context, initiator *userpb.UserId, remoteUser *userpb.UserId) error
}

// ErrTokenNotFound is the error returned when the token does not exist.
var ErrTokenNotFound = errors.New("token not found")

// ErrUserAlreadyAccepted is the error returned when the user was
// already added to the accepted users list.
var ErrUserAlreadyAccepted = errors.New("user already added to accepted users")
