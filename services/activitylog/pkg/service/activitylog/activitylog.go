package activitylog

import (
	"context"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/jellydator/ttlcache/v2"
	"github.com/nats-io/nats.go"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/data"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/vmihailenco/msgpack/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func init() {
	tracer = otel.Tracer("github.com/opencloud-eu/opencloud/services/activitylog/pkg/service/activitylog")
}

var (
	_maxActivitiesDefault = 6000
	_writeBufferDuration  = 10 * time.Second
)

// Activitylog stores and retrieves activities for resources and their parents from a nats kv
type ActivityLog struct {
	log log.Logger
	// FIXME the lock does not protect agains concurrent resource activities on multiple instances
	// known since https://github.com/owncloud/ocis/pull/9361#pullrequestreview-2135350157
	// current ocis discussion in https://github.com/owncloud/ocis/issues/12475
	lock          sync.RWMutex
	debouncer     *Debouncer
	parentIdCache *ttlcache.Cache
	natskv        nats.KeyValue

	maxActivities int
}

type batchInfo struct {
	key       string
	count     int
	timestamp time.Time
}

// New creates a new ActivitylogService
func New(kv nats.KeyValue, opts ...Option) (*ActivityLog, error) {
	o := &Options{
		MaxActivities:       _maxActivitiesDefault,
		WriteBufferDuration: _writeBufferDuration,
		Logger:              log.NopLogger(),
	}
	for _, opt := range opts {
		opt(o)
	}

	cache := ttlcache.NewCache()
	err := cache.SetTTL(30 * time.Second)
	if err != nil {
		return nil, err
	}

	s := &ActivityLog{
		log:           o.Logger,
		lock:          sync.RWMutex{},
		parentIdCache: cache,
		maxActivities: o.MaxActivities,
		natskv:        kv,
	}
	s.debouncer = NewDebouncer(o.WriteBufferDuration, s.StoreActivity)

	// run migrations
	err = s.runMigrations(context.Background(), kv)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// RemoveResource removes the resource from the store
func (a *ActivityLog) RemoveResource(rid *provider.ResourceId) error {
	if rid == nil {
		return fmt.Errorf("resource id is required")
	}

	a.lock.Lock()
	defer a.lock.Unlock()

	return a.natskv.Delete(storagespace.FormatResourceID(rid))
}

func (a *ActivityLog) AddActivity(ctx context.Context, initRef *provider.Reference, parentId *provider.ResourceId, eventID string, timestamp time.Time, getResource func(context.Context, *provider.Reference) (*provider.ResourceInfo, error)) error {
	var (
		err   error
		depth int
		ref   = initRef
	)
	ctx, span := tracer.Start(ctx, "AddActivity")
	defer span.End()
	for {
		var info *provider.ResourceInfo
		id := ref.GetResourceId()
		if ref.Path != "" {
			// Path based reference, we need to resolve the resource id
			ctx, span = tracer.Start(ctx, "AddActivity.getResource")
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
		a.debouncer.Debounce(key, data.RawActivity{
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
					ctx, span := tracer.Start(ctx, "AddActivity.getResource parent")
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

func (a *ActivityLog) StoreActivity(resourceID string, activities []data.RawActivity) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	ctx, span := tracer.Start(context.Background(), "storeActivity")
	defer span.End()

	_, subspan := tracer.Start(ctx, "storeActivity.Marshal")
	b, err := msgpack.Marshal(activities)
	if err != nil {
		return err
	}
	subspan.End()

	_, subspan = tracer.Start(ctx, "storeActivity.natskv.Put")
	key := natsKey(resourceID, len(activities))
	_, err = a.natskv.Put(key, b)
	if err != nil {
		return err
	}
	subspan.End()

	ctx, subspan = tracer.Start(ctx, "storeActivity.enforceMaxActivities")
	a.enforceMaxActivities(ctx, resourceID)
	subspan.End()
	return nil
}

func (a *ActivityLog) enforceMaxActivities(ctx context.Context, resourceID string) {
	if a.maxActivities <= 0 {
		return
	}

	key := fmt.Sprintf("%s.>", base32.StdEncoding.EncodeToString([]byte(resourceID)))

	_, subspan := tracer.Start(ctx, "enforceMaxActivities.watch")
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

		var batchActivities []data.RawActivity
		if err := msgpack.Unmarshal(update.Value(), &batchActivities); err != nil {
			a.log.Debug().Err(err).Str("resourceID", resourceID).Msg("could not unmarshal messagepack, trying json")
		}
		keys = append(keys, update.Key())
	}
	subspan.End()

	_, subspan = tracer.Start(ctx, "enforceMaxActivities.compile")
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

	_, subspan = tracer.Start(ctx, "enforceMaxActivities.delete")
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

func (a *ActivityLog) InvalidateCachedParentID(purgeId *provider.ResourceId) {
	// The parent id cache is populated lazily and its entries expire, so a
	// missing key is the expected case rather than an error.
	if err := a.parentIdCache.Remove(storagespace.FormatResourceID(purgeId)); err != nil {
		a.log.Debug().Interface("event", purgeId).Err(err).Msg("could not delete parent id cache")
	}
}

func natsKey(resourceID string, activitiesCount int) string {
	return fmt.Sprintf("%s.%d.%d",
		base32.StdEncoding.EncodeToString([]byte(resourceID)),
		activitiesCount,
		time.Now().UnixNano())
}

func (a *ActivityLog) Activities(rid *provider.ResourceId) ([]data.RawActivity, error) {
	a.lock.RLock()
	defer a.lock.RUnlock()

	return a.activities(rid)
}

func (a *ActivityLog) activities(rid *provider.ResourceId) ([]data.RawActivity, error) {
	resourceID := storagespace.FormatResourceID(rid)

	glob := fmt.Sprintf("%s.>", base32.StdEncoding.EncodeToString([]byte(resourceID)))

	watcher, err := a.natskv.Watch(glob, nats.IgnoreDeletes())
	if err != nil {
		return nil, err
	}
	defer watcher.Stop()

	var activities []data.RawActivity
	for update := range watcher.Updates() {
		if update == nil {
			break
		}

		var batchActivities []data.RawActivity
		if err := msgpack.Unmarshal(update.Value(), &batchActivities); err != nil {
			a.log.Debug().Err(err).Str("resourceID", resourceID).Msg("could not unmarshal messagepack")
		}
		activities = append(activities, batchActivities...)
	}

	return activities, nil
}

// RemoveActivities removes the activities from the given resource
func (a *ActivityLog) RemoveActivities(rid *provider.ResourceId, toDelete map[string]struct{}) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	curActivities, err := a.activities(rid)
	if err != nil {
		return err
	}

	var acts []data.RawActivity
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
