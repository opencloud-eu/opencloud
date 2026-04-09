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
	"fmt"
	"net/url"
	"strings"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc"
	"github.com/opencloud-eu/reva/v2/pkg/sharedconf"
	"github.com/opencloud-eu/reva/v2/pkg/storage/cache"
	"github.com/opencloud-eu/reva/v2/pkg/token"
	"github.com/opencloud-eu/reva/v2/pkg/token/manager/registry"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

const (
	_spaceTypePersonal = "personal"
	_spaceTypeProject  = "project"
	_spaceTypeVirtual  = "virtual"
)

func init() {
	rgrpc.Register("gateway", New)
}

type config struct {
	AuthRegistryEndpoint          string `mapstructure:"authregistrysvc"`
	ApplicationAuthEndpoint       string `mapstructure:"applicationauthsvc"`
	StorageRegistryEndpoint       string `mapstructure:"storageregistrysvc"`
	AppRegistryEndpoint           string `mapstructure:"appregistrysvc"`
	PreferencesEndpoint           string `mapstructure:"preferencessvc"`
	UserShareProviderEndpoint     string `mapstructure:"usershareprovidersvc"`
	PublicShareProviderEndpoint   string `mapstructure:"publicshareprovidersvc"`
	OCMShareProviderEndpoint      string `mapstructure:"ocmshareprovidersvc"`
	OCMInviteManagerEndpoint      string `mapstructure:"ocminvitemanagersvc"`
	OCMProviderAuthorizerEndpoint string `mapstructure:"ocmproviderauthorizersvc"`
	OCMCoreEndpoint               string `mapstructure:"ocmcoresvc"`
	UserProviderEndpoint          string `mapstructure:"userprovidersvc"`
	GroupProviderEndpoint         string `mapstructure:"groupprovidersvc"`
	TenantProviderEndpoint        string `mapstructure:"tenantprovidersvc"`
	DataTxEndpoint                string `mapstructure:"datatx"`
	DataGatewayEndpoint           string `mapstructure:"datagateway"`
	PermissionsEndpoint           string `mapstructure:"permissionssvc"`
	CommitShareToStorageGrant     bool   `mapstructure:"commit_share_to_storage_grant"`
	DisableHomeCreationOnLogin    bool   `mapstructure:"disable_home_creation_on_login"`
	TransferSharedSecret          string `mapstructure:"transfer_shared_secret"`
	TransferExpires               int64  `mapstructure:"transfer_expires"`
	TokenManager                  string `mapstructure:"token_manager"`
	// ShareFolder is the location where to create shares in the recipient's storage provider.
	// FIXME get rid of ShareFolder, there are findByPath calls in the ocmshareporvider.go and usershareprovider.go
	ShareFolder                    string                            `mapstructure:"share_folder"`
	DataTransfersFolder            string                            `mapstructure:"data_transfers_folder"`
	TokenManagers                  map[string]map[string]interface{} `mapstructure:"token_managers"`
	AllowedUserAgents              map[string][]string               `mapstructure:"allowed_user_agents"` // map[path][]user-agent
	CreatePersonalSpaceCacheConfig cache.Config                      `mapstructure:"create_personal_space_cache_config"`
	ProviderCacheConfig            cache.Config                      `mapstructure:"provider_cache_config"`
	UseCommonSpaceRootShareLogic   bool                              `mapstructure:"use_common_space_root_share_logic"`
}

// sets defaults
func (c *config) init() {
	if c.ShareFolder == "" {
		c.ShareFolder = "MyShares"
	}

	c.ShareFolder = strings.Trim(c.ShareFolder, "/")

	if c.DataTransfersFolder == "" {
		c.DataTransfersFolder = "DataTransfers"
	}

	if c.TokenManager == "" {
		c.TokenManager = "jwt"
	}

	// if services address are not specified we used the shared conf
	// for the gatewaysvc to have dev setups very quickly.
	c.AuthRegistryEndpoint = sharedconf.GetGatewaySVC(c.AuthRegistryEndpoint)
	c.ApplicationAuthEndpoint = sharedconf.GetGatewaySVC(c.ApplicationAuthEndpoint)
	c.StorageRegistryEndpoint = sharedconf.GetGatewaySVC(c.StorageRegistryEndpoint)
	c.AppRegistryEndpoint = sharedconf.GetGatewaySVC(c.AppRegistryEndpoint)
	c.PreferencesEndpoint = sharedconf.GetGatewaySVC(c.PreferencesEndpoint)
	c.UserShareProviderEndpoint = sharedconf.GetGatewaySVC(c.UserShareProviderEndpoint)
	c.PublicShareProviderEndpoint = sharedconf.GetGatewaySVC(c.PublicShareProviderEndpoint)
	c.OCMShareProviderEndpoint = sharedconf.GetGatewaySVC(c.OCMShareProviderEndpoint)
	c.OCMInviteManagerEndpoint = sharedconf.GetGatewaySVC(c.OCMInviteManagerEndpoint)
	c.OCMProviderAuthorizerEndpoint = sharedconf.GetGatewaySVC(c.OCMProviderAuthorizerEndpoint)
	c.OCMCoreEndpoint = sharedconf.GetGatewaySVC(c.OCMCoreEndpoint)
	c.UserProviderEndpoint = sharedconf.GetGatewaySVC(c.UserProviderEndpoint)
	c.GroupProviderEndpoint = sharedconf.GetGatewaySVC(c.GroupProviderEndpoint)
	// Fall back to userprovidersvc when no dedicated tenant provider is configured.
	if c.TenantProviderEndpoint == "" {
		c.TenantProviderEndpoint = c.UserProviderEndpoint
	} else {
		c.TenantProviderEndpoint = sharedconf.GetGatewaySVC(c.TenantProviderEndpoint)
	}
	c.DataTxEndpoint = sharedconf.GetGatewaySVC(c.DataTxEndpoint)

	c.DataGatewayEndpoint = sharedconf.GetDataGateway(c.DataGatewayEndpoint)

	// use shared secret if not set
	c.TransferSharedSecret = sharedconf.GetJWTSecret(c.TransferSharedSecret)

	// lifetime for the transfer token (TUS upload)
	if c.TransferExpires == 0 {
		c.TransferExpires = 100 * 60 // seconds
	}

	// caching needs to be explicitly enabled
	if c.ProviderCacheConfig.Store == "" {
		c.ProviderCacheConfig.Store = "noop"
	}

	if c.ProviderCacheConfig.Database == "" {
		c.ProviderCacheConfig.Database = "reva"
	}

	if c.CreatePersonalSpaceCacheConfig.Store == "" {
		c.CreatePersonalSpaceCacheConfig.Store = "memory"
	}

	if c.CreatePersonalSpaceCacheConfig.Database == "" {
		c.CreatePersonalSpaceCacheConfig.Database = "reva"
	}
}

type svc struct {
	c                        *config
	dataGatewayURL           url.URL
	tokenmgr                 token.Manager
	providerCache            cache.ProviderCache
	createPersonalSpaceCache cache.CreatePersonalSpaceCache
}

// New creates a new gateway svc that acts as a proxy for any grpc operation.
// The gateway is responsible for high-level controls: rate-limiting, coordination between svcs
// like sharing and storage acls, asynchronous transactions, ...
func New(m map[string]interface{}, _ *grpc.Server, _ *zerolog.Logger) (rgrpc.Service, error) {
	c, err := parseConfig(m)
	if err != nil {
		return nil, err
	}

	c.init()

	// ensure DataGatewayEndpoint is a valid URI
	u, err := url.Parse(c.DataGatewayEndpoint)
	if err != nil {
		return nil, err
	}

	tokenManager, err := getTokenManager(c.TokenManager, c.TokenManagers)
	if err != nil {
		return nil, err
	}

	s := &svc{
		c:                        c,
		dataGatewayURL:           *u,
		tokenmgr:                 tokenManager,
		providerCache:            cache.GetProviderCache(c.ProviderCacheConfig),
		createPersonalSpaceCache: cache.GetCreatePersonalSpaceCache(c.CreatePersonalSpaceCacheConfig),
	}

	return s, nil
}

func (s *svc) Register(ss *grpc.Server) {
	gateway.RegisterGatewayAPIServer(ss, s)
}

func (s *svc) Close() error {
	s.providerCache.Close()
	s.createPersonalSpaceCache.Close()
	return nil
}

func (s *svc) UnprotectedEndpoints() []string {
	return []string{"/cs3.gateway.v1beta1.GatewayAPI"}
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "gateway: error decoding conf")
		return nil, err
	}
	return c, nil
}

func getTokenManager(manager string, m map[string]map[string]interface{}) (token.Manager, error) {
	if f, ok := registry.NewFuncs[manager]; ok {
		return f(m[manager])
	}

	return nil, errtypes.NotFound(fmt.Sprintf("driver %s not found for token manager", manager))
}
