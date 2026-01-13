package config

import (
	"context"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// Config holds Config config
type Config struct {
	Commons  *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service
	Service  Service         `yaml:"-"`
	LogLevel string          `yaml:"loglevel" env:"OC_LOG_LEVEL;STORAGE_SYSTEM_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"1.0.0"`
	Debug    Debug           `yaml:"debug"`

	GRPC GRPCConfig `yaml:"grpc"`
	HTTP HTTPConfig `yaml:"http"`

	TokenManager *TokenManager `yaml:"token_manager"`
	Reva         *shared.Reva  `yaml:"reva"`

	SystemUserID     string `yaml:"system_user_id" env:"OC_SYSTEM_USER_ID" desc:"ID of the OpenCloud storage-system system user. Admins need to set the ID for the STORAGE-SYSTEM system user in this config option which is then used to reference the user. Any reasonable long string is possible, preferably this would be an UUIDv4 format." introductionVersion:"1.0.0"`
	SystemUserAPIKey string `yaml:"system_user_api_key" env:"OC_SYSTEM_USER_API_KEY" desc:"API key for the STORAGE-SYSTEM system user." introductionVersion:"1.0.0"`

	SkipUserGroupsInToken bool `yaml:"skip_user_groups_in_token" env:"STORAGE_SYSTEM_SKIP_USER_GROUPS_IN_TOKEN" desc:"Disables the loading of user's group memberships from the reva access token." introductionVersion:"1.0.0"`

	FileMetadataCache Cache   `yaml:"cache"`
	Driver            string  `yaml:"driver" env:"STORAGE_SYSTEM_DRIVER" desc:"The driver which should be used by the service. The only supported driver is 'decomposed'. For backwards compatibility reasons it's also possible to use the 'ocis' driver and configure it using the 'decomposed' options. " introductionVersion:"1.0.0"`
	Drivers           Drivers `yaml:"drivers"`
	DataServerURL     string  `yaml:"data_server_url" env:"STORAGE_SYSTEM_DATA_SERVER_URL" desc:"URL of the data server, needs to be reachable by other services using this service." introductionVersion:"1.0.0"`

	Context context.Context `yaml:"-"`
}

// Service holds Service config
type Service struct {
	Name string `yaml:"-"`
}

// Debug holds Debug config
type Debug struct {
	Addr   string `yaml:"addr" env:"STORAGE_SYSTEM_DEBUG_ADDR" desc:"Bind address of the debug server, where metrics, health, config and debug endpoints will be exposed." introductionVersion:"1.0.0"`
	Token  string `yaml:"token" env:"STORAGE_SYSTEM_DEBUG_TOKEN" desc:"Token to secure the metrics endpoint" introductionVersion:"1.0.0"`
	Pprof  bool   `yaml:"pprof" env:"STORAGE_SYSTEM_DEBUG_PPROF" desc:"Enables pprof, which can be used for profiling" introductionVersion:"1.0.0"`
	Zpages bool   `yaml:"zpages" env:"STORAGE_SYSTEM_DEBUG_ZPAGES" desc:"Enables zpages, which can be used for collecting and viewing in-memory traces." introductionVersion:"1.0.0"`
}

// GRPCConfig holds GRPCConfig config
type GRPCConfig struct {
	Addr      string                 `yaml:"addr" env:"STORAGE_SYSTEM_GRPC_ADDR" desc:"The bind address of the GRPC service." introductionVersion:"1.0.0"`
	TLS       *shared.GRPCServiceTLS `yaml:"tls"`
	Namespace string                 `yaml:"-"`
	Protocol  string                 `yaml:"protocol" env:"OC_GRPC_PROTOCOL;STORAGE_SYSTEM_GRPC_PROTOCOL" desc:"The transport protocol of the GPRC service." introductionVersion:"1.0.0"`
}

// HTTPConfig holds HTTPConfig config
type HTTPConfig struct {
	Addr      string `yaml:"addr" env:"STORAGE_SYSTEM_HTTP_ADDR" desc:"The bind address of the HTTP service." introductionVersion:"1.0.0"`
	Namespace string `yaml:"-"`
	Protocol  string `yaml:"protocol" env:"STORAGE_SYSTEM_HTTP_PROTOCOL" desc:"The transport protocol of the HTTP service." introductionVersion:"1.0.0"`
}

// Drivers holds Drivers config
type Drivers struct {
	Decomposed DecomposedDriver `yaml:"decomposed"`
}

// DecomposedDriver holds the decomposed Driver config
type DecomposedDriver struct {
	// Root is the absolute path to the location of the data
	Root string `yaml:"root" env:"STORAGE_SYSTEM_OC_ROOT" desc:"Path for the directory where the STORAGE-SYSTEM service stores it's persistent data. If not defined, the root directory derives from $OC_BASE_DATA_PATH/storage." introductionVersion:"1.0.0"`

	MaxAcquireLockCycles    int `yaml:"max_acquire_lock_cycles" env:"STORAGE_SYSTEM_OC_MAX_ACQUIRE_LOCK_CYCLES" desc:"When trying to lock files, OpenCloud will try this amount of times to acquire the lock before failing. After each try it will wait for an increasing amount of time. Values of 0 or below will be ignored and the default value of 20 will be used." introductionVersion:"1.0.0"`
	LockCycleDurationFactor int `yaml:"lock_cycle_duration_factor" env:"STORAGE_SYSTEM_OC_LOCK_CYCLE_DURATION_FACTOR" desc:"When trying to lock files, OpenCloud will multiply the cycle with this factor and use it as a millisecond timeout. Values of 0 or below will be ignored and the default value of 30 will be used." introductionVersion:"1.0.0"`
}

// Cache holds cache config
type Cache struct {
	Store              string        `yaml:"store" env:"OC_CACHE_STORE;STORAGE_SYSTEM_CACHE_STORE" desc:"The type of the cache store. Supported values are: 'memory', 'redis-sentinel', 'nats-js-kv', 'noop'. See the text description for details." introductionVersion:"1.0.0"`
	Nodes              []string      `yaml:"nodes" env:"OC_CACHE_STORE_NODES;STORAGE_SYSTEM_CACHE_STORE_NODES" desc:"A list of nodes to access the configured store. This has no effect when 'memory' store is configured. Note that the behaviour how nodes are used is dependent on the library of the configured store. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	Database           string        `yaml:"database" env:"OC_CACHE_DATABASE" desc:"The database name the configured store should use." introductionVersion:"1.0.0"`
	TTL                time.Duration `yaml:"ttl" env:"OC_CACHE_TTL;STORAGE_SYSTEM_CACHE_TTL" desc:"Default time to live for user info in the user info cache. Only applied when access tokens has no expiration. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	DisablePersistence bool          `yaml:"disable_persistence" env:"OC_CACHE_DISABLE_PERSISTENCE;STORAGE_SYSTEM_CACHE_DISABLE_PERSISTENCE" desc:"Disables persistence of the cache. Only applies when store type 'nats-js-kv' is configured. Defaults to false." introductionVersion:"1.0.0"`
	AuthUsername       string        `yaml:"auth_username" env:"OC_CACHE_AUTH_USERNAME;STORAGE_SYSTEM_CACHE_AUTH_USERNAME" desc:"Username for the configured store. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"1.0.0"`
	AuthPassword       string        `yaml:"auth_password" env:"OC_CACHE_AUTH_PASSWORD;STORAGE_SYSTEM_CACHE_AUTH_PASSWORD" desc:"Password for the configured store. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"1.0.0"`
}
