package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/opencloud-eu/opencloud/services/postprocessing/pkg/config"
	"github.com/opencloud-eu/opencloud/services/postprocessing/pkg/metrics"
	"github.com/opencloud-eu/opencloud/services/postprocessing/pkg/postprocessing"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/raw"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"go-micro.dev/v4/store"
	"go.opentelemetry.io/otel/trace"
)

// PostprocessingService is an instance of the service handling postprocessing of files
type PostprocessingService struct {
	ctx     context.Context
	log     log.Logger
	events  <-chan raw.Event
	pub     events.Publisher
	steps   []events.Postprocessingstep
	store   store.Store
	c       config.Postprocessing
	tp      trace.TracerProvider
	metrics *metrics.Metrics
}

var (
	// ErrFatal is returned when a fatal error occurs and we want to exit.
	ErrFatal = errors.New("fatal error")
	// ErrEvent is returned when something went wrong with a specific event.
	ErrEvent = errors.New("event error")
	// ErrNotFound is returned when a postprocessing is not found in the store.
	ErrNotFound = errors.New("postprocessing not found")
)

// NewPostprocessingService returns a new instance of a postprocessing service
func NewPostprocessingService(ctx context.Context, logger log.Logger, sto store.Store, tp trace.TracerProvider, cfg *config.Config) (*PostprocessingService, error) {
	pub, err := stream.NatsFromConfig(cfg.Service.Name, false, stream.NatsConfig{
		Endpoint:             cfg.Postprocessing.Events.Endpoint,
		Cluster:              cfg.Postprocessing.Events.Cluster,
		EnableTLS:            cfg.Postprocessing.Events.EnableTLS,
		TLSInsecure:          cfg.Postprocessing.Events.TLSInsecure,
		TLSRootCACertificate: cfg.Postprocessing.Events.TLSRootCACertificate,
		AuthUsername:         cfg.Postprocessing.Events.AuthUsername,
		AuthPassword:         cfg.Postprocessing.Events.AuthPassword,
	})

	if err != nil {
		return nil, err
	}
	raw, err := raw.FromConfig(ctx, cfg.Service.Name, raw.Config{
		Endpoint:             cfg.Postprocessing.Events.Endpoint,
		Cluster:              cfg.Postprocessing.Events.Cluster,
		EnableTLS:            cfg.Postprocessing.Events.EnableTLS,
		TLSInsecure:          cfg.Postprocessing.Events.TLSInsecure,
		TLSRootCACertificate: cfg.Postprocessing.Events.TLSRootCACertificate,
		AuthUsername:         cfg.Postprocessing.Events.AuthUsername,
		AuthPassword:         cfg.Postprocessing.Events.AuthPassword,
		MaxAckPending:        cfg.Postprocessing.Events.MaxAckPending,
		AckWait:              cfg.Postprocessing.Events.AckWait,
	})

	evs, err := raw.Consume("postprocessing-pull",
		events.BytesReceived{},
		events.StartPostprocessingStep{},
		events.UploadReady{},
		events.PostprocessingStepFinished{},
		events.ResumePostprocessing{})
	if err != nil {
		return nil, err
	}

	m := metrics.New()
	m.BuildInfo.WithLabelValues(version.GetString()).Set(1)
	monitorMetrics(raw, "postprocessing-pull", m, logger)

	return &PostprocessingService{
		ctx:     ctx,
		log:     logger,
		events:  evs,
		pub:     pub,
		steps:   getSteps(cfg.Postprocessing),
		store:   sto,
		c:       cfg.Postprocessing,
		tp:      tp,
		metrics: m,
	}, nil
}

// Run to fulfil Runner interface
func (pps *PostprocessingService) Run() error {
	wg := sync.WaitGroup{}

	for i := 0; i < pps.c.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for e := range pps.events {
				if err := pps.processEvent(e); err != nil {
					switch {
					case errors.Is(err, ErrFatal):
						pps.log.Fatal().Err(err).Msg("fatal error - exiting")
					case errors.Is(err, ErrEvent):
						pps.log.Error().Err(err).Msg("continuing")
					default:
						pps.log.Fatal().Err(err).Msg("unknown error - exiting")
					}
				}
			}
		}()
	}
	wg.Wait()

	return nil
}

func (pps *PostprocessingService) processEvent(e raw.Event) error {
	var (
		next interface{}
		pp   *postprocessing.Postprocessing
		err  error
	)

	ctx := e.GetTraceContext(pps.ctx)
	ctx, span := pps.tp.Tracer("postprocessing").Start(ctx, "processEvent")
	defer span.End()

	ackEvent := true
	defer func() {
		if ackEvent {
			if err := e.Ack(); err != nil {
				pps.log.Error().Err(err).Msg("unable to ack event")
			}
		}
	}()

	switch ev := e.Event.Event.(type) {
	case events.BytesReceived:
		pp = &postprocessing.Postprocessing{
			ID:                ev.UploadID,
			URL:               ev.URL,
			User:              ev.ExecutingUser,
			Filename:          ev.Filename,
			Filesize:          ev.Filesize,
			ResourceID:        ev.ResourceID,
			Steps:             pps.steps,
			InitiatorID:       e.InitiatorID,
			ImpersonatingUser: ev.ImpersonatingUser,
			StartTime:         time.Now(),
		}
		pps.metrics.InProgress.Inc()
		next = pp.Init(ev)
	case events.PostprocessingStepFinished:
		if ev.UploadID == "" {
			// no current upload - this was an on demand scan
			return nil
		}
		pp, err = pps.getPP(pps.store, ev.UploadID)
		if err != nil {
			pps.log.Error().Str("uploadID", ev.UploadID).Err(err).Msg("cannot get upload")
			return fmt.Errorf("%w: cannot get upload", ErrEvent)
		}
		next = pp.NextStep(ev)

		switch pp.Status.Outcome {
		case events.PPOutcomeRetry:
			// schedule retry
			backoff := pp.BackoffDuration()
			go func() {
				time.Sleep(backoff)
				retryEvent := events.StartPostprocessingStep{
					UploadID:          pp.ID,
					URL:               pp.URL,
					ExecutingUser:     pp.User,
					Filename:          pp.Filename,
					Filesize:          pp.Filesize,
					ResourceID:        pp.ResourceID,
					StepToStart:       pp.Status.CurrentStep,
					ImpersonatingUser: pp.ImpersonatingUser,
				}
				err := events.Publish(ctx, pps.pub, retryEvent)
				if err != nil {
					pps.log.Error().Str("uploadID", ev.UploadID).Err(err).Msg("cannot publish RestartPostprocessing event")
				}
			}()
		}
	case events.StartPostprocessingStep:
		if ev.StepToStart != events.PPStepDelay {
			return nil
		}
		pp, err = pps.getPP(pps.store, ev.UploadID)
		if err != nil {
			pps.log.Error().Str("uploadID", ev.UploadID).Err(err).Msg("cannot get upload")
			return fmt.Errorf("%w: cannot get upload", ErrEvent)
		}
		pp.Delay(func(next interface{}) {
			if err := events.Publish(ctx, pps.pub, next); err != nil {
				pps.log.Error().Err(err).Msg("cannot publish event")
			}
		})
	case events.UploadReady:
		pps.metrics.InProgress.Dec()
		// the upload failed - let's keep it around for a while - but mark it as finished
		pp, err = pps.getPP(pps.store, ev.UploadID)
		if err != nil {
			pps.log.Error().Str("uploadID", ev.UploadID).Err(err).Msg("cannot get upload")
			return fmt.Errorf("%w: cannot get upload", ErrEvent)
		}

		if ev.Failed {
			pps.metrics.Finished.WithLabelValues("failed").Inc()
			if !pp.StartTime.IsZero() {
				pps.metrics.Duration.WithLabelValues("failed").Observe(time.Since(pp.StartTime).Seconds())
			}
			pp.Finished = true
			return storePP(pps.store, pp)
		}

		pps.metrics.Finished.WithLabelValues("succeeded").Inc()
		if !pp.StartTime.IsZero() {
			pps.metrics.Duration.WithLabelValues("succeeded").Observe(time.Since(pp.StartTime).Seconds())
		}
		// the storage provider thinks the upload is done - so no need to keep it any more
		if err := pps.store.Delete(ev.UploadID); err != nil {
			pps.log.Error().Str("uploadID", ev.UploadID).Err(err).Msg("cannot delete upload")
			return fmt.Errorf("%w: cannot delete upload", ErrEvent)
		}
	case events.ResumePostprocessing:
		return pps.handleResumePPEvent(ctx, ev)
	}

	if pp != nil {
		ctx = ctxpkg.ContextSetInitiator(ctx, pp.InitiatorID)

		if err := storePP(pps.store, pp); err != nil {
			ackEvent = false
			pps.log.Error().Str("uploadID", pp.ID).Err(err).Msg("cannot store upload")
			return fmt.Errorf("%w: cannot store upload", ErrEvent)
		}
	}

	if next != nil {
		if err := events.Publish(ctx, pps.pub, next); err != nil {
			pps.log.Error().Err(err).Msg("unable to publish event")
			return fmt.Errorf("%w: unable to publish event", ErrFatal) // we can't publish -> we are screwed
		}
	}
	return nil
}

func (pps *PostprocessingService) getPP(sto store.Store, uploadID string) (*postprocessing.Postprocessing, error) {
	recs, err := sto.Read(uploadID)
	if err != nil {
		if err == store.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if len(recs) == 0 {
		return nil, ErrNotFound
	}

	if len(recs) > 1 {
		return nil, fmt.Errorf("expected only one result for '%s', got %d", uploadID, len(recs))
	}

	pp := postprocessing.New(pps.c)
	err = json.Unmarshal(recs[0].Value, pp)
	if err != nil {
		return nil, err
	}

	return pp, nil
}

func getSteps(c config.Postprocessing) []events.Postprocessingstep {
	// NOTE: improved version only allows configuring order of postprocessing steps
	// But we aim for a system where postprocessing steps can be configured per space, ideally by the spaceadmin itself
	// We need to iterate over configuring PP service when we see fit
	steps := make([]events.Postprocessingstep, 0, len(c.Steps))
	for _, s := range c.Steps {
		steps = append(steps, events.Postprocessingstep(s))
	}

	return steps
}

func storePP(sto store.Store, pp *postprocessing.Postprocessing) error {
	b, err := json.Marshal(pp)
	if err != nil {
		return err
	}

	return sto.Write(&store.Record{
		Key:   pp.ID,
		Value: b,
	})
}

func (pps *PostprocessingService) handleResumePPEvent(ctx context.Context, ev events.ResumePostprocessing) error {
	ids := []string{ev.UploadID}
	if ev.Step != "" {
		ids = pps.findUploadsByStep(ev.Step)
	}

	for _, id := range ids {
		if err := pps.resumePP(ctx, id); err != nil {
			pps.log.Error().Str("uploadID", id).Err(err).Msg("cannot resume upload")
		}
	}
	return nil
}

func (pps *PostprocessingService) resumePP(ctx context.Context, uploadID string) error {
	pp, err := pps.getPP(pps.store, uploadID)
	if err != nil {
		if err == ErrNotFound {
			if err := events.Publish(ctx, pps.pub, events.RestartPostprocessing{
				UploadID:  uploadID,
				Timestamp: utils.TSNow(),
			}); err != nil {
				return err
			}
			return nil
		}
		return fmt.Errorf("cannot get upload: %w", err)
	}

	if pp.Finished {
		// dont retry finished uploads
		return nil
	}

	return events.Publish(ctx, pps.pub, pp.CurrentStep())
}

func (pps *PostprocessingService) findUploadsByStep(step events.Postprocessingstep) []string {
	var ids []string

	keys, err := pps.store.List()
	if err != nil {
		pps.log.Error().Err(err).Msg("cannot list uploads")
	}

	for _, k := range keys {
		rec, err := pps.store.Read(k)
		if err != nil {
			pps.log.Error().Err(err).Msg("cannot read upload")
			continue
		}

		if len(rec) != 1 {
			pps.log.Error().Err(err).Msg("expected only one result")
			continue
		}

		pp := &postprocessing.Postprocessing{}
		err = json.Unmarshal(rec[0].Value, pp)
		if err != nil {
			pps.log.Error().Err(err).Msg("cannot unmarshal upload")
			continue
		}

		if pp.Status.CurrentStep == step {
			ids = append(ids, pp.ID)
		}
	}

	return ids
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
			logger.Trace().Msg("updated postprocessing event metrics")
		}
	}()
}
