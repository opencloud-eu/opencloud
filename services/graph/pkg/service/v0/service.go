package svc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	ldapv3 "github.com/go-ldap/ldap/v3"
	"github.com/jellydator/ttlcache/v3"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/identity/cache"
	"github.com/riandyrn/otelchi"
	microstore "go-micro.dev/v4/store"

	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/store"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/opencloud-eu/reva/v2/pkg/utils/ldap"

	ocldap "github.com/opencloud-eu/opencloud/pkg/ldap"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/pkg/roles"
	"github.com/opencloud-eu/opencloud/pkg/service/grpc"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/identity"
	graphm "github.com/opencloud-eu/opencloud/services/graph/pkg/middleware"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/unifiedrole"
)

const (
	// HeaderPurge defines the header name for the purge header.
	HeaderPurge     = "Purge"
	displayNameAttr = "displayName"
)

// Service defines the service handlers.
type Service interface { //nolint:interfacebloat
	ServeHTTP(w http.ResponseWriter, r *http.Request)

	ListApplications(w http.ResponseWriter, r *http.Request)
	GetApplication(w http.ResponseWriter, r *http.Request)

	GetMe(w http.ResponseWriter, r *http.Request)
	GetUsers(w http.ResponseWriter, r *http.Request)
	GetUser(w http.ResponseWriter, r *http.Request)
	PostUser(w http.ResponseWriter, r *http.Request)
	DeleteUser(w http.ResponseWriter, r *http.Request)
	PatchUser(w http.ResponseWriter, r *http.Request)
	ChangeOwnPassword(w http.ResponseWriter, r *http.Request)

	ListAppRoleAssignments(w http.ResponseWriter, r *http.Request)
	CreateAppRoleAssignment(w http.ResponseWriter, r *http.Request)
	DeleteAppRoleAssignment(w http.ResponseWriter, r *http.Request)

	GetGroups(w http.ResponseWriter, r *http.Request)
	GetGroup(w http.ResponseWriter, r *http.Request)
	PostGroup(w http.ResponseWriter, r *http.Request)
	PatchGroup(w http.ResponseWriter, r *http.Request)
	DeleteGroup(w http.ResponseWriter, r *http.Request)
	GetGroupMembers(w http.ResponseWriter, r *http.Request)
	PostGroupMember(w http.ResponseWriter, r *http.Request)
	DeleteGroupMember(w http.ResponseWriter, r *http.Request)

	GetEducationSchools(w http.ResponseWriter, r *http.Request)
	GetEducationSchool(w http.ResponseWriter, r *http.Request)
	PostEducationSchool(w http.ResponseWriter, r *http.Request)
	PatchEducationSchool(w http.ResponseWriter, r *http.Request)
	DeleteEducationSchool(w http.ResponseWriter, r *http.Request)
	GetEducationSchoolUsers(w http.ResponseWriter, r *http.Request)
	PostEducationSchoolUser(w http.ResponseWriter, r *http.Request)
	DeleteEducationSchoolUser(w http.ResponseWriter, r *http.Request)
	GetEducationSchoolClasses(w http.ResponseWriter, r *http.Request)
	PostEducationSchoolClass(w http.ResponseWriter, r *http.Request)
	DeleteEducationSchoolClass(w http.ResponseWriter, r *http.Request)

	GetEducationClasses(w http.ResponseWriter, r *http.Request)
	GetEducationClass(w http.ResponseWriter, r *http.Request)
	PostEducationClass(w http.ResponseWriter, r *http.Request)
	PatchEducationClass(w http.ResponseWriter, r *http.Request)
	DeleteEducationClass(w http.ResponseWriter, r *http.Request)
	GetEducationClassMembers(w http.ResponseWriter, r *http.Request)
	PostEducationClassMember(w http.ResponseWriter, r *http.Request)

	GetEducationUsers(w http.ResponseWriter, r *http.Request)
	GetEducationUser(w http.ResponseWriter, r *http.Request)
	PostEducationUser(w http.ResponseWriter, r *http.Request)
	DeleteEducationUser(w http.ResponseWriter, r *http.Request)
	PatchEducationUser(w http.ResponseWriter, r *http.Request)
	DeleteEducationClassMember(w http.ResponseWriter, r *http.Request)

	GetEducationClassTeachers(w http.ResponseWriter, r *http.Request)
	PostEducationClassTeacher(w http.ResponseWriter, r *http.Request)
	DeleteEducationClassTeacher(w http.ResponseWriter, r *http.Request)

	GetDrivesV1(w http.ResponseWriter, r *http.Request)
	GetDrivesV1Beta1(w http.ResponseWriter, r *http.Request)
	GetSingleDrive(w http.ResponseWriter, r *http.Request)
	GetAllDrivesV1(w http.ResponseWriter, r *http.Request)
	GetAllDrivesV1Beta1(w http.ResponseWriter, r *http.Request)
	CreateDrive(w http.ResponseWriter, r *http.Request)
	UpdateDrive(w http.ResponseWriter, r *http.Request)
	DeleteDrive(w http.ResponseWriter, r *http.Request)

	GetSharedByMe(w http.ResponseWriter, r *http.Request)
	ListSharedWithMe(w http.ResponseWriter, r *http.Request)

	GetRootDriveChildren(w http.ResponseWriter, r *http.Request)
	GetDriveItem(w http.ResponseWriter, r *http.Request)
	GetDriveItemChildren(w http.ResponseWriter, r *http.Request)

	CreateUploadSession(w http.ResponseWriter, r *http.Request)

	GetTags(w http.ResponseWriter, r *http.Request)
	AssignTags(w http.ResponseWriter, r *http.Request)
	UnassignTags(w http.ResponseWriter, r *http.Request)
}

// NewService returns a service implementation for Service.
func NewService(opts ...Option) (Graph, error) { //nolint:maintidx
	options := newOptions(opts...)

	m := chi.NewMux()
	m.Use(options.Middleware...)
	m.Use(
		otelchi.Middleware(
			"graph",
			otelchi.WithChiRoutes(m),
			otelchi.WithTracerProvider(options.TraceProvider),
			otelchi.WithPropagators(tracing.GetPropagator()),
		),
	)

	spacePropertiesCache := ttlcache.New(
		ttlcache.WithTTL[string, interface{}](
			time.Duration(options.Config.Spaces.ExtendedSpacePropertiesCacheTTL),
		),
		ttlcache.WithDisableTouchOnHit[string, interface{}](),
	)
	go spacePropertiesCache.Start()

	identityCache := cache.NewIdentityCache(
		cache.IdentityCacheWithGatewaySelector(options.GatewaySelector),
		cache.IdentityCacheWithUsersTTL(time.Duration(options.Config.Spaces.UsersCacheTTL)),
		cache.IdentityCacheWithGroupsTTL(time.Duration(options.Config.Spaces.GroupsCacheTTL)),
	)

	baseGraphService := BaseGraphService{
		logger:          &options.Logger,
		identityCache:   identityCache,
		gatewaySelector: options.GatewaySelector,
		config:          options.Config,
		availableRoles:  unifiedrole.GetRoles(unifiedrole.RoleFilterIDs(options.Config.UnifiedRoles.AvailableRoles...)),
	}

	drivesDriveItemService, err := NewDrivesDriveItemService(options.Logger, options.GatewaySelector)
	if err != nil {
		return Graph{}, err
	}

	drivesDriveItemApi, err := NewDrivesDriveItemApi(drivesDriveItemService, baseGraphService, options.Logger)
	if err != nil {
		return Graph{}, err
	}

	driveItemPermissionsService, err := NewDriveItemPermissionsService(options.Logger, options.GatewaySelector, identityCache, options.Config)
	if err != nil {
		return Graph{}, err
	}

	driveItemPermissionsApi, err := NewDriveItemPermissionsApi(driveItemPermissionsService, options.Logger, options.Config)
	if err != nil {
		return Graph{}, err
	}

	usersUserProfilePhotoApi, err := NewUsersUserProfilePhotoApi(options.UserProfilePhotoService, options.Logger)
	if err != nil {
		return Graph{}, err
	}

	svc := Graph{
		BaseGraphService:         baseGraphService,
		mux:                      m,
		specialDriveItemsCache:   spacePropertiesCache,
		eventsPublisher:          options.EventsPublisher,
		eventsConsumer:           options.EventsConsumer,
		searchService:            options.SearchService,
		identityEducationBackend: options.IdentityEducationBackend,
		keycloakClient:           options.KeycloakClient,
		historyClient:            options.EventHistoryClient,
		traceProvider:            options.TraceProvider,
		valueService:             options.ValueService,
		natskv:                   options.NatsKeyValue,
	}

	if err := setIdentityBackends(options, &svc); err != nil {
		return svc, err
	}

	if options.PermissionService == nil {
		grpcClient, err := grpc.NewClient(append(grpc.GetClientOptions(options.Config.GRPCClientTLS), grpc.WithTraceProvider(options.TraceProvider))...)
		if err != nil {
			return svc, err
		}
		svc.permissionsService = settingssvc.NewPermissionService("eu.opencloud.api.settings", grpcClient)
	} else {
		svc.permissionsService = options.PermissionService
	}

	svc.roleService = options.RoleService

	roleManager := options.RoleManager
	if roleManager == nil {
		storeOptions := []microstore.Option{
			store.Store(options.Config.Cache.Store),
			store.TTL(options.Config.Cache.TTL),
			microstore.Nodes(options.Config.Cache.Nodes...),
			microstore.Database(options.Config.Cache.Database),
			microstore.Table(options.Config.Cache.Table),
			store.DisablePersistence(options.Config.Cache.DisablePersistence),
			store.Authentication(options.Config.Cache.AuthUsername, options.Config.Cache.AuthPassword),
		}
		m := roles.NewManager(
			roles.StoreOptions(storeOptions),
			roles.Logger(options.Logger),
			roles.RoleService(options.RoleService),
		)
		roleManager = &m
	}

	var requireAdmin func(http.Handler) http.Handler
	if options.RequireAdminMiddleware == nil {
		requireAdmin = graphm.RequireAdmin(roleManager, options.Logger)
	} else {
		requireAdmin = options.RequireAdminMiddleware
	}

	m.Route(options.Config.HTTP.Root, func(r chi.Router) {
		r.Use(middleware.StripSlashes)

		r.Route("/v1beta1", func(r chi.Router) {
			r.Route("/me", func(r chi.Router) {
				r.Get("/drives", svc.GetDrives(APIVersion_1_Beta_1))
				r.Route("/drive", func(r chi.Router) {
					r.Get("/sharedByMe", svc.GetSharedByMe)
					r.Get("/sharedWithMe", svc.ListSharedWithMe)
				})
			})
			r.Route("/drives", func(r chi.Router) {
				r.Get("/", svc.GetAllDrives(APIVersion_1_Beta_1))
				r.Route("/{driveID}", func(r chi.Router) {
					r.Route("/root", func(r chi.Router) {
						r.Post("/children", drivesDriveItemApi.CreateDriveItem)
						r.Post("/invite", driveItemPermissionsApi.SpaceRootInvite)
						r.Post("/createLink", driveItemPermissionsApi.CreateSpaceRootLink)
						r.Route("/permissions", func(r chi.Router) {
							r.Get("/", driveItemPermissionsApi.ListSpaceRootPermissions)
							r.Route("/{permissionID}", func(r chi.Router) {
								r.Delete("/", driveItemPermissionsApi.DeleteSpaceRootPermission)
								r.Patch("/", driveItemPermissionsApi.UpdateSpaceRootPermission)
								r.Post("/setPassword", driveItemPermissionsApi.SetSpaceRootLinkPassword)
							})
						})
					})
					r.Route("/items/{itemID}", func(r chi.Router) {
						r.Get("/", drivesDriveItemApi.GetDriveItem)
						r.Patch("/", drivesDriveItemApi.UpdateDriveItem)
						r.Delete("/", drivesDriveItemApi.DeleteDriveItem)
						r.Post("/invite", driveItemPermissionsApi.Invite)
						r.Post("/createLink", driveItemPermissionsApi.CreateLink)
						r.Route("/permissions", func(r chi.Router) {
							r.Get("/", driveItemPermissionsApi.ListPermissions)
							r.Route("/{permissionID}", func(r chi.Router) {
								r.Delete("/", driveItemPermissionsApi.DeletePermission)
								r.Patch("/", driveItemPermissionsApi.UpdatePermission)
								r.Post("/setPassword", driveItemPermissionsApi.SetLinkPassword)
							})
						})
					})
				})
			})
			r.Route("/roleManagement/permissions/roleDefinitions", func(r chi.Router) {
				r.Get("/", svc.GetRoleDefinitions)
				r.Get("/{roleID}", svc.GetRoleDefinition)
			})
		})
		r.Route("/v1.0", func(r chi.Router) {
			r.Route("/extensions/org.libregraph", func(r chi.Router) {
				r.Get("/tags", svc.GetTags)
				r.Put("/tags", svc.AssignTags)
				r.Delete("/tags", svc.UnassignTags)
			})
			r.Route("/applications", func(r chi.Router) {
				r.Get("/", svc.ListApplications)
				r.Get("/{applicationID}", svc.GetApplication)
			})
			r.Route("/me", func(r chi.Router) {
				r.Get("/", svc.GetMe)
				r.Patch("/", svc.PatchMe)
				r.Route("/drive", func(r chi.Router) {
					r.Get("/", svc.GetUserDrive)
					r.Get("/root/children", svc.GetRootDriveChildren)
				})
				r.Get("/drives", svc.GetDrives(APIVersion_1))
				r.Post("/changePassword", svc.ChangeOwnPassword)
				r.Route("/photo/$value", func(r chi.Router) {
					r.Get("/", usersUserProfilePhotoApi.GetProfilePhoto(GetUserIDFromCTX))
					r.Put("/", usersUserProfilePhotoApi.UpsertProfilePhoto(GetUserIDFromCTX))
					r.Patch("/", usersUserProfilePhotoApi.UpsertProfilePhoto(GetUserIDFromCTX))
					r.Delete("/", usersUserProfilePhotoApi.DeleteProfilePhoto(GetUserIDFromCTX))
				})
			})
			r.Route("/users", func(r chi.Router) {
				r.Get("/", svc.GetUsers)
				r.With(requireAdmin).Post("/", svc.PostUser)
				r.Route("/{userID}", func(r chi.Router) {
					r.Get("/", svc.GetUser)
					r.Get("/drive", svc.GetUserDrive)
					r.Post("/exportPersonalData", svc.ExportPersonalData)
					r.Route("/photo/$value", func(r chi.Router) {
						r.Get("/", usersUserProfilePhotoApi.GetProfilePhoto(GetSlugValue("userID")))
					})
					r.With(requireAdmin).Delete("/", svc.DeleteUser)
					r.With(requireAdmin).Patch("/", svc.PatchUser)
					if svc.roleService != nil {
						r.With(requireAdmin).Route("/appRoleAssignments", func(r chi.Router) {
							r.Get("/", svc.ListAppRoleAssignments)
							r.Post("/", svc.CreateAppRoleAssignment)
							r.Delete("/{appRoleAssignmentID}", svc.DeleteAppRoleAssignment)
						})
					}
				})
			})
			r.Route("/groups", func(r chi.Router) {
				r.Get("/", svc.GetGroups)
				r.With(requireAdmin).Post("/", svc.PostGroup)
				r.Route("/{groupID}", func(r chi.Router) {
					r.Get("/", svc.GetGroup)
					r.With(requireAdmin).Delete("/", svc.DeleteGroup)
					r.With(requireAdmin).Patch("/", svc.PatchGroup)
					r.Route("/members", func(r chi.Router) {
						r.With(requireAdmin).Get("/", svc.GetGroupMembers)
						r.With(requireAdmin).Post("/$ref", svc.PostGroupMember)
						r.With(requireAdmin).Delete("/{memberID}/$ref", svc.DeleteGroupMember)
					})
				})
			})
			r.Route("/drives", func(r chi.Router) {
				r.Get("/", svc.GetAllDrives(APIVersion_1))
				r.Post("/", svc.CreateDrive)
				r.Route("/{driveID}", func(r chi.Router) {
					r.Patch("/", svc.UpdateDrive)
					r.Get("/", svc.GetSingleDrive)
					r.Delete("/", svc.DeleteDrive)
					r.Route("/items/{driveItemID}", func(r chi.Router) {
						r.Get("/", svc.GetDriveItem)
						r.Get("/children", svc.GetDriveItemChildren)
						r.Post("/createUploadSession", svc.CreateUploadSession)
					})
				})
			})
			r.With(requireAdmin).Route("/education", func(r chi.Router) {
				r.Route("/schools", func(r chi.Router) {
					r.Get("/", svc.GetEducationSchools)
					r.Post("/", svc.PostEducationSchool)
					r.Route("/{schoolID}", func(r chi.Router) {
						r.Get("/", svc.GetEducationSchool)
						r.Delete("/", svc.DeleteEducationSchool)
						r.Patch("/", svc.PatchEducationSchool)
						r.Route("/users", func(r chi.Router) {
							r.Get("/", svc.GetEducationSchoolUsers)
							r.Post("/$ref", svc.PostEducationSchoolUser)
							r.Delete("/{userID}/$ref", svc.DeleteEducationSchoolUser)
						})
						r.Route("/classes", func(r chi.Router) {
							r.Get("/", svc.GetEducationSchoolClasses)
							r.Post("/$ref", svc.PostEducationSchoolClass)
							r.Delete("/{classID}/$ref", svc.DeleteEducationSchoolClass)
						})
					})
				})
				r.Route("/users", func(r chi.Router) {
					r.Get("/", svc.GetEducationUsers)
					r.Post("/", svc.PostEducationUser)
					r.Route("/{userID}", func(r chi.Router) {
						r.Get("/", svc.GetEducationUser)
						r.Delete("/", svc.DeleteEducationUser)
						r.Patch("/", svc.PatchEducationUser)
					})
				})
				r.Route("/classes", func(r chi.Router) {
					r.Get("/", svc.GetEducationClasses)
					r.Post("/", svc.PostEducationClass)
					r.Route("/{classID}", func(r chi.Router) {
						r.Get("/", svc.GetEducationClass)
						r.Delete("/", svc.DeleteEducationClass)
						r.Patch("/", svc.PatchEducationClass)
						r.Route("/members", func(r chi.Router) {
							r.Get("/", svc.GetEducationClassMembers)
							r.Post("/$ref", svc.PostEducationClassMember)
							r.Delete("/{memberID}/$ref", svc.DeleteEducationClassMember)
						})
						r.Route("/teachers", func(r chi.Router) {
							r.Get("/", svc.GetEducationClassTeachers)
							r.Post("/$ref", svc.PostEducationClassTeacher)
							r.Delete("/{teacherID}/$ref", svc.DeleteEducationClassTeacher)
						})
					})
				})
			})
		})
	})

	_ = chi.Walk(m, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		options.Logger.Debug().Str("method", method).Str("route", route).Int("middlewares", len(middlewares)).Msg("serving endpoint")
		return nil
	})

	return svc, nil
}

func setIdentityBackends(options Options, svc *Graph) error {
	if options.IdentityBackend == nil {
		switch options.Config.Identity.Backend {
		case "cs3":
			gatewaySelector, err := pool.GatewaySelector(
				options.Config.Reva.Address,
				append(
					options.Config.Reva.GetRevaOptions(),
					pool.WithRegistry(registry.GetRegistry()),
					pool.WithTracerProvider(options.TraceProvider),
				)...,
			)
			if err != nil {
				return err
			}

			svc.identityBackend = &identity.CS3{
				Config:          options.Config.Reva,
				Logger:          &options.Logger,
				GatewaySelector: gatewaySelector,
			}
		case "ldap":
			var err error

			var tlsConf *tls.Config
			if options.Config.Identity.LDAP.Insecure {

				// When insecure is set to true then we don't need a certificate.
				options.Config.Identity.LDAP.CACert = ""
				tlsConf = &tls.Config{
					MinVersion: tls.VersionTLS12,

					//nolint:gosec // We need the ability to run with "insecure" (dev/testing)
					InsecureSkipVerify: options.Config.Identity.LDAP.Insecure,
				}
			}

			if options.Config.Identity.LDAP.CACert != "" {
				if err := ocldap.WaitForCA(options.Logger,
					options.Config.Identity.LDAP.Insecure,
					options.Config.Identity.LDAP.CACert); err != nil {
					options.Logger.Fatal().Err(err).Msg("The configured LDAP CA cert does not exist")
				}
				if tlsConf == nil {
					tlsConf = &tls.Config{
						MinVersion: tls.VersionTLS12,
					}
				}
				certs := x509.NewCertPool()
				pemData, err := os.ReadFile(options.Config.Identity.LDAP.CACert)
				if err != nil {
					options.Logger.Error().Err(err).Msg("Error initializing LDAP Backend")
					return err
				}
				if !certs.AppendCertsFromPEM(pemData) {
					options.Logger.Error().Msg("Error initializing LDAP Backend. Adding CA cert failed")
					return err
				}
				tlsConf.RootCAs = certs
			}

			conn := ldap.NewLDAPWithReconnect(
				ldap.Config{
					URI:          options.Config.Identity.LDAP.URI,
					BindDN:       options.Config.Identity.LDAP.BindDN,
					BindPassword: options.Config.Identity.LDAP.BindPassword,
					TLSConfig:    tlsConf,
				},
			)
			conn.SetLogger(&options.Logger.Logger)
			lb, err := identity.NewLDAPBackend(conn, options.Config.Identity.LDAP, &options.Logger)
			if err != nil {
				options.Logger.Error().Err(err).Msg("Error initializing LDAP Backend")
				return err
			}
			svc.identityBackend = lb
			if options.IdentityEducationBackend == nil {
				if options.Config.Identity.LDAP.EducationResourcesEnabled {
					svc.identityEducationBackend = lb
				} else {
					errEduBackend := &identity.ErrEducationBackend{}
					svc.identityEducationBackend = errEduBackend
				}
			}

			disableMechanismType, err := identity.ParseDisableMechanismType(options.Config.Identity.LDAP.DisableUserMechanism)
			if err != nil {
				options.Logger.Error().Err(err).Msg("Error initializing LDAP Backend")
				return err
			}

			if disableMechanismType == identity.DisableMechanismGroup {
				options.Logger.Info().Msg("LocalUserDisable is true, will create group if not exists")
				err := lb.CreateLDAPGroupByDN(options.Config.Identity.LDAP.LdapDisabledUsersGroupDN)
				if err != nil {
					isAnError := false
					var lerr *ldapv3.Error
					if errors.As(err, &lerr) {
						if lerr.ResultCode != ldapv3.LDAPResultEntryAlreadyExists {
							isAnError = true
						}
					} else {
						isAnError = true
					}

					if isAnError {
						msg := "error adding group for disabling users"
						options.Logger.Error().Err(err).Str("local_user_disable", options.Config.Identity.LDAP.LdapDisabledUsersGroupDN).Msg(msg)
						return err
					}
				}
			}

		default:
			err := fmt.Errorf("unknown identity backend: '%s'", options.Config.Identity.Backend)
			options.Logger.Err(err)
			return err
		}
	} else {
		svc.identityBackend = options.IdentityBackend
	}

	return svc.StartListenForLogonEvents(options.Context, options.Logger)
}

func (g *Graph) StartListenForLogonEvents(ctx context.Context, l log.Logger) error {
	if g.eventsConsumer == nil {
		return nil
	}
	var _registeredEvents = []events.Unmarshaller{
		events.UserSignedIn{},
	}
	evChannel, err := events.Consume(g.eventsConsumer, "graph", _registeredEvents...)
	if err != nil {
		l.Error().Err(err).Msg("cannot consume from nats")
		return err
	}
	go func() {
		for loop := true; loop; {
			select {
			case e := <-evChannel:
				switch ev := e.Event.(type) {
				default:
					l.Error().Interface("event", e).Msg("unhandled event")
				case events.UserSignedIn:
					if err := g.identityBackend.UpdateLastSignInDate(ctx, ev.Executant.OpaqueId, utils.TSToTime(ev.Timestamp)); err != nil {
						l.Error().Err(err).Str("userid", ev.Executant.OpaqueId).Msg("Error updating last sign in date")
					}
				}
			case <-ctx.Done():
				l.Info().Msg("context cancelled")
				loop = false
			}
		}
	}()
	return nil
}

// parseHeaderPurge parses the 'Purge' header.
// '1', 't', 'T', 'TRUE', 'true', 'True' are parsed as true
// all other values are false.
func parsePurgeHeader(h http.Header) bool {
	val := h.Get(HeaderPurge)

	if b, err := strconv.ParseBool(val); err == nil {
		return b
	}
	return false
}
