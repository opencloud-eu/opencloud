package config

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// Config defines the root config structure
type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service
	Service Service         `yaml:"-"`
	Tracing *Tracing        `yaml:"tracing"`
	Log     *Log            `yaml:"log"`
	Debug   Debug           `yaml:"debug"`

	GRPC GRPCConfig `yaml:"grpc"`
	HTTP HTTP       `yaml:"http"`

	GRPCClientTLS *shared.GRPCClientTLS `yaml:"grpc_client_tls"`

	TokenManager *TokenManager `yaml:"token_manager"`
	Reva         *shared.Reva  `yaml:"reva"`

	SkipUserGroupsInToken bool `yaml:"skip_user_groups_in_token" env:"AUTH_APP_SKIP_USER_GROUPS_IN_TOKEN" desc:"Disables the encoding of the user's group memberships in the access token. This reduces the token size, especially when users are members of a large number of groups." introductionVersion:"1.0.0"`

	MachineAuthAPIKey string `yaml:"machine_auth_api_key" env:"OC_MACHINE_AUTH_API_KEY;AUTH_APP_MACHINE_AUTH_API_KEY" desc:"The machine auth API key used to validate internal requests necessary to access resources from other services." introductionVersion:"1.0.0"`

	AllowImpersonation bool `yaml:"allow_impersonation" env:"AUTH_APP_ENABLE_IMPERSONATION" desc:"Allows admins to create app tokens for other users. Used for migration. Do NOT use in productive deployments." introductionVersion:"1.0.0"`

	StorageDriver  string         `yaml:"storage_driver" env:"AUTH_APP_STORAGE_DRIVER" desc:"Driver to be used to persist the app tokes . Supported values are 'jsoncs3', 'json'." introductionVersion:"4.0.0"`
	StorageDrivers StorageDrivers `yaml:"storage_drivers"`

	Context context.Context `yaml:"-"`
}

type StorageDrivers struct {
	JSONCS3 JSONCS3Driver `yaml:"jsoncs3"`
}

type JSONCS3Driver struct {
	ProviderAddr             string                   `yaml:"provider_addr" env:"AUTH_APP_JSONCS3_PROVIDER_ADDR" desc:"GRPC address of the STORAGE-SYSTEM service." introductionVersion:"4.0.0"`
	SystemUserID             string                   `yaml:"system_user_id" env:"OC_SYSTEM_USER_ID;AUTH_APP_JSONCS3_SYSTEM_USER_ID" desc:"ID of the OpenCloud STORAGE-SYSTEM system user. Admins need to set the ID for the STORAGE-SYSTEM system user in this config option which is then used to reference the user. Any reasonable long string is possible, preferably this would be an UUIDv4 format." introductionVersion:"4.0.0"`
	SystemUserIDP            string                   `yaml:"system_user_idp" env:"OC_SYSTEM_USER_IDP;AUTH_APP_JSONCS3_SYSTEM_USER_IDP" desc:"IDP of the OpenCloud STORAGE-SYSTEM system user." introductionVersion:"4.0.0"`
	SystemUserAPIKey         string                   `yaml:"system_user_api_key" env:"OC_SYSTEM_USER_API_KEY;AUTH_APP_JSONCS3_SYSTEM_USER_API_KEY" desc:"API key for the STORAGE-SYSTEM system user." introductionVersion:"4.0.0"`
	PasswordGenerator        string                   `yaml:"password_generator" env:"AUTH_APP_JSONCS3_PASSWORD_GENERATOR" desc:"The password generator that should be used for generating app tokens. Supported values are: 'diceware' and 'random'." introductionVersion:"4.0.0"`
	PasswordGeneratorOptions PasswordGeneratorOptions `yaml:"password_generator_options"`
}

type PasswordGeneratorOptions struct {
	DicewareOptions DicewareOptions `yaml:"diceware"`
	RandPWOpts      RandPWOpts      `yaml:"randon"`
}

// DicewareOptions defines the config options for the "diceware" password generator
type DicewareOptions struct {
	NumberOfWords int `yaml:"number_of_words" env:"AUTH_APP_JSONCS3_DICEWARE_NUMBER_OF_WORDS" desc:"The number of words the generated passphrase will have." introductionVersion:"4.0.0"`
}

// RandPWOpts defines the config options for the "random" password generator
type RandPWOpts struct {
	PasswordLength int `yaml:"password_length" env:"AUTH_APP_JSONCS3_RANDOM_PASSWORD_LENGTH" desc:"The number of charactors the generated passwords will have." introductionVersion:"4.0.0"`
}

// Log defines the loging configuration
type Log struct {
	Level  string `yaml:"level" env:"OC_LOG_LEVEL;AUTH_APP_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"1.0.0"`
	Pretty bool   `yaml:"pretty" env:"OC_LOG_PRETTY;AUTH_APP_LOG_PRETTY" desc:"Activates pretty log output." introductionVersion:"1.0.0"`
	Color  bool   `yaml:"color" env:"OC_LOG_COLOR;AUTH_APP_LOG_COLOR" desc:"Activates colorized log output." introductionVersion:"1.0.0"`
	File   string `yaml:"file" env:"OC_LOG_FILE;AUTH_APP_LOG_FILE" desc:"The path to the log file. Activates logging to this file if set." introductionVersion:"1.0.0"`
}

// Service defines the service configuration
type Service struct {
	Name string `yaml:"-"`
}

// Debug defines the debug configuration
type Debug struct {
	Addr   string `yaml:"addr" env:"AUTH_APP_DEBUG_ADDR" desc:"Bind address of the debug server, where metrics, health, config and debug endpoints will be exposed." introductionVersion:"1.0.0"`
	Token  string `yaml:"token" env:"AUTH_APP_DEBUG_TOKEN" desc:"Token to secure the metrics endpoint." introductionVersion:"1.0.0"`
	Pprof  bool   `yaml:"pprof" env:"AUTH_APP_DEBUG_PPROF" desc:"Enables pprof, which can be used for profiling." introductionVersion:"1.0.0"`
	Zpages bool   `yaml:"zpages" env:"AUTH_APP_DEBUG_ZPAGES" desc:"Enables zpages, which can  be used for collecting and viewing traces in-memory." introductionVersion:"1.0.0"`
}

// GRPCConfig defines the GRPC configuration
type GRPCConfig struct {
	Addr      string                 `yaml:"addr" env:"AUTH_APP_GRPC_ADDR" desc:"The bind address of the GRPC service." introductionVersion:"1.0.0"`
	TLS       *shared.GRPCServiceTLS `yaml:"tls"`
	Namespace string                 `yaml:"-"`
	Protocol  string                 `yaml:"protocol" env:"OC_GRPC_PROTOCOL;AUTH_APP_GRPC_PROTOCOL" desc:"The transport protocol of the GRPC service." introductionVersion:"1.0.0"`
}

// HTTP defines the available http configuration.
type HTTP struct {
	Addr      string                `yaml:"addr" env:"AUTH_APP_HTTP_ADDR" desc:"The bind address of the HTTP service." introductionVersion:"1.0.0"`
	Namespace string                `yaml:"-"`
	Root      string                `yaml:"root" env:"AUTH_APP_HTTP_ROOT" desc:"Subdirectory that serves as the root for this HTTP service." introductionVersion:"1.0.0"`
	CORS      CORS                  `yaml:"cors"`
	TLS       shared.HTTPServiceTLS `yaml:"tls"`
}

// CORS defines the available cors configuration.
type CORS struct {
	AllowedOrigins   []string `yaml:"allow_origins" env:"OC_CORS_ALLOW_ORIGINS;AUTH_APP_CORS_ALLOW_ORIGINS" desc:"A list of allowed CORS origins. See following chapter for more details: *Access-Control-Allow-Origin* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Origin. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	AllowedMethods   []string `yaml:"allow_methods" env:"OC_CORS_ALLOW_METHODS;AUTH_APP_CORS_ALLOW_METHODS" desc:"A list of allowed CORS methods. See following chapter for more details: *Access-Control-Request-Method* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Request-Method. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	AllowedHeaders   []string `yaml:"allow_headers" env:"OC_CORS_ALLOW_HEADERS;AUTH_APP_CORS_ALLOW_HEADERS" desc:"A list of allowed CORS headers. See following chapter for more details: *Access-Control-Request-Headers* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Request-Headers. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	AllowCredentials bool     `yaml:"allow_credentials" env:"OC_CORS_ALLOW_CREDENTIALS;AUTH_APP_CORS_ALLOW_CREDENTIALS" desc:"Allow credentials for CORS.See following chapter for more details: *Access-Control-Allow-Credentials* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Credentials." introductionVersion:"1.0.0"`
}
