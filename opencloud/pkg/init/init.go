package init

import (
	"fmt"
	"os"
	"path"

	"github.com/google/uuid"
	"gopkg.in/yaml.v2"

	"github.com/opencloud-eu/opencloud/pkg/generators"
)

const (
	configFilename = "opencloud.yaml"
	passwordLength = 32
)

var (
	_insecureService = InsecureService{Insecure: true}
	_insecureEvents  = Events{TLSInsecure: true}
)

// CreateConfig creates a config file with random passwords at configPath
func CreateConfig(insecure, forceOverwrite, diff bool, configPath, adminPassword string) error {
	if diff && forceOverwrite {
		return fmt.Errorf("diff and force-overwrite flags are mutually exclusive")
	}
	if diff && adminPassword != "" {
		return fmt.Errorf("diff and admin-password flags are mutually exclusive")
	}

	if configExists(configPath) && !forceOverwrite && !diff {
		return fmt.Errorf("config file already exists, use --force-overwrite to overwrite or --diff to show diff")
	}

	err := checkConfigPath(configPath)
	if err != nil && (!forceOverwrite && !diff) {
		fmt.Println("off")
		return err
	}
	targetBackupConfig := ""
	if err != nil {
		targetBackupConfig, err = backupOpenCloudConfigFile(configPath)
		if err != nil {
			return err
		}
	}
	err = os.MkdirAll(configPath, 0700)
	if err != nil {
		return err
	}

	// Load old config
	var oldCfg OpenCloudConfig
	if diff {
		fp, err := os.ReadFile(path.Join(configPath, configFilename))
		if err != nil {
			return err
		}
		err = yaml.Unmarshal(fp, &oldCfg)
		if err != nil {
			return err
		}
	}

	var (
		systemUserID, adminUserID, graphApplicationID, storageUsersMountID, serviceAccountID string
		idmServicePassword, idpServicePassword, ocAdminServicePassword, revaServicePassword  string
		tokenManagerJwtSecret, collaborationWOPISecret, machineAuthAPIKey, systemUserAPIKey  string
		revaTransferSecret, thumbnailsTransferSecret, serviceAccountSecret, urlSigningSecret string
	)

	if diff {
		systemUserID = oldCfg.SystemUserID
		adminUserID = oldCfg.AdminUserID
		graphApplicationID = oldCfg.Graph.Application.ID
		storageUsersMountID = oldCfg.Gateway.StorageRegistry.StorageUsersMountID
		serviceAccountID = oldCfg.Graph.ServiceAccount.ServiceAccountID

		idmServicePassword = oldCfg.Idm.ServiceUserPasswords.IdmPassword
		idpServicePassword = oldCfg.Idm.ServiceUserPasswords.IdpPassword
		ocAdminServicePassword = oldCfg.Idm.ServiceUserPasswords.AdminPassword
		revaServicePassword = oldCfg.Idm.ServiceUserPasswords.RevaPassword
		tokenManagerJwtSecret = oldCfg.TokenManager.JWTSecret
		collaborationWOPISecret = oldCfg.Collaboration.WopiApp.Secret
		if collaborationWOPISecret == "" {
			collaborationWOPISecret, err = generators.GenerateRandomPassword(passwordLength)
			if err != nil {
				return fmt.Errorf("could not generate random wopi secret for collaboration service: %s", err)
			}
		}
		machineAuthAPIKey = oldCfg.MachineAuthAPIKey
		systemUserAPIKey = oldCfg.SystemUserAPIKey
		revaTransferSecret = oldCfg.TransferSecret
		thumbnailsTransferSecret = oldCfg.Thumbnails.Thumbnail.TransferSecret
		serviceAccountSecret = oldCfg.Graph.ServiceAccount.ServiceAccountSecret
		urlSigningSecret = oldCfg.URLSigningSecret
		if urlSigningSecret == "" {
			urlSigningSecret, err = generators.GenerateRandomPassword(passwordLength)
			if err != nil {
				return fmt.Errorf("could not generate random secret for urlSigningSecret: %s", err)
			}
		}
	} else {
		systemUserID = uuid.NewString()
		adminUserID = uuid.NewString()
		graphApplicationID = uuid.NewString()
		storageUsersMountID = uuid.NewString()
		serviceAccountID = uuid.NewString()

		idmServicePassword, err = generators.GenerateRandomPassword(passwordLength)
		if err != nil {
			return fmt.Errorf("could not generate random password for idm: %s", err)
		}
		idpServicePassword, err = generators.GenerateRandomPassword(passwordLength)
		if err != nil {
			return fmt.Errorf("could not generate random password for idp: %s", err)
		}
		ocAdminServicePassword = adminPassword
		if ocAdminServicePassword == "" {
			ocAdminServicePassword, err = generators.GenerateRandomPassword(passwordLength)
			if err != nil {
				return fmt.Errorf("could not generate random password for opencloud admin: %s", err)
			}
		}

		revaServicePassword, err = generators.GenerateRandomPassword(passwordLength)
		if err != nil {
			return fmt.Errorf("could not generate random password for reva: %s", err)
		}
		tokenManagerJwtSecret, err = generators.GenerateRandomPassword(passwordLength)
		if err != nil {
			return fmt.Errorf("could not generate random password for tokenmanager: %s", err)
		}
		collaborationWOPISecret, err = generators.GenerateRandomPassword(passwordLength)
		if err != nil {
			return fmt.Errorf("could not generate random wopi secret for collaboration service: %s", err)
		}
		machineAuthAPIKey, err = generators.GenerateRandomPassword(passwordLength)
		if err != nil {
			return fmt.Errorf("could not generate random password for machineauthsecret: %s", err)
		}
		systemUserAPIKey, err = generators.GenerateRandomPassword(passwordLength)
		if err != nil {
			return fmt.Errorf("could not generate random system user API key: %s", err)
		}
		revaTransferSecret, err = generators.GenerateRandomPassword(passwordLength)
		if err != nil {
			return fmt.Errorf("could not generate random password for revaTransferSecret: %s", err)
		}
		urlSigningSecret, err = generators.GenerateRandomPassword(passwordLength)
		if err != nil {
			return fmt.Errorf("could not generate random secret for urlSigningSecret: %s", err)
		}
		thumbnailsTransferSecret, err = generators.GenerateRandomPassword(passwordLength)
		if err != nil {
			return fmt.Errorf("could not generate random password for thumbnailsTransferSecret: %s", err)
		}
		serviceAccountSecret, err = generators.GenerateRandomPassword(passwordLength)
		if err != nil {
			return fmt.Errorf("could not generate random secret for serviceAccountSecret: %s", err)
		}
	}

	serviceAccount := ServiceAccount{
		ServiceAccountID:     serviceAccountID,
		ServiceAccountSecret: serviceAccountSecret,
	}

	cfg := OpenCloudConfig{
		TokenManager: TokenManager{
			JWTSecret: tokenManagerJwtSecret,
		},
		MachineAuthAPIKey: machineAuthAPIKey,
		SystemUserAPIKey:  systemUserAPIKey,
		TransferSecret:    revaTransferSecret,
		URLSigningSecret:  urlSigningSecret,
		SystemUserID:      systemUserID,
		AdminUserID:       adminUserID,
		Idm: IdmService{
			ServiceUserPasswords: ServiceUserPasswordsSettings{
				AdminPassword: ocAdminServicePassword,
				IdpPassword:   idpServicePassword,
				RevaPassword:  revaServicePassword,
				IdmPassword:   idmServicePassword,
			},
		},
		Idp: LdapBasedService{
			Ldap: LdapSettings{
				BindPassword: idpServicePassword,
			},
		},
		AuthBasic: AuthbasicService{
			AuthProviders: LdapBasedService{
				Ldap: LdapSettings{
					BindPassword: revaServicePassword,
				},
			},
		},
		Collaboration: Collaboration{
			WopiApp: WopiApp{
				Secret: collaborationWOPISecret,
			},
		},
		Groups: UsersAndGroupsService{
			Drivers: LdapBasedService{
				Ldap: LdapSettings{
					BindPassword: revaServicePassword,
				},
			},
		},
		Users: UsersAndGroupsService{
			Drivers: LdapBasedService{
				Ldap: LdapSettings{
					BindPassword: revaServicePassword,
				},
			},
		},
		Graph: GraphService{
			Application: GraphApplication{
				ID: graphApplicationID,
			},
			Identity: LdapBasedService{
				Ldap: LdapSettings{
					BindPassword: idmServicePassword,
				},
			},
			ServiceAccount: serviceAccount,
		},
		Thumbnails: ThumbnailService{
			Thumbnail: ThumbnailSettings{
				TransferSecret: thumbnailsTransferSecret,
			},
		},
		Gateway: Gateway{
			StorageRegistry: StorageRegistry{
				StorageUsersMountID: storageUsersMountID,
			},
		},
		StorageUsers: StorageUsers{
			MountID:        storageUsersMountID,
			ServiceAccount: serviceAccount,
		},
		Userlog: Userlog{
			ServiceAccount: serviceAccount,
		},
		AuthService: AuthService{
			ServiceAccount: serviceAccount,
		},
		Search: Search{
			ServiceAccount: serviceAccount,
		},
		Notifications: Notifications{
			ServiceAccount: serviceAccount,
		},
		Frontend: FrontendService{
			ServiceAccount: serviceAccount,
		},
		Ocm: OcmService{
			ServiceAccount: serviceAccount,
		},
		Clientlog: Clientlog{
			ServiceAccount: serviceAccount,
		},
		Proxy: ProxyService{
			ServiceAccount: serviceAccount,
		},
		Settings: SettingsService{
			ServiceAccountIDs: []string{serviceAccount.ServiceAccountID},
		},
		Activitylog: Activitylog{
			ServiceAccount: serviceAccount,
		},
	}

	if insecure {
		cfg.AuthBearer = AuthbearerService{
			AuthProviders: AuthProviderSettings{Oidc: _insecureService},
		}
		cfg.Collaboration.App.Insecure = true
		cfg.Frontend.AppHandler = _insecureService
		cfg.Frontend.Archiver = _insecureService
		cfg.Frontend.OCDav = _insecureService
		cfg.Graph.Spaces = _insecureService
		cfg.Graph.Events = _insecureEvents
		cfg.Notifications.Notifications.Events = _insecureEvents
		cfg.Search.Events = _insecureEvents
		cfg.Audit.Events = _insecureEvents
		cfg.Sharing.Events = _insecureEvents
		cfg.StorageUsers.Events = _insecureEvents
		cfg.Nats.Nats.TLSSkipVerifyClientCert = true
		cfg.Proxy = ProxyService{
			InsecureBackends: true,
			OIDC: InsecureProxyOIDC{
				Insecure: true,
			},
			ServiceAccount: serviceAccount,
		}

		cfg.Thumbnails.Thumbnail.WebdavAllowInsecure = true
		cfg.Thumbnails.Thumbnail.Cs3AllowInsecure = true
	}
	yamlOutput, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("could not marshall config into yaml: %s", err)
	}
	if diff {
		return writePatch(configPath, yamlOutput)
	}
	return writeConfig(configPath, ocAdminServicePassword, targetBackupConfig, yamlOutput)
}
