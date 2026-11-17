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

package gateway

import (
	"context"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	"github.com/pkg/errors"
)

func (s *svc) GetUser(ctx context.Context, req *user.GetUserRequest) (*user.GetUserResponse, error) {
	c, err := pool.GetUserProviderServiceClient(s.c.UserProviderEndpoint)
	if err != nil {
		return &user.GetUserResponse{
			Status: status.NewInternal(ctx, "error getting auth client"),
		}, nil
	}

	res, err := c.GetUser(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling GetUser")
	}

	return res, nil
}

func (s *svc) GetUserByClaim(ctx context.Context, req *user.GetUserByClaimRequest) (*user.GetUserByClaimResponse, error) {
	c, err := pool.GetUserProviderServiceClient(s.c.UserProviderEndpoint)
	if err != nil {
		return &user.GetUserByClaimResponse{
			Status: status.NewInternal(ctx, "error getting auth client"),
		}, nil
	}

	res, err := c.GetUserByClaim(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling GetUserByClaim")
	}

	return res, nil
}

func (s *svc) FindUsers(ctx context.Context, req *user.FindUsersRequest) (*user.FindUsersResponse, error) {
	c, err := pool.GetUserProviderServiceClient(s.c.UserProviderEndpoint)
	if err != nil {
		return &user.FindUsersResponse{
			Status: status.NewInternal(ctx, "error getting auth client"),
		}, nil
	}

	res, err := c.FindUsers(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling FindUsers")
	}

	return res, nil
}

func (s *svc) GetUserGroups(ctx context.Context, req *user.GetUserGroupsRequest) (*user.GetUserGroupsResponse, error) {
	c, err := pool.GetUserProviderServiceClient(s.c.UserProviderEndpoint)
	if err != nil {
		return &user.GetUserGroupsResponse{
			Status: status.NewInternal(ctx, "error getting auth client"),
		}, nil
	}

	res, err := c.GetUserGroups(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling GetUserGroups")
	}

	return res, nil
}
