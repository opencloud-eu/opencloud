package config

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service
	Service Service         `yaml:"-"`
	Log     *Log            `yaml:"log"`
	Debug   Debug           `yaml:"debug"`

	GRPC GRPCConfig `yaml:"grpc"`

	TokenManager *TokenManager `yaml:"token_manager"`
	Reva         *shared.Reva  `yaml:"reva"`
	Events       Events        `yaml:"events"`

	SkipUserGroupsInToken bool `yaml:"skip_user_groups_in_token" env:"SHARING_SKIP_USER_GROUPS_IN_TOKEN" desc:"Disables the loading of user's group memberships from the reva access token." introductionVersion:"1.0.0"`

	UserSharingDriver              string               `yaml:"user_sharing_driver" env:"SHARING_USER_DRIVER" desc:"Driver to be used to persist shares. Supported values are 'jsoncs3', 'json', 'cs3' (deprecated) and 'owncloudsql'." introductionVersion:"1.0.0"`
	UserSharingDrivers             UserSharingDrivers   `yaml:"user_sharing_drivers"`
	PublicSharingDriver            string               `yaml:"public_sharing_driver" env:"SHARING_PUBLIC_DRIVER" desc:"Driver to be used to persist public shares. Supported values are 'jsoncs3', 'json' and 'cs3' (deprecated)." introductionVersion:"1.0.0"`
	PublicSharingDrivers           PublicSharingDrivers `yaml:"public_sharing_drivers"`
	WriteableShareMustHavePassword bool                 `yaml:"public_sharing_writeableshare_must_have_password" env:"OC_SHARING_PUBLIC_WRITEABLE_SHARE_MUST_HAVE_PASSWORD;SHARING_PUBLIC_WRITEABLE_SHARE_MUST_HAVE_PASSWORD" desc:"Set this to true if you want to enforce passwords on Uploader, Editor or Contributor shares. If not using the global OC_SHARING_PUBLIC_WRITEABLE_SHARE_MUST_HAVE_PASSWORD, you must define the FRONTEND_OCS_PUBLIC_WRITEABLE_SHARE_MUST_HAVE_PASSWORD (deprecated) in the frontend service." introductionVersion:"1.0.0"`
	PublicShareMustHavePassword    bool                 `yaml:"public_sharing_share_must_have_password" env:"OC_SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD;SHARING_PUBLIC_SHARE_MUST_HAVE_PASSWORD" desc:"Set this to true if you want to enforce passwords on all public shares." introductionVersion:"1.0.0"`
	EnableExpiredSharesCleanup     bool                 `yaml:"enable_expired_shares_cleanup"`
	PasswordPolicy                 PasswordPolicy       `yaml:"password_policy"`

	Context context.Context `yaml:"-"`
}

type Log struct {
	Level  string `yaml:"level" env:"OC_LOG_LEVEL;SHARING_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"1.0.0"`
	Pretty bool   `yaml:"pretty" env:"OC_LOG_PRETTY;SHARING_LOG_PRETTY" desc:"Activates pretty log output." introductionVersion:"1.0.0"`
	Color  bool   `yaml:"color" env:"OC_LOG_COLOR;SHARING_LOG_COLOR" desc:"Activates colorized log output." introductionVersion:"1.0.0"`
	File   string `yaml:"file" env:"OC_LOG_FILE;SHARING_LOG_FILE" desc:"The path to the log file. Activates logging to this file if set." introductionVersion:"1.0.0"`
}

type Service struct {
	Name string `yaml:"-"`
}

type Debug struct {
	Addr   string `yaml:"addr" env:"SHARING_DEBUG_ADDR" desc:"Bind address of the debug server, where metrics, health, config and debug endpoints will be exposed." introductionVersion:"1.0.0"`
	Token  string `yaml:"token" env:"SHARING_DEBUG_TOKEN" desc:"Token to secure the metrics endpoint." introductionVersion:"1.0.0"`
	Pprof  bool   `yaml:"pprof" env:"SHARING_DEBUG_PPROF" desc:"Enables pprof, which can be used for profiling." introductionVersion:"1.0.0"`
	Zpages bool   `yaml:"zpages" env:"SHARING_DEBUG_ZPAGES" desc:"Enables zpages, which can be used for collecting and viewing in-memory traces." introductionVersion:"1.0.0"`
}

type GRPCConfig struct {
	Addr      string                 `yaml:"addr" env:"SHARING_GRPC_ADDR" desc:"The bind address of the GRPC service." introductionVersion:"1.0.0"`
	TLS       *shared.GRPCServiceTLS `yaml:"tls"`
	Namespace string                 `yaml:"-"`
	Protocol  string                 `yaml:"protocol" env:"OC_GRPC_PROTOCOL;SHARING_GRPC_PROTOCOL" desc:"The transport protocol of the GRPC service." introductionVersion:"1.0.0"`
}

type UserSharingDrivers struct {
	JSONCS3     UserSharingJSONCS3Driver     `yaml:"jsoncs3"`
	JSON        UserSharingJSONDriver        `yaml:"json"`
	CS3         UserSharingCS3Driver         `yaml:"cs3"`
	OwnCloudSQL UserSharingOwnCloudSQLDriver `yaml:"owncloudsql"`

	SQL UserSharingSQLDriver `yaml:"sql,omitempty"` // not supported by the OpenCloud product, therefore not part of docs
}

type UserSharingJSONDriver struct {
	File string `yaml:"file" env:"SHARING_USER_JSON_FILE" desc:"Path to the JSON file where shares will be persisted. If not defined, the root directory derives from $OC_BASE_DATA_PATH/storage." introductionVersion:"1.0.0"`
}

type UserSharingSQLDriver struct {
	DBUsername                 string `yaml:"db_username"`
	DBPassword                 string `yaml:"db_password"`
	DBHost                     string `yaml:"db_host"`
	DBPort                     int    `yaml:"db_port"`
	DBName                     string `yaml:"db_name"`
	PasswordHashCost           int    `yaml:"password_hash_cost"`
	EnableExpiredSharesCleanup bool   `yaml:"enable_expired_shares_cleanup"`
	JanitorRunInterval         int    `yaml:"janitor_run_interval"`
	UserStorageMountID         string `yaml:"user_storage_mount_id"`
}

type UserSharingOwnCloudSQLDriver struct {
	DBUsername         string `yaml:"db_username" env:"SHARING_USER_OWNCLOUDSQL_DB_USERNAME" desc:"Username for the database." introductionVersion:"1.0.0"`
	DBPassword         string `yaml:"db_password" env:"SHARING_USER_OWNCLOUDSQL_DB_PASSWORD" desc:"Password for the database." introductionVersion:"1.0.0"`
	DBHost             string `yaml:"db_host" env:"SHARING_USER_OWNCLOUDSQL_DB_HOST" desc:"Hostname or IP of the database server." introductionVersion:"1.0.0"`
	DBPort             int    `yaml:"db_port" env:"SHARING_USER_OWNCLOUDSQL_DB_PORT" desc:"Port that the database server is listening on." introductionVersion:"1.0.0"`
	DBName             string `yaml:"db_name" env:"SHARING_USER_OWNCLOUDSQL_DB_NAME" desc:"Name of the database to be used." introductionVersion:"1.0.0"`
	UserStorageMountID string `yaml:"user_storage_mount_id" env:"SHARING_USER_OWNCLOUDSQL_USER_STORAGE_MOUNT_ID" desc:"Mount ID of the ownCloudSQL users storage for mapping ownCloud 10 shares." introductionVersion:"1.0.0"`
}

type UserSharingCS3Driver struct {
	ProviderAddr     string `yaml:"provider_addr" env:"SHARING_USER_CS3_PROVIDER_ADDR" desc:"GRPC address of the STORAGE-SYSTEM service." introductionVersion:"1.0.0"`
	SystemUserID     string `yaml:"system_user_id" env:"OC_SYSTEM_USER_ID;SHARING_USER_CS3_SYSTEM_USER_ID" desc:"ID of the OpenCloud STORAGE-SYSTEM system user. Admins need to set the ID for the STORAGE-SYSTEM system user in this config option which is then used to reference the user. Any reasonable long string is possible, preferably this would be an UUIDv4 format." introductionVersion:"1.0.0"`
	SystemUserIDP    string `yaml:"system_user_idp" env:"OC_SYSTEM_USER_IDP;SHARING_USER_CS3_SYSTEM_USER_IDP" desc:"IDP of the OpenCloud STORAGE-SYSTEM system user." introductionVersion:"1.0.0"`
	SystemUserAPIKey string `yaml:"system_user_api_key" env:"OC_SYSTEM_USER_API_KEY;SHARING_USER_CS3_SYSTEM_USER_API_KEY" desc:"API key for the STORAGE-SYSTEM system user." introductionVersion:"1.0.0"`
}

// UserSharingJSONCS3Driver holds the jsoncs3 driver config
type UserSharingJSONCS3Driver struct {
	ProviderAddr     string `yaml:"provider_addr" env:"SHARING_USER_JSONCS3_PROVIDER_ADDR" desc:"GRPC address of the STORAGE-SYSTEM service." introductionVersion:"1.0.0"`
	SystemUserID     string `yaml:"system_user_id" env:"OC_SYSTEM_USER_ID;SHARING_USER_JSONCS3_SYSTEM_USER_ID" desc:"ID of the OpenCloud STORAGE-SYSTEM system user. Admins need to set the ID for the STORAGE-SYSTEM system user in this config option which is then used to reference the user. Any reasonable long string is possible, preferably this would be an UUIDv4 format." introductionVersion:"1.0.0"`
	SystemUserIDP    string `yaml:"system_user_idp" env:"OC_SYSTEM_USER_IDP;SHARING_USER_JSONCS3_SYSTEM_USER_IDP" desc:"IDP of the OpenCloud STORAGE-SYSTEM system user." introductionVersion:"1.0.0"`
	SystemUserAPIKey string `yaml:"system_user_api_key" env:"OC_SYSTEM_USER_API_KEY;SHARING_USER_JSONCS3_SYSTEM_USER_API_KEY" desc:"API key for the STORAGE-SYSTEM system user." introductionVersion:"1.0.0"`
	CacheTTL         int    `yaml:"cache_ttl" env:"SHARING_USER_JSONCS3_CACHE_TTL" desc:"TTL for the internal caches in seconds." introductionVersion:"1.0.0"`
	MaxConcurrency   int    `yaml:"max_concurrency" env:"OC_MAX_CONCURRENCY;SHARING_USER_JSONCS3_MAX_CONCURRENCY" desc:"Maximum number of concurrent go-routines. Higher values can potentially get work done faster but will also cause more load on the system. Values of 0 or below will be ignored and the default value will be used." introductionVersion:"1.0.0"`
}
type PublicSharingDrivers struct {
	JSON    PublicSharingJSONDriver    `yaml:"json"`
	JSONCS3 PublicSharingJSONCS3Driver `yaml:"jsoncs3"`
	CS3     PublicSharingCS3Driver     `yaml:"cs3"`

	SQL PublicSharingSQLDriver `yaml:"sql,omitempty"` // not supported by the OpenCloud product, therefore not part of docs
}

type PublicSharingJSONDriver struct {
	File string `yaml:"file" env:"SHARING_PUBLIC_JSON_FILE" desc:"Path to the JSON file where public share meta-data will be stored. This JSON file contains the information about public shares that have been created. If not defined, the root directory derives from $OC_BASE_DATA_PATH/storage." introductionVersion:"1.0.0"`
}

type PublicSharingSQLDriver struct {
	DBUsername                 string `yaml:"db_username"`
	DBPassword                 string `yaml:"db_password"`
	DBHost                     string `yaml:"db_host"`
	DBPort                     int    `yaml:"db_port"`
	DBName                     string `yaml:"db_name"`
	PasswordHashCost           int    `yaml:"password_hash_cost"`
	EnableExpiredSharesCleanup bool   `yaml:"enable_expired_shares_cleanup"`
	JanitorRunInterval         int    `yaml:"janitor_run_interval"`
	UserStorageMountID         string `yaml:"user_storage_mount_id"`
}

type PublicSharingCS3Driver struct {
	ProviderAddr     string `yaml:"provider_addr" env:"SHARING_PUBLIC_CS3_PROVIDER_ADDR" desc:"GRPC address of the STORAGE-SYSTEM service." introductionVersion:"1.0.0"`
	SystemUserID     string `yaml:"system_user_id" env:"OC_SYSTEM_USER_ID;SHARING_PUBLIC_CS3_SYSTEM_USER_ID" desc:"ID of the OpenCloud STORAGE-SYSTEM system user. Admins need to set the ID for the STORAGE-SYSTEM system user in this config option which is then used to reference the user. Any reasonable long string is possible, preferably this would be an UUIDv4 format." introductionVersion:"1.0.0"`
	SystemUserIDP    string `yaml:"system_user_idp" env:"OC_SYSTEM_USER_IDP;SHARING_PUBLIC_CS3_SYSTEM_USER_IDP" desc:"IDP of the OpenCloud STORAGE-SYSTEM system user." introductionVersion:"1.0.0"`
	SystemUserAPIKey string `yaml:"system_user_api_key" env:"OC_SYSTEM_USER_API_KEY;SHARING_PUBLIC_CS3_SYSTEM_USER_API_KEY" desc:"API key for the STORAGE-SYSTEM system user." introductionVersion:"1.0.0"`
}

// PublicSharingJSONCS3Driver holds the jsoncs3 driver config
type PublicSharingJSONCS3Driver struct {
	ProviderAddr     string `yaml:"provider_addr" env:"SHARING_PUBLIC_JSONCS3_PROVIDER_ADDR" desc:"GRPC address of the STORAGE-SYSTEM service." introductionVersion:"1.0.0"`
	SystemUserID     string `yaml:"system_user_id" env:"OC_SYSTEM_USER_ID;SHARING_PUBLIC_JSONCS3_SYSTEM_USER_ID" desc:"ID of the OpenCloud STORAGE-SYSTEM system user. Admins need to set the ID for the STORAGE-SYSTEM system user in this config option which is then used to reference the user. Any reasonable long string is possible, preferably this would be an UUIDv4 format." introductionVersion:"1.0.0"`
	SystemUserIDP    string `yaml:"system_user_idp" env:"OC_SYSTEM_USER_IDP;SHARING_PUBLIC_JSONCS3_SYSTEM_USER_IDP" desc:"IDP of the OpenCloud STORAGE-SYSTEM system user." introductionVersion:"1.0.0"`
	SystemUserAPIKey string `yaml:"system_user_api_key" env:"OC_SYSTEM_USER_API_KEY;SHARING_PUBLIC_JSONCS3_SYSTEM_USER_API_KEY" desc:"API key for the STORAGE-SYSTEM system user." introductionVersion:"1.0.0"`
}

type Events struct {
	Addr              string `yaml:"endpoint" env:"OC_EVENTS_ENDPOINT;SHARING_EVENTS_ENDPOINT" desc:"The address of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture." introductionVersion:"1.0.0"`
	ClusterID         string `yaml:"cluster" env:"OC_EVENTS_CLUSTER;SHARING_EVENTS_CLUSTER" desc:"The clusterID of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture. Mandatory when using NATS as event system." introductionVersion:"1.0.0"`
	TLSInsecure       bool   `yaml:"tls_insecure" env:"OC_INSECURE;OC_EVENTS_TLS_INSECURE;SHARING_EVENTS_TLS_INSECURE" desc:"Whether to verify the server TLS certificates." introductionVersion:"1.0.0"`
	TLSRootCaCertPath string `yaml:"tls_root_ca_cert_path" env:"OC_EVENTS_TLS_ROOT_CA_CERTIFICATE;SHARING_EVENTS_TLS_ROOT_CA_CERTIFICATE" desc:"The root CA certificate used to validate the server's TLS certificate. If provided SHARING_EVENTS_TLS_INSECURE will be seen as false." introductionVersion:"1.0.0"`
	EnableTLS         bool   `yaml:"enable_tls" env:"OC_EVENTS_ENABLE_TLS;SHARING_EVENTS_ENABLE_TLS" desc:"Enable TLS for the connection to the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
	AuthUsername      string `yaml:"auth_username" env:"OC_EVENTS_AUTH_USERNAME;SHARING_EVENTS_AUTH_USERNAME" desc:"Username for the events broker." introductionVersion:"1.0.0"`
	AuthPassword      string `yaml:"auth_password" env:"OC_EVENTS_AUTH_PASSWORD;SHARING_EVENTS_AUTH_PASSWORD" desc:"Password for the events broker." introductionVersion:"1.0.0"`
}

// PasswordPolicy configures reva password policy
type PasswordPolicy struct {
	Disabled               bool   `yaml:"disabled,omitempty" env:"OC_PASSWORD_POLICY_DISABLED;SHARING_PASSWORD_POLICY_DISABLED" desc:"Disable the password policy. Defaults to false if not set." introductionVersion:"1.0.0"`
	MinCharacters          int    `yaml:"min_characters,omitempty" env:"OC_PASSWORD_POLICY_MIN_CHARACTERS;SHARING_PASSWORD_POLICY_MIN_CHARACTERS" desc:"Define the minimum password length. Defaults to 8 if not set." introductionVersion:"1.0.0"`
	MinLowerCaseCharacters int    `yaml:"min_lowercase_characters" env:"OC_PASSWORD_POLICY_MIN_LOWERCASE_CHARACTERS;SHARING_PASSWORD_POLICY_MIN_LOWERCASE_CHARACTERS" desc:"Define the minimum number of uppercase letters. Defaults to 1 if not set." introductionVersion:"1.0.0"`
	MinUpperCaseCharacters int    `yaml:"min_uppercase_characters" env:"OC_PASSWORD_POLICY_MIN_UPPERCASE_CHARACTERS;SHARING_PASSWORD_POLICY_MIN_UPPERCASE_CHARACTERS" desc:"Define the minimum number of lowercase letters. Defaults to 1 if not set." introductionVersion:"1.0.0"`
	MinDigits              int    `yaml:"min_digits" env:"OC_PASSWORD_POLICY_MIN_DIGITS;SHARING_PASSWORD_POLICY_MIN_DIGITS" desc:"Define the minimum number of digits. Defaults to 1 if not set." introductionVersion:"1.0.0"`
	MinSpecialCharacters   int    `yaml:"min_special_characters" env:"OC_PASSWORD_POLICY_MIN_SPECIAL_CHARACTERS;SHARING_PASSWORD_POLICY_MIN_SPECIAL_CHARACTERS" desc:"Define the minimum number of characters from the special characters list to be present. Defaults to 1 if not set." introductionVersion:"1.0.0"`
	BannedPasswordsList    string `yaml:"banned_passwords_list" env:"OC_PASSWORD_POLICY_BANNED_PASSWORDS_LIST;SHARING_PASSWORD_POLICY_BANNED_PASSWORDS_LIST" desc:"Path to the 'banned passwords list' file. This only impacts public link password validation. See the documentation for more details." introductionVersion:"1.0.0"`
}
