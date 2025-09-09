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

package userprovider

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/plugin"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/user"
	"github.com/opencloud-eu/reva/v2/pkg/user/manager/registry"
)

func init() {
	rgrpc.Register("userprovider", New)
}

type config struct {
	Driver  string                            `mapstructure:"driver"`
	Drivers map[string]map[string]interface{} `mapstructure:"drivers"`
}

func (c *config) init() {
	if c.Driver == "" {
		c.Driver = "json"
	}
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}
	c.init()
	return c, nil
}

func getDriver(c *config) (user.Manager, *plugin.RevaPlugin, error) {
	p, err := plugin.Load("userprovider", c.Driver)
	if err == nil {
		manager, ok := p.Plugin.(user.Manager)
		if !ok {
			return nil, nil, fmt.Errorf("could not assert the loaded plugin")
		}
		pluginConfig := filepath.Base(c.Driver)
		err = manager.Configure(c.Drivers[pluginConfig])
		if err != nil {
			return nil, nil, err
		}
		return manager, p, nil
	} else if _, ok := err.(errtypes.NotFound); ok {
		// plugin not found, fetch the driver from the in-memory registry
		if f, ok := registry.NewFuncs[c.Driver]; ok {
			mgr, err := f(c.Drivers[c.Driver])
			return mgr, nil, err
		}
	} else {
		return nil, nil, err
	}
	return nil, nil, errtypes.NotFound(fmt.Sprintf("driver %s not found for user manager", c.Driver))
}

// New returns a new UserProviderServiceServer.
func New(m map[string]interface{}, ss *grpc.Server, _ *zerolog.Logger) (rgrpc.Service, error) {
	c, err := parseConfig(m)
	if err != nil {
		return nil, err
	}
	userManager, plug, err := getDriver(c)
	if err != nil {
		return nil, err
	}
	svc := &service{
		usermgr: userManager,
		plugin:  plug,
	}

	return svc, nil
}

type service struct {
	usermgr user.Manager
	plugin  *plugin.RevaPlugin
}

func (s *service) Close() error {
	if s.plugin != nil {
		s.plugin.Kill()
	}
	return nil
}

func (s *service) UnprotectedEndpoints() []string {
	return []string{"/cs3.identity.user.v1beta1.UserAPI/GetUser", "/cs3.identity.user.v1beta1.UserAPI/GetUserByClaim", "/cs3.identity.user.v1beta1.UserAPI/GetUserGroups"}
}

func (s *service) Register(ss *grpc.Server) {
	userpb.RegisterUserAPIServer(ss, s)
}

func (s *service) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.GetUserResponse, error) {
	if req.UserId == nil {
		res := &userpb.GetUserResponse{
			Status: status.NewInvalid(ctx, "userid missing"),
		}
		return res, nil
	}

	// Only request users from the same tenant as the current user
	if currentUser, ok := revactx.ContextGetUser(ctx); ok {
		req.UserId.TenantId = currentUser.GetId().GetTenantId()
	}

	user, err := s.usermgr.GetUser(ctx, req.UserId, req.SkipFetchingUserGroups)
	if err != nil {
		res := &userpb.GetUserResponse{}
		if _, ok := err.(errtypes.NotFound); ok {
			res.Status = status.NewNotFound(ctx, "user not found")
		} else {
			res.Status = status.NewInternal(ctx, "error getting user")
		}
		return res, nil
	}

	res := &userpb.GetUserResponse{
		Status: status.NewOK(ctx),
		User:   user,
	}
	return res, nil
}

func (s *service) GetUserByClaim(ctx context.Context, req *userpb.GetUserByClaimRequest) (*userpb.GetUserByClaimResponse, error) {
	tenantID := ""
	if currentUser, ok := revactx.ContextGetUser(ctx); ok {
		tenantID = currentUser.GetId().GetTenantId()
	}
	user, err := s.usermgr.GetUserByClaim(ctx, req.Claim, req.Value, tenantID, req.SkipFetchingUserGroups)
	if err != nil {
		res := &userpb.GetUserByClaimResponse{}
		if _, ok := err.(errtypes.NotFound); ok {
			res.Status = status.NewNotFound(ctx, fmt.Sprintf("user not found %s %s", req.Claim, req.Value))
		} else {
			res.Status = status.NewInternal(ctx, "error getting user by claim")
		}
		return res, nil
	}

	res := &userpb.GetUserByClaimResponse{
		Status: status.NewOK(ctx),
		User:   user,
	}
	return res, nil
}

func (s *service) FindUsers(ctx context.Context, req *userpb.FindUsersRequest) (*userpb.FindUsersResponse, error) {
	currentUser := revactx.ContextMustGetUser(ctx)

	users, err := s.usermgr.FindUsers(ctx, req.Filter, currentUser.GetId().GetTenantId(), req.SkipFetchingUserGroups)
	if err != nil {
		res := &userpb.FindUsersResponse{
			Status: status.NewInternal(ctx, "error finding users"),
		}
		return res, nil
	}

	// sort users by username
	sort.Slice(users, func(i, j int) bool {
		return users[i].Username <= users[j].Username
	})

	res := &userpb.FindUsersResponse{
		Status: status.NewOK(ctx),
		Users:  users,
	}
	return res, nil
}

func (s *service) GetUserGroups(ctx context.Context, req *userpb.GetUserGroupsRequest) (*userpb.GetUserGroupsResponse, error) {
	log := appctx.GetLogger(ctx)
	if req.UserId == nil {
		res := &userpb.GetUserGroupsResponse{
			Status: status.NewInvalid(ctx, "userid missing"),
		}
		return res, nil
	}
	groups, err := s.usermgr.GetUserGroups(ctx, req.UserId)
	if err != nil {
		log.Warn().Err(err).Interface("userid", req.UserId).Msg("error getting user groups")
		res := &userpb.GetUserGroupsResponse{
			Status: status.NewInternal(ctx, "error getting user groups"),
		}
		return res, nil
	}

	res := &userpb.GetUserGroupsResponse{
		Status: status.NewOK(ctx),
		Groups: groups,
	}
	return res, nil
}
