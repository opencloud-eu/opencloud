package service

import (
	"context"
	"crypto/tls"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/jellydator/ttlcache/v2"
	"github.com/nats-io/nats.go"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack/v5"
	"go.opentelemetry.io/otel/trace"

	"github.com/opencloud-eu/opencloud/pkg/log"
	ehsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/eventhistory/v0"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/config"
)

// Nats runs into max payload exceeded errors at around 7k activities. Let's keep a buffer.
var _maxActivitiesDefault = 6000

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
	gws           pool.Selectable[gateway.GatewayAPIClient]
	mux           *chi.Mux
	evHistory     ehsvc.EventHistoryService
	valService    settingssvc.ValueService
	lock          sync.RWMutex
	tp            trace.TracerProvider
	tracer        trace.Tracer
	debouncer     *Debouncer
	parentIdCache *ttlcache.Cache
	natskv        nats.KeyValue

	maxActivities int

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

type batchInfo struct {
	key       string
	count     int
	timestamp time.Time
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
	o := &Options{
		MaxActivities: _maxActivitiesDefault,
	}
	for _, opt := range opts {
		opt(o)
	}

	if o.Stream == nil {
		return nil, errors.New("stream is required")
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

	// Connect to NATS servers
	natsOptions := nats.Options{
		Servers: o.Config.Store.Nodes,
	}
	if o.Config.Store.EnableTLS {
		if o.Config.Store.TLSRootCACertificate != "" {
			// when root ca is configured use it. an insecure flag is ignored.
			nats.RootCAs(o.Config.Store.TLSRootCACertificate)(&natsOptions)
		} else {
			// enable tls and use insecure flag
			nats.Secure(&tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: o.Config.Store.TLSInsecure})(&natsOptions)
		}
	}
	if o.Config.Store.AuthUsername != "" && o.Config.Store.AuthPassword != "" {
		nats.UserInfo(o.Config.Store.AuthUsername, o.Config.Store.AuthPassword)(&natsOptions)
	}
	conn, err := natsOptions.Connect()
	if err != nil {
		return nil, err
	}

	js, err := conn.JetStream()
	if err != nil {
		return nil, err
	}

	kv, err := js.KeyValue(o.Config.Store.Database)
	if err != nil {
		if !errors.Is(err, nats.ErrBucketNotFound) {
			return nil, errors.Wrapf(err, "Failed to get bucket (%s)", o.Config.Store.Database)
		}

		kv, err = js.CreateKeyValue(&nats.KeyValueConfig{
			Bucket: o.Config.Store.Database,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create bucket (%s)", o.Config.Store.Database)
		}
	}
	if err != nil {
		return nil, err
	}

	s := &ActivitylogService{
		log:              o.Logger,
		cfg:              o.Config,
		events:           ch,
		gws:              o.GatewaySelector,
		mux:              o.Mux,
		evHistory:        o.HistoryClient,
		valService:       o.ValueClient,
		lock:             sync.RWMutex{},
		registeredEvents: make(map[string]events.Unmarshaller),
		tp:               o.TraceProvider,
		tracer:           o.TraceProvider.Tracer("github.com/opencloud-eu/opencloud/services/activitylog/pkg/service"),
		parentIdCache:    cache,
		maxActivities:    o.Config.MaxActivities,
		natskv:           kv,
	}
	s.debouncer = NewDebouncer(o.Config.WriteBufferDuration, s.storeActivity)

	// run migrations
	err = s.runMigrations(context.Background(), kv)
	if err != nil {
		return nil, err
	}

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

	return a.addActivity(ctx, initRef, parentId, eventID, timestamp, func(ctx context.Context, ref *provider.Reference) (*provider.ResourceInfo, error) {
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
	ctx, span = a.tracer.Start(ctx, "AddActivityTrashed")
	defer span.End()

	return a.addActivity(ctx, ref, parentId, eventID, timestamp, func(ctx context.Context, ref *provider.Reference) (*provider.ResourceInfo, error) {
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

	_, err = a.natskv.Put(storagespace.FormatResourceID(rid), b)
	return err
}

// RemoveResource removes the resource from the store
func (a *ActivitylogService) RemoveResource(rid *provider.ResourceId) error {
	if rid == nil {
		return fmt.Errorf("resource id is required")
	}

	a.lock.Lock()
	defer a.lock.Unlock()

	return a.natskv.Delete(storagespace.FormatResourceID(rid))
}

func (a *ActivitylogService) activities(rid *provider.ResourceId) ([]RawActivity, error) {
	resourceID := storagespace.FormatResourceID(rid)

	glob := fmt.Sprintf("%s.>", base32.StdEncoding.EncodeToString([]byte(resourceID)))

	watcher, err := a.natskv.Watch(glob, nats.IgnoreDeletes())
	if err != nil {
		return nil, err
	}
	defer watcher.Stop()

	var activities []RawActivity
	for update := range watcher.Updates() {
		if update == nil {
			break
		}

		var batchActivities []RawActivity
		if err := msgpack.Unmarshal(update.Value(), &batchActivities); err != nil {
			a.log.Debug().Err(err).Str("resourceID", resourceID).Msg("could not unmarshal messagepack, trying json")
		}
		activities = append(activities, batchActivities...)
	}

	return activities, nil
}

// note: getResource is abstracted to allow unit testing, in general this will just be utils.GetResource
func (a *ActivitylogService) addActivity(ctx context.Context, initRef *provider.Reference, parentId *provider.ResourceId, eventID string, timestamp time.Time, getResource func(context.Context, *provider.Reference) (*provider.ResourceInfo, error)) error {
	var (
		err   error
		depth int
		ref   = initRef
	)
	ctx, span := a.tracer.Start(ctx, "addActivity")
	defer span.End()
	for {
		var info *provider.ResourceInfo
		id := ref.GetResourceId()
		if ref.Path != "" {
			// Path based reference, we need to resolve the resource id
			ctx, span = a.tracer.Start(ctx, "addActivity.getResource")
			info, err = getResource(ctx, ref)
			span.End()
			if err != nil {
				return fmt.Errorf("could not get resource info: %w", err)
			}
			id = info.GetId()
		}
		if id == nil {
			return fmt.Errorf("resource id is required")
		}

		key := storagespace.FormatResourceID(id)
		a.debouncer.Debounce(key, RawActivity{
			EventID:   eventID,
			Depth:     depth,
			Timestamp: timestamp,
		})

		if id.OpaqueId == id.SpaceId {
			// we are at the root of the space, no need to go further
			break
		}

		// check if parent id is cached
		// parent id is cached in the format <storageid>$<spaceid>!<resourceid>
		// if it is not cached, get the resource info and cache it
		if parentId == nil {
			if v, err := a.parentIdCache.Get(key); err != nil {
				if info == nil {
					ctx, span := a.tracer.Start(ctx, "addActivity.getResource parent")
					info, err = getResource(ctx, ref)
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

	return nil
}

func (a *ActivitylogService) storeActivity(resourceID string, activities []RawActivity) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	ctx, span := a.tracer.Start(context.Background(), "storeActivity")
	defer span.End()

	_, subspan := a.tracer.Start(ctx, "storeActivity.Marshal")
	b, err := msgpack.Marshal(activities)
	if err != nil {
		return err
	}
	subspan.End()

	_, subspan = a.tracer.Start(ctx, "storeActivity.natskv.Put")
	key := natsKey(resourceID, len(activities))
	_, err = a.natskv.Put(key, b)
	if err != nil {
		return err
	}
	subspan.End()

	ctx, subspan = a.tracer.Start(ctx, "storeActivity.enforceMaxActivities")
	a.enforceMaxActivities(ctx, resourceID)
	subspan.End()
	return nil
}

func (a *ActivitylogService) enforceMaxActivities(ctx context.Context, resourceID string) {
	if a.maxActivities <= 0 {
		return
	}

	key := fmt.Sprintf("%s.>", base32.StdEncoding.EncodeToString([]byte(resourceID)))

	_, subspan := a.tracer.Start(ctx, "enforceMaxActivities.watch")
	watcher, err := a.natskv.Watch(key, nats.IgnoreDeletes())
	if err != nil {
		a.log.Error().Err(err).Str("resourceID", resourceID).Msg("could not watch")
		return
	}
	defer watcher.Stop()

	var keys []string
	for update := range watcher.Updates() {
		if update == nil {
			break
		}

		var batchActivities []RawActivity
		if err := msgpack.Unmarshal(update.Value(), &batchActivities); err != nil {
			a.log.Debug().Err(err).Str("resourceID", resourceID).Msg("could not unmarshal messagepack, trying json")
		}
		keys = append(keys, update.Key())
	}
	subspan.End()

	_, subspan = a.tracer.Start(ctx, "enforceMaxActivities.compile")
	// Parse keys into batches
	batches := make([]batchInfo, 0)
	var activitiesCount int
	for _, k := range keys {
		parts := strings.SplitN(k, ".", 3)
		if len(parts) < 3 {
			a.log.Warn().Str("key", k).Msg("skipping key, not enough parts")
			continue
		}

		c, err := strconv.Atoi(parts[1])
		if err != nil {
			a.log.Warn().Str("key", k).Msg("skipping key, can not parse count")
			continue
		}

		// parse timestamp
		nano, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			a.log.Warn().Str("key", k).Msg("skipping key, can not parse timestamp")
			continue
		}

		batches = append(batches, batchInfo{
			key:       k,
			count:     c,
			timestamp: time.Unix(0, nano),
		})
		activitiesCount += c
	}

	// sort batches by timestamp
	sort.Slice(batches, func(i, j int) bool {
		return batches[i].timestamp.Before(batches[j].timestamp)
	})
	subspan.End()

	_, subspan = a.tracer.Start(ctx, "enforceMaxActivities.delete")
	// remove oldest keys until we are at max activities
	for _, b := range batches {
		if activitiesCount-b.count < a.maxActivities {
			break
		}

		activitiesCount -= b.count
		err = a.natskv.Delete(b.key)
		if err != nil {
			a.log.Error().Err(err).Str("key", b.key).Msg("could not delete key")
			break
		}
	}
	subspan.End()
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

func natsKey(resourceID string, activitiesCount int) string {
	return fmt.Sprintf("%s.%d.%d",
		base32.StdEncoding.EncodeToString([]byte(resourceID)),
		activitiesCount,
		time.Now().UnixNano())
}
