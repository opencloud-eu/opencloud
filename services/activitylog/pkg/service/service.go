package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/jellydator/ttlcache/v2"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/vmihailenco/msgpack/v5"
	microstore "go-micro.dev/v4/store"
	"go.opentelemetry.io/otel/trace"

	"github.com/opencloud-eu/opencloud/pkg/log"
	ehsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/eventhistory/v0"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/config"
)

// Nats runs into max payload exceeded errors at around 7k activities. Let's keep a buffer.
var _maxActivities = 6000

// RawActivity represents an activity as it is stored in the activitylog store
type RawActivity struct {
	EventID   string    `json:"event_id"`
	Depth     int       `json:"depth"`
	Timestamp time.Time `json:"timestamp"`
}

// ActivitylogService logs events per resource
type ActivitylogService struct {
	cfg           *config.Config
	log           log.Logger
	events        <-chan events.Event
	store         microstore.Store
	gws           pool.Selectable[gateway.GatewayAPIClient]
	mux           *chi.Mux
	evHistory     ehsvc.EventHistoryService
	valService    settingssvc.ValueService
	lock          sync.RWMutex
	tp            trace.TracerProvider
	tracer        trace.Tracer
	debouncer     *Debouncer
	parentIdCache *ttlcache.Cache

	registeredEvents map[string]events.Unmarshaller
}

type Debouncer struct {
	after      time.Duration
	f          func(id string, ra []RawActivity) error
	pending    sync.Map
	inProgress sync.Map

	mutex sync.Mutex
}

type queueItem struct {
	activities []RawActivity
	timer      *time.Timer
}

// NewDebouncer returns a new Debouncer instance
func NewDebouncer(d time.Duration, f func(id string, ra []RawActivity) error) *Debouncer {
	return &Debouncer{
		after:      d,
		f:          f,
		pending:    sync.Map{},
		inProgress: sync.Map{},
	}
}

// Debounce restarts the debounce timer for the given space
func (d *Debouncer) Debounce(id string, ra RawActivity) {
	if d.after == 0 {
		d.f(id, []RawActivity{ra})
		return
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	activities := []RawActivity{ra}
	item := &queueItem{
		activities: activities,
	}
	if i, ok := d.pending.Load(id); ok {
		// if the item is already in the queue, append the new activities
		item, ok = i.(*queueItem)
		if ok {
			item.activities = append(item.activities, ra)
		}
	}

	if item.timer == nil {
		item.timer = time.AfterFunc(d.after, func() {
			if _, ok := d.inProgress.Load(id); ok {
				// Reschedule this run for when the previous run has finished
				d.mutex.Lock()
				if i, ok := d.pending.Load(id); ok {
					i.(*queueItem).timer.Reset(d.after)
				}

				d.mutex.Unlock()
				return
			}

			d.pending.Delete(id)
			d.inProgress.Store(id, true)
			defer d.inProgress.Delete(id)
			d.f(id, item.activities)
		})
	}

	d.pending.Store(id, item)
}

// New creates a new ActivitylogService
func New(opts ...Option) (*ActivitylogService, error) {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}

	if o.Stream == nil {
		return nil, errors.New("stream is required")
	}

	if o.Store == nil {
		return nil, errors.New("store is required")
	}

	ch, err := events.Consume(o.Stream, o.Config.Service.Name, o.RegisteredEvents...)
	if err != nil {
		return nil, err
	}

	cache := ttlcache.NewCache()
	err = cache.SetTTL(30 * time.Second)
	if err != nil {
		return nil, err
	}

	s := &ActivitylogService{
		log:              o.Logger,
		cfg:              o.Config,
		events:           ch,
		store:            o.Store,
		gws:              o.GatewaySelector,
		mux:              o.Mux,
		evHistory:        o.HistoryClient,
		valService:       o.ValueClient,
		lock:             sync.RWMutex{},
		registeredEvents: make(map[string]events.Unmarshaller),
		tp:               o.TraceProvider,
		tracer:           o.TraceProvider.Tracer("github.com/opencloud-eu/opencloud/services/activitylog/pkg/service"),
		parentIdCache:    cache,
	}
	s.debouncer = NewDebouncer(o.WriteBufferDuration, s.storeActivity)

	s.mux.Get("/graph/v1beta1/extensions/org.libregraph/activities", s.HandleGetItemActivities)

	for _, e := range o.RegisteredEvents {
		typ := reflect.TypeOf(e)
		s.registeredEvents[typ.String()] = e
	}

	go s.Run()

	return s, nil
}

// Run runs the service
func (a *ActivitylogService) Run() {
	for e := range a.events {
		var err error
		switch ev := e.Event.(type) {
		case events.UploadReady:
			err = a.AddActivity(ev.FileRef, ev.ParentID, e.ID, utils.TSToTime(ev.Timestamp))
		case events.FileTouched:
			err = a.AddActivity(ev.Ref, ev.ParentID, e.ID, utils.TSToTime(ev.Timestamp))
		// Disabled https://github.com/owncloud/ocis/issues/10293
		//case events.FileDownloaded:
		// we are only interested in public link downloads - so no need to store others.
		//if ev.ImpersonatingUser.GetDisplayName() == "Public" {
		//	err = a.AddActivity(ev.Ref, e.ID, utils.TSToTime(ev.Timestamp))
		//}
		case events.ContainerCreated:
			err = a.AddActivity(ev.Ref, ev.ParentID, e.ID, utils.TSToTime(ev.Timestamp))
		case events.ItemTrashed:
			err = a.AddActivityTrashed(ev.ID, ev.Ref, nil, e.ID, utils.TSToTime(ev.Timestamp))
		case events.ItemPurged:
			err = a.RemoveResource(ev.ID)
		case events.ItemMoved:
			// remove the cached parent id for this resource
			a.removeCachedParentID(ev.Ref)

			err = a.AddActivity(ev.Ref, nil, e.ID, utils.TSToTime(ev.Timestamp))
		case events.ShareCreated:
			err = a.AddActivity(toRef(ev.ItemID), nil, e.ID, utils.TSToTime(ev.CTime))
		case events.ShareUpdated:
			if ev.Sharer != nil && ev.ItemID != nil && ev.Sharer.GetOpaqueId() != ev.ItemID.GetSpaceId() {
				err = a.AddActivity(toRef(ev.ItemID), nil, e.ID, utils.TSToTime(ev.MTime))
			}
		case events.ShareRemoved:
			err = a.AddActivity(toRef(ev.ItemID), nil, e.ID, ev.Timestamp)
		case events.LinkCreated:
			err = a.AddActivity(toRef(ev.ItemID), nil, e.ID, utils.TSToTime(ev.CTime))
		case events.LinkUpdated:
			if ev.Sharer != nil && ev.ItemID != nil && ev.Sharer.GetOpaqueId() != ev.ItemID.GetSpaceId() {
				err = a.AddActivity(toRef(ev.ItemID), nil, e.ID, utils.TSToTime(ev.MTime))
			}
		case events.LinkRemoved:
			err = a.AddActivity(toRef(ev.ItemID), nil, e.ID, utils.TSToTime(ev.Timestamp))
		case events.SpaceShared:
			err = a.AddSpaceActivity(ev.ID, e.ID, ev.Timestamp)
		case events.SpaceUnshared:
			err = a.AddSpaceActivity(ev.ID, e.ID, ev.Timestamp)
		}

		if err != nil {
			a.log.Error().Err(err).Interface("event", e).Msg("could not process event")
		}
	}
}

// AddActivity adds the activity to the given resource and all its parents
func (a *ActivitylogService) AddActivity(initRef *provider.Reference, parentId *provider.ResourceId, eventID string, timestamp time.Time) error {
	gwc, err := a.gws.Next()
	if err != nil {
		return fmt.Errorf("cant get gateway client: %w", err)
	}

	ctx, err := utils.GetServiceUserContext(a.cfg.ServiceAccount.ServiceAccountID, gwc, a.cfg.ServiceAccount.ServiceAccountSecret)
	if err != nil {
		return fmt.Errorf("cant get service user context: %w", err)
	}
	var span trace.Span
	ctx, span = a.tracer.Start(ctx, "AddActivity")
	defer span.End()

	return a.addActivity(ctx, initRef, parentId, eventID, timestamp, func(ref *provider.Reference) (*provider.ResourceInfo, error) {
		return utils.GetResource(ctx, ref, gwc)
	})
}

// AddActivityTrashed adds the activity to given trashed resource and all its former parents
func (a *ActivitylogService) AddActivityTrashed(resourceID *provider.ResourceId, reference *provider.Reference, parentId *provider.ResourceId, eventID string, timestamp time.Time) error {
	gwc, err := a.gws.Next()
	if err != nil {
		return fmt.Errorf("cant get gateway client: %w", err)
	}

	ctx, err := utils.GetServiceUserContext(a.cfg.ServiceAccount.ServiceAccountID, gwc, a.cfg.ServiceAccount.ServiceAccountSecret)
	if err != nil {
		return fmt.Errorf("cant get service user context: %w", err)
	}

	// store activity on trashed item
	if err := a.storeActivity(storagespace.FormatResourceID(resourceID), []RawActivity{
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

	var span trace.Span
	ctx, span = a.tracer.Start(ctx, "AddActivity")
	defer span.End()

	return a.addActivity(ctx, ref, parentId, eventID, timestamp, func(ref *provider.Reference) (*provider.ResourceInfo, error) {
		return utils.GetResource(ctx, ref, gwc)
	})
}

// AddSpaceActivity adds the activity to the given spaceroot
func (a *ActivitylogService) AddSpaceActivity(spaceID *provider.StorageSpaceId, eventID string, timestamp time.Time) error {
	// spaceID is in format <providerid>$<spaceid>
	// activitylog service uses format <providerid>$<spaceid>!<resourceid>
	// lets do some converting, shall we?
	rid, err := storagespace.ParseID(spaceID.GetOpaqueId())
	if err != nil {
		return fmt.Errorf("could not parse space id: %w", err)
	}
	rid.OpaqueId = rid.GetSpaceId()
	return a.storeActivity(storagespace.FormatResourceID(&rid), []RawActivity{
		{
			EventID:   eventID,
			Depth:     0,
			Timestamp: timestamp,
		},
	})

}

// Activities returns the activities for the given resource
func (a *ActivitylogService) Activities(rid *provider.ResourceId) ([]RawActivity, error) {
	a.lock.RLock()
	defer a.lock.RUnlock()

	return a.activities(rid)
}

// RemoveActivities removes the activities from the given resource
func (a *ActivitylogService) RemoveActivities(rid *provider.ResourceId, toDelete map[string]struct{}) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	curActivities, err := a.activities(rid)
	if err != nil {
		return err
	}

	var acts []RawActivity
	for _, a := range curActivities {
		if _, ok := toDelete[a.EventID]; !ok {
			acts = append(acts, a)
		}
	}

	b, err := json.Marshal(acts)
	if err != nil {
		return err
	}

	return a.store.Write(&microstore.Record{
		Key:   storagespace.FormatResourceID(rid),
		Value: b,
	})
}

// RemoveResource removes the resource from the store
func (a *ActivitylogService) RemoveResource(rid *provider.ResourceId) error {
	if rid == nil {
		return fmt.Errorf("resource id is required")
	}

	a.lock.Lock()
	defer a.lock.Unlock()

	return a.store.Delete(storagespace.FormatResourceID(rid))
}

func (a *ActivitylogService) activities(rid *provider.ResourceId) ([]RawActivity, error) {
	resourceID := storagespace.FormatResourceID(rid)

	records, err := a.store.Read(resourceID)
	if err != nil && err != microstore.ErrNotFound {
		return nil, fmt.Errorf("could not read activities: %w", err)
	}

	if len(records) == 0 {
		return []RawActivity{}, nil
	}

	var activities []RawActivity
	if err := msgpack.Unmarshal(records[0].Value, &activities); err != nil {
		a.log.Debug().Err(err).Str("resourceID", resourceID).Msg("could not unmarshal messagepack, trying json")
		if err := json.Unmarshal(records[0].Value, &activities); err != nil {
			return nil, fmt.Errorf("could not unmarshal activities: %w", err)
		}
	}

	return activities, nil
}

// note: getResource is abstracted to allow unit testing, in general this will just be utils.GetResource
func (a *ActivitylogService) addActivity(ctx context.Context, initRef *provider.Reference, parentId *provider.ResourceId, eventID string, timestamp time.Time, getResource func(*provider.Reference) (*provider.ResourceInfo, error)) error {
	var (
		err   error
		depth int
		ref   = initRef
	)
	for {
		var info *provider.ResourceInfo
		id := ref.GetResourceId()
		if ref.Path != "" {
			// Path based reference, we need to resolve the resource id
			info, err = getResource(ref)
			if err != nil {
				return fmt.Errorf("could not get resource info: %w", err)
			}
			id = info.GetId()
		}
		if id == nil {
			return fmt.Errorf("resource id is required")
		}

		key := storagespace.FormatResourceID(id)
		_, span := a.tracer.Start(ctx, "queueStoreActivity")
		a.debouncer.Debounce(key, RawActivity{
			EventID:   eventID,
			Depth:     depth,
			Timestamp: timestamp,
		})
		span.End()

		if id.OpaqueId == id.SpaceId {
			// we are at the root of the space, no need to go further
			return nil
		}

		// check if parent id is cached
		// parent id is cached in the format <storageid>$<spaceid>!<resourceid>
		// if it is not cached, get the resource info and cache it
		if parentId == nil {
			if v, err := a.parentIdCache.Get(key); err != nil {
				if info == nil {
					_, span = a.tracer.Start(ctx, "getResource")
					info, err = getResource(ref)
					span.End()
					if err != nil || info.GetParentId() == nil || info.GetParentId().GetOpaqueId() == "" {
						return fmt.Errorf("could not get parent id: %w", err)
					}
				}
				parentId = info.GetParentId()
				a.parentIdCache.Set(key, parentId)
			} else {
				parentId = v.(*provider.ResourceId)
			}
		} else {
			a.log.Debug().Msg("parent id is cached")
		}

		depth++
		ref = &provider.Reference{ResourceId: parentId}
		parentId = nil // reset parent id so it's not reused in the next iteration
	}
}

func (a *ActivitylogService) storeActivity(resourceID string, activities []RawActivity) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	ctx, span := a.tracer.Start(context.Background(), "storeActivity")
	defer span.End()
	_, subspan := a.tracer.Start(ctx, "store.Read")
	records, err := a.store.Read(resourceID)
	if err != nil && err != microstore.ErrNotFound {
		return err
	}
	subspan.End()

	_, subspan = a.tracer.Start(ctx, "Unmarshal")
	var existingActivities []RawActivity
	if len(records) > 0 {
		if err := msgpack.Unmarshal(records[0].Value, &existingActivities); err != nil {
			a.log.Debug().Err(err).Str("resourceID", resourceID).Msg("could not unmarshal messagepack, trying json")
			if err := json.Unmarshal(records[0].Value, &existingActivities); err != nil {
				return err
			}
		}
	}
	subspan.End()

	if l := len(existingActivities) + len(activities); l >= _maxActivities {
		start := min(len(existingActivities), l-_maxActivities+1)
		existingActivities = existingActivities[start:]
	}

	activities = append(existingActivities, activities...)

	_, subspan = a.tracer.Start(ctx, "Unmarshal")
	b, err := msgpack.Marshal(activities)
	if err != nil {
		return err
	}
	subspan.End()

	return a.store.Write(&microstore.Record{
		Key:   resourceID,
		Value: b,
	})
}

func toRef(r *provider.ResourceId) *provider.Reference {
	return &provider.Reference{
		ResourceId: r,
	}
}

func toSpace(r *provider.Reference) *provider.StorageSpaceId {
	return &provider.StorageSpaceId{
		OpaqueId: storagespace.FormatStorageID(r.GetResourceId().GetStorageId(), r.GetResourceId().GetSpaceId()),
	}
}

func (a *ActivitylogService) removeCachedParentID(ref *provider.Reference) {
	purgeId := ref.GetResourceId()
	if ref.GetPath() != "" {
		gwc, err := a.gws.Next()
		if err != nil {
			a.log.Error().Err(err).Msg("could not get gateway client")
			return
		}

		ctx, err := utils.GetServiceUserContext(a.cfg.ServiceAccount.ServiceAccountID, gwc, a.cfg.ServiceAccount.ServiceAccountSecret)
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
	if err := a.parentIdCache.Remove(storagespace.FormatResourceID(purgeId)); err != nil {
		a.log.Error().Interface("event", ref).Err(err).Msg("could not delete parent id cache")
	}
}
