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

	tenantpb "github.com/cs3org/go-cs3apis/cs3/identity/tenant/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/plugin"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/sharedconf"
	"github.com/opencloud-eu/reva/v2/pkg/tenant"
	tenantRegistry "github.com/opencloud-eu/reva/v2/pkg/tenant/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/user"
	userRegistry "github.com/opencloud-eu/reva/v2/pkg/user/manager/registry"
)

func init() {
	rgrpc.Register("userprovider", New)
}

type config struct {
	Driver        string                            `mapstructure:"driver"`
	Drivers       map[string]map[string]interface{} `mapstructure:"drivers"`
	TenantDriver  string                            `mapstructure:"tenant_driver"`
	TenantDrivers map[string]map[string]interface{} `mapstructure:"tenant_drivers"`
}

func (c *config) init() {
	if c.Driver == "" {
		c.Driver = "json"
	}

	// Fall back to user driver/drivers when no tenant-specific config is provided.
	if c.TenantDriver == "" {
		c.TenantDriver = c.Driver
	}
	if c.TenantDrivers == nil {
		c.TenantDrivers = c.Drivers
	}

	// Force "null" driver if multi-tenancy is disabled
	if !sharedconf.MultiTenantEnabled() {
		c.TenantDriver = "null"
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
		if f, ok := userRegistry.NewFuncs[c.Driver]; ok {
			mgr, err := f(c.Drivers[c.Driver])
			return mgr, nil, err
		}
	} else {
		return nil, nil, err
	}
	return nil, nil, errtypes.NotFound(fmt.Sprintf("driver %s not found for user manager", c.Driver))
}

func getTenantManager(c *config) (tenant.Manager, error) {
	if f, ok := tenantRegistry.NewFuncs[c.TenantDriver]; ok {
		mgr, err := f(c.TenantDrivers[c.TenantDriver])
		return mgr, err
	}
	return nil, errtypes.NotFound(fmt.Sprintf("driver %s not found for tenant manager", c.TenantDriver))
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
	tenantManager, err := getTenantManager(c)
	if err != nil {
		return nil, err
	}

	return NewWithManagers(userManager, tenantManager, plug), nil
}

// NewWithManagers returns a new UserProviderService with the given managers.
func NewWithManagers(um user.Manager, tm tenant.Manager, plug *plugin.RevaPlugin) rgrpc.Service {
	return &service{
		usermgr:   um,
		tenantmgr: tm,
		plugin:    plug,
	}
}

type service struct {
	usermgr   user.Manager
	tenantmgr tenant.Manager
	plugin    *plugin.RevaPlugin
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
	tenantpb.RegisterTenantAPIServer(ss, s)
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
		switch err.(type) {
		case errtypes.NotFound:
			res.Status = status.NewNotFound(ctx, "user not found")
		case errtypes.Unavailable:
			res.Status = status.NewUnavailable(ctx, "user provider temporarily unavailable")
		default:
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
		switch err.(type) {
		case errtypes.NotFound:
			res.Status = status.NewNotFound(ctx, fmt.Sprintf("user not found %s %s", req.Claim, req.Value))
		case errtypes.Unavailable:
			res.Status = status.NewUnavailable(ctx, "user provider temporarily unavailable")
		default:
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
	if len(req.Filters) > 1 || req.Filters[0].GetType() != userpb.Filter_TYPE_QUERY {
		return nil, fmt.Errorf("only one query filter supported")
	}

	currentUser := revactx.ContextMustGetUser(ctx)

	users, err := s.usermgr.FindUsers(ctx, req.Filters[0].GetQuery(), currentUser.GetId().GetTenantId(), req.SkipFetchingUserGroups)
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

func (s *service) GetTenant(ctx context.Context, req *tenantpb.GetTenantRequest) (*tenantpb.GetTenantResponse, error) {
	log := appctx.GetLogger(ctx)
	t, err := s.tenantmgr.GetTenant(ctx, req.GetTenantId())
	if err != nil {
		log.Warn().Err(err).Interface("tenantid", req.GetTenantId()).Msg("error getting tenant")
		res := &tenantpb.GetTenantResponse{
			Status: status.NewInternal(ctx, "error getting tenant"),
		}
		if _, ok := err.(errtypes.NotFound); ok {
			res.Status = status.NewNotFound(ctx, "tenant not found")
		}
		return res, nil
	}
	return &tenantpb.GetTenantResponse{
		Status: status.NewOK(ctx),
		Tenant: t,
	}, nil
}

func (s *service) GetTenantByClaim(ctx context.Context, req *tenantpb.GetTenantByClaimRequest) (*tenantpb.GetTenantByClaimResponse, error) {
	log := appctx.GetLogger(ctx)
	t, err := s.tenantmgr.GetTenantByClaim(ctx, req.GetClaim(), req.GetValue())
	if err != nil {
		log.Warn().Err(err).Interface("claim", req.GetClaim()).Interface("value", req.GetValue()).Msg("error getting tenant")
		res := &tenantpb.GetTenantByClaimResponse{
			Status: status.NewInternal(ctx, "error getting tenant"),
		}
		if _, ok := err.(errtypes.NotFound); ok {
			res.Status = status.NewNotFound(ctx, "tenant not found")
		}
		return res, nil
	}
	return &tenantpb.GetTenantByClaimResponse{
		Status: status.NewOK(ctx),
		Tenant: t,
	}, nil
}
