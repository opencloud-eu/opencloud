// Copyright 2018-2020 CERN
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

	group "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	"github.com/pkg/errors"
)

func (s *svc) GetGroup(ctx context.Context, req *group.GetGroupRequest) (*group.GetGroupResponse, error) {
	c, err := pool.GetGroupProviderServiceClient(s.c.GroupProviderEndpoint)
	if err != nil {
		return &group.GetGroupResponse{
			Status: status.NewInternal(ctx, "error getting auth client"),
		}, nil
	}

	res, err := c.GetGroup(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling GetGroup")
	}

	return res, nil
}

func (s *svc) GetGroupByClaim(ctx context.Context, req *group.GetGroupByClaimRequest) (*group.GetGroupByClaimResponse, error) {
	c, err := pool.GetGroupProviderServiceClient(s.c.GroupProviderEndpoint)
	if err != nil {
		return &group.GetGroupByClaimResponse{
			Status: status.NewInternal(ctx, "error getting auth client"),
		}, nil
	}

	res, err := c.GetGroupByClaim(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling GetGroupByClaim")
	}

	return res, nil
}

func (s *svc) FindGroups(ctx context.Context, req *group.FindGroupsRequest) (*group.FindGroupsResponse, error) {
	c, err := pool.GetGroupProviderServiceClient(s.c.GroupProviderEndpoint)
	if err != nil {
		return &group.FindGroupsResponse{
			Status: status.NewInternal(ctx, "error getting auth client"),
		}, nil
	}

	res, err := c.FindGroups(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling FindGroups")
	}

	return res, nil
}

func (s *svc) GetMembers(ctx context.Context, req *group.GetMembersRequest) (*group.GetMembersResponse, error) {
	c, err := pool.GetGroupProviderServiceClient(s.c.GroupProviderEndpoint)
	if err != nil {
		return &group.GetMembersResponse{
			Status: status.NewInternal(ctx, "error getting auth client"),
		}, nil
	}

	res, err := c.GetMembers(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling GetMembers")
	}

	return res, nil
}

func (s *svc) HasMember(ctx context.Context, req *group.HasMemberRequest) (*group.HasMemberResponse, error) {
	c, err := pool.GetGroupProviderServiceClient(s.c.GroupProviderEndpoint)
	if err != nil {
		return &group.HasMemberResponse{
			Status: status.NewInternal(ctx, "error getting auth client"),
		}, nil
	}

	res, err := c.HasMember(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "gateway: error calling HasMember")
	}

	return res, nil
}
