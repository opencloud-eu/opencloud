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

package rest

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/gomodule/redigo/redis"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	utils "github.com/opencloud-eu/reva/v2/pkg/cbox/utils"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/user"
	"github.com/opencloud-eu/reva/v2/pkg/user/manager/registry"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func init() {
	registry.Register("rest", New)
}

type manager struct {
	conf            *config
	redisPool       *redis.Pool
	apiTokenManager *utils.APITokenManager
}

type config struct {
	// The address at which the redis server is running
	RedisAddress string `mapstructure:"redis_address" docs:"localhost:6379"`
	// The username for connecting to the redis server
	RedisUsername string `mapstructure:"redis_username" docs:""`
	// The password for connecting to the redis server
	RedisPassword string `mapstructure:"redis_password" docs:""`
	// The time in minutes for which the groups to which a user belongs would be cached
	UserGroupsCacheExpiration int `mapstructure:"user_groups_cache_expiration" docs:"5"`
	// The OIDC Provider
	IDProvider string `mapstructure:"id_provider" docs:"http://cernbox.cern.ch"`
	// Base API Endpoint
	APIBaseURL string `mapstructure:"api_base_url" docs:"https://authorization-service-api-dev.web.cern.ch"`
	// Client ID needed to authenticate
	ClientID string `mapstructure:"client_id" docs:"-"`
	// Client Secret
	ClientSecret string `mapstructure:"client_secret" docs:"-"`

	// Endpoint to generate token to access the API
	OIDCTokenEndpoint string `mapstructure:"oidc_token_endpoint" docs:"https://keycloak-dev.cern.ch/auth/realms/cern/api-access/token"`
	// The target application for which token needs to be generated
	TargetAPI string `mapstructure:"target_api" docs:"authorization-service-api"`
	// The time in seconds between bulk fetch of user accounts
	UserFetchInterval int `mapstructure:"user_fetch_interval" docs:"3600"`
}

func (c *config) init() {
	if c.UserGroupsCacheExpiration == 0 {
		c.UserGroupsCacheExpiration = 5
	}
	if c.RedisAddress == "" {
		c.RedisAddress = ":6379"
	}
	if c.APIBaseURL == "" {
		c.APIBaseURL = "https://authorization-service-api-dev.web.cern.ch"
	}
	if c.TargetAPI == "" {
		c.TargetAPI = "authorization-service-api"
	}
	if c.OIDCTokenEndpoint == "" {
		c.OIDCTokenEndpoint = "https://keycloak-dev.cern.ch/auth/realms/cern/api-access/token"
	}
	if c.IDProvider == "" {
		c.IDProvider = "http://cernbox.cern.ch"
	}
	if c.UserFetchInterval == 0 {
		c.UserFetchInterval = 3600
	}
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		return nil, err
	}
	return c, nil
}

// New returns a user manager implementation that makes calls to the GRAPPA API.
func New(m map[string]interface{}) (user.Manager, error) {
	mgr := &manager{}
	err := mgr.Configure(m)
	if err != nil {
		return nil, err
	}
	return mgr, err
}

func (m *manager) Configure(ml map[string]interface{}) error {
	c, err := parseConfig(ml)
	if err != nil {
		return err
	}
	c.init()
	redisPool := initRedisPool(c.RedisAddress, c.RedisUsername, c.RedisPassword)
	apiTokenManager := utils.InitAPITokenManager(c.TargetAPI, c.OIDCTokenEndpoint, c.ClientID, c.ClientSecret)
	m.conf = c
	m.redisPool = redisPool
	m.apiTokenManager = apiTokenManager

	// Since we're starting a subroutine which would take some time to execute,
	// we can't wait to see if it works before returning the user.Manager object
	// TODO: return err if the fetch fails
	go m.fetchAllUsers()
	return nil
}

func (m *manager) fetchAllUsers() {
	_ = m.fetchAllUserAccounts()
	ticker := time.NewTicker(time.Duration(m.conf.UserFetchInterval) * time.Second)
	work := make(chan os.Signal, 1)
	signal.Notify(work, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	for {
		select {
		case <-work:
			return
		case <-ticker.C:
			_ = m.fetchAllUserAccounts()
		}
	}
}

func (m *manager) fetchAllUserAccounts() error {
	ctx := context.Background()
	url := fmt.Sprintf("%s/api/v1.0/Identity?field=upn&field=primaryAccountEmail&field=displayName&field=uid&field=gid&field=type", m.conf.APIBaseURL)

	for url != "" {
		result, err := m.apiTokenManager.SendAPIGetRequest(ctx, url, false)
		if err != nil {
			return err
		}

		responseData, ok := result["data"].([]interface{})
		if !ok {
			return errors.New("rest: error in type assertion")
		}
		for _, usr := range responseData {
			userData, ok := usr.(map[string]interface{})
			if !ok {
				continue
			}

			_, err = m.parseAndCacheUser(ctx, userData)
			if err != nil {
				continue
			}
		}

		url = ""
		if pagination, ok := result["pagination"].(map[string]interface{}); ok {
			if links, ok := pagination["links"].(map[string]interface{}); ok {
				if next, ok := links["next"].(string); ok {
					url = fmt.Sprintf("%s%s", m.conf.APIBaseURL, next)
				}
			}
		}
	}

	return nil
}

func (m *manager) parseAndCacheUser(ctx context.Context, userData map[string]interface{}) (*userpb.User, error) {
	upn, ok := userData["upn"].(string)
	if !ok {
		return nil, errors.New("rest: missing upn in user data")
	}
	mail, _ := userData["primaryAccountEmail"].(string)
	name, _ := userData["displayName"].(string)
	uidNumber, _ := userData["uid"].(float64)
	gidNumber, _ := userData["gid"].(float64)
	t, _ := userData["type"].(string)
	userType := getUserType(t, upn)

	userID := &userpb.UserId{
		OpaqueId: upn,
		Idp:      m.conf.IDProvider,
		Type:     userType,
	}
	u := &userpb.User{
		Id:          userID,
		Username:    upn,
		Mail:        mail,
		DisplayName: name,
		UidNumber:   int64(uidNumber),
		GidNumber:   int64(gidNumber),
	}

	if err := m.cacheUserDetails(u); err != nil {
		log.Error().Err(err).Msg("rest: error caching user details")
	}
	return u, nil
}

func (m *manager) GetUser(ctx context.Context, uid *userpb.UserId, skipFetchingGroups bool) (*userpb.User, error) {
	u, err := m.fetchCachedUserDetails(uid)
	if err != nil {
		return nil, err
	}

	if !skipFetchingGroups {
		userGroups, err := m.GetUserGroups(ctx, uid)
		if err != nil {
			return nil, err
		}
		u.Groups = userGroups
	}

	return u, nil
}

func (m *manager) GetUserByClaim(ctx context.Context, claim, value string, skipFetchingGroups bool) (*userpb.User, error) {
	u, err := m.fetchCachedUserByParam(claim, value)
	if err != nil {
		return nil, err
	}

	if !skipFetchingGroups {
		userGroups, err := m.GetUserGroups(ctx, u.Id)
		if err != nil {
			return nil, err
		}
		u.Groups = userGroups
	}

	return u, nil
}

func (m *manager) FindUsers(ctx context.Context, query, tenantID string, skipFetchingGroups bool) ([]*userpb.User, error) {
	if tenantID != "" {
		return nil, errtypes.NotSupported("tenant filter not supported in rest user manager")
	}

	// Look at namespaces filters. If the query starts with:
	// "a" => look into primary/secondary/service accounts
	// "l" => look into lightweight/federated accounts
	// none => look into primary

	parts := strings.SplitN(query, ":", 2)

	var namespace string
	if len(parts) == 2 {
		// the query contains a namespace filter
		namespace, query = parts[0], parts[1]
	}

	users, err := m.findCachedUsers(query)
	if err != nil {
		return nil, err
	}

	userSlice := []*userpb.User{}

	var accountsFilters []userpb.UserType
	switch namespace {
	case "":
		accountsFilters = []userpb.UserType{userpb.UserType_USER_TYPE_PRIMARY}
	case "a":
		accountsFilters = []userpb.UserType{userpb.UserType_USER_TYPE_PRIMARY, userpb.UserType_USER_TYPE_SECONDARY, userpb.UserType_USER_TYPE_SERVICE}
	case "l":
		accountsFilters = []userpb.UserType{userpb.UserType_USER_TYPE_LIGHTWEIGHT, userpb.UserType_USER_TYPE_FEDERATED}
	}

	for _, u := range users {
		if isUserAnyType(u, accountsFilters) {
			userSlice = append(userSlice, u)
		}
	}

	return userSlice, nil
}

// isUserAnyType returns true if the user's type is one of types list
func isUserAnyType(user *userpb.User, types []userpb.UserType) bool {
	for _, t := range types {
		if user.GetId().Type == t {
			return true
		}
	}
	return false
}

func (m *manager) GetUserGroups(ctx context.Context, uid *userpb.UserId) ([]string, error) {
	groups, err := m.fetchCachedUserGroups(uid)
	if err == nil {
		return groups, nil
	}

	url := fmt.Sprintf("%s/api/v1.0/Identity/%s/groups?recursive=true", m.conf.APIBaseURL, uid.OpaqueId)
	result, err := m.apiTokenManager.SendAPIGetRequest(ctx, url, false)
	if err != nil {
		return nil, err
	}

	groupData := result["data"].([]interface{})
	groups = []string{}

	for _, g := range groupData {
		groupInfo, ok := g.(map[string]interface{})
		if !ok {
			return nil, errors.New("rest: error in type assertion")
		}
		name, ok := groupInfo["displayName"].(string)
		if ok {
			groups = append(groups, name)
		}
	}

	if err = m.cacheUserGroups(uid, groups); err != nil {
		log := appctx.GetLogger(ctx)
		log.Error().Err(err).Msg("rest: error caching user groups")
	}

	return groups, nil
}

func (m *manager) IsInGroup(ctx context.Context, uid *userpb.UserId, group string) (bool, error) {
	userGroups, err := m.GetUserGroups(ctx, uid)
	if err != nil {
		return false, err
	}

	for _, g := range userGroups {
		if group == g {
			return true, nil
		}
	}
	return false, nil
}

func getUserType(userType, upn string) userpb.UserType {
	var t userpb.UserType
	switch userType {
	case "Application":
		t = userpb.UserType_USER_TYPE_APPLICATION
	case "Service":
		t = userpb.UserType_USER_TYPE_SERVICE
	case "Secondary":
		t = userpb.UserType_USER_TYPE_SECONDARY
	case "Person":
		switch {
		case strings.HasPrefix(upn, "guest"):
			t = userpb.UserType_USER_TYPE_LIGHTWEIGHT
		case strings.Contains(upn, "@"):
			t = userpb.UserType_USER_TYPE_FEDERATED
		default:
			t = userpb.UserType_USER_TYPE_PRIMARY
		}
	default:
		t = userpb.UserType_USER_TYPE_INVALID
	}
	return t

}
