package config

import (
	"context"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service

	Service Service `yaml:"-"`
	Log     *Log    `yaml:"log"`
	Debug   Debug   `yaml:"debug"`

	GRPC GRPCConfig `yaml:"grpc"`

	TokenManager *TokenManager `yaml:"token_manager"`
	Reva         *shared.Reva  `yaml:"reva"`

	SkipUserGroupsInToken bool `yaml:"skip_user_groups_in_token" env:"GATEWAY_SKIP_USER_GROUPS_IN_TOKEN" desc:"Disables the loading of user's group memberships from the reva access token." introductionVersion:"1.0.0"`

	CommitShareToStorageGrant  bool   `yaml:"commit_share_to_storage_grant" env:"GATEWAY_COMMIT_SHARE_TO_STORAGE_GRANT" desc:"Commit shares to storage grants. This grants access to shared resources for the share receiver directly on the storage." introductionVersion:"1.0.0"`
	ShareFolder                string `yaml:"share_folder_name" env:"GATEWAY_SHARE_FOLDER_NAME" desc:"Name of the share folder in users' home space." introductionVersion:"1.0.0"`
	DisableHomeCreationOnLogin bool   `yaml:"disable_home_creation_on_login" env:"GATEWAY_DISABLE_HOME_CREATION_ON_LOGIN" desc:"Disable creation of the home space on login." introductionVersion:"1.0.0"`
	TransferSecret             string `yaml:"transfer_secret" env:"OC_TRANSFER_SECRET" desc:"The storage transfer secret." introductionVersion:"1.0.0"`
	TransferExpires            int    `yaml:"transfer_expires" env:"GATEWAY_TRANSFER_EXPIRES" desc:"Expiry for the gateway tokens." introductionVersion:"1.0.0"`
	Cache                      Cache  `yaml:"cache"`

	FrontendPublicURL string `yaml:"frontend_public_url" env:"OC_URL;GATEWAY_FRONTEND_PUBLIC_URL" desc:"The public facing URL of the OpenCloud frontend." introductionVersion:"1.0.0"`

	UsersEndpoint             string `yaml:"users_endpoint" env:"GATEWAY_USERS_ENDPOINT" desc:"The endpoint of the users service. Can take a service name or a gRPC URI with the dns, kubernetes or unix protocol." introductionVersion:"1.0.0"`
	GroupsEndpoint            string `yaml:"groups_endpoint" env:"GATEWAY_GROUPS_ENDPOINT" desc:"The endpoint of the groups service. Can take a service name or a gRPC URI with the dns, kubernetes or unix protocol." introductionVersion:"1.0.0"`
	PermissionsEndpoint       string `yaml:"permissions_endpoint" env:"GATEWAY_PERMISSIONS_ENDPOINT" desc:"The endpoint of the permissions service. Can take a service name or a gRPC URI with the dns, kubernetes or unix protocol." introductionVersion:"1.0.0"`
	SharingEndpoint           string `yaml:"sharing_endpoint" env:"GATEWAY_SHARING_ENDPOINT" desc:"The endpoint of the shares service. Can take a service name or a gRPC URI with the dns, kubernetes or unix protocol." introductionVersion:"1.0.0"`
	AuthAppEndpoint           string `yaml:"auth_app_endpoint" env:"GATEWAY_AUTH_APP_ENDPOINT" desc:"The endpoint of the auth-app service. Can take a service name or a gRPC URI with the dns, kubernetes or unix protocol." introductionVersion:"1.0.0"`
	AuthBasicEndpoint         string `yaml:"auth_basic_endpoint" env:"GATEWAY_AUTH_BASIC_ENDPOINT" desc:"The endpoint of the auth-basic service. Can take a service name or a gRPC URI with the dns, kubernetes or unix protocol." introductionVersion:"1.0.0"`
	AuthBearerEndpoint        string `yaml:"auth_bearer_endpoint" env:"GATEWAY_AUTH_BEARER_ENDPOINT" desc:"The endpoint of the auth-bearer service. Can take a service name or a gRPC URI with the dns, kubernetes or unix protocol." introductionVersion:"1.0.0"`
	AuthMachineEndpoint       string `yaml:"auth_machine_endpoint" env:"GATEWAY_AUTH_MACHINE_ENDPOINT" desc:"The endpoint of the auth-machine service. Can take a service name or a gRPC URI with the dns, kubernetes or unix protocol." introductionVersion:"1.0.0"`
	AuthServiceEndpoint       string `yaml:"auth_service_endpoint" env:"GATEWAY_AUTH_SERVICE_ENDPOINT" desc:"The endpoint of the auth-service service. Can take a service name or a gRPC URI with the dns, kubernetes or unix protocol." introductionVersion:"1.0.0"`
	StoragePublicLinkEndpoint string `yaml:"storage_public_link_endpoint" env:"GATEWAY_STORAGE_PUBLIC_LINK_ENDPOINT" desc:"The endpoint of the storage-publiclink service. Can take a service name or a gRPC URI with the dns, kubernetes or unix protocol." introductionVersion:"1.0.0"`
	StorageUsersEndpoint      string `yaml:"storage_users_endpoint" env:"GATEWAY_STORAGE_USERS_ENDPOINT" desc:"The endpoint of the storage-users service. Can take a service name or a gRPC URI with the dns, kubernetes or unix protocol." introductionVersion:"1.0.0"`
	StorageSharesEndpoint     string `yaml:"storage_shares_endpoint" env:"GATEWAY_STORAGE_SHARES_ENDPOINT" desc:"The endpoint of the storage-shares service. Can take a service name or a gRPC URI with the dns, kubernetes or unix protocol." introductionVersion:"1.0.0"`
	AppRegistryEndpoint       string `yaml:"app_registry_endpoint" env:"GATEWAY_APP_REGISTRY_ENDPOINT" desc:"The endpoint of the app-registry service. Can take a service name or a gRPC URI with the dns, kubernetes or unix protocol." introductionVersion:"1.0.0"`
	OCMEndpoint               string `yaml:"ocm_endpoint" env:"GATEWAY_OCM_ENDPOINT" desc:"The endpoint of the ocm service. Can take a service name or a gRPC URI with the dns, kubernetes or unix protocol." introductionVersion:"1.0.0"`

	StorageRegistry StorageRegistry `yaml:"storage_registry"` // TODO: should we even support switching this?

	Context context.Context `yaml:"-"`
}

type Log struct {
	Level  string `yaml:"level" env:"OC_LOG_LEVEL;GATEWAY_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"1.0.0"`
	Pretty bool   `yaml:"pretty" env:"OC_LOG_PRETTY;GATEWAY_LOG_PRETTY" desc:"Activates pretty log output." introductionVersion:"1.0.0"`
	Color  bool   `yaml:"color" env:"OC_LOG_COLOR;GATEWAY_LOG_COLOR" desc:"Activates colorized log output." introductionVersion:"1.0.0"`
	File   string `yaml:"file" env:"OC_LOG_FILE;GATEWAY_LOG_FILE" desc:"The path to the log file. Activates logging to this file if set." introductionVersion:"1.0.0"`
}

type Service struct {
	Name string `yaml:"-"`
}

type Debug struct {
	Addr   string `yaml:"addr" env:"GATEWAY_DEBUG_ADDR" desc:"Bind address of the debug server, where metrics, health, config and debug endpoints will be exposed." introductionVersion:"1.0.0"`
	Token  string `yaml:"token" env:"GATEWAY_DEBUG_TOKEN" desc:"Token to secure the metrics endpoint." introductionVersion:"1.0.0"`
	Pprof  bool   `yaml:"pprof" env:"GATEWAY_DEBUG_PPROF" desc:"Enables pprof, which can be used for profiling." introductionVersion:"1.0.0"`
	Zpages bool   `yaml:"zpages" env:"GATEWAY_DEBUG_ZPAGES" desc:"Enables zpages, which can be used for collecting and viewing in-memory traces." introductionVersion:"1.0.0"`
}

type GRPCConfig struct {
	Addr      string                 `yaml:"addr" env:"OC_GATEWAY_GRPC_ADDR;GATEWAY_GRPC_ADDR" desc:"The bind address of the GRPC service." introductionVersion:"1.0.0"`
	TLS       *shared.GRPCServiceTLS `yaml:"tls"`
	Namespace string                 `yaml:"-"`
	Protocol  string                 `yaml:"protocol" env:"OC_GRPC_PROTOCOL;GATEWAY_GRPC_PROTOCOL" desc:"The transport protocol of the GRPC service." introductionVersion:"1.0.0"`
}

type StorageRegistry struct {
	Driver              string   `yaml:"driver" env:"GATEWAY_STORAGE_REGISTRY_DRIVER" desc:"The driver name of the storage registry to use." introductionVersion:"1.0.0"`
	Rules               []string `yaml:"rules" env:"GATEWAY_STORAGE_REGISTRY_RULES" desc:"The rules for the storage registry. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	JSON                string   `yaml:"json" env:"GATEWAY_STORAGE_REGISTRY_CONFIG_JSON" desc:"Additional configuration for the storage registry in json format." introductionVersion:"1.0.0"`
	StorageUsersMountID string   `yaml:"storage_users_mount_id" env:"GATEWAY_STORAGE_USERS_MOUNT_ID" desc:"Mount ID of this storage. Admins can set the ID for the storage in this config option manually which is then used to reference the storage. Any reasonable long string is possible, preferably this would be an UUIDv4 format." introductionVersion:"1.0.0"`
}

// Cache holds cache config
type Cache struct {
	ProviderCacheStore                string        `yaml:"provider_cache_store" env:"OC_CACHE_STORE;GATEWAY_PROVIDER_CACHE_STORE" desc:"The type of the cache store. Supported values are: 'memory', 'redis-sentinel', 'nats-js-kv', 'noop'. See the text description for details." introductionVersion:"1.0.0"`
	ProviderCacheNodes                []string      `yaml:"provider_cache_nodes" env:"OC_CACHE_STORE_NODES;GATEWAY_PROVIDER_CACHE_STORE_NODES" desc:"A list of nodes to access the configured store. This has no effect when 'memory' store is configured. Note that the behaviour how nodes are used is dependent on the library of the configured store. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	ProviderCacheDatabase             string        `yaml:"provider_cache_database" env:"OC_CACHE_DATABASE" desc:"The database name the configured store should use." introductionVersion:"1.0.0"`
	ProviderCacheTTL                  time.Duration `yaml:"provider_cache_ttl" env:"OC_CACHE_TTL;GATEWAY_PROVIDER_CACHE_TTL" desc:"Default time to live for user info in the cache. Only applied when access tokens has no expiration. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	ProviderCacheDisablePersistence   bool          `yaml:"provider_cache_disable_persistence" env:"OC_CACHE_DISABLE_PERSISTENCE;GATEWAY_PROVIDER_CACHE_DISABLE_PERSISTENCE" desc:"Disables persistence of the provider cache. Only applies when store type 'nats-js-kv' is configured. Defaults to false." introductionVersion:"1.0.0"`
	ProviderCacheAuthUsername         string        `yaml:"provider_cache_auth_username" env:"OC_CACHE_AUTH_USERNAME;GATEWAY_PROVIDER_CACHE_AUTH_USERNAME" desc:"The username to use for authentication. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"1.0.0"`
	ProviderCacheAuthPassword         string        `yaml:"provider_cache_auth_password" env:"OC_CACHE_AUTH_PASSWORD;GATEWAY_PROVIDER_CACHE_AUTH_PASSWORD" desc:"The password to use for authentication. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"1.0.0"`
	CreateHomeCacheStore              string        `yaml:"create_home_cache_store" env:"OC_CACHE_STORE;GATEWAY_CREATE_HOME_CACHE_STORE" desc:"The type of the cache store. Supported values are: 'memory', 'redis-sentinel', 'nats-js-kv', 'noop'. See the text description for details." introductionVersion:"1.0.0"`
	CreateHomeCacheNodes              []string      `yaml:"create_home_cache_nodes" env:"OC_CACHE_STORE_NODES;GATEWAY_CREATE_HOME_CACHE_STORE_NODES" desc:"A list of nodes to access the configured store. This has no effect when 'memory' store is configured. Note that the behaviour how nodes are used is dependent on the library of the configured store. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	CreateHomeCacheDatabase           string        `yaml:"create_home_cache_database" env:"OC_CACHE_DATABASE" desc:"The database name the configured store should use." introductionVersion:"1.0.0"`
	CreateHomeCacheTTL                time.Duration `yaml:"create_home_cache_ttl" env:"OC_CACHE_TTL;GATEWAY_CREATE_HOME_CACHE_TTL" desc:"Default time to live for user info in the cache. Only applied when access tokens has no expiration. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	CreateHomeCacheDisablePersistence bool          `yaml:"create_home_cache_disable_persistence" env:"OC_CACHE_DISABLE_PERSISTENCE;GATEWAY_CREATE_HOME_CACHE_DISABLE_PERSISTENCE" desc:"Disables persistence of the create home cache. Only applies when store type 'nats-js-kv' is configured. Defaults to false." introductionVersion:"1.0.0"`
	CreateHomeCacheAuthUsername       string        `yaml:"create_home_cache_auth_username" env:"OC_CACHE_AUTH_USERNAME;GATEWAY_CREATE_HOME_CACHE_AUTH_USERNAME" desc:"The username to use for authentication. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"1.0.0"`
	CreateHomeCacheAuthPassword       string        `yaml:"create_home_cache_auth_password" env:"OC_CACHE_AUTH_PASSWORD;GATEWAY_CREATE_HOME_CACHE_AUTH_PASSWORD" desc:"The password to use for authentication. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"1.0.0"`
}
