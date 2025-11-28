package config

import (
	"context"
	"time"

	"go-micro.dev/v4/client"

	"github.com/opencloud-eu/opencloud/pkg/shared"
)

// Config combines all available configuration parts.
type Config struct {
	Commons *shared.Commons `yaml:"-"` // don't use this directly as configuration for a service

	Service Service `yaml:"-"`

	Log   *Log  `yaml:"log"`
	Debug Debug `yaml:"debug"`

	HTTP           HTTPConfig            `yaml:"http"`
	Middleware     Middleware            `yaml:"middleware"`
	GRPC           GRPCConfig            `yaml:"grpc"`
	GRPCClientTLS  *shared.GRPCClientTLS `yaml:"grpc_client_tls"`
	GrpcClient     client.Client         `yaml:"-"`
	ServiceAccount ServiceAccount        `yaml:"service_account"`
	Events         Events                `yaml:"-"`

	TokenManager                 *TokenManager                `yaml:"token_manager"`
	Reva                         *shared.Reva                 `yaml:"reva"`
	OCMD                         OCMD                         `yaml:"ocmd"`
	ScienceMesh                  ScienceMesh                  `yaml:"sciencemesh"`
	OCMInviteManager             OCMInviteManager             `yaml:"ocm_invite_manager"`
	OCMProviderAuthorizerDriver  string                       `yaml:"ocm_provider_authorizer_driver" env:"SHARING_OCM_PROVIDER_AUTHORIZER_DRIVER" desc:"Driver to be used to persist ocm invites. Supported value is only 'json'." introductionVersion:"1.0.0"`
	OCMProviderAuthorizerDrivers OCMProviderAuthorizerDrivers `yaml:"ocm_provider_authorizer_drivers"`
	OCMShareProvider             OCMShareProvider             `yaml:"ocm_share_provider"`
	OCMCore                      OCMCore                      `yaml:"ocm_core"`
	OCMStorageProvider           OCMStorageProvider           `yaml:"ocm_storage_provider"`

	Context context.Context `yaml:"-"`
}

// HTTPConfig defines the available http configuration.
type HTTPConfig struct {
	Addr      string `yaml:"addr" env:"OCM_HTTP_ADDR" desc:"The bind address of the HTTP service." introductionVersion:"1.0.0"`
	Namespace string `yaml:"-"`
	Protocol  string `yaml:"protocol" env:"OCM_HTTP_PROTOCOL" desc:"The transport protocol of the HTTP service." introductionVersion:"1.0.0"`
	Prefix    string `yaml:"prefix" env:"OCM_HTTP_PREFIX" desc:"The path prefix where OCM can be accessed (defaults to /)." introductionVersion:"1.0.0"`
	CORS      CORS   `yaml:"cors"`
}

// Middleware configures reva middlewares.
type Middleware struct {
	Auth Auth `yaml:"auth"`
}

// Auth configures reva http auth middleware.
type Auth struct {
	CredentialsByUserAgent map[string]string `yaml:"credentials_by_user_agent"`
}

// ServiceAccount is the configuration for the used service account
type ServiceAccount struct {
	ID     string `yaml:"service_account_id" env:"OC_SERVICE_ACCOUNT_ID;OCM_SERVICE_ACCOUNT_ID" desc:"The ID of the service account the service should use. See the 'auth-service' service description for more details." introductionVersion:"1.0.0"`
	Secret string `yaml:"service_account_secret" env:"OC_SERVICE_ACCOUNT_SECRET;OCM_SERVICE_ACCOUNT_SECRET" desc:"The service account secret." introductionVersion:"1.0.0"`
}

// CORS defines the available cors configuration.
type CORS struct {
	AllowedOrigins   []string `yaml:"allow_origins" env:"OC_CORS_ALLOW_ORIGINS;OCM_CORS_ALLOW_ORIGINS" desc:"A list of allowed CORS origins. See following chapter for more details: *Access-Control-Allow-Origin* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Origin. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	AllowedMethods   []string `yaml:"allow_methods" env:"OC_CORS_ALLOW_METHODS;OCM_CORS_ALLOW_METHODS" desc:"A list of allowed CORS methods. See following chapter for more details: *Access-Control-Request-Method* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Request-Method. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	AllowedHeaders   []string `yaml:"allow_headers" env:"OC_CORS_ALLOW_HEADERS;OCM_CORS_ALLOW_HEADERS" desc:"A list of allowed CORS headers. See following chapter for more details: *Access-Control-Request-Headers* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Request-Headers. See the Environment Variable Types description for more details." introductionVersion:"1.0.0"`
	AllowCredentials bool     `yaml:"allow_credentials" env:"OC_CORS_ALLOW_CREDENTIALS;OCM_CORS_ALLOW_CREDENTIALS" desc:"Allow credentials for CORS.See following chapter for more details: *Access-Control-Allow-Credentials* at https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Credentials." introductionVersion:"1.0.0"`
}

// GRPCConfig defines the available grpc configuration.
type GRPCConfig struct {
	Addr      string                 `yaml:"addr" env:"OCM_GRPC_ADDR" desc:"The bind address of the GRPC service." introductionVersion:"1.0.0"`
	Namespace string                 `yaml:"-"`
	TLS       *shared.GRPCServiceTLS `yaml:"tls"`
	Protocol  string                 `yaml:"protocol" env:"OC_GRPC_PROTOCOL;OCM_GRPC_PROTOCOL" desc:"The transport protocol of the GRPC service." introductionVersion:"1.0.0"`
}

type ScienceMesh struct {
	Prefix               string `yaml:"prefix" env:"OCM_SCIENCEMESH_PREFIX" desc:"URL path prefix for the ScienceMesh service. Note that the string must not start with '/'." introductionVersion:"1.0.0"`
	MeshDirectoryURL     string `yaml:"science_mesh_directory_url" env:"OCM_MESH_DIRECTORY_URL" desc:"URL of the mesh directory service." introductionVersion:"1.0.0"`
	DirectoryServiceURLs string `yaml:"directory_service_urls" env:"OCM_DIRECTORY_SERVICE_URLS" desc:"Space delimited URLs of the directory services." introductionVersion:"3.5.0"`
	InviteAcceptDialog   string `yaml:"invite_accept_dialog" env:"OCM_INVITE_ACCEPT_DIALOG" desc:"/open-cloud-mesh/accept-invite;The frontend URL where to land when receiving an invitation" introductionVersion:"3.5.0"`
}

type OCMD struct {
	Prefix                     string `yaml:"prefix" env:"OCM_OCMD_PREFIX" desc:"URL path prefix for the OCMD service. Note that the string must not start with '/'." introductionVersion:"1.0.0"`
	ExposeRecipientDisplayName bool   `yaml:"expose_recipient_display_name" env:"OCM_OCMD_EXPOSE_RECIPIENT_DISPLAY_NAME" desc:"Expose the display name of OCM share recipients." introductionVersion:"1.0.0"`
}

type OCMInviteManager struct {
	Driver          string                  `yaml:"driver" env:"OCM_OCM_INVITE_MANAGER_DRIVER" desc:"Driver to be used to persist OCM invites. Supported value is only 'json'." introductionVersion:"1.0.0"`
	Drivers         OCMInviteManagerDrivers `yaml:"drivers"`
	TokenExpiration time.Duration           `yaml:"token_expiration" env:"OCM_OCM_INVITE_MANAGER_TOKEN_EXPIRATION" desc:"Expiry duration for invite tokens." introductionVersion:"1.0.0"`
	Timeout         time.Duration           `yaml:"timeout" env:"OCM_OCM_INVITE_MANAGER_TIMEOUT" desc:"Timeout specifies a time limit for requests made to OCM endpoints." introductionVersion:"1.0.0"`
	Insecure        bool                    `yaml:"insecure" env:"OCM_OCM_INVITE_MANAGER_INSECURE" desc:"Disable TLS certificate validation for the OCM connections. Do not set this in production environments." introductionVersion:"1.0.0"`
}

type OCMInviteManagerDrivers struct {
	JSON OCMInviteManagerJSONDriver `yaml:"json"`
}

type OCMInviteManagerJSONDriver struct {
	File string `yaml:"file" env:"OCM_OCM_INVITE_MANAGER_JSON_FILE" desc:"Path to the JSON file where OCM invite data will be stored. This file is maintained by the instance and must not be changed manually. If not defined, the root directory derives from $OC_BASE_DATA_PATH/storage/ocm." introductionVersion:"1.0.0"`
}

type OCMProviderAuthorizerDrivers struct {
	JSON OCMProviderAuthorizerJSONDriver `yaml:"json"`
}

type OCMProviderAuthorizerJSONDriver struct {
	Providers string `yaml:"providers" env:"OCM_OCM_PROVIDER_AUTHORIZER_PROVIDERS_FILE" desc:"Path to the JSON file where ocm invite data will be stored. Defaults to $OC_CONFIG_DIR/ocmproviders.json." introductionVersion:"1.0.0"`
}

type OCMCore struct {
	Driver  string         `yaml:"driver" env:"OCM_OCM_CORE_DRIVER" desc:"Driver to be used for the OCM core. Supported value is only 'json'." introductionVersion:"1.0.0"`
	Drivers OCMCoreDrivers `yaml:"drivers"`
}
type OCMStorageProvider struct {
	Insecure      bool   `yaml:"insecure" env:"OCM_OCM_STORAGE_PROVIDER_INSECURE" desc:"Disable TLS certificate validation for the OCM connections. Do not set this in production environments." introductionVersion:"1.0.0"`
	StorageRoot   string `yaml:"storage_root" env:"OCM_OCM_STORAGE_PROVIDER_STORAGE_ROOT" desc:"Directory where the ocm storage provider persists its data like tus upload info files." introductionVersion:"1.0.0"`
	DataServerURL string `yaml:"data_server_url" env:"OCM_OCM_STORAGE_DATA_SERVER_URL" desc:"URL of the data server, needs to be reachable by the data gateway provided by the frontend service or the user if directly exposed." introductionVersion:"1.0.0"`
}

type OCMCoreDrivers struct {
	JSON OCMCoreJSONDriver `yaml:"json"`
}

type OCMCoreJSONDriver struct {
	File string `yaml:"file" env:"OCM_OCM_CORE_JSON_FILE" desc:"Path to the JSON file where OCM share data will be stored. If not defined, the root directory derives from $OC_BASE_DATA_PATH/storage." introductionVersion:"1.0.0"`
}

type OCMShareProvider struct {
	Driver         string                  `yaml:"driver" env:"OCM_OCM_SHARE_PROVIDER_DRIVER" desc:"Driver to be used for the OCM share provider. Supported value is only 'json'." introductionVersion:"1.0.0"`
	Drivers        OCMShareProviderDrivers `yaml:"drivers"`
	Insecure       bool                    `yaml:"insecure" env:"OCM_OCM_SHARE_PROVIDER_INSECURE" desc:"Disable TLS certificate validation for the OCM connections. Do not set this in production environments." introductionVersion:"1.0.0"`
	WebappTemplate string                  `yaml:"webapp_template" env:"OCM_WEBAPP_TEMPLATE" desc:"Template for the webapp url." introductionVersion:"1.0.0"`
}

type OCMShareProviderDrivers struct {
	JSON OCMShareProviderJSONDriver `yaml:"json"`
}

type OCMShareProviderJSONDriver struct {
	File string `yaml:"file" env:"OCM_OCM_SHAREPROVIDER_JSON_FILE" desc:"Path to the JSON file where OCM share data will be stored. If not defined, the root directory derives from $OC_BASE_DATA_PATH/storage." introductionVersion:"1.0.0"`
}

// Events combine the configuration options for the event bus.
type Events struct {
	Endpoint             string `yaml:"endpoint" env:"OC_EVENTS_ENDPOINT;OCM_EVENTS_ENDPOINT" desc:"The address of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture." introductionVersion:"1.0.0"`
	Cluster              string `yaml:"cluster" env:"OC_EVENTS_CLUSTER;OCM_EVENTS_CLUSTER" desc:"The clusterID of the event system. The event system is the message queuing service. It is used as message broker for the microservice architecture. Mandatory when using NATS as event system." introductionVersion:"1.0.0"`
	TLSInsecure          bool   `yaml:"tls_insecure" env:"OC_INSECURE;OC_EVENTS_TLS_INSECURE;OCM_EVENTS_TLS_INSECURE" desc:"Whether to verify the server TLS certificates." introductionVersion:"1.0.0"`
	TLSRootCACertificate string `yaml:"tls_root_ca_certificate" env:"OC_EVENTS_TLS_ROOT_CA_CERTIFICATE;OCM_EVENTS_TLS_ROOT_CA_CERTIFICATE" desc:"The root CA certificate used to validate the server's TLS certificate. If provided OCM_EVENTS_TLS_INSECURE will be seen as false." introductionVersion:"1.0.0"`
	EnableTLS            bool   `yaml:"enable_tls" env:"OC_EVENTS_ENABLE_TLS;OCM_EVENTS_ENABLE_TLS" desc:"Enable TLS for the connection to the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
	AuthUsername         string `yaml:"username" env:"OC_EVENTS_AUTH_USERNAME;OCM_EVENTS_AUTH_USERNAME" desc:"The username to authenticate with the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
	AuthPassword         string `yaml:"password" env:"OC_EVENTS_AUTH_PASSWORD;OCM_EVENTS_AUTH_PASSWORD" desc:"The password to authenticate with the events broker. The events broker is the OpenCloud service which receives and delivers events between the services." introductionVersion:"1.0.0"`
}
