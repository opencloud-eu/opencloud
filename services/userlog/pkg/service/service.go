package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
	ehmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/eventhistory/v0"
	ehsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/eventhistory/v0"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"go-micro.dev/v4/store"
	"go.opentelemetry.io/otel/trace"
)

// UserlogService is the service responsible for user activities
type UserlogService struct {
	log           log.Logger
	store         store.Store
	historyClient ehsvc.EventHistoryService
	tp            trace.TracerProvider
	tracer        trace.Tracer
}

// NewUserlogService returns an EventHistory service
func NewUserlogService(opts ...Option) (*UserlogService, error) {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}

	if o.Store == nil {
		return nil, fmt.Errorf("need non nil store to work properly")
	}

	ul := &UserlogService{
		log:           o.Logger,
		store:         o.Store,
		historyClient: o.HistoryClient,
		tp:            o.TraceProvider,
		tracer:        o.TraceProvider.Tracer("github.com/opencloud-eu/opencloud/services/userlog/pkg/service"),
	}

	return ul, nil
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

func (ul *UserlogService) AddEventToUser(userid string, event events.Event) error {
	return ul.alterUserEventList(userid, func(ids []string) []string {
		return append(ids, event.ID)
	})
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
