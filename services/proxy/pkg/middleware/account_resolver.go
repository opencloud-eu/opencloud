package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/router"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/user/backend"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/userroles"
	"go.opentelemetry.io/otel/trace"

	cs3user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// AccountResolver provides a middleware which mints a jwt and adds it to the proxied request based
// on the oidc-claims
func AccountResolver(optionSetters ...Option) func(next http.Handler) http.Handler {
	options := newOptions(optionSetters...)
	logger := options.Logger
	tracer := getTraceProvider(options).Tracer("proxy.middleware.account_resolver")

	lastGroupSyncCache := ttlcache.New(
		ttlcache.WithTTL[string, struct{}](5*time.Minute),
		ttlcache.WithDisableTouchOnHit[string, struct{}](),
	)
	go lastGroupSyncCache.Start()

	return func(next http.Handler) http.Handler {
		return &accountResolver{
			next:                  next,
			logger:                logger,
			tracer:                tracer,
			userProvider:          options.UserProvider,
			userOIDCClaim:         options.UserOIDCClaim,
			userCS3Claim:          options.UserCS3Claim,
			userRoleAssigner:      options.UserRoleAssigner,
			autoProvisionAccounts: options.AutoprovisionAccounts,
			multiTenantEnabled:    options.MultiTenantEnabled,
			lastGroupSyncCache:    lastGroupSyncCache,
			eventsPublisher:       options.EventsPublisher,
		}
	}
}

type accountResolver struct {
	next                  http.Handler
	logger                log.Logger
	tracer                trace.Tracer
	userProvider          backend.UserBackend
	userRoleAssigner      userroles.UserRoleAssigner
	autoProvisionAccounts bool
	multiTenantEnabled    bool
	userOIDCClaim         string
	userCS3Claim          string
	// lastGroupSyncCache is used to keep track of when the last sync of group
	// memberships was done for a specific user. This is used to trigger a sync
	// with every single request.
	lastGroupSyncCache *ttlcache.Cache[string, struct{}]
	eventsPublisher    events.Publisher
}

func readUserIDClaim(path string, claims map[string]interface{}) (string, error) {
	// happy path
	value, _ := claims[path].(string)
	if value != "" {
		return value, nil
	}

	// try splitting path at .
	segments := oidc.SplitWithEscaping(path, ".", "\\")
	subclaims := claims
	lastSegment := len(segments) - 1
	for i := range segments {
		if i < lastSegment {
			if castedClaims, ok := subclaims[segments[i]].(map[string]interface{}); ok {
				subclaims = castedClaims
			} else if castedClaims, ok := subclaims[segments[i]].(map[interface{}]interface{}); ok {
				subclaims = make(map[string]interface{}, len(castedClaims))
				for k, v := range castedClaims {
					if s, ok := k.(string); ok {
						subclaims[s] = v
					} else {
						return "", fmt.Errorf("could not walk claims path, key '%v' is not a string", k)
					}
				}
			}
		} else {
			if value, _ = subclaims[segments[i]].(string); value != "" {
				return value, nil
			}
		}
	}

	return value, fmt.Errorf("claim path '%s' not set or empty", path)
}

// TODO do not use the context to store values: https://medium.com/@cep21/how-to-correctly-use-context-context-in-go-1-7-8f2c0fafdf39
func (m accountResolver) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx, span := m.tracer.Start(req.Context(), fmt.Sprintf("%s %s", req.Method, req.URL.Path), trace.WithSpanKind(trace.SpanKindServer))
	claims := oidc.FromContext(ctx)
	user, ok := revactx.ContextGetUser(ctx)
	token, hasToken := revactx.ContextGetToken(ctx)
	req = req.WithContext(ctx)
	defer span.End()
	if claims == nil && !ok {
		span.End()
		m.next.ServeHTTP(w, req)
		return
	}

	if user == nil && claims != nil {
		value, err := readUserIDClaim(m.userOIDCClaim, claims)
		if err != nil {
			m.logger.Error().Err(err).Msg("could not read user id claim")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		user, token, err = m.userProvider.GetUserByClaims(req.Context(), m.userCS3Claim, value)

		if errors.Is(err, backend.ErrAccountNotFound) {
			m.logger.Debug().Str("claim", m.userOIDCClaim).Str("value", value).Msg("User by claim not found")
			if !m.autoProvisionAccounts {
				m.logger.Debug().Interface("claims", claims).Msg("Autoprovisioning disabled")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			m.logger.Debug().Interface("claims", claims).Msg("Autoprovisioning user")
			var newuser *cs3user.User
			newuser, err = m.userProvider.CreateUserFromClaims(req.Context(), claims)
			if err != nil {
				m.logger.Error().Err(err).Msg("Autoprovisioning user failed")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			user, token, err = m.userProvider.GetUserByClaims(req.Context(), "userid", newuser.Id.OpaqueId)
			if err != nil {
				m.logger.Error().Err(err).Str("userid", newuser.Id.OpaqueId).Msg("Error getting token for autoprovisioned user")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}

		if errors.Is(err, backend.ErrAccountDisabled) {
			m.logger.Debug().Interface("claims", claims).Msg("Disabled")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if err != nil {
			m.logger.Error().Err(err).Msg("Could not get user by claim")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// if this is a multi-tenant setup, make sure the resolved user has a tenant id set
		if m.multiTenantEnabled && user.GetId().GetTenantId() == "" {
			m.logger.Error().Str("userid", user.Id.OpaqueId).Msg("User does not have a tenantId assigned")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// update user if needed
		if m.autoProvisionAccounts {
			if err = m.userProvider.UpdateUserIfNeeded(req.Context(), user, claims); err != nil {
				m.logger.Error().Err(err).Str("userid", user.GetId().GetOpaqueId()).Interface("claims", claims).Msg("Failed to update autoprovisioned user")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// Only	sync group memberships if the user has not been synced since the last cache invalidation
			if !m.lastGroupSyncCache.Has(user.GetId().GetOpaqueId()) {
				if err = m.userProvider.SyncGroupMemberships(req.Context(), user, claims); err != nil {
					m.logger.Error().Err(err).Str("userid", user.GetId().GetOpaqueId()).Interface("claims", claims).Msg("Failed to sync group memberships for autoprovisioned user")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				m.lastGroupSyncCache.Set(user.GetId().GetOpaqueId(), struct{}{}, ttlcache.DefaultTTL)
			}
		}

		// resolve the user's roles
		user, err = m.userRoleAssigner.UpdateUserRoleAssignment(ctx, user, claims)
		if err != nil {
			m.logger.Error().Err(err).Msg("Could not get user roles")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// If this is a new session, publish user login event
		if newSession := oidc.NewSessionFlagFromContext(ctx); newSession && m.eventsPublisher != nil {
			event := events.UserSignedIn{
				Executant: user.Id,
				Timestamp: utils.TimeToTS(time.Now()),
			}
			if err := events.Publish(req.Context(), m.eventsPublisher, event); err != nil {
				m.logger.Error().Err(err).Msg("could not publish user signin event.")
			}
		}

		// add user to context for selectors
		ctx = revactx.ContextSetUser(ctx, user)
		req = req.WithContext(ctx)

		m.logger.Debug().Interface("claims", claims).Interface("user", user).Msg("associated claims with user")
	} else if user != nil && !hasToken {
		// if this is a multi-tenant setup, make sure the resolved user has a tenant id set
		if m.multiTenantEnabled && user.GetId().GetTenantId() == "" {
			m.logger.Error().Str("userid", user.Id.OpaqueId).Msg("User does not have a tenantId assigned")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// If we already have a token (e.g. the app auth middleware adds the token to the context) there is no need
		// to get yet another one here.
		var err error
		_, token, err = m.userProvider.GetUserByClaims(req.Context(), "username", user.Username)
		if errors.Is(err, backend.ErrAccountDisabled) {
			m.logger.Debug().Interface("user", user).Msg("Disabled")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if err != nil {
			m.logger.Error().Err(err).Msg("Could not get user by claim")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	ri := router.ContextRoutingInfo(ctx)
	if ri.RemoteUserHeader() != "" {
		req.Header.Set(ri.RemoteUserHeader(), user.GetId().GetOpaqueId())
	}
	if !ri.SkipXAccessToken() {
		req.Header.Set(revactx.TokenHeader, token)
	}
	span.End()
	m.next.ServeHTTP(w, req)
}
