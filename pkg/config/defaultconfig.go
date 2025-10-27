package config

import (
	"github.com/opencloud-eu/opencloud/pkg/shared"
	activitylog "github.com/opencloud-eu/opencloud/services/activitylog/pkg/config/defaults"
	antivirus "github.com/opencloud-eu/opencloud/services/antivirus/pkg/config/defaults"
	appProvider "github.com/opencloud-eu/opencloud/services/app-provider/pkg/config/defaults"
	appRegistry "github.com/opencloud-eu/opencloud/services/app-registry/pkg/config/defaults"
	audit "github.com/opencloud-eu/opencloud/services/audit/pkg/config/defaults"
	authapp "github.com/opencloud-eu/opencloud/services/auth-app/pkg/config/defaults"
	authbasic "github.com/opencloud-eu/opencloud/services/auth-basic/pkg/config/defaults"
	authbearer "github.com/opencloud-eu/opencloud/services/auth-bearer/pkg/config/defaults"
	authmachine "github.com/opencloud-eu/opencloud/services/auth-machine/pkg/config/defaults"
	authservice "github.com/opencloud-eu/opencloud/services/auth-service/pkg/config/defaults"
	clientlog "github.com/opencloud-eu/opencloud/services/clientlog/pkg/config/defaults"
	collaboration "github.com/opencloud-eu/opencloud/services/collaboration/pkg/config/defaults"
	eventhistory "github.com/opencloud-eu/opencloud/services/eventhistory/pkg/config/defaults"
	frontend "github.com/opencloud-eu/opencloud/services/frontend/pkg/config/defaults"
	gateway "github.com/opencloud-eu/opencloud/services/gateway/pkg/config/defaults"
	graph "github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	groups "github.com/opencloud-eu/opencloud/services/groups/pkg/config/defaults"
	idm "github.com/opencloud-eu/opencloud/services/idm/pkg/config/defaults"
	idp "github.com/opencloud-eu/opencloud/services/idp/pkg/config/defaults"
	invitations "github.com/opencloud-eu/opencloud/services/invitations/pkg/config/defaults"
	nats "github.com/opencloud-eu/opencloud/services/nats/pkg/config/defaults"
	notifications "github.com/opencloud-eu/opencloud/services/notifications/pkg/config/defaults"
	ocdav "github.com/opencloud-eu/opencloud/services/ocdav/pkg/config/defaults"
	ocm "github.com/opencloud-eu/opencloud/services/ocm/pkg/config/defaults"
	ocs "github.com/opencloud-eu/opencloud/services/ocs/pkg/config/defaults"
	policies "github.com/opencloud-eu/opencloud/services/policies/pkg/config/defaults"
	postprocessing "github.com/opencloud-eu/opencloud/services/postprocessing/pkg/config/defaults"
	proxy "github.com/opencloud-eu/opencloud/services/proxy/pkg/config/defaults"
	search "github.com/opencloud-eu/opencloud/services/search/pkg/config/defaults"
	settings "github.com/opencloud-eu/opencloud/services/settings/pkg/config/defaults"
	sharing "github.com/opencloud-eu/opencloud/services/sharing/pkg/config/defaults"
	sse "github.com/opencloud-eu/opencloud/services/sse/pkg/config/defaults"
	storagepublic "github.com/opencloud-eu/opencloud/services/storage-publiclink/pkg/config/defaults"
	storageshares "github.com/opencloud-eu/opencloud/services/storage-shares/pkg/config/defaults"
	storageSystem "github.com/opencloud-eu/opencloud/services/storage-system/pkg/config/defaults"
	storageusers "github.com/opencloud-eu/opencloud/services/storage-users/pkg/config/defaults"
	thumbnails "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/config/defaults"
	userlog "github.com/opencloud-eu/opencloud/services/userlog/pkg/config/defaults"
	users "github.com/opencloud-eu/opencloud/services/users/pkg/config/defaults"
	web "github.com/opencloud-eu/opencloud/services/web/pkg/config/defaults"
	webdav "github.com/opencloud-eu/opencloud/services/webdav/pkg/config/defaults"
	webfinger "github.com/opencloud-eu/opencloud/services/webfinger/pkg/config/defaults"
)

func DefaultConfig() *Config {
	return &Config{
		OpenCloudURL: "https://localhost:9200",
		Runtime: Runtime{
			Port:          "9250",
			Host:          "localhost",
			ShutdownOrder: []string{"proxy"},
		},
		Reva: &shared.Reva{
			Address: "eu.opencloud.api.gateway",
		},

		Activitylog:       activitylog.DefaultConfig(),
		Antivirus:         antivirus.DefaultConfig(),
		AppProvider:       appProvider.DefaultConfig(),
		AppRegistry:       appRegistry.DefaultConfig(),
		Audit:             audit.DefaultConfig(),
		AuthApp:           authapp.DefaultConfig(),
		AuthBasic:         authbasic.DefaultConfig(),
		AuthBearer:        authbearer.DefaultConfig(),
		AuthMachine:       authmachine.DefaultConfig(),
		AuthService:       authservice.DefaultConfig(),
		Clientlog:         clientlog.DefaultConfig(),
		Collaboration:     collaboration.DefaultConfig(),
		EventHistory:      eventhistory.DefaultConfig(),
		Frontend:          frontend.DefaultConfig(),
		Gateway:           gateway.DefaultConfig(),
		Graph:             graph.DefaultConfig(),
		Groups:            groups.DefaultConfig(),
		IDM:               idm.DefaultConfig(),
		IDP:               idp.DefaultConfig(),
		Invitations:       invitations.DefaultConfig(),
		Nats:              nats.DefaultConfig(),
		Notifications:     notifications.DefaultConfig(),
		OCDav:             ocdav.DefaultConfig(),
		OCM:               ocm.DefaultConfig(),
		OCS:               ocs.DefaultConfig(),
		Postprocessing:    postprocessing.DefaultConfig(),
		Policies:          policies.DefaultConfig(),
		Proxy:             proxy.DefaultConfig(),
		Search:            search.DefaultConfig(),
		Settings:          settings.DefaultConfig(),
		Sharing:           sharing.DefaultConfig(),
		SSE:               sse.DefaultConfig(),
		StoragePublicLink: storagepublic.DefaultConfig(),
		StorageShares:     storageshares.DefaultConfig(),
		StorageSystem:     storageSystem.DefaultConfig(),
		StorageUsers:      storageusers.DefaultConfig(),
		Thumbnails:        thumbnails.DefaultConfig(),
		Userlog:           userlog.DefaultConfig(),
		Users:             users.DefaultConfig(),
		Web:               web.DefaultConfig(),
		WebDAV:            webdav.DefaultConfig(),
		Webfinger:         webfinger.DefaultConfig(),
	}
}
