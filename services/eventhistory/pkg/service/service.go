package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/opencloud-eu/opencloud/pkg/log"
	ehmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/eventhistory/v0"
	ehsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/eventhistory/v0"
	"github.com/opencloud-eu/opencloud/services/eventhistory/pkg/config"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"go-micro.dev/v4/store"
)

// StoreEvent is data structure in the store
type StoreEvent struct {
	ID    string
	Type  string
	Event []byte
}

// EventHistoryService is the service responsible for event history
type EventHistoryService struct {
	ch    <-chan events.Event
	store store.Store
	cfg   *config.Config
	log   log.Logger
}

// NewEventHistoryService returns an EventHistory service
func NewEventHistoryService(cfg *config.Config, consumer events.Consumer, store store.Store, log log.Logger) (*EventHistoryService, error) {
	if consumer == nil || store == nil {
		return nil, fmt.Errorf("need non nil consumer (%v) and store (%v) to work properly", consumer, store)
	}

	ch, err := events.ConsumeAll(consumer, "evhistory")
	if err != nil {
		return nil, err
	}

	eh := &EventHistoryService{ch: ch, store: store, cfg: cfg, log: log}
	go eh.StoreEvents()

	return eh, nil
}

// StoreEvents consumes all events and stores them in the store. Will block
func (eh *EventHistoryService) StoreEvents() {
	for event := range eh.ch {
		ev, err := json.Marshal(StoreEvent{
			ID:    event.ID,
			Type:  event.Type,
			Event: event.Event.([]byte),
		})
		if err != nil {
			eh.log.Error().Err(err).Str("eventid", event.ID).Msg("could not marshal event")
			continue
		}
		if err := eh.store.Write(&store.Record{
			Key:    event.ID,
			Value:  ev,
			Expiry: eh.cfg.Store.TTL,
			Metadata: map[string]any{
				"type": event.Type,
			},
		}); err != nil {
			// we can't store. That's it for us.
			eh.log.Error().Err(err).Str("eventid", event.ID).Msg("could not store event")
			continue
		}
	}
}

// GetEvents allows retrieving events from the eventstore by id
func (eh *EventHistoryService) GetEvents(ctx context.Context, req *ehsvc.GetEventsRequest, resp *ehsvc.GetEventsResponse) error {
	for _, id := range req.Ids {
		ev, err := eh.getEvent(id)
		if err != nil {
			continue
		}

		resp.Events = append(resp.Events, ev)
	}

	return nil
}

// GetEventsForUser allows retrieving events from the eventstore by userID
// This function will match all events that contains the user ID between two non-word characters.
// The reasoning behind this is that events put the userID in many different fields, which can differ
// per event type. This function will match all events that contain the userID by using a regex.
// This should also cover future events that might contain the userID in a different field.
func (eh *EventHistoryService) GetEventsForUser(ctx context.Context, req *ehsvc.GetEventsForUserRequest, resp *ehsvc.GetEventsResponse) error {
	idx, err := eh.store.List(store.ListPrefix(""))
	if err != nil {
		eh.log.Error().Err(err).Msg("could not list events")
		return err
	}

	// Match all events that contains the user ID between two non-word characters.
	userID, err := regexp.Compile(fmt.Sprintf(`\W%s\W`, req.UserID))
	if err != nil {
		eh.log.Error().Err(err).Str("userID", req.UserID).Msg("could not compile regex")
		return err
	}

	for _, i := range idx {
		e, err := eh.getEvent(i)
		if err != nil {
			continue
		}

		if userID.Match(e.Event) {
			resp.Events = append(resp.Events, e)
		}
	}

	return nil
}

func (eh *EventHistoryService) getEvent(id string) (*ehmsg.Event, error) {
	evs, err := eh.store.Read(id)
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			eh.log.Error().Err(err).Str("eventid", id).Msg("could not read event")
		}
		return nil, err
	}

	if len(evs) == 0 {
		return nil, store.ErrNotFound
	}

	var ev StoreEvent
	if err := json.Unmarshal(evs[0].Value, &ev); err != nil {
		eh.log.Error().Err(err).Str("eventid", id).Msg("could not unmarshal event")
		return nil, err
	}

	return &ehmsg.Event{
		Id:    ev.ID,
		Event: ev.Event,
		Type:  ev.Type,
	}, nil
}
