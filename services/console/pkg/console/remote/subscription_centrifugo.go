package remote

import (
	"fmt"

	"github.com/centrifugal/centrifuge-go"
	"github.com/golang-jwt/jwt/v5"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/console/pkg/console"
)

type CentrifugoSubscriptionOptions struct {
	Token                        *jwt.Token                    `validate:"required"`
	Claims                       *console.Claims               `validate:"required"`
	CentrifugeConfig             centrifuge.Config             `validate:"required"`
	CentrifugeSubscriptionConfig centrifuge.SubscriptionConfig `validate:"required"`
	URLBuilder                   console.URLBuilder
	Logger                       log.Logger
}

func (o CentrifugoSubscriptionOptions) Validate() error {
	if err := console.Validate.Struct(o); err != nil {
		return fmt.Errorf("(%w): %w", console.ErrOptionsInvalid, err)
	}

	return nil
}

type CentrifugoSubscription struct {
	client             *centrifuge.Client
	subscriptionConfig centrifuge.SubscriptionConfig
	channel            string
	logger             log.Logger
}

func NewCentrifugoSubscription(o CentrifugoSubscriptionOptions) (CentrifugoSubscription, error) {
	if err := o.Validate(); err != nil {
		return CentrifugoSubscription{}, err
	}

	client := centrifuge.NewJsonClient(o.URLBuilder.SubscriptionUrl().String(), o.CentrifugeConfig)
	if err := client.Connect(); err != nil {
		return CentrifugoSubscription{}, fmt.Errorf("failed to connect to centrifugo: %w", err)
	}

	client.OnConnected(func(_ centrifuge.ConnectedEvent) {
		o.Logger.Info().Msg("console connection established")
	})

	client.OnConnecting(func(_ centrifuge.ConnectingEvent) {
		o.Logger.Info().Msg("connecting to console")
	})

	client.OnDisconnected(func(e centrifuge.DisconnectedEvent) {
		o.Logger.Info().Interface("event", e).Msg("console connection closed")
	})

	client.OnError(func(e centrifuge.ErrorEvent) {
		o.Logger.Error().Err(e.Error).Msg("console connection closed")
	})

	return CentrifugoSubscription{
		client:             client,
		channel:            fmt.Sprintf("#%s", o.Claims.TenantId),
		subscriptionConfig: o.CentrifugeSubscriptionConfig,
		logger:             o.Logger,
	}, nil
}

func (cs CentrifugoSubscription) Close() error {
	if cs.client != nil {
		cs.client.Close()
	}

	return nil
}

func (cs CentrifugoSubscription) Handle(handler Handler) error {
	if cs.client == nil {
		return fmt.Errorf("client not initialized")
	}

	subscription, err := cs.client.NewSubscription(cs.channel, cs.subscriptionConfig)
	if err != nil {
		return err
	}

	subscription.OnSubscribed(func(_ centrifuge.SubscribedEvent) {
		cs.logger.Info().Msg("personal channel subscription established")
	})

	subscription.OnError(func(e centrifuge.SubscriptionErrorEvent) {
		cs.logger.Error().Err(e.Error).Msg("personal channel subscription failed")
	})

	subscription.OnUnsubscribed(func(e centrifuge.UnsubscribedEvent) {
		cs.logger.Info().Interface("event", e).Msg("personal channel subscription closed")
	})

	subscription.OnPublication(func(e centrifuge.PublicationEvent) {
		topic, ok := e.Tags["topic"]
		if !ok {
			cs.logger.Debug().Interface("event", e).Msg("event without topic")
			return
		}

		if err := cs.handle(Topic(topic), handler, e.Data); err != nil {
			cs.logger.Error().Err(err)
		}
	})

	return subscription.Subscribe()
}

func (cs CentrifugoSubscription) handle(topic Topic, handler Handler, b []byte) error {
	switch {
	case topic == TopicThemeAssigned && handler.ThemeAssigned != nil:
		return dispatchMessage(handler.ThemeAssigned, b)
	case topic == TopicThemeUnassigned && handler.ThemeUnassigned != nil:
		return dispatchMessage(handler.ThemeUnassigned, b)
	default:
		return fmt.Errorf("no handler registered for topic: %s", topic)
	}
}
