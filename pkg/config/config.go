package config

import (
	"github.com/opencloud-eu/opencloud/pkg/shared"
	activitylog "github.com/opencloud-eu/opencloud/services/activitylog/pkg/config"
	antivirus "github.com/opencloud-eu/opencloud/services/antivirus/pkg/config"
	appProvider "github.com/opencloud-eu/opencloud/services/app-provider/pkg/config"
	appRegistry "github.com/opencloud-eu/opencloud/services/app-registry/pkg/config"
	audit "github.com/opencloud-eu/opencloud/services/audit/pkg/config"
	authapp "github.com/opencloud-eu/opencloud/services/auth-app/pkg/config"
	authbasic "github.com/opencloud-eu/opencloud/services/auth-basic/pkg/config"
	authbearer "github.com/opencloud-eu/opencloud/services/auth-bearer/pkg/config"
	authmachine "github.com/opencloud-eu/opencloud/services/auth-machine/pkg/config"
	authservice "github.com/opencloud-eu/opencloud/services/auth-service/pkg/config"
	clientlog "github.com/opencloud-eu/opencloud/services/clientlog/pkg/config"
	collaboration "github.com/opencloud-eu/opencloud/services/collaboration/pkg/config"
	eventhistory "github.com/opencloud-eu/opencloud/services/eventhistory/pkg/config"
	frontend "github.com/opencloud-eu/opencloud/services/frontend/pkg/config"
	gateway "github.com/opencloud-eu/opencloud/services/gateway/pkg/config"
	graph "github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	groups "github.com/opencloud-eu/opencloud/services/groups/pkg/config"
	idm "github.com/opencloud-eu/opencloud/services/idm/pkg/config"
	idp "github.com/opencloud-eu/opencloud/services/idp/pkg/config"
	invitations "github.com/opencloud-eu/opencloud/services/invitations/pkg/config"
	nats "github.com/opencloud-eu/opencloud/services/nats/pkg/config"
	notifications "github.com/opencloud-eu/opencloud/services/notifications/pkg/config"
	ocdav "github.com/opencloud-eu/opencloud/services/ocdav/pkg/config"
	ocm "github.com/opencloud-eu/opencloud/services/ocm/pkg/config"
	ocs "github.com/opencloud-eu/opencloud/services/ocs/pkg/config"
	policies "github.com/opencloud-eu/opencloud/services/policies/pkg/config"
	postprocessing "github.com/opencloud-eu/opencloud/services/postprocessing/pkg/config"
	proxy "github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	search "github.com/opencloud-eu/opencloud/services/search/pkg/config"
	settings "github.com/opencloud-eu/opencloud/services/settings/pkg/config"
	sharing "github.com/opencloud-eu/opencloud/services/sharing/pkg/config"
	sse "github.com/opencloud-eu/opencloud/services/sse/pkg/config"
	storagepublic "github.com/opencloud-eu/opencloud/services/storage-publiclink/pkg/config"
	storageshares "github.com/opencloud-eu/opencloud/services/storage-shares/pkg/config"
	storagesystem "github.com/opencloud-eu/opencloud/services/storage-system/pkg/config"
	storageusers "github.com/opencloud-eu/opencloud/services/storage-users/pkg/config"
	thumbnails "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/config"
	userlog "github.com/opencloud-eu/opencloud/services/userlog/pkg/config"
	users "github.com/opencloud-eu/opencloud/services/users/pkg/config"
	web "github.com/opencloud-eu/opencloud/services/web/pkg/config"
	webdav "github.com/opencloud-eu/opencloud/services/webdav/pkg/config"
	webfinger "github.com/opencloud-eu/opencloud/services/webfinger/pkg/config"
)

type Mode int

// Runtime configures the OpenCloud runtime when running in supervised mode.
type Runtime struct {
	Port          string   `yaml:"port" env:"OC_RUNTIME_PORT" desc:"The TCP port at which OpenCloud will be available" introductionVersion:"1.0.0"`
	Host          string   `yaml:"host" env:"OC_RUNTIME_HOST" desc:"The host at which OpenCloud will be available" introductionVersion:"1.0.0"`
	Services      []string `yaml:"services" env:"OC_RUN_EXTENSIONS;OC_RUN_SERVICES" desc:"A comma-separated list of service names. Will start only the listed services." introductionVersion:"1.0.0"`
	Disabled      []string `yaml:"disabled_services" env:"OC_EXCLUDE_RUN_SERVICES" desc:"A comma-separated list of service names. Will start all default services except of the ones listed. Has no effect when OC_RUN_SERVICES is set." introductionVersion:"1.0.0"`
	Additional    []string `yaml:"add_services" env:"OC_ADD_RUN_SERVICES" desc:"A comma-separated list of service names. Will add the listed services to the default configuration. Has no effect when OC_RUN_SERVICES is set. Note that one can add services not started by the default list and exclude services from the default list by using both envvars at the same time." introductionVersion:"1.0.0"`
	ShutdownOrder []string `yaml:"shutdown_order" env:"OC_SHUTDOWN_ORDER" desc:"A comma-separated list of service names defining the order in which services are shut down. Services not listed will be stopped after the listed ones in random order." introductionVersion:"%%NEXT%%"`
}

// Config combines all available configuration parts.
type Config struct {
	*shared.Commons `yaml:"shared"`

	Tracing        *shared.Tracing        `yaml:"tracing"`
	Log            *shared.Log            `yaml:"log"`
	Cache          *shared.Cache          `yaml:"cache"`
	GRPCClientTLS  *shared.GRPCClientTLS  `yaml:"grpc_client_tls"`
	GRPCServiceTLS *shared.GRPCServiceTLS `yaml:"grpc_service_tls"`
	HTTPServiceTLS shared.HTTPServiceTLS  `yaml:"http_service_tls"`
	Reva           *shared.Reva           `yaml:"reva"`

	Mode         Mode // DEPRECATED
	File         string
	OpenCloudURL string `yaml:"opencloud_url" env:"OC_URL" desc:"URL, where OpenCloudURL is reachable for users." introductionVersion:"1.0.0"`

	Registry          string               `yaml:"registry"`
	TokenManager      *shared.TokenManager `yaml:"token_manager"`
	MachineAuthAPIKey string               `yaml:"machine_auth_api_key" env:"OC_MACHINE_AUTH_API_KEY" desc:"Machine auth API key used to validate internal requests necessary for the access to resources from other services." introductionVersion:"1.0.0"`
	TransferSecret    string               `yaml:"transfer_secret" env:"OC_TRANSFER_SECRET" desc:"Transfer secret for signing file up- and download requests." introductionVersion:"1.0.0"`
	SystemUserID      string               `yaml:"system_user_id" env:"OC_SYSTEM_USER_ID" desc:"ID of the OpenCloud storage-system system user. Admins need to set the ID for the storage-system system user in this config option which is then used to reference the user. Any reasonable long string is possible, preferably this would be an UUIDv4 format." introductionVersion:"1.0.0"`
	SystemUserAPIKey  string               `yaml:"system_user_api_key" env:"OC_SYSTEM_USER_API_KEY" desc:"API key for the storage-system system user." introductionVersion:"1.0.0"`
	AdminUserID       string               `yaml:"admin_user_id" env:"OC_ADMIN_USER_ID" desc:"ID of a user, that should receive admin privileges. Consider that the UUID can be encoded in some LDAP deployment configurations like in .ldif files. These need to be decoded beforehand." introductionVersion:"1.0.0"`
	Runtime           Runtime              `yaml:"runtime"`

	Activitylog       *activitylog.Config    `yaml:"activitylog"`
	Antivirus         *antivirus.Config      `yaml:"antivirus"`
	AppProvider       *appProvider.Config    `yaml:"app_provider"`
	AppRegistry       *appRegistry.Config    `yaml:"app_registry"`
	Audit             *audit.Config          `yaml:"audit"`
	AuthApp           *authapp.Config        `yaml:"auth_app"`
	AuthBasic         *authbasic.Config      `yaml:"auth_basic"`
	AuthBearer        *authbearer.Config     `yaml:"auth_bearer"`
	AuthMachine       *authmachine.Config    `yaml:"auth_machine"`
	AuthService       *authservice.Config    `yaml:"auth_service"`
	Clientlog         *clientlog.Config      `yaml:"clientlog"`
	Collaboration     *collaboration.Config  `yaml:"collaboration"`
	EventHistory      *eventhistory.Config   `yaml:"eventhistory"`
	Frontend          *frontend.Config       `yaml:"frontend"`
	Gateway           *gateway.Config        `yaml:"gateway"`
	Graph             *graph.Config          `yaml:"graph"`
	Groups            *groups.Config         `yaml:"groups"`
	IDM               *idm.Config            `yaml:"idm"`
	IDP               *idp.Config            `yaml:"idp"`
	Invitations       *invitations.Config    `yaml:"invitations"`
	Nats              *nats.Config           `yaml:"nats"`
	Notifications     *notifications.Config  `yaml:"notifications"`
	OCDav             *ocdav.Config          `yaml:"ocdav"`
	OCM               *ocm.Config            `yaml:"ocm"`
	OCS               *ocs.Config            `yaml:"ocs"`
	Postprocessing    *postprocessing.Config `yaml:"postprocessing"`
	Policies          *policies.Config       `yaml:"policies"`
	Proxy             *proxy.Config          `yaml:"proxy"`
	Settings          *settings.Config       `yaml:"settings"`
	Sharing           *sharing.Config        `yaml:"sharing"`
	SSE               *sse.Config            `yaml:"sse"`
	StorageSystem     *storagesystem.Config  `yaml:"storage_system"`
	StoragePublicLink *storagepublic.Config  `yaml:"storage_public"`
	StorageShares     *storageshares.Config  `yaml:"storage_shares"`
	StorageUsers      *storageusers.Config   `yaml:"storage_users"`
	Thumbnails        *thumbnails.Config     `yaml:"thumbnails"`
	Userlog           *userlog.Config        `yaml:"userlog"`
	Users             *users.Config          `yaml:"users"`
	Web               *web.Config            `yaml:"web"`
	WebDAV            *webdav.Config         `yaml:"webdav"`
	Webfinger         *webfinger.Config      `yaml:"webfinger"`
	Search            *search.Config         `yaml:"search"`
}
