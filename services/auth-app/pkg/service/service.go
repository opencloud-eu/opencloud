package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	applications "github.com/cs3org/go-cs3apis/cs3/auth/applications/v1beta1"
	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/roles"
	"github.com/opencloud-eu/opencloud/services/auth-app/pkg/config"
	settings "github.com/opencloud-eu/opencloud/services/settings/pkg/service/v0"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/auth/scope"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"google.golang.org/grpc/metadata"
)

var ErrBadRequest = errors.New("bad request")

// AuthAppToken represents an app token.
type AuthAppToken struct {
	Token          string    `json:"token"`
	ExpirationDate time.Time `json:"expiration_date"`
	CreatedDate    time.Time `json:"created_date"`
	Label          string    `json:"label"`
}

// AuthAppService defines the service interface.
type AuthAppService struct {
	log log.Logger
	cfg *config.Config
	gws pool.Selectable[gateway.GatewayAPIClient]
	m   *chi.Mux
	r   *roles.Manager
}

// NewAuthAppService initializes a new AuthAppService.
func NewAuthAppService(opts ...Option) (*AuthAppService, error) {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}

	r := roles.NewManager(
		roles.Logger(o.Logger),
		roles.RoleService(o.RoleClient),
	)

	a := &AuthAppService{
		log: o.Logger,
		cfg: o.Config,
		gws: o.GatewaySelector,
		m:   o.Mux,
		r:   &r,
	}

	a.m.Route("/auth-app/tokens", func(r chi.Router) {
		r.Get("/", a.HandleList)
		r.Post("/", a.HandleCreate)
		r.Delete("/", a.HandleDelete)
	})

	return a, nil
}

// ServeHTTP implements the http.Handler interface.
func (a *AuthAppService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.m.ServeHTTP(w, r)
}

// HandleCreate handles the creation of app tokens
func (a *AuthAppService) HandleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := getContext(r)
	sublog := a.log.With().Str("actor", ctxpkg.ContextMustGetUser(ctx).GetId().GetOpaqueId()).Logger()

	gwc, err := a.gws.Next()
	if err != nil {
		sublog.Error().Err(err).Msg("error getting gateway client")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	q := r.URL.Query()
	expiry, err := time.ParseDuration(q.Get("expiry"))
	if err != nil {
		sublog.Info().Err(err).Str("duration", q.Get("expiry")).Msg("error parsing expiry")
		http.Error(w, "error parsing expiry. Use e.g. 30m or 72h", http.StatusBadRequest)
		return
	}

	label := q.Get("label")
	if label == "" {
		label = "Generated via API"
	}

	// Impersonated request
	userID, userName := q.Get("userID"), q.Get("userName")
	if userID != "" || userName != "" {
		if !a.cfg.AllowImpersonation {
			sublog.Error().Msg("impersonation is not allowed")
			http.Error(w, "impersonation is not allowed", http.StatusForbidden)
			return
		}
		ok, err := isAdmin(ctx, a.r)
		if err != nil {
			sublog.Error().Err(err).Msg("error checking if user is admin")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !ok {
			sublog.Error().Msg("user is not admin")
			w.WriteHeader(http.StatusForbidden)
			return
		}
		ctx, err = a.authenticateUser(userID, userName, gwc)
		if err != nil {
			sublog.Error().Err(err).Msg("error authenticating user")
			if errors.Is(err, ErrBadRequest) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		label = label + " (Impersonation)"
	}

	scopes, err := scope.AddOwnerScope(map[string]*authpb.Scope{})
	if err != nil {
		sublog.Error().Err(err).Msg("error adding owner scope")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	res, err := gwc.GenerateAppPassword(ctx, &applications.GenerateAppPasswordRequest{
		TokenScope: scopes,
		Label:      label,
		Expiration: utils.TimeToTS(time.Now().Add(expiry)),
	})
	if err != nil {
		sublog.Error().Err(err).Msg("error generating app password")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if res.GetStatus().GetCode() != rpc.Code_CODE_OK {
		sublog.Error().Str("status", res.GetStatus().GetCode().String()).Msg("error generating app password")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(convert(res.GetAppPassword()))
	if err != nil {
		sublog.Error().Err(err).Msg("error marshaling app password")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(b); err != nil {
		sublog.Error().Err(err).Msg("error writing response")
	}
	w.WriteHeader(http.StatusCreated)
}

// HandleList handles listing of app tokens
func (a *AuthAppService) HandleList(w http.ResponseWriter, r *http.Request) {
	ctx := getContext(r)
	sublog := a.log.With().Str("actor", ctxpkg.ContextMustGetUser(ctx).GetId().GetOpaqueId()).Logger()

	gwc, err := a.gws.Next()
	if err != nil {
		sublog.Error().Err(err).Msg("error getting gateway client")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	res, err := gwc.ListAppPasswords(ctx, &applications.ListAppPasswordsRequest{})
	if err != nil {
		sublog.Error().Err(err).Msg("error listing app passwords")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if res.GetStatus().GetCode() != rpc.Code_CODE_OK {
		sublog.Error().Str("status", res.GetStatus().GetCode().String()).Msg("error listing app passwords")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tokens := make([]AuthAppToken, 0, len(res.GetAppPasswords()))
	for _, ap := range res.GetAppPasswords() {
		tokens = append(tokens, convert(ap))
	}

	b, err := json.Marshal(tokens)
	if err != nil {
		sublog.Error().Err(err).Msg("error marshaling app passwords")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(b); err != nil {
		sublog.Error().Err(err).Msg("error writing response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// HandleDelete handles deletion of app tokens
func (a *AuthAppService) HandleDelete(w http.ResponseWriter, r *http.Request) {
	ctx := getContext(r)
	sublog := a.log.With().Str("actor", ctxpkg.ContextMustGetUser(ctx).GetId().GetOpaqueId()).Logger()

	gwc, err := a.gws.Next()
	if err != nil {
		sublog.Error().Err(err).Msg("error getting gateway client")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	pw := r.URL.Query().Get("token")
	if pw == "" {
		sublog.Info().Msg("missing token")
		http.Error(w, "missing auth-app token. Set 'token' parameter", http.StatusBadRequest)
		return
	}

	res, err := gwc.InvalidateAppPassword(ctx, &applications.InvalidateAppPasswordRequest{Password: pw})
	if err != nil {
		sublog.Error().Err(err).Msg("error invalidating app password")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if res.GetStatus().GetCode() != rpc.Code_CODE_OK {
		sublog.Error().Str("status", res.GetStatus().GetCode().String()).Msg("error invalidating app password")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *AuthAppService) authenticateUser(userID, userName string, gwc gateway.GatewayAPIClient) (context.Context, error) {
	ctx := context.Background()
	authRes, err := gwc.Authenticate(ctx, &gateway.AuthenticateRequest{
		Type:         "machine",
		ClientId:     buildClientID(userID, userName),
		ClientSecret: a.cfg.MachineAuthAPIKey,
	})
	if err != nil {
		return nil, err
	}

	if authRes.GetStatus().GetCode() != rpc.Code_CODE_OK {
		return nil, errors.New("error authenticating user: " + authRes.GetStatus().GetMessage())
	}

	if (userID != "" && authRes.GetUser().GetId().GetOpaqueId() != userID) || (userName != "" && authRes.GetUser().GetUsername() != userName) {
		return nil, fmt.Errorf("requested user does not match authenticated user: userID:%s, userName:%s, %w", authRes.GetUser().GetId().GetOpaqueId(), authRes.GetUser().GetUsername(), ErrBadRequest)
	}

	ctx = ctxpkg.ContextSetUser(ctx, &userpb.User{Id: authRes.GetUser().GetId()})
	return metadata.AppendToOutgoingContext(ctx, ctxpkg.TokenHeader, authRes.GetToken()), nil
}

func getContext(r *http.Request) context.Context {
	ctx := r.Context()
	return metadata.AppendToOutgoingContext(ctx, ctxpkg.TokenHeader, r.Header.Get(ctxpkg.TokenHeader))
}

func buildClientID(userID, userName string) string {
	switch {
	default:
		return ""
	case userID != "":
		return "userid:" + userID
	case userName != "":
		return "username:" + userName
	}
}

// isAdmin determines if the user in the context is an admin / has account management permissions
func isAdmin(ctx context.Context, rm *roles.Manager) (bool, error) {
	logger := appctx.GetLogger(ctx)

	u, ok := ctxpkg.ContextGetUser(ctx)
	uid := u.GetId().GetOpaqueId()
	if !ok || uid == "" {
		logger.Error().Str("userid", uid).Msg("user not in context")
		return false, errors.New("no user in context")
	}
	// get roles from context
	roleIDs, ok := roles.ReadRoleIDsFromContext(ctx)
	if !ok {
		logger.Debug().Str("userid", uid).Msg("No roles in context, contacting settings service")
		var err error
		roleIDs, err = rm.FindRoleIDsForUser(ctx, uid)
		if err != nil {
			logger.Err(err).Str("userid", uid).Msg("failed to get roles for user")
			return false, err
		}

		if len(roleIDs) == 0 {
			logger.Err(err).Str("userid", uid).Msg("user has no roles")
			return false, errors.New("user has no roles")
		}
	}

	// check if permission is present in roles of the authenticated account
	return rm.FindPermissionByID(ctx, roleIDs, settings.AccountManagementPermissionID) != nil, nil
}

func convert(ap *applications.AppPassword) AuthAppToken {
	return AuthAppToken{
		Token:          ap.GetPassword(),
		ExpirationDate: utils.TSToTime(ap.GetExpiration()),
		CreatedDate:    utils.TSToTime(ap.GetCtime()),
		Label:          ap.GetLabel(),
	}
}
