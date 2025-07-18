package search

import (
	"context"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
	"github.com/opencloud-eu/opencloud/services/search/pkg/metrics"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/raw"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
)

// HandleEvents listens to the needed events,
// it handles the whole resource indexing livecycle.
func HandleEvents(s Searcher, stream raw.Stream, cfg *config.Config, m *metrics.Metrics, logger log.Logger) error {
	evts := []events.Unmarshaller{
		events.ItemTrashed{},
		events.ItemRestored{},
		events.ItemMoved{},
		events.ContainerCreated{},
		events.FileTouched{},
		events.FileVersionRestored{},
		events.TagsAdded{},
		events.TagsRemoved{},
		events.SpaceRenamed{},
	}

	if cfg.Events.AsyncUploads {
		evts = append(evts, events.UploadReady{})
	} else {
		evts = append(evts, events.FileUploaded{})
	}

	ch, err := stream.Consume("search-pull", evts...)
	if err != nil {
		return err
	}

	if m != nil {
		monitorMetrics(stream, "search-pull", m, logger)
	}

	if cfg.Events.NumConsumers == 0 {
		cfg.Events.NumConsumers = 1
	}

	getSpaceID := func(ref *provider.Reference) *provider.StorageSpaceId {
		return &provider.StorageSpaceId{
			OpaqueId: storagespace.FormatResourceID(
				&provider.ResourceId{
					StorageId: ref.GetResourceId().GetStorageId(),
					SpaceId:   ref.GetResourceId().GetSpaceId(),
				},
			),
		}
	}

	indexSpaceDebouncer := NewSpaceDebouncer(time.Duration(cfg.Events.DebounceDuration)*time.Millisecond, 30*time.Second, func(id *provider.StorageSpaceId) {
		if err := s.IndexSpace(id); err != nil {
			logger.Error().Err(err).Interface("spaceID", id).Msg("error while indexing a space")
		}
	}, logger)

	for i := 0; i < cfg.Events.NumConsumers; i++ {
		go func(s Searcher, ch <-chan raw.Event) {
			for event := range ch {
				e := event
				go func() {
					e.InProgress() // let nats know that we are processing this event
					logger.Debug().Interface("event", e).Msg("updating index")

					switch ev := e.Event.Event.(type) {
					case events.ItemTrashed:
						s.TrashItem(ev.ID)
						indexSpaceDebouncer.Debounce(getSpaceID(ev.Ref), e.Ack)
					case events.ItemMoved:
						s.MoveItem(ev.Ref)
						indexSpaceDebouncer.Debounce(getSpaceID(ev.Ref), e.Ack)
					case events.ItemRestored:
						s.RestoreItem(ev.Ref)
						indexSpaceDebouncer.Debounce(getSpaceID(ev.Ref), e.Ack)
					case events.ContainerCreated:
						indexSpaceDebouncer.Debounce(getSpaceID(ev.Ref), e.Ack)
					case events.FileTouched:
						indexSpaceDebouncer.Debounce(getSpaceID(ev.Ref), e.Ack)
					case events.FileVersionRestored:
						indexSpaceDebouncer.Debounce(getSpaceID(ev.Ref), e.Ack)
					case events.TagsAdded:
						s.UpsertItem(ev.Ref)
					case events.TagsRemoved:
						s.UpsertItem(ev.Ref)
					case events.FileUploaded:
						indexSpaceDebouncer.Debounce(getSpaceID(ev.Ref), e.Ack)
					case events.UploadReady:
						indexSpaceDebouncer.Debounce(getSpaceID(ev.FileRef), e.Ack)
					case events.SpaceRenamed:
						indexSpaceDebouncer.Debounce(ev.ID, e.Ack)
					}
				}()
			}
		}(
			s,
			ch,
		)
	}

	return nil
}

func monitorMetrics(stream raw.Stream, name string, m *metrics.Metrics, logger log.Logger) {
	ctx := context.Background()
	consumer, err := stream.JetStream().Consumer(ctx, name)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get consumer")
	}
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for range ticker.C {
			info, err := consumer.Info(ctx)
			if err != nil {
				logger.Error().Err(err).Msg("failed to get consumer")
			}

			m.EventsOutstandingAcks.Set(float64(info.NumAckPending))
			m.EventsUnprocessed.Set(float64(info.NumPending))
			m.EventsRedelivered.Set(float64(info.NumRedelivered))
			logger.Trace().Msg("updated search event metrics")
		}
	}()
}
