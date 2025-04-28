package svc

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/mux"
	"github.com/libregraph/lico/bootstrap"
	guestBackendSupport "github.com/libregraph/lico/bootstrap/backends/guest"
	ldapBackendSupport "github.com/libregraph/lico/bootstrap/backends/ldap"
	libreGraphBackendSupport "github.com/libregraph/lico/bootstrap/backends/libregraph"
	licoconfig "github.com/libregraph/lico/config"
	"github.com/libregraph/lico/server"
	"github.com/opencloud-eu/opencloud/pkg/ldap"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/services/idp/pkg/assets"
	cs3BackendSupport "github.com/opencloud-eu/opencloud/services/idp/pkg/backends/cs3/bootstrap"
	"github.com/opencloud-eu/opencloud/services/idp/pkg/config"
	"github.com/opencloud-eu/opencloud/services/idp/pkg/middleware"
	"github.com/riandyrn/otelchi"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/yaml.v2"
	"stash.kopano.io/kgol/rndm"
)

// Service defines the service handlers.
type Service interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}

// NewService returns a service implementation for Service.
func NewService(opts ...Option) Service {
	ctx := context.Background()
	options := newOptions(opts...)
	logger := options.Logger.Logger
	assetVFS := assets.New(
		assets.Logger(options.Logger),
		assets.Config(options.Config),
	)

	if err := createTemporaryClientsConfig(
		options.Config.IDP.IdentifierRegistrationConf,
		options.Config.Commons.OpenCloudURL,
		options.Config.Clients,
	); err != nil {
		logger.Fatal().Err(err).Msg("could not create default config")
	}

	switch options.Config.IDP.IdentityManager {
	case "cs3":
		cs3BackendSupport.MustRegister()
		if err := initCS3EnvVars(options.Config.Reva.Address, options.Config.MachineAuthAPIKey); err != nil {
			logger.Fatal().Err(err).Msg("could not initialize cs3 backend env vars")
		}
	case "ldap":

		if err := ldap.WaitForCA(options.Logger, options.Config.IDP.Insecure, options.Config.Ldap.TLSCACert); err != nil {
			logger.Fatal().Err(err).Msg("The configured LDAP CA cert does not exist")
		}
		if options.Config.IDP.Insecure {
			// force CACert to be empty to avoid lico try to load it
			options.Config.Ldap.TLSCACert = ""
		}

		ldapBackendSupport.MustRegister()
		if err := initLicoInternalLDAPEnvVars(&options.Config.Ldap); err != nil {
			logger.Fatal().Err(err).Msg("could not initialize ldap env vars")
		}
	default:
		guestBackendSupport.MustRegister()
		libreGraphBackendSupport.MustRegister()
	}

	idpSettings := bootstrap.Settings{
		Iss:                               options.Config.IDP.Iss,
		IdentityManager:                   options.Config.IDP.IdentityManager,
		URIBasePath:                       options.Config.IDP.URIBasePath,
		SignInURI:                         options.Config.IDP.SignInURI,
		SignedOutURI:                      options.Config.IDP.SignedOutURI,
		AuthorizationEndpointURI:          options.Config.IDP.AuthorizationEndpointURI,
		EndsessionEndpointURI:             options.Config.IDP.EndsessionEndpointURI,
		Insecure:                          options.Config.IDP.Insecure,
		TrustedProxy:                      options.Config.IDP.TrustedProxy,
		AllowScope:                        options.Config.IDP.AllowScope,
		AllowClientGuests:                 options.Config.IDP.AllowClientGuests,
		AllowDynamicClientRegistration:    options.Config.IDP.AllowDynamicClientRegistration,
		EncryptionSecretFile:              options.Config.IDP.EncryptionSecretFile,
		Listen:                            options.Config.IDP.Listen,
		IdentifierClientDisabled:          options.Config.IDP.IdentifierClientDisabled,
		IdentifierClientPath:              options.Config.IDP.IdentifierClientPath,
		IdentifierRegistrationConf:        options.Config.IDP.IdentifierRegistrationConf,
		IdentifierScopesConf:              options.Config.IDP.IdentifierScopesConf,
		IdentifierDefaultBannerLogo:       options.Config.IDP.IdentifierDefaultBannerLogo,
		IdentifierDefaultSignInPageText:   options.Config.IDP.IdentifierDefaultSignInPageText,
		IdentifierDefaultLogoTargetURI:    options.Config.IDP.IdentifierDefaultLogoTargetURI,
		IdentifierDefaultUsernameHintText: options.Config.IDP.IdentifierDefaultUsernameHintText,
		IdentifierUILocales:               options.Config.IDP.IdentifierUILocales,
		SigningKid:                        options.Config.IDP.SigningKid,
		SigningMethod:                     options.Config.IDP.SigningMethod,
		SigningPrivateKeyFiles:            options.Config.IDP.SigningPrivateKeyFiles,
		ValidationKeysPath:                options.Config.IDP.ValidationKeysPath,
		CookieBackendURI:                  options.Config.IDP.CookieBackendURI,
		CookieNames:                       options.Config.IDP.CookieNames,
		CookieSameSite:                    options.Config.IDP.CookieSameSite,
		AccessTokenDurationSeconds:        options.Config.IDP.AccessTokenDurationSeconds,
		IDTokenDurationSeconds:            options.Config.IDP.IDTokenDurationSeconds,
		RefreshTokenDurationSeconds:       options.Config.IDP.RefreshTokenDurationSeconds,
		DyamicClientSecretDurationSeconds: options.Config.IDP.DynamicClientSecretDurationSeconds,
	}
	bs, err := bootstrap.Boot(ctx, &idpSettings, &licoconfig.Config{
		Logger: log.LogrusWrap(logger),
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("could not bootstrap idp")
	}

	managers := bs.Managers()
	routes := []server.WithRoutes{managers.Must("identity").(server.WithRoutes)}
	handlers := managers.Must("handler").(http.Handler)

	svc := IDP{
		logger: options.Logger,
		config: options.Config,
		assets: assetVFS,
		tp:     options.TraceProvider,
	}

	svc.initMux(ctx, routes, handlers, options)

	return svc
}

type temporaryClientConfig struct {
	Clients []config.Client `yaml:"clients"`
}

func createTemporaryClientsConfig(filePath, openCloudURL string, clients []config.Client) error {
	folder := path.Dir(filePath)
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		if err := os.MkdirAll(folder, 0o700); err != nil {
			return err
		}
	}

	for i, client := range clients {

		for i, entry := range client.RedirectURIs {
			client.RedirectURIs[i] = strings.ReplaceAll(entry, "{{OC_URL}}", strings.TrimRight(openCloudURL, "/"))
		}
		for i, entry := range client.Origins {
			client.Origins[i] = strings.ReplaceAll(entry, "{{OC_URL}}", strings.TrimRight(openCloudURL, "/"))
		}
		clients[i] = client
	}

	c := temporaryClientConfig{
		Clients: clients,
	}

	conf, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	confOnDisk, err := os.Create(filePath)
	if err != nil {
		return err
	}

	defer confOnDisk.Close()

	err = os.WriteFile(filePath, conf, 0o600)
	if err != nil {
		return err
	}

	return nil
}

// Init cs3 backend vars which are currently not accessible via idp api
func initCS3EnvVars(cs3Addr, machineAuthAPIKey string) error {
	defaults := map[string]string{
		"CS3_GATEWAY":              cs3Addr,
		"CS3_MACHINE_AUTH_API_KEY": machineAuthAPIKey,
	}

	for k, v := range defaults {
		if err := os.Setenv(k, v); err != nil {
			return fmt.Errorf("could not set cs3 env var %s=%s", k, v)
		}
	}

	return nil
}

// Init ldap backend vars which are currently not accessible via idp api
func initLicoInternalLDAPEnvVars(ldap *config.Ldap) error {
	filter := fmt.Sprintf("(objectclass=%s)", ldap.ObjectClass)

	var needsAnd bool
	if ldap.Filter != "" {
		filter += ldap.Filter
		needsAnd = true
	}

	if ldap.UserEnabledAttribute != "" {
		// Using a (!(enabled=FALSE)) filter here to allow user without
		// any value for the enable flag to log in
		filter += fmt.Sprintf("(!(%s=FALSE))", ldap.UserEnabledAttribute)
		needsAnd = true
	}

	if needsAnd {
		filter = fmt.Sprintf("(&%s)", filter)
	}

	defaults := map[string]string{
		"LDAP_URI":                 ldap.URI,
		"LDAP_BINDDN":              ldap.BindDN,
		"LDAP_BINDPW":              ldap.BindPassword,
		"LDAP_BASEDN":              ldap.BaseDN,
		"LDAP_SCOPE":               ldap.Scope,
		"LDAP_LOGIN_ATTRIBUTE":     ldap.LoginAttribute,
		"LDAP_EMAIL_ATTRIBUTE":     ldap.EmailAttribute,
		"LDAP_NAME_ATTRIBUTE":      ldap.NameAttribute,
		"LDAP_UUID_ATTRIBUTE":      ldap.UUIDAttribute,
		"LDAP_SUB_ATTRIBUTES":      ldap.UUIDAttribute,
		"LDAP_UUID_ATTRIBUTE_TYPE": ldap.UUIDAttributeType,
		"LDAP_FILTER":              filter,
	}

	if ldap.TLSCACert != "" {
		defaults["LDAP_TLS_CACERT"] = ldap.TLSCACert
	}

	for k, v := range defaults {
		if err := os.Setenv(k, v); err != nil {
			return fmt.Errorf("could not set ldap env var %s=%s", k, v)
		}
	}

	return nil
}

// IDP defines implements the business logic for Service.
type IDP struct {
	logger log.Logger
	config *config.Config
	mux    *chi.Mux
	assets http.FileSystem
	tp     trace.TracerProvider
}

// initMux initializes the internal idp gorilla mux and mounts it in to an OpenCloud chi-router
func (idp *IDP) initMux(ctx context.Context, r []server.WithRoutes, h http.Handler, options Options) {
	gm := mux.NewRouter()
	for _, route := range r {
		route.AddRoutes(ctx, gm)
	}

	// Delegate rest to provider which is also a handler.
	if h != nil {
		gm.NotFoundHandler = h
	}

	idp.mux = chi.NewMux()
	idp.mux.Use(options.Middleware...)

	idp.mux.Use(middleware.Static(
		"/signin/v1/",
		assets.New(
			assets.Logger(options.Logger),
			assets.Config(options.Config),
		),
		idp.tp,
	))

	idp.mux.Use(
		otelchi.Middleware(
			"idp",
			otelchi.WithChiRoutes(idp.mux),
			otelchi.WithTracerProvider(idp.tp),
			otelchi.WithPropagators(tracing.GetPropagator()),
		),
	)

	// handle / | index.html with a template that needs to have the BASE_PREFIX replaced
	idp.mux.Get("/signin/v1/identifier", idp.Index())
	idp.mux.Get("/signin/v1/identifier/", idp.Index())
	idp.mux.Get("/signin/v1/identifier/index.html", idp.Index())

	idp.mux.Mount("/", gm)

	_ = chi.Walk(idp.mux, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		options.Logger.Debug().Str("method", method).Str("route", route).Int("middlewares", len(middlewares)).Msg("serving endpoint")
		return nil
	})
}

// ServeHTTP implements the Service interface.
func (idp IDP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	idp.mux.ServeHTTP(w, r)
}

// Index renders the static html with templated variables.
func (idp IDP) Index() http.HandlerFunc {
	f, err := idp.assets.Open("/identifier/index.html")
	if err != nil {
		idp.logger.Fatal().Err(err).Msg("Could not open index template")
	}

	template, err := io.ReadAll(f)
	if err != nil {
		idp.logger.Fatal().Err(err).Msg("Could not read index template")
	}
	if err = f.Close(); err != nil {
		idp.logger.Fatal().Err(err).Msg("Could not close body")
	}

	// TODO add environment variable to make the path prefix configurable
	pp := "/signin/v1"
	indexHTML := bytes.Replace(template, []byte("__PATH_PREFIX__"), []byte(pp), 1)

	background := idp.config.Asset.LoginBackgroundUrl
	indexHTML = bytes.Replace(template, []byte("__BG_IMG_URL__"), []byte(background), 1)

	nonce := rndm.GenerateRandomString(32)
	indexHTML = bytes.Replace(indexHTML, []byte("__CSP_NONCE__"), []byte(nonce), 1)

	indexHTML = bytes.Replace(indexHTML, []byte("__PASSWORD_RESET_LINK__"), []byte(idp.config.Service.PasswordResetURI), 1)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(indexHTML); err != nil {
			idp.logger.Error().Err(err).Msg("could not write to response writer")
		}
	})
}
