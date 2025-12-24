package command

import (
	"fmt"

	"github.com/opencloud-eu/opencloud/opencloud/pkg/register"
	"github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/config/parser"
	activitylog "github.com/opencloud-eu/opencloud/services/activitylog/pkg/command"
	antivirus "github.com/opencloud-eu/opencloud/services/antivirus/pkg/command"
	appprovider "github.com/opencloud-eu/opencloud/services/app-provider/pkg/command"
	appregistry "github.com/opencloud-eu/opencloud/services/app-registry/pkg/command"
	audit "github.com/opencloud-eu/opencloud/services/audit/pkg/command"
	authapp "github.com/opencloud-eu/opencloud/services/auth-app/pkg/command"
	authbasic "github.com/opencloud-eu/opencloud/services/auth-basic/pkg/command"
	authbearer "github.com/opencloud-eu/opencloud/services/auth-bearer/pkg/command"
	authmachine "github.com/opencloud-eu/opencloud/services/auth-machine/pkg/command"
	authservice "github.com/opencloud-eu/opencloud/services/auth-service/pkg/command"
	clientlog "github.com/opencloud-eu/opencloud/services/clientlog/pkg/command"
	collaboration "github.com/opencloud-eu/opencloud/services/collaboration/pkg/command"
	eventhistory "github.com/opencloud-eu/opencloud/services/eventhistory/pkg/command"
	frontend "github.com/opencloud-eu/opencloud/services/frontend/pkg/command"
	gateway "github.com/opencloud-eu/opencloud/services/gateway/pkg/command"
	graph "github.com/opencloud-eu/opencloud/services/graph/pkg/command"
	groups "github.com/opencloud-eu/opencloud/services/groups/pkg/command"
	idm "github.com/opencloud-eu/opencloud/services/idm/pkg/command"
	idp "github.com/opencloud-eu/opencloud/services/idp/pkg/command"
	invitations "github.com/opencloud-eu/opencloud/services/invitations/pkg/command"
	nats "github.com/opencloud-eu/opencloud/services/nats/pkg/command"
	notifications "github.com/opencloud-eu/opencloud/services/notifications/pkg/command"
	ocdav "github.com/opencloud-eu/opencloud/services/ocdav/pkg/command"
	ocm "github.com/opencloud-eu/opencloud/services/ocm/pkg/command"
	ocs "github.com/opencloud-eu/opencloud/services/ocs/pkg/command"
	policies "github.com/opencloud-eu/opencloud/services/policies/pkg/command"
	postprocessing "github.com/opencloud-eu/opencloud/services/postprocessing/pkg/command"
	proxy "github.com/opencloud-eu/opencloud/services/proxy/pkg/command"
	search "github.com/opencloud-eu/opencloud/services/search/pkg/command"
	settings "github.com/opencloud-eu/opencloud/services/settings/pkg/command"
	sharing "github.com/opencloud-eu/opencloud/services/sharing/pkg/command"
	sse "github.com/opencloud-eu/opencloud/services/sse/pkg/command"
	storagepubliclink "github.com/opencloud-eu/opencloud/services/storage-publiclink/pkg/command"
	storageshares "github.com/opencloud-eu/opencloud/services/storage-shares/pkg/command"
	storagesystem "github.com/opencloud-eu/opencloud/services/storage-system/pkg/command"
	storageusers "github.com/opencloud-eu/opencloud/services/storage-users/pkg/command"
	thumbnails "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/command"
	userlog "github.com/opencloud-eu/opencloud/services/userlog/pkg/command"
	users "github.com/opencloud-eu/opencloud/services/users/pkg/command"
	web "github.com/opencloud-eu/opencloud/services/web/pkg/command"
	webdav "github.com/opencloud-eu/opencloud/services/webdav/pkg/command"
	webfinger "github.com/opencloud-eu/opencloud/services/webfinger/pkg/command"

	"github.com/spf13/cobra"
)

var serviceCommands = []register.Command{
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Activitylog.Service.Name, activitylog.GetCommands(cfg.Activitylog), func(c *config.Config) {
			cfg.Activitylog.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Antivirus.Service.Name, antivirus.GetCommands(cfg.Antivirus), func(c *config.Config) {
			cfg.Antivirus.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.AppProvider.Service.Name, appprovider.GetCommands(cfg.AppProvider), func(c *config.Config) {
			cfg.AppProvider.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.AppRegistry.Service.Name, appregistry.GetCommands(cfg.AppRegistry), func(c *config.Config) {
			cfg.AppRegistry.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Audit.Service.Name, audit.GetCommands(cfg.Audit), func(c *config.Config) {
			cfg.Audit.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.AuthApp.Service.Name, authapp.GetCommands(cfg.AuthApp), func(_ *config.Config) {
			cfg.AuthApp.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.AuthBasic.Service.Name, authbasic.GetCommands(cfg.AuthBasic), func(c *config.Config) {
			cfg.AuthBasic.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.AuthBearer.Service.Name, authbearer.GetCommands(cfg.AuthBearer), func(c *config.Config) {
			cfg.AuthBearer.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.AuthMachine.Service.Name, authmachine.GetCommands(cfg.AuthMachine), func(c *config.Config) {
			cfg.AuthMachine.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.AuthService.Service.Name, authservice.GetCommands(cfg.AuthService), func(c *config.Config) {
			cfg.AuthService.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Clientlog.Service.Name, clientlog.GetCommands(cfg.Clientlog), func(c *config.Config) {
			cfg.Clientlog.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Collaboration.Service.Name, collaboration.GetCommands(cfg.Collaboration), func(c *config.Config) {
			cfg.Collaboration.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.EventHistory.Service.Name, eventhistory.GetCommands(cfg.EventHistory), func(c *config.Config) {
			cfg.EventHistory.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Frontend.Service.Name, frontend.GetCommands(cfg.Frontend), func(c *config.Config) {
			cfg.Frontend.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Gateway.Service.Name, gateway.GetCommands(cfg.Gateway), func(c *config.Config) {
			cfg.Gateway.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Graph.Service.Name, graph.GetCommands(cfg.Graph), func(c *config.Config) {
			cfg.Graph.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Groups.Service.Name, groups.GetCommands(cfg.Groups), func(c *config.Config) {
			cfg.Groups.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.IDM.Service.Name, idm.GetCommands(cfg.IDM), func(c *config.Config) {
			cfg.IDM.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.IDP.Service.Name, idp.GetCommands(cfg.IDP), func(c *config.Config) {
			cfg.IDP.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Invitations.Service.Name, invitations.GetCommands(cfg.Invitations), func(c *config.Config) {
			cfg.Invitations.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Nats.Service.Name, nats.GetCommands(cfg.Nats), func(c *config.Config) {
			cfg.Nats.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Notifications.Service.Name, notifications.GetCommands(cfg.Notifications), func(c *config.Config) {
			cfg.Notifications.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.OCDav.Service.Name, ocdav.GetCommands(cfg.OCDav), func(c *config.Config) {
			cfg.OCDav.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.OCM.Service.Name, ocm.GetCommands(cfg.OCM), func(c *config.Config) {
			cfg.OCM.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.OCS.Service.Name, ocs.GetCommands(cfg.OCS), func(c *config.Config) {
			cfg.OCS.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Policies.Service.Name, policies.GetCommands(cfg.Policies), func(c *config.Config) {
			cfg.Policies.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Postprocessing.Service.Name, postprocessing.GetCommands(cfg.Postprocessing), func(c *config.Config) {
			cfg.Postprocessing.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Proxy.Service.Name, proxy.GetCommands(cfg.Proxy), func(c *config.Config) {
			cfg.Proxy.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Search.Service.Name, search.GetCommands(cfg.Search), func(c *config.Config) {
			cfg.Search.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Settings.Service.Name, settings.GetCommands(cfg.Settings), func(c *config.Config) {
			cfg.Settings.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Sharing.Service.Name, sharing.GetCommands(cfg.Sharing), func(c *config.Config) {
			cfg.Sharing.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.SSE.Service.Name, sse.GetCommands(cfg.SSE), func(c *config.Config) {
			cfg.SSE.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.StoragePublicLink.Service.Name, storagepubliclink.GetCommands(cfg.StoragePublicLink), func(c *config.Config) {
			cfg.StoragePublicLink.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.StorageShares.Service.Name, storageshares.GetCommands(cfg.StorageShares), func(c *config.Config) {
			cfg.StorageShares.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.StorageSystem.Service.Name, storagesystem.GetCommands(cfg.StorageSystem), func(c *config.Config) {
			cfg.StorageSystem.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.StorageUsers.Service.Name, storageusers.GetCommands(cfg.StorageUsers), func(c *config.Config) {
			cfg.StorageUsers.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Thumbnails.Service.Name, thumbnails.GetCommands(cfg.Thumbnails), func(c *config.Config) {
			cfg.Thumbnails.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Userlog.Service.Name, userlog.GetCommands(cfg.Userlog), func(c *config.Config) {
			cfg.Userlog.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Users.Service.Name, users.GetCommands(cfg.Users), func(c *config.Config) {
			cfg.Users.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Web.Service.Name, web.GetCommands(cfg.Web), func(c *config.Config) {
			cfg.Web.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.WebDAV.Service.Name, webdav.GetCommands(cfg.WebDAV), func(c *config.Config) {
			cfg.WebDAV.Commons = cfg.Commons
		})
	},
	func(cfg *config.Config) *cobra.Command {
		return ServiceCommand(cfg, cfg.Webfinger.Service.Name, webfinger.GetCommands(cfg.Webfinger), func(c *config.Config) {
			cfg.Webfinger.Commons = cfg.Commons
		})
	},
}

// ServiceCommand composes a cobra command from the given inputs.
func ServiceCommand(cfg *config.Config, serviceName string, subCommands []*cobra.Command, f func(*config.Config)) *cobra.Command {
	command := &cobra.Command{
		Use:     serviceName,
		Short:   fmt.Sprintf("%s service commands", serviceName),
		GroupID: CommandGroupServices,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			configlog.Error(parser.ParseConfig(cfg, true))
			f(cfg)
			return nil
		},
	}

	command.AddCommand(subCommands...)

	return command
}

func init() {
	for _, c := range serviceCommands {
		register.AddCommand(c)
	}
}
