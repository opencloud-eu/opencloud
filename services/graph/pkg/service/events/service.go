package events

import (
	"context"
	"io"
	"sync/atomic"
	"time"

	ocevents "github.com/opencloud-eu/opencloud/pkg/events"

	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/identity"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/metrics"
)

// PhotoSyncer downloads and stores a profile picture from an allowed URL.
type PhotoSyncer interface {
	SyncPhotoFromURL(context.Context, string, string) error
}

// Option configures the graph event consumer.
type Option func(*GraphEventConsumerImpl)

// WithProfilePictureSync enables profile picture synchronization and update events.
func WithProfilePictureSync(syncer PhotoSyncer, publisher events.Publisher) Option {
	return func(g *GraphEventConsumerImpl) {
		g.photoSyncer = syncer
		g.publisher = publisher
	}
}

type GraphEventConsumer interface {
	Start() error
	io.Closer
}

type GraphEventConsumerImpl struct {
	ctx         context.Context
	consumer    events.Consumer
	backend     identity.Backend
	metrics     *metrics.Metrics
	logger      *log.Logger
	stopped     atomic.Bool
	stopCh      chan struct{}
	cancel      context.CancelFunc
	photoSyncer PhotoSyncer
	publisher   events.Publisher
}

var _ GraphEventConsumer = &GraphEventConsumerImpl{}

func (g *GraphEventConsumerImpl) Start() error {
	defer g.cancel()
	return processEvents(g.ctx, g.consumer, &g.stopped, g.stopCh, g.backend, g.metrics, g.logger, g.photoSyncer, g.publisher)
}

func (g *GraphEventConsumerImpl) Close() error {
	g.cancel()
	if g.stopped.CompareAndSwap(false, true) {
		close(g.stopCh)
	}
	return nil
}

type NullGraphEventConsumer struct {
}

var _ GraphEventConsumer = &NullGraphEventConsumer{}

func (n *NullGraphEventConsumer) Start() error {
	return nil
}

func (n *NullGraphEventConsumer) Close() error {
	return nil
}

func NewService(ctx context.Context, consumer events.Consumer, backend identity.Backend, metrics *metrics.Metrics, logger *log.Logger, options ...Option) (GraphEventConsumer, error) {
	if consumer == nil {
		return &NullGraphEventConsumer{}, nil
	} else {
		stopCh := make(chan struct{}, 1)
		ctx, cancel := context.WithCancel(ctx)
		g := &GraphEventConsumerImpl{
			ctx:      ctx,
			consumer: consumer,
			backend:  backend,
			metrics:  metrics,
			logger:   logger,
			stopCh:   stopCh,
			cancel:   cancel,
		}
		for _, option := range options {
			option(g)
		}
		return g, nil
	}
}

func processEvents(ctx context.Context, consumer events.Consumer, stop *atomic.Bool, stopCh chan struct{},
	backend identity.Backend, m *metrics.Metrics, logger *log.Logger, photoSyncer PhotoSyncer, publisher events.Publisher) error {
	var _registeredEvents = []events.Unmarshaller{
		events.UserSignedIn{},
		ocevents.UserProfilePictureSyncRequested{},
	}
	evChannel, err := events.Consume(consumer, "graph", _registeredEvents...)
	if err != nil {
		logger.Error().Err(err).Msg("cannot consume from nats")
		return err
	}
	logger.Debug().Msg("listening for events")
	for loop := true; loop; {
		select {
		case e := <-evChannel:
			switch ev := e.Event.(type) {
			default:
				// this branch is currently impossible to test and run into because we pick which events we're interested in
				// through the _registeredEvents above, and the stream won't hand us events we didn't register for
				m.UnsupportedEvents.Inc()
				logger.Error().Interface("event", e).Msg("unhandled event")
			case ocevents.UserProfilePictureSyncRequested:
				if ev.Executant.GetOpaqueId() == "" || ev.PictureURL == "" {
					m.InvalidEvents.Inc()
					logger.Error().Interface("event", ev).Msg("Received invalid profile picture sync event")
					continue
				}
				if photoSyncer != nil {
					go syncProfilePicture(ctx, logger, photoSyncer, publisher, ev)
				}
			case events.UserSignedIn:
				name := "UserSignedIn"
				userId := ""
				if ev.Executant != nil && ev.Executant.OpaqueId != "" {
					userId = ev.Executant.OpaqueId
				} else {
					m.InvalidEvents.Inc()
					logger.Error().Err(err).Interface("event", ev).Msg("Received invalid event: executant.opaqueId not set")
					continue
				}
				if err := backend.UpdateLastSignInDate(ctx, userId, utils.TSToTime(ev.Timestamp)); err != nil {
					result := metrics.ResultFailure
					if errorcode.IsErrorCode(err, errorcode.ItemNotFound) {
						result = metrics.ResultNotFound
					}
					m.EventsProcessed.WithLabelValues(name, result).Inc()
					logger.Error().Err(err).Str("userid", userId).Str("event", name).Msg("Error updating last sign in date")
				} else {
					m.EventsProcessed.WithLabelValues(name, metrics.ResultSuccess).Inc()
					logger.Debug().Str("userid", userId).Str("event", name).Msg("Successfully updated last sign in date")
				}
			}
			if stop.Load() {
				loop = false
			}
		case <-stopCh:
			logger.Info().Msg("instructed to stop")
			loop = false
		case <-ctx.Done():
			logger.Info().Msg("context cancelled")
			loop = false
		}
	}
	return nil
}

// syncProfilePicture runs outside the event loop so downloads do not delay sign-in events.
func syncProfilePicture(ctx context.Context, logger *log.Logger, syncer PhotoSyncer, publisher events.Publisher, ev ocevents.UserProfilePictureSyncRequested) {
	if ctx.Err() != nil {
		return
	}
	if err := syncer.SyncPhotoFromURL(ctx, ev.Executant.GetOpaqueId(), ev.PictureURL); err != nil {
		logger.Warn().Err(err).Str("userid", ev.Executant.GetOpaqueId()).Msg("Failed to sync profile picture from OIDC claim")
		return
	}
	if ctx.Err() != nil || publisher == nil {
		return
	}
	if err := events.Publish(ctx, publisher, ocevents.UserProfilePictureUpdated{
		Executant: ev.Executant,
		Timestamp: utils.TimeToTS(time.Now()),
	}); err != nil {
		logger.Warn().Err(err).Str("userid", ev.Executant.GetOpaqueId()).Msg("Failed to publish profile picture updated event")
	}
}
