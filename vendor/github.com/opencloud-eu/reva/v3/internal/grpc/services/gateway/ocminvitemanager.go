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

package gateway

import (
	"context"

	invitepb "github.com/cs3org/go-cs3apis/cs3/ocm/invite/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	"github.com/pkg/errors"
)

func (s *svc) GenerateInviteToken(ctx context.Context, req *invitepb.GenerateInviteTokenRequest) (*invitepb.GenerateInviteTokenResponse, error) {
	c, err := pool.GetOCMInviteManagerClient(s.c.OCMInviteManagerEndpoint)
	if err != nil {
		return &invitepb.GenerateInviteTokenResponse{
			Status: status.NewInternal(ctx, "error getting user invite provider client"),
		}, nil
	}

	res, err := c.GenerateInviteToken(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling GenerateInviteToken")
	}

	return res, nil
}

func (s *svc) ListInviteTokens(ctx context.Context, req *invitepb.ListInviteTokensRequest) (*invitepb.ListInviteTokensResponse, error) {
	c, err := pool.GetOCMInviteManagerClient(s.c.OCMInviteManagerEndpoint)
	if err != nil {
		return &invitepb.ListInviteTokensResponse{
			Status: status.NewInternal(ctx, "error getting user invite provider client"),
		}, nil
	}

	res, err := c.ListInviteTokens(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling ListInviteTokens")
	}

	return res, nil
}

func (s *svc) ForwardInvite(ctx context.Context, req *invitepb.ForwardInviteRequest) (*invitepb.ForwardInviteResponse, error) {
	c, err := pool.GetOCMInviteManagerClient(s.c.OCMInviteManagerEndpoint)
	if err != nil {
		return &invitepb.ForwardInviteResponse{
			Status: status.NewInternal(ctx, "error getting user invite provider client"),
		}, nil
	}

	res, err := c.ForwardInvite(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling ForwardInvite")
	}

	return res, nil
}

func (s *svc) AcceptInvite(ctx context.Context, req *invitepb.AcceptInviteRequest) (*invitepb.AcceptInviteResponse, error) {
	c, err := pool.GetOCMInviteManagerClient(s.c.OCMInviteManagerEndpoint)
	if err != nil {
		return &invitepb.AcceptInviteResponse{
			Status: status.NewInternal(ctx, "error getting user invite provider client"),
		}, nil
	}

	res, err := c.AcceptInvite(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling AcceptInvite")
	}

	return res, nil
}

func (s *svc) GetAcceptedUser(ctx context.Context, req *invitepb.GetAcceptedUserRequest) (*invitepb.GetAcceptedUserResponse, error) {
	c, err := pool.GetOCMInviteManagerClient(s.c.OCMInviteManagerEndpoint)
	if err != nil {
		return &invitepb.GetAcceptedUserResponse{
			Status: status.NewInternal(ctx, "error getting user invite provider client"),
		}, nil
	}

	res, err := c.GetAcceptedUser(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling GetAcceptedUser")
	}

	return res, nil
}

func (s *svc) FindAcceptedUsers(ctx context.Context, req *invitepb.FindAcceptedUsersRequest) (*invitepb.FindAcceptedUsersResponse, error) {
	c, err := pool.GetOCMInviteManagerClient(s.c.OCMInviteManagerEndpoint)
	if err != nil {
		return &invitepb.FindAcceptedUsersResponse{
			Status: status.NewInternal(ctx, "error getting user invite provider client"),
		}, nil
	}

	res, err := c.FindAcceptedUsers(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling FindAcceptedUsers")
	}

	return res, nil
}

func (s *svc) DeleteAcceptedUser(ctx context.Context, req *invitepb.DeleteAcceptedUserRequest) (*invitepb.DeleteAcceptedUserResponse, error) {
	c, err := pool.GetOCMInviteManagerClient(s.c.OCMInviteManagerEndpoint)
	if err != nil {
		return &invitepb.DeleteAcceptedUserResponse{
			Status: status.NewInternal(ctx, "error getting user invite provider client"),
		}, nil
	}

	res, err := c.DeleteAcceptedUser(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling FindAcceptedUsers")
	}

	return res, nil
}
