package defaults

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/opencloud-eu/opencloud/services/idp/pkg/config"
)

// FullDefaultConfig returns a fully initialized default configuration
func FullDefaultConfig() *config.Config {
	cfg := DefaultConfig()
	EnsureDefaults(cfg)
	Sanitize(cfg)
	return cfg
}

// DefaultConfig returns a basic default configuration
func DefaultConfig() *config.Config {
	return &config.Config{
		Debug: config.Debug{
			Addr: "127.0.0.1:9134",
		},
		HTTP: config.HTTP{
			Addr:      "127.0.0.1:9130",
			Root:      "/",
			Namespace: "eu.opencloud.web",
			TLSCert:   filepath.Join(defaults.BaseDataPath(), "idp", "server.crt"),
			TLSKey:    filepath.Join(defaults.BaseDataPath(), "idp", "server.key"),
			TLS:       false,
		},
		Reva: shared.DefaultRevaConfig(),
		Service: config.Service{
			Name: "idp",
		},
		IDP: config.Settings{
			Iss:                                "https://localhost:9200",
			IdentityManager:                    "ldap",
			URIBasePath:                        "",
			SignInURI:                          "",
			SignedOutURI:                       "",
			AuthorizationEndpointURI:           "",
			EndsessionEndpointURI:              "",
			Insecure:                           false,
			TrustedProxy:                       nil,
			AllowScope:                         nil,
			AllowClientGuests:                  false,
			AllowDynamicClientRegistration:     false,
			EncryptionSecretFile:               filepath.Join(defaults.BaseDataPath(), "idp", "encryption.key"),
			Listen:                             "",
			IdentifierClientDisabled:           true,
			IdentifierClientPath:               filepath.Join(defaults.BaseDataPath(), "idp"),
			IdentifierRegistrationConf:         filepath.Join(defaults.BaseDataPath(), "idp", "tmp", "identifier-registration.yaml"),
			IdentifierScopesConf:               "",
			IdentifierDefaultBannerLogo:        "",
			IdentifierDefaultSignInPageText:    "",
			IdentifierDefaultLogoTargetURI:     "https://opencloud.eu",
			IdentifierDefaultUsernameHintText:  "",
			SigningKid:                         "private-key",
			SigningMethod:                      "PS256",
			SigningPrivateKeyFiles:             []string{filepath.Join(defaults.BaseDataPath(), "idp", "private-key.pem")},
			ValidationKeysPath:                 "",
			CookieBackendURI:                   "",
			CookieNames:                        nil,
			CookieSameSite:                     http.SameSiteStrictMode,
			AccessTokenDurationSeconds:         60 * 5,            // 5 minutes
			IDTokenDurationSeconds:             60 * 5,            // 5 minutes
			RefreshTokenDurationSeconds:        60 * 60 * 24 * 30, // 30 days
			DynamicClientSecretDurationSeconds: 0,
		},
		Clients: []config.Client{
			{
				ID:      "web",
				Name:    "OpenCloud Web App",
				Trusted: true,
				RedirectURIs: []string{
					"{{OC_URL}}/",
					"{{OC_URL}}/oidc-callback.html",
					"{{OC_URL}}/oidc-silent-redirect.html",
				},
				Origins: []string{
					"{{OC_URL}}",
				},
			},
			{
				ID:              "OpenCloudDesktop",
				Name:            "OpenCloud Desktop Client",
				ApplicationType: "native",
				RedirectURIs: []string{
					"http://127.0.0.1",
					"http://localhost",
				},
			},
			{
				ID:              "OpenCloudAndroid",
				Name:            "OpenCloud Android App",
				ApplicationType: "native",
				RedirectURIs: []string{
					"oc://android.opencloud.eu",
				},
				PostLogoutRedirectURIs: []string{
					"oc://android.opencloud.eu",
				},
			},
			{
				ID:              "OpenCloudIOS",
				Name:            "OpenCloud iOS App",
				ApplicationType: "native",
				RedirectURIs: []string{
					"oc://ios.opencloud.eu",
				},
				PostLogoutRedirectURIs: []string{
					"oc://ios.opencloud.eu",
				},
			},
		},
		Ldap: config.Ldap{
			URI:                  "ldaps://localhost:9235",
			TLSCACert:            filepath.Join(defaults.BaseDataPath(), "idm", "ldap.crt"),
			BindDN:               "uid=idp,ou=sysusers,o=libregraph-idm",
			BaseDN:               "ou=users,o=libregraph-idm",
			Scope:                "sub",
			LoginAttribute:       "uid",
			EmailAttribute:       "mail",
			NameAttribute:        "displayName",
			UUIDAttribute:        "openCloudUUID",
			UUIDAttributeType:    "text",
			Filter:               "",
			ObjectClass:          "inetOrgPerson",
			UserEnabledAttribute: "openCloudUserEnabled",
		},
	}
}

// EnsureDefaults adds default values to the configuration if they are not set yet
func EnsureDefaults(cfg *config.Config) {
	// provide with defaults for shared logging, since we need a valid destination address for "envdecode".
	if cfg.Log == nil && cfg.Commons != nil && cfg.Commons.Log != nil {
		cfg.Log = &config.Log{
			Level:  cfg.Commons.Log.Level,
			Pretty: cfg.Commons.Log.Pretty,
			Color:  cfg.Commons.Log.Color,
			File:   cfg.Commons.Log.File,
		}
	} else if cfg.Log == nil {
		cfg.Log = &config.Log{}
	}

	if cfg.Reva == nil && cfg.Commons != nil {
		cfg.Reva = structs.CopyOrZeroValue(cfg.Commons.Reva)
	}

	if cfg.MachineAuthAPIKey == "" && cfg.Commons != nil && cfg.Commons.MachineAuthAPIKey != "" {
		cfg.MachineAuthAPIKey = cfg.Commons.MachineAuthAPIKey
	}
}

// Sanitize sanitizes the configuration
func Sanitize(cfg *config.Config) {
	// sanitize config
	if cfg.HTTP.Root != "/" {
		cfg.HTTP.Root = strings.TrimSuffix(cfg.HTTP.Root, "/")
	}
}
