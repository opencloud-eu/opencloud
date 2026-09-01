package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/roles"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	settings "github.com/opencloud-eu/opencloud/services/settings/pkg/service/v0"
	"github.com/opencloud-eu/opencloud/services/userlog/pkg/config"
	"github.com/opencloud-eu/opencloud/services/userlog/pkg/service"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/reva/v2/pkg/events"
)

// Service is the http service responsible for serving userlog requests
type Service struct {
	log              log.Logger
	cfg              *config.Config
	userlog          *service.UserlogService
	gatewaySelector  pool.Selectable[gateway.GatewayAPIClient]
	valueClient      settingssvc.ValueService
	registeredEvents map[string]events.Unmarshaller
	tp               trace.TracerProvider
	tracer           trace.Tracer
}

// HeaderAcceptLanguage is the header where the client can set the locale
var HeaderAcceptLanguage = "Accept-Language"

// HandleGetEvents is the GET handler for events
func (s *Service) HandleGetEvents(w http.ResponseWriter, r *http.Request) {
	ctx, span := s.tracer.Start(r.Context(), "HandleGetEvents")
	defer span.End()
	u, ok := revactx.ContextGetUser(ctx)
	if !ok {
		s.log.Error().Int("returned statuscode", http.StatusUnauthorized).Msg("user unauthorized")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	evs, err := s.userlog.GetEvents(ctx, u.GetId().GetOpaqueId())
	if err != nil {
		s.log.Error().Err(err).Int("returned statuscode", http.StatusInternalServerError).Msg("get events failed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	span.SetAttributes(attribute.KeyValue{
		Key:   "events",
		Value: attribute.IntValue(len(evs)),
	})

	gwc, err := s.gatewaySelector.Next()
	if err != nil {
		s.log.Error().Err(err).Msg("cant get gateway client")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	ctx, err = utils.GetServiceUserContext(s.cfg.ServiceAccount.ServiceAccountID, gwc, s.cfg.ServiceAccount.ServiceAccountSecret)
	if err != nil {
		s.log.Error().Err(err).Msg("cant get service account")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	conv := service.NewConverter(ctx, r.Header.Get(HeaderAcceptLanguage), s.gatewaySelector, s.cfg.Service.Name, s.cfg.TranslationPath, s.cfg.DefaultLanguage)

	var outdatedEvents []string
	resp := GetEventResponseOC10{}
	for _, e := range evs {
		etype, ok := s.registeredEvents[e.Type]
		if !ok {
			s.log.Error().Str("eventid", e.Id).Str("eventtype", e.Type).Msg("event not registered")
			continue
		}

		einterface, err := etype.Unmarshal(e.Event)
		if err != nil {
			s.log.Error().Str("eventid", e.Id).Str("eventtype", e.Type).Msg("failed to umarshal event")
			continue
		}

		noti, err := conv.ConvertEvent(e.Id, einterface)
		if err != nil {
			if utils.IsErrNotFound(err) || utils.IsErrPermissionDenied(err) {
				outdatedEvents = append(outdatedEvents, e.Id)
				continue
			}
			s.log.Error().Err(err).Str("eventid", e.Id).Str("eventtype", e.Type).Msg("failed to convert event")
			continue
		}

		resp.OCS.Data = append(resp.OCS.Data, noti)
	}

	// delete outdated events asynchronously
	if len(outdatedEvents) > 0 {
		go func() {
			err := s.userlog.DeleteEvents(u.GetId().GetOpaqueId(), outdatedEvents)
			if err != nil {
				s.log.Error().Err(err).Msg("failed to delete events")
			}
		}()
	}

	glevs, err := s.userlog.GetGlobalEvents(ctx)
	if err != nil {
		s.log.Error().Err(err).Int("returned statuscode", http.StatusInternalServerError).Msg("get global events failed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for t, data := range glevs {
		noti, err := conv.ConvertGlobalEvent(t, data)
		if err != nil {
			s.log.Error().Err(err).Str("eventtype", t).Msg("failed to convert event")
			continue
		}

		resp.OCS.Data = append(resp.OCS.Data, noti)
	}

	resp.OCS.Meta.StatusCode = http.StatusOK
	b, _ := json.Marshal(resp)
	w.Write(b)
}

// HandlePostGlobaelEvent is the POST handler for global events
func (s *Service) HandlePostGlobalEvent(w http.ResponseWriter, r *http.Request) {
	var req PostEventsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.log.Error().Err(err).Int("returned statuscode", http.StatusBadRequest).Msg("request body is malformed")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := s.userlog.StoreGlobalEvent(r.Context(), req.Type, req.Data); err != nil {
		s.log.Error().Err(err).Msg("post: error storing global event")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// HandleDeleteGlobalEvent is the DELETE handler for global events
func (s *Service) HandleDeleteGlobalEvent(w http.ResponseWriter, r *http.Request) {
	var req DeleteEventsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.log.Error().Err(err).Int("returned statuscode", http.StatusBadRequest).Msg("request body is malformed")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := s.userlog.DeleteGlobalEvents(r.Context(), req.IDs); err != nil {
		s.log.Error().Err(err).Int("returned statuscode", http.StatusInternalServerError).Msg("delete events failed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// HandleDeleteEvents is the DELETE handler for events
func (s *Service) HandleDeleteEvents(w http.ResponseWriter, r *http.Request) {
	u, ok := revactx.ContextGetUser(r.Context())
	if !ok {
		s.log.Error().Int("returned statuscode", http.StatusUnauthorized).Msg("user unauthorized")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var req DeleteEventsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.log.Error().Err(err).Int("returned statuscode", http.StatusBadRequest).Msg("request body is malformed")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := s.userlog.DeleteEvents(u.GetId().GetOpaqueId(), req.IDs); err != nil {
		s.log.Error().Err(err).Int("returned statuscode", http.StatusInternalServerError).Msg("delete events failed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// RequireAdminOrSecret middleware allows only requests if the requesting user is an admin or knows the static secret
func RequireAdminOrSecret(rm *roles.Manager, secret string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// allow bypassing admin requirement by sending the correct secret
			if secret != "" && r.Header.Get("secret") == secret {
				next.ServeHTTP(w, r)
				return
			}

			isadmin, err := isAdmin(r.Context(), rm)
			if err != nil {
				errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "")
				return
			}

			if isadmin {
				next.ServeHTTP(w, r)
				return
			}

			errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "Not found")
		}
	}
}

// isAdmin determines if the user in the context is an admin / has account management permissions
func isAdmin(ctx context.Context, rm *roles.Manager) (bool, error) {
	logger := appctx.GetLogger(ctx)

	u, ok := revactx.ContextGetUser(ctx)
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
