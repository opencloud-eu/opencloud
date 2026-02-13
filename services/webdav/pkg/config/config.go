package config

import (
	"context"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	"go-micro.dev/v4/client"
)

// Config combines all available configuration parts.
type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service

	Service Service `yaml:"-"`

	LogLevel string `yaml:"loglevel" env:"OC_LOG_LEVEL;WEBDAV_LOG_LEVEL" desc:"The log level. Valid values are: 'panic', 'fatal', 'error', 'warn', 'info', 'debug', 'trace'." introductionVersion:"1.0.0"`
	Debug    Debug  `yaml:"debug"`

	GRPCClientTLS *shared.GRPCClientTLS `yaml:"grpc_client_tls"`
	GrpcClient    client.Client         `yaml:"-"`

	HTTP HTTP `yaml:"http"`

	DisablePreviews    bool            `yaml:"disablePreviews" env:"OC_DISABLE_PREVIEWS;WEBDAV_DISABLE_PREVIEWS" desc:"Set this option to 'true' to disable rendering of thumbnails triggered via webdav access. Note that when disabled, all access to preview related webdav paths will return a 404." introductionVersion:"1.0.0"`
	OpenCloudPublicURL string          `yaml:"opencloud_public_url" env:"OC_URL;OC_PUBLIC_URL" desc:"URL, where OpenCloud is reachable for users." introductionVersion:"1.0.0"`
	WebdavNamespace    string          `yaml:"webdav_namespace" env:"WEBDAV_WEBDAV_NAMESPACE" desc:"CS3 path layout to use when forwarding /webdav requests" introductionVersion:"1.0.0"`
	RevaGateway        string          `yaml:"reva_gateway" env:"OC_REVA_GATEWAY" desc:"CS3 gateway used to look up user metadata" introductionVersion:"1.0.0"`
	Context            context.Context `yaml:"-"`

	OCDav          OCDav          `yaml:"ocdav"`
	FavoritesStore FavoritesStore `yaml:"favorites_store"`
}

type OCDav struct {
	Prefix string `yaml:"prefix" env:"OCDAV_HTTP_PREFIX;FRONTENT_OCDAV_HTTP_PREFIX" desc:"A URL path prefix for the handler." introductionVersion:"1.0.0"`

	SkipUserGroupsInToken bool `yaml:"skip_user_groups_in_token" env:"OCDAV_SKIP_USER_GROUPS_IN_TOKEN;FRONTENT_OCDAV_SKIP_USER_GROUPS_IN_TOKEN" desc:"Disables the loading of user's group memberships from the reva access token." introductionVersion:"1.0.0"`

	WebdavNamespace string `yaml:"webdav_namespace" env:"OCDAV_WEBDAV_NAMESPACE;FRONTENT_OCDAV_WEBDAV_NAMESPACE" desc:"Jail requests to /dav/webdav into this CS3 namespace. Supports template layouting with CS3 User properties." introductionVersion:"1.0.0"`
	FilesNamespace  string `yaml:"files_namespace" env:"OCDAV_FILES_NAMESPACE;FRONTENT_OCDAV_FILES_NAMESPACE" desc:"Jail requests to /dav/files/{username} into this CS3 namespace. Supports template layouting with CS3 User properties." introductionVersion:"1.0.0"`
	SharesNamespace string `yaml:"shares_namespace" env:"OCDAV_SHARES_NAMESPACE;FRONTENT_OCDAV_SHARES_NAMESPACE" desc:"The human readable path for the share jail. Relative to a users personal space root. Upcased intentionally." introductionVersion:"1.0.0"`
	OCMNamespace    string `yaml:"ocm_namespace" env:"OCDAV_OCM_NAMESPACE;FRONTENT_OCDAV_OCM_NAMESPACE" desc:"The human readable path prefix for the ocm shares." introductionVersion:"1.0.0"`
	// PublicURL used to redirect /s/{token} URLs to
	PublicURL string `yaml:"public_url" env:"OC_URL;OCDAV_PUBLIC_URL;FRONTENT_OCDAV_PUBLIC_URL" desc:"URL where OpenCloud is reachable for users." introductionVersion:"1.0.0"`

	// Insecure certificates allowed when making requests to the gateway
	Insecure      bool `yaml:"insecure" env:"OC_INSECURE;OCDAV_INSECURE;FRONTENT_OCDAV_INSECURE" desc:"Allow insecure connections to the GATEWAY service." introductionVersion:"1.0.0"`
	EnableHTTPTPC bool `yaml:"enable_http_tpc" env:"OCDAV_ENABLE_HTTP_TPC;FRONTENT_OCDAV_ENABLE_HTTP_TPC" desc:"Enable HTTP / WebDAV Third-Party-Copy support." introductionVersion:"%%NEXT%%"`
	// Timeout in seconds when making requests to the gateway
	Timeout int64 `yaml:"gateway_request_timeout" env:"OCDAV_GATEWAY_REQUEST_TIME;FRONTENT_OUTOCDAV_GATEWAY_REQUEST_TIMEOUT" desc:"Request timeout in seconds for requests from the oCDAV service to the GATEWAY service." introductionVersion:"1.0.0"`

	MachineAuthAPIKey string `yaml:"machine_auth_api_key" env:"OC_MACHINE_AUTH_API_KEY;OCDAV_MACHINE_AUTH_API_KEY;FRONTENT_OCDAV_MACHINE_AUTH_API_KEY" desc:"Machine auth API key used to validate internal requests necessary for the access to resources from other services." introductionVersion:"1.0.0"`

	AllowPropfindDepthInfinity bool `yaml:"allow_propfind_depth_infinity" env:"OCDAV_ALLOW_PROPFIND_DEPTH_INFINITY;FRONTENT_OCDAV_ALLOW_PROPFIND_DEPTH_INFINITY" desc:"Allow the use of depth infinity in PROPFINDS. When enabled, a propfind will traverse through all subfolders. If many subfolders are expected, depth infinity can cause heavy server load and/or delayed response times." introductionVersion:"1.0.0"`

	NameValidation NameValidation `yaml:"name_validation"`
}

type NameValidation struct {
	InvalidChars []string `yaml:"invalid_chars" env:"OCDAV_NAME_VALIDATION_INVALID_CHARS;FRONTENT_OCDAV_NAME_VALIDATION_INVALID_CHARS" desc:"List of characters that are not allowed in file or folder names." introductionVersion:"%%NEXT%%"`
	MaxLength    int      `yaml:"max_length" env:"OCDAV_NAME_VALIDATION_MAX_LENGTH;FRONTENT_OCDAV_NAME_VALIDATION_MAX_LENGTH" desc:"Max length of file or folder names." introductionVersion:"%%NEXT%%"`
}

// FavoritesStore configures the store to use
type FavoritesStore struct {
	Store        string        `yaml:"store" env:"OC_PERSISTENT_STORE;WEBDAV_FAVORITES_STORE" desc:"The type of the store. Supported values are: 'memory', 'nats-js-kv'. See the text description for details." introductionVersion:"%%NEXT%%"`
	Nodes        []string      `yaml:"nodes" env:"OC_PERSISTENT_STORE_NODES;WEBDAV_FAVORITES_STORE_NODES" desc:"A list of nodes to access the configured store. This has no effect when 'memory' store is configured. Note that the behaviour how nodes are used is dependent on the library of the configured store. See the Environment Variable Types description for more details." introductionVersion:"%%NEXT%%"`
	Database     string        `yaml:"database" env:"WEBDAV_FAVORITES_STORE_DATABASE" desc:"The database name the configured store should use." introductionVersion:"%%NEXT%%"`
	Table        string        `yaml:"table" env:"WEBDAV_FAVORITES_STORE_TABLE" desc:"The database table the store should use." introductionVersion:"%%NEXT%%"`
	TTL          time.Duration `yaml:"ttl" env:"OC_PERSISTENT_STORE_TTL;WEBDAV_FAVORITES_STORE_TTL" desc:"Time to live for entries in the store. See the Environment Variable Types description for more details." introductionVersion:"%%NEXT%%"`
	AuthUsername string        `yaml:"username" env:"OC_PERSISTENT_STORE_AUTH_USERNAME;WEBDAV_FAVORITES_STORE_AUTH_USERNAME" desc:"The username to authenticate with the store. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"%%NEXT%%"`
	AuthPassword string        `yaml:"password" env:"OC_PERSISTENT_STORE_AUTH_PASSWORD;WEBDAV_FAVORITES_STORE_AUTH_PASSWORD" desc:"The password to authenticate with the store. Only applies when store type 'nats-js-kv' is configured." introductionVersion:"%%NEXT%%"`
}
