package init

// TODO: use the opencloud config struct instead of this custom struct
// We can't use it right now, because it would need  "omitempty" on
// all elements, in order to produce a slim config file with `opencloud init`.
// We can't just add these "omitempty" tags, since we want to generate
// full example configuration files with that struct, too.
// Proposed solution to  get rid of this temporary solution:
// - use the opencloud config struct
// - set the needed values like below
// - marshal it to yaml
// - unmarshal it into yaml.Node
// - recurse through the nodes and delete empty / default ones
// - marshal it to yaml

// OpenCloudConfig is the configuration for the OpenCloud services
type OpenCloudConfig struct {
	TokenManager      TokenManager          `yaml:"token_manager"`
	MachineAuthAPIKey string                `yaml:"machine_auth_api_key"`
	SystemUserAPIKey  string                `yaml:"system_user_api_key"`
	TransferSecret    string                `yaml:"transfer_secret"`
	URLSigningSecret  string                `yaml:"url_signing_secret"`
	SystemUserID      string                `yaml:"system_user_id"`
	AdminUserID       string                `yaml:"admin_user_id"`
	Graph             GraphService          `yaml:"graph"`
	Idp               LdapBasedService      `yaml:"idp"`
	Idm               IdmService            `yaml:"idm"`
	Collaboration     Collaboration         `yaml:"collaboration"`
	Proxy             ProxyService          `yaml:"proxy"`
	Frontend          FrontendService       `yaml:"frontend"`
	AuthBasic         AuthbasicService      `yaml:"auth_basic"`
	AuthBearer        AuthbearerService     `yaml:"auth_bearer"`
	Users             UsersAndGroupsService `yaml:"users"`
	Groups            UsersAndGroupsService `yaml:"groups"`
	Ocm               OcmService            `yaml:"ocm"`
	Thumbnails        ThumbnailService      `yaml:"thumbnails"`
	Search            Search                `yaml:"search"`
	Audit             Audit                 `yaml:"audit"`
	Settings          SettingsService       `yaml:"settings"`
	Sharing           Sharing               `yaml:"sharing"`
	StorageUsers      StorageUsers          `yaml:"storage_users"`
	Notifications     Notifications         `yaml:"notifications"`
	Nats              Nats                  `yaml:"nats"`
	Gateway           Gateway               `yaml:"gateway"`
	Userlog           Userlog               `yaml:"userlog"`
	AuthService       AuthService           `yaml:"auth_service"`
	Clientlog         Clientlog             `yaml:"clientlog"`
	Activitylog       Activitylog           `yaml:"activitylog"`
}

// Activitylog is the configuration for the activitylog service
type Activitylog struct {
	ServiceAccount ServiceAccount `yaml:"service_account"`
}

// App is the configuration for the collaboration service
type App struct {
	Insecure bool `yaml:"insecure"`
}

// Audit is the configuration for the audit service
type Audit struct {
	Events Events
}

// AuthbasicService is the configuration for the authbasic service
type AuthbasicService struct {
	AuthProviders LdapBasedService `yaml:"auth_providers"`
}

// AuthbearerService is the configuration for the authbearer service
type AuthbearerService struct {
	AuthProviders AuthProviderSettings `yaml:"auth_providers"`
}

// AuthProviderSettings is the configuration for the auth provider settings
type AuthProviderSettings struct {
	Oidc InsecureService
}

// AuthService is the configuration for the auth service
type AuthService struct {
	ServiceAccount ServiceAccount `yaml:"service_account"`
}

// Clientlog is the configuration for the clientlog service
type Clientlog struct {
	ServiceAccount ServiceAccount `yaml:"service_account"`
}

// Collaboration is the configuration for the collaboration service
type Collaboration struct {
	WopiApp WopiApp `yaml:"wopi"`
	App     App     `yaml:"app"`
}

// Events is the configuration for events
type Events struct {
	TLSInsecure bool `yaml:"tls_insecure"`
}

// FrontendService is the configuration for the frontend service
type FrontendService struct {
	AppHandler     InsecureService `yaml:"app_handler"`
	Archiver       InsecureService
	ServiceAccount ServiceAccount `yaml:"service_account"`
	OCDav          InsecureService
}

// Gateway is the configuration for the gateway
type Gateway struct {
	StorageRegistry StorageRegistry `yaml:"storage_registry"`
}

// GraphApplication is the configuration for the graph application
type GraphApplication struct {
	ID string `yaml:"id"`
}

// GraphService is the configuration for the graph service
type GraphService struct {
	Application    GraphApplication
	Events         Events
	Spaces         InsecureService
	Identity       LdapBasedService
	ServiceAccount ServiceAccount `yaml:"service_account"`
}

// IdmService is the configuration for the IDM service
type IdmService struct {
	ServiceUserPasswords ServiceUserPasswordsSettings `yaml:"service_user_passwords"`
}

// InsecureProxyOIDC is the configuration for the insecure proxy OIDC
type InsecureProxyOIDC struct {
	Insecure bool `yaml:"insecure"`
}

// InsecureService is the configuration for services that can be insecure
type InsecureService struct {
	Insecure bool
}

// LdapBasedService is the configuration for LDAP based services
type LdapBasedService struct {
	Ldap LdapSettings
}

// LdapSettings is the configuration for LDAP settings
type LdapSettings struct {
	BindPassword string `yaml:"bind_password"`
}

// Nats is the configuration for the nats service
type Nats struct {
	// The nats config has a field called nats
	Nats struct {
		TLSSkipVerifyClientCert bool `yaml:"tls_skip_verify_client_cert"`
	}
}

// Notifications is the configuration for the notifications service
type Notifications struct {
	Notifications  struct{ Events Events } // The notifications config has a field called notifications
	ServiceAccount ServiceAccount          `yaml:"service_account"`
}

// OcmService is the configuration for the OCM service
type OcmService struct {
	ServiceAccount ServiceAccount `yaml:"service_account"`
}

// ProxyService is the configuration for the proxy service
type ProxyService struct {
	OIDC             InsecureProxyOIDC `yaml:"oidc"`
	InsecureBackends bool              `yaml:"insecure_backends"`
	ServiceAccount   ServiceAccount    `yaml:"service_account"`
}

// Search is the configuration for the search service
type Search struct {
	Events         Events
	ServiceAccount ServiceAccount `yaml:"service_account"`
}

// ServiceAccount is the configuration for the used service account
type ServiceAccount struct {
	ServiceAccountID     string `yaml:"service_account_id"`
	ServiceAccountSecret string `yaml:"service_account_secret"`
}

// ServiceUserPasswordsSettings is the configuration for service user passwords
type ServiceUserPasswordsSettings struct {
	AdminPassword string `yaml:"admin_password"`
	IdmPassword   string `yaml:"idm_password"`
	RevaPassword  string `yaml:"reva_password"`
	IdpPassword   string `yaml:"idp_password"`
}

// SettingsService is the configuration for the settings service
type SettingsService struct {
	ServiceAccountIDs []string `yaml:"service_account_ids"`
}

// Sharing is the configuration for the sharing service
type Sharing struct {
	Events Events
}

// StorageRegistry is the configuration for the storage registry
type StorageRegistry struct {
	StorageUsersMountID string `yaml:"storage_users_mount_id"`
}

// StorageUsers is the configuration for the storage users
type StorageUsers struct {
	Events         Events
	MountID        string         `yaml:"mount_id"`
	ServiceAccount ServiceAccount `yaml:"service_account"`
}

// ThumbnailSettings is the configuration for the thumbnail settings
type ThumbnailSettings struct {
	TransferSecret      string `yaml:"transfer_secret"`
	WebdavAllowInsecure bool   `yaml:"webdav_allow_insecure"`
	Cs3AllowInsecure    bool   `yaml:"cs3_allow_insecure"`
}

// ThumbnailService is the configuration for the thumbnail service
type ThumbnailService struct {
	Thumbnail ThumbnailSettings
}

// TokenManager is the configuration for the token manager
type TokenManager struct {
	JWTSecret string `yaml:"jwt_secret"`
}

// UsersAndGroupsService is the configuration for the users and groups service
type UsersAndGroupsService struct {
	Drivers LdapBasedService
}

// Userlog is the configuration for the userlog service
type Userlog struct {
	ServiceAccount ServiceAccount `yaml:"service_account"`
}

// WopiApp is the configuration for the WOPI app
type WopiApp struct {
	Secret string `yaml:"secret"`
}
