package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"go-micro.dev/v4/store"
	"go.opentelemetry.io/otel/trace"

	ocEvents "github.com/opencloud-eu/opencloud/pkg/events"
	"github.com/opencloud-eu/opencloud/pkg/l10n"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/roles"
	ehmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/eventhistory/v0"
	ehsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/eventhistory/v0"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/userlog/pkg/config"
)

// UserlogService is the service responsible for user activities
type UserlogService struct {
	log              log.Logger
	m                *chi.Mux
	store            store.Store
	cfg              *config.Config
	historyClient    ehsvc.EventHistoryService
	gatewaySelector  pool.Selectable[gateway.GatewayAPIClient]
	valueClient      settingssvc.ValueService
	registeredEvents map[string]events.Unmarshaller
	tp               trace.TracerProvider
	tracer           trace.Tracer
	publisher        events.Publisher
	filter           *userlogFilter
}

// NewUserlogService returns an EventHistory service
func NewUserlogService(opts ...Option) (*UserlogService, error) {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}

	if o.Stream == nil || o.Store == nil {
		return nil, fmt.Errorf("need non nil stream (%v) and store (%v) to work properly", o.Stream, o.Store)
	}

	ch, err := events.Consume(o.Stream, "userlog", o.RegisteredEvents...)
	if err != nil {
		return nil, err
	}

	ul := &UserlogService{
		log:              o.Logger,
		m:                o.Mux,
		store:            o.Store,
		cfg:              o.Config,
		historyClient:    o.HistoryClient,
		gatewaySelector:  o.GatewaySelector,
		valueClient:      o.ValueClient,
		registeredEvents: make(map[string]events.Unmarshaller),
		tp:               o.TraceProvider,
		tracer:           o.TraceProvider.Tracer("github.com/opencloud-eu/opencloud/services/userlog/pkg/service"),
		publisher:        o.Stream,
		filter:           newUserlogFilter(o.Logger, o.ValueClient),
	}

	for _, e := range o.RegisteredEvents {
		typ := reflect.TypeOf(e)
		ul.registeredEvents[typ.String()] = e
	}

	m := roles.NewManager(
		// TODO: caching?
		roles.Logger(o.Logger),
		roles.RoleService(o.RoleClient),
	)

	ul.m.Route("/ocs/v2.php/apps/notifications/api/v1/notifications", func(r chi.Router) {
		r.Get("/", ul.HandleGetEvents)
		r.Delete("/", ul.HandleDeleteEvents)
		r.Post("/global", RequireAdminOrSecret(&m, o.Config.GlobalNotificationsSecret)(ul.HandlePostGlobalEvent))
		r.Delete("/global", RequireAdminOrSecret(&m, o.Config.GlobalNotificationsSecret)(ul.HandleDeleteGlobalEvent))
	})

	go ul.MemorizeEvents(ch)

	return ul, nil
}

// MemorizeEvents stores eventIDs a user wants to receive
func (ul *UserlogService) MemorizeEvents(ch <-chan events.Event) {
	for i := 0; i < ul.cfg.MaxConcurrency; i++ {
		go func(ch <-chan events.Event) {
			for event := range ch {
				ul.processEvent(event)
			}
		}(ch)
	}
}

func (ul *UserlogService) processEvent(event events.Event) {
	// for each event we need to:
	// I) find users eligible to receive the event
	var (
		users     []string
		executant *user.UserId
		err       error
	)

	gwc, err := ul.gatewaySelector.Next()
	if err != nil {
		ul.log.Error().Err(err).Msg("cannot get gateway client")
		return
	}

	ctx, err := utils.GetServiceUserContext(ul.cfg.ServiceAccount.ServiceAccountID, gwc, ul.cfg.ServiceAccount.ServiceAccountSecret)
	if err != nil {
		ul.log.Error().Err(err).Msg("cannot get service account")
		return
	}

	gwc, err = ul.gatewaySelector.Next()
	if err != nil {
		ul.log.Error().Err(err).Msg("cannot get gateway client")
		return
	}
	switch e := event.Event.(type) {
	default:
		err = errors.New("unhandled event")
	// file related
	case events.PostprocessingStepFinished:
		switch e.FinishedStep {
		case events.PPStepAntivirus:
			result := e.Result.(events.VirusscanResult)
			if !result.Infected {
				return
			}

			// TODO: should space mangers also be informed?
			users = append(users, e.ExecutingUser.GetId().GetOpaqueId())
		case events.PPStepPolicies:
			if e.Outcome == events.PPOutcomeContinue {
				return
			}
			users = append(users, e.ExecutingUser.GetId().GetOpaqueId())
		default:
			return
		}

	// space related // TODO: how to find spaceadmins?
	case events.SpaceDisabled:
		executant = e.Executant
		users, err = utils.GetSpaceMembers(ctx, e.ID.GetOpaqueId(), gwc, utils.ViewerRole)
	case events.SpaceDeleted:
		executant = e.Executant
		for u := range e.FinalMembers {
			users = append(users, u)
		}
	case events.SpaceShared:
		executant = e.Executant
		users, err = utils.ResolveID(ctx, e.GranteeUserID, e.GranteeGroupID, gwc)
	case ocEvents.ResourceMention:
		executant = e.Executant
		for _, userID := range e.UserIDs {
			users = append(users, userID.GetOpaqueId())
		}
	case events.SpaceUnshared:
		executant = e.Executant
		users, err = utils.ResolveID(ctx, e.GranteeUserID, e.GranteeGroupID, gwc)
	case events.SpaceMembershipExpired:
		users, err = utils.ResolveID(ctx, e.GranteeUserID, e.GranteeGroupID, gwc)

	// share related
	case events.ShareCreated:
		executant = e.Executant
		users, err = utils.ResolveID(ctx, e.GranteeUserID, e.GranteeGroupID, gwc)
	case events.ShareRemoved:
		executant = e.Executant
		users, err = utils.ResolveID(ctx, e.GranteeUserID, e.GranteeGroupID, gwc)
	case events.ShareExpired:
		users, err = utils.ResolveID(ctx, e.GranteeUserID, e.GranteeGroupID, gwc)
	}

	if err != nil {
		// TODO: Find out why this errors on ci pipeline
		ul.log.Debug().Err(err).Interface("event", event).Msg("error gathering members for event")
		return
	}

	// II) filter users who want to receive the event
	users = ul.filter.execute(ctx, event, executant, users)

	// III) store the eventID for each user
	for _, id := range users {
		if err := ul.addEventToUser(id, event); err != nil {
			ul.log.Error().Err(err).Str("userID", id).Str("eventid", event.ID).Msg("failed to store event for user")
			return
		}
	}

	// IV) send sses
	if !ul.cfg.DisableSSE {
		if err := ul.sendSSE(ctx, users, event, ul.gatewaySelector); err != nil {
			ul.log.Error().Err(err).Interface("userid", users).Str("eventid", event.ID).Msg("cannot create sse event")
		}
	}
}

// GetEvents allows retrieving events from the eventhistory by userid
func (ul *UserlogService) GetEvents(ctx context.Context, userid string) ([]*ehmsg.Event, error) {
	ctx, span := ul.tracer.Start(ctx, "GetEvents")
	defer span.End()
	rec, err := ul.store.Read(userid)
	if err != nil && err != store.ErrNotFound {
		ul.log.Error().Err(err).Str("userid", userid).Msg("failed to read record from store")
		return nil, err
	}

	if len(rec) == 0 {
		// no events available
		return []*ehmsg.Event{}, nil
	}

	var eventIDs []string
	if err := json.Unmarshal(rec[0].Value, &eventIDs); err != nil {
		ul.log.Error().Err(err).Str("userid", userid).Msg("failed to umarshal record from store")
		return nil, err
	}

	resp, err := ul.historyClient.GetEvents(ctx, &ehsvc.GetEventsRequest{Ids: eventIDs})
	if err != nil {
		return nil, err
	}

	// remove expired events from list asynchronously
	go func() {
		if err := ul.removeExpiredEvents(userid, eventIDs, resp.GetEvents()); err != nil {
			ul.log.Error().Err(err).Str("userid", userid).Msg("could not remove expired events from user")
		}
	}()

	return resp.GetEvents(), nil
}

// DeleteEvents will delete the specified events
func (ul *UserlogService) DeleteEvents(userid string, evids []string) error {
	toDelete := make(map[string]struct{})
	for _, e := range evids {
		toDelete[e] = struct{}{}
	}

	return ul.alterUserEventList(userid, func(ids []string) []string {
		var newids []string
		for _, id := range ids {
			if _, del := toDelete[id]; del {
				continue
			}

			newids = append(newids, id)
		}
		return newids
	})
}

// StoreGlobalEvent will store a global event that will be returned with each `GetEvents` request
func (ul *UserlogService) StoreGlobalEvent(ctx context.Context, typ string, data map[string]string) error {
	ctx, span := ul.tracer.Start(ctx, "StoreGlobalEvent")
	defer span.End()
	switch typ {
	default:
		return fmt.Errorf("unknown event type: %s", typ)
	case "deprovision":
		dps, ok := data["deprovision_date"]
		if !ok {
			return errors.New("need 'deprovision_date' in request body")
		}

		format := data["deprovision_date_format"]
		if format == "" {
			format = time.RFC3339
		}

		date, err := time.Parse(format, dps)
		if err != nil {
			fmt.Println("", format, "\n", dps)
			return fmt.Errorf("cannot parse time to format. time: '%s' format: '%s'", dps, format)
		}

		ev := DeprovisionData{
			DeprovisionDate:   date,
			DeprovisionFormat: format,
		}

		b, err := json.Marshal(ev)
		if err != nil {
			return err
		}

		return ul.alterGlobalEvents(ctx, func(evs map[string]json.RawMessage) error {
			evs[typ] = b
			return nil
		})
	}
}

// GetGlobalEvents will return all global events
func (ul *UserlogService) GetGlobalEvents(ctx context.Context) (map[string]json.RawMessage, error) {
	_, span := ul.tracer.Start(ctx, "GetGlobalEvents")
	defer span.End()
	out := make(map[string]json.RawMessage)

	recs, err := ul.store.Read(_globalEventsKey)
	if err != nil && err != store.ErrNotFound {
		return out, err
	}

	if len(recs) > 0 {
		if err := json.Unmarshal(recs[0].Value, &out); err != nil {
			return out, err
		}
	}

	return out, nil
}

// DeleteGlobalEvents will delete the specified event
func (ul *UserlogService) DeleteGlobalEvents(ctx context.Context, evnames []string) error {
	_, span := ul.tracer.Start(ctx, "DeleteGlobalEvents")
	defer span.End()
	return ul.alterGlobalEvents(ctx, func(evs map[string]json.RawMessage) error {
		for _, name := range evnames {
			delete(evs, name)
		}
		return nil
	})
}

func (ul *UserlogService) addEventToUser(userid string, event events.Event) error {
	return ul.alterUserEventList(userid, func(ids []string) []string {
		return append(ids, event.ID)
	})
}

func (ul *UserlogService) sendSSE(ctx context.Context, userIDs []string, event events.Event, gatewaySelector pool.Selectable[gateway.GatewayAPIClient]) error {
	m := make(map[string]events.SendSSE)

	for _, userid := range userIDs {
		loc := l10n.MustGetUserLocale(ctx, userid, "", ul.valueClient)
		if ev, ok := m[loc]; ok {
			ev.UserIDs = append(m[loc].UserIDs, userid)
			m[loc] = ev
			continue
		}

		ev, err := NewConverter(ctx, loc, gatewaySelector, ul.cfg.Service.Name, ul.cfg.TranslationPath, ul.cfg.DefaultLanguage).ConvertEvent(event.ID, event.Event)
		if err != nil {
			if utils.IsErrNotFound(err) || utils.IsErrPermissionDenied(err) {
				// the resource was not found, we assume it is deleted
				continue
			}
			return err
		}

		b, err := json.Marshal(ev)
		if err != nil {
			return err
		}

		m[loc] = events.SendSSE{
			UserIDs: []string{userid},
			Type:    "userlog-notification",
			Message: b,
		}

	}

	for _, ev := range m {
		if err := events.Publish(ctx, ul.publisher, ev); err != nil {
			return err
		}
	}

	return nil
}

func (ul *UserlogService) removeExpiredEvents(userid string, all []string, received []*ehmsg.Event) error {
	exists := make(map[string]struct{}, len(received))
	for _, e := range received {
		exists[e.Id] = struct{}{}
	}

	var toDelete []string
	for _, eid := range all {
		if _, ok := exists[eid]; !ok {
			toDelete = append(toDelete, eid)
		}
	}

	if len(toDelete) == 0 {
		return nil
	}

	return ul.DeleteEvents(userid, toDelete)
}

func (ul *UserlogService) alterUserEventList(userid string, alter func([]string) []string) error {
	recs, err := ul.store.Read(userid)
	if err != nil && err != store.ErrNotFound {
		return err
	}

	var ids []string
	if len(recs) > 0 {
		if err := json.Unmarshal(recs[0].Value, &ids); err != nil {
			return err
		}
	}

	ids = alter(ids)

	// store reacts unforseeable when trying to store nil values
	if len(ids) == 0 {
		return ul.store.Delete(userid)
	}

	b, err := json.Marshal(ids)
	if err != nil {
		return err
	}

	return ul.store.Write(&store.Record{
		Key:   userid,
		Value: b,
	})
}

func (ul *UserlogService) alterGlobalEvents(ctx context.Context, alter func(map[string]json.RawMessage) error) error {
	_, span := ul.tracer.Start(ctx, "alterGlobalEvents")
	defer span.End()
	evs, err := ul.GetGlobalEvents(ctx)
	if err != nil && err != store.ErrNotFound {
		return err
	}

	if err := alter(evs); err != nil {
		return err
	}

	val, err := json.Marshal(evs)
	if err != nil {
		return err
	}

	return ul.store.Write(&store.Record{
		Key:   "global-events",
		Value: val,
	})
}
