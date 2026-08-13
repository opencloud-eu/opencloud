package events

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/config"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/data"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/service/activitylog"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func init() {
	tracer = otel.Tracer("github.com/opencloud-eu/opencloud/services/activitylog/pkg/service/events")
}

var (
	_numConsumersDefault = 1
)

// ActivitylogService logs events per resource
type ActivitylogService struct {
	ctx    context.Context
	sa     config.ServiceAccount
	log    log.Logger
	stream events.Stream
	gws    pool.Selectable[gateway.GatewayAPIClient]
	al     *activitylog.ActivityLog

	numConsumers int

	events []events.Unmarshaller

	stopCh  chan struct{}
	stopped *atomic.Bool
}

// New creates a new ActivitylogService
func New(al *activitylog.ActivityLog, stream events.Stream, opts ...Option) (*ActivitylogService, error) {
	o := &Options{
		NumConsumers: _numConsumersDefault,
	}
	for _, opt := range opts {
		opt(o)
	}

	s := &ActivitylogService{
		ctx:          o.Context,
		log:          o.Logger,
		sa:           o.ServiceAccount,
		stream:       stream,
		gws:          o.GatewaySelector,
		events:       o.RegisteredEvents,
		numConsumers: o.NumConsumers,
		al:           al,
		stopCh:       make(chan struct{}, 1),
		stopped:      new(atomic.Bool),
	}

	return s, nil
}

// Run to fulfil Runner interface
func (s *ActivitylogService) Run() error {
	ch, err := events.Consume(s.stream, "activitylog-pull", s.events...)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	s.log.Debug().Int("worker.count", s.numConsumers).
		Str("messaging.consumer.group.name", "activitylog").
		Str("messaging.system", "nats").
		Str("messaging.operation.name", "receive").
		Msg("starting event processing workers")

	// start workers
	for i := 0; i < s.numConsumers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case e, ok := <-ch:
					if !ok {
						return
					}
					if err := s.processEvent(e); err != nil {
						s.log.Error().Err(err).
							Int("worker", workerID).
							Interface("event", e).
							Msg("failed to process event")
					}
				}
			}
		}(i)
	}

	// wait for stop signal
	<-s.stopCh
	cancel() // signal workers to stop
	wg.Wait()

	return nil
}

// Close will make the service to stop processing, so the `Run`
// method can finish.
func (s *ActivitylogService) Close() {
	if s.stopped.CompareAndSwap(false, true) {
		close(s.stopCh)
	}
}

func (s *ActivitylogService) processEvent(e events.Event) error {
	ctx := e.GetTraceContext(s.ctx)
	ctx, span := tracer.Start(ctx, "processEvent")
	defer span.End()

	s.log.Debug().Interface("event", e).Msg("updating activitylog")

	switch ev := e.Event.(type) {
	case events.UploadReady:
		return s.AddActivity(ctx, ev.FileRef, ev.ParentID, e.ID, utils.TSToTime(ev.Timestamp))
	case events.FileTouched:
		return s.AddActivity(ctx, ev.Ref, ev.ParentID, e.ID, utils.TSToTime(ev.Timestamp))
	// Disabled https://github.com/owncloud/ocis/issues/10293
	//case events.FileDownloaded:
	// we are only interested in public link downloads - so no need to store others.
	//if ev.ImpersonatingUser.GetDisplayName() == "Public" {
	//	err = a.AddActivity(ev.Ref, e.ID, utils.TSToTime(ev.Timestamp))
	//}
	case events.ContainerCreated:
		return s.AddActivity(ctx, ev.Ref, ev.ParentID, e.ID, utils.TSToTime(ev.Timestamp))
	case events.ItemTrashed:
		return s.AddActivityTrashed(ctx, ev.ID, ev.Ref, nil, e.ID, utils.TSToTime(ev.Timestamp))
	case events.ItemPurged:
		return s.al.RemoveResource(ev.ID)
	case events.ItemMoved:
		// remove the cached parent id for this resource
		s.removeCachedParentID(ctx, ev.Ref)

		return s.AddActivity(ctx, ev.Ref, nil, e.ID, utils.TSToTime(ev.Timestamp))
	case events.ShareCreated:
		return s.AddActivity(ctx, toRef(ev.ItemID), nil, e.ID, utils.TSToTime(ev.CTime))
	case events.ShareUpdated:
		if ev.Sharer != nil && ev.ItemID != nil && ev.Sharer.GetOpaqueId() != ev.ItemID.GetSpaceId() {
			return s.AddActivity(ctx, toRef(ev.ItemID), nil, e.ID, utils.TSToTime(ev.MTime))
		}
	case events.ShareRemoved:
		return s.AddActivity(ctx, toRef(ev.ItemID), nil, e.ID, ev.Timestamp)
	case events.LinkCreated:
		return s.AddActivity(ctx, toRef(ev.ItemID), nil, e.ID, utils.TSToTime(ev.CTime))
	case events.LinkUpdated:
		if ev.Sharer != nil && ev.ItemID != nil && ev.Sharer.GetOpaqueId() != ev.ItemID.GetSpaceId() {
			return s.AddActivity(ctx, toRef(ev.ItemID), nil, e.ID, utils.TSToTime(ev.MTime))
		}
	case events.LinkRemoved:
		return s.AddActivity(ctx, toRef(ev.ItemID), nil, e.ID, utils.TSToTime(ev.Timestamp))
	case events.SpaceShared:
		return s.AddSpaceActivity(ctx, ev.ID, e.ID, ev.Timestamp)
	case events.SpaceUnshared:
		return s.AddSpaceActivity(ctx, ev.ID, e.ID, ev.Timestamp)
	}

	return nil
}

// AddActivity adds the activity to the given resource and all its parents
func (a *ActivitylogService) AddActivity(ctx context.Context, initRef *provider.Reference, parentId *provider.ResourceId, eventID string, timestamp time.Time) error {
	ctx, span := tracer.Start(ctx, "AddActivity")
	defer span.End()

	gwc, err := a.gws.Next()
	if err != nil {
		return fmt.Errorf("cant get gateway client: %w", err)
	}

	ctx, err = utils.GetServiceUserContextWithContext(ctx, gwc, a.sa.ServiceAccountID, a.sa.ServiceAccountSecret)
	if err != nil {
		return fmt.Errorf("cant get service user context: %w", err)

	}
	return a.al.AddActivity(ctx, initRef, parentId, eventID, timestamp, func(ctx context.Context, ref *provider.Reference) (*provider.ResourceInfo, error) {
		return utils.GetResource(ctx, ref, gwc)
	})
}

// AddActivityTrashed adds the activity to given trashed resource and all its former parents
func (a *ActivitylogService) AddActivityTrashed(ctx context.Context, resourceID *provider.ResourceId, reference *provider.Reference, parentId *provider.ResourceId, eventID string, timestamp time.Time) error {
	ctx, span := tracer.Start(ctx, "AddActivityTrashed")
	defer span.End()

	gwc, err := a.gws.Next()
	if err != nil {
		return fmt.Errorf("cant get gateway client: %w", err)
	}

	ctx, err = utils.GetServiceUserContextWithContext(ctx, gwc, a.sa.ServiceAccountID, a.sa.ServiceAccountSecret)
	if err != nil {
		return fmt.Errorf("cant get service user context: %w", err)
	}

	// store activity on trashed item
	if err := a.al.StoreActivity(storagespace.FormatResourceID(resourceID), []data.RawActivity{
		{
			EventID:   eventID,
			Depth:     0,
			Timestamp: timestamp,
		},
	}); err != nil {
		return fmt.Errorf("could not store activity: %w", err)
	}

	// get previous parent
	ref := &provider.Reference{
		ResourceId: reference.GetResourceId(),
		Path:       filepath.Dir(reference.GetPath()),
	}

	return a.al.AddActivity(ctx, ref, parentId, eventID, timestamp, func(ctx context.Context, ref *provider.Reference) (*provider.ResourceInfo, error) {
		return utils.GetResource(ctx, ref, gwc)
	})
}

// AddSpaceActivity adds the activity to the given spaceroot
func (a *ActivitylogService) AddSpaceActivity(ctx context.Context, spaceID *provider.StorageSpaceId, eventID string, timestamp time.Time) error {
	_, span := tracer.Start(ctx, "AddSpaceActivity")
	defer span.End()
	// spaceID is in format <providerid>$<spaceid>
	// activitylog service uses format <providerid>$<spaceid>!<resourceid>
	// lets do some converting, shall we?
	rid, err := storagespace.ParseID(spaceID.GetOpaqueId())
	if err != nil {
		return fmt.Errorf("could not parse space id: %w", err)
	}
	rid.OpaqueId = rid.GetSpaceId()
	err = a.al.StoreActivity(storagespace.FormatResourceID(&rid), []data.RawActivity{
		{
			EventID:   eventID,
			Depth:     0,
			Timestamp: timestamp,
		},
	})
	if err != nil {
		return fmt.Errorf("could not store activity: %w", err)
	}
	return nil
}

func toRef(r *provider.ResourceId) *provider.Reference {
	return &provider.Reference{
		ResourceId: r,
	}
}

func (a *ActivitylogService) removeCachedParentID(ctx context.Context, ref *provider.Reference) {
	var span trace.Span
	ctx, span = tracer.Start(ctx, "removeCachedParentID")
	defer span.End()

	purgeId := ref.GetResourceId()
	if ref.GetPath() != "" {
		gwc, err := a.gws.Next()
		if err != nil {
			a.log.Error().Err(err).Msg("could not get gateway client")
			return
		}

		ctx, err = utils.GetServiceUserContextWithContext(ctx, gwc, a.sa.ServiceAccountID, a.sa.ServiceAccountSecret)
		if err != nil {
			a.log.Error().Err(err).Msg("could not get service user context")
			return
		}

		info, err := utils.GetResource(ctx, ref, gwc)
		if err != nil {
			a.log.Error().Err(err).Msg("could not get resource info")
			return
		}
		purgeId = info.GetId()
	}
	a.al.InvalidateCachedParentID(purgeId)
}
