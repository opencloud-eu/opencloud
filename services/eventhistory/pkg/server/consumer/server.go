package consumer

import (
	"encoding/json"

	"github.com/opencloud-eu/opencloud/pkg/generators"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/eventhistory/pkg/config"
	svc "github.com/opencloud-eu/opencloud/services/eventhistory/pkg/service"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"
	"go-micro.dev/v4/store"
)

// Consumer consumes all events and stores them in the store
type Consumer struct {
	ch    <-chan events.Event
	store store.Store
	cfg   *config.Config
	log   log.Logger
}

// NewConsumer initializes the event consumer and starts storing events
func NewConsumer(opts ...Option) (*Consumer, error) {
	options := newOptions(opts...)

	cons := options.Consumer
	if cons == nil {
		connName := generators.GenerateConnectionName(options.Config.Service.Name, generators.NTypeBus)
		var err error
		cons, err = stream.NatsFromConfig(connName, false, stream.NatsConfig{
			Endpoint:             options.Config.Events.Endpoint,
			Cluster:              options.Config.Events.Cluster,
			TLSInsecure:          options.Config.Events.TLSInsecure,
			TLSRootCACertificate: options.Config.Events.TLSRootCACertificate,
			EnableTLS:            options.Config.Events.EnableTLS,
			AuthUsername:         options.Config.Events.AuthUsername,
			AuthPassword:         options.Config.Events.AuthPassword,
		})
		if err != nil {
			return nil, err
		}
	}

	ch, err := events.ConsumeAll(cons, "evhistory")
	if err != nil {
		return nil, err
	}

	c := &Consumer{ch: ch, store: options.Persistence, cfg: options.Config, log: options.Logger}
	go c.StoreEvents()

	return c, nil
}

// StoreEvents consumes all events and stores them in the store. Will block
func (c *Consumer) StoreEvents() {
	for event := range c.ch {
		ev, err := json.Marshal(svc.StoreEvent{
			ID:    event.ID,
			Type:  event.Type,
			Event: event.Event.([]byte),
		})
		if err != nil {
			c.log.Error().Err(err).Str("eventid", event.ID).Msg("could not marshal event")
			continue
		}
		if err := c.store.Write(&store.Record{
			Key:    event.ID,
			Value:  ev,
			Expiry: c.cfg.Store.TTL,
			Metadata: map[string]any{
				"type": event.Type,
			},
		}); err != nil {
			// we can't store. That's it for us.
			c.log.Error().Err(err).Str("eventid", event.ID).Msg("could not store event")
			continue
		}
	}
}
