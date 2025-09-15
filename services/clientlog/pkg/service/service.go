package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sync/atomic"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	group "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"go.opentelemetry.io/otel/trace"

	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/clientlog/pkg/config"
)

// ClientlogService is the service responsible for user activities
type ClientlogService struct {
	log              log.Logger
	cfg              *config.Config
	gatewaySelector  pool.Selectable[gateway.GatewayAPIClient]
	registeredEvents map[string]events.Unmarshaller // ?
	tp               trace.TracerProvider
	tracer           trace.Tracer
	publisher        events.Publisher
	ch               <-chan events.Event
	stopCh           chan struct{}
	stopped          atomic.Bool
}

// NewClientlogService returns a clientlog service
func NewClientlogService(opts ...Option) (*ClientlogService, error) {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}

	if o.Stream == nil {
		return nil, fmt.Errorf("need non nil stream (%v) to work properly", o.Stream)
	}

	ch, err := events.Consume(o.Stream, "clientlog", o.RegisteredEvents...)
	if err != nil {
		return nil, err
	}

	cl := &ClientlogService{
		log:              o.Logger,
		cfg:              o.Config,
		gatewaySelector:  o.GatewaySelector,
		registeredEvents: make(map[string]events.Unmarshaller),
		tp:               o.TraceProvider,
		tracer:           o.TraceProvider.Tracer("github.com/opencloud-eu/opencloud/services/clientlog/pkg/service"),
		publisher:        o.Stream,
		ch:               ch,
		stopCh:           make(chan struct{}, 1),
	}

	for _, e := range o.RegisteredEvents {
		typ := reflect.TypeOf(e)
		cl.registeredEvents[typ.String()] = e
	}

	return cl, nil
}

// Run runs the service
func (cl *ClientlogService) Run() error {
EventLoop:
	for {
		select {
		case event, ok := <-cl.ch:
			if !ok {
				break EventLoop
			}
			cl.processEvent(event)

			if cl.stopped.Load() {
				break EventLoop
			}
		case <-cl.stopCh:
			break EventLoop
		}
	}

	return nil
}

func (cl *ClientlogService) Close() {
	if cl.stopped.CompareAndSwap(false, true) {
		close(cl.stopCh)
	}
}

func (cl *ClientlogService) processEvent(event events.Event) {
	gwc, err := cl.gatewaySelector.Next()
	if err != nil {
		cl.log.Error().Err(err).Interface("event", event).Msg("error getting gateway client")
		return
	}

	ctx, err := utils.GetServiceUserContextWithContext(context.Background(), gwc, cl.cfg.ServiceAccount.ServiceAccountID, cl.cfg.ServiceAccount.ServiceAccountSecret)
	if err != nil {
		cl.log.Error().Err(err).Interface("event", event).Msg("error authenticating service user")
		return
	}

	var (
		users  []string
		evType string
		data   interface{}
	)

	fileEv := func(typ string, ref *provider.Reference) {
		evType = typ
		users, data, err = processFileEvent(ctx, ref, gwc, event.InitiatorID)
	}

	shareEv := func(typ string, ref *provider.Reference, uid *user.UserId, gid *group.GroupId) {
		evType = typ
		users, data, err = processShareEvent(ctx, ref, gwc, event.InitiatorID, uid, gid)
	}

	switch e := event.Event.(type) {
	default:
		err = errors.New("unhandled event")
	case events.UploadReady:
		if e.Failed {
			// we don't inform about failed uploads yet
			return
		}
		fileEv("postprocessing-finished", e.FileRef)
	case events.ItemTrashed:
		evType = "item-trashed"
		users, data, err = processItemTrashedEvent(ctx, e.Ref, gwc, event.InitiatorID, e.ID)
	case events.ItemRestored:
		fileEv("item-restored", e.Ref)
	case events.ContainerCreated:
		fileEv("folder-created", e.Ref)
	case events.ItemMoved:
		// we send a dedicated event in case the item was only renamed
		if isRename(e.OldReference, e.Ref) {
			fileEv("item-renamed", e.Ref)
		} else {
			fileEv("item-moved", e.Ref)
		}
	case events.FileLocked:
		fileEv("file-locked", e.Ref)
	case events.FileUnlocked:
		fileEv("file-unlocked", e.Ref)
	case events.FileTouched:
		fileEv("file-touched", e.Ref)
	case events.SpaceShared:
		r, _ := storagespace.ParseReference(e.ID.GetOpaqueId())
		shareEv("space-member-added", &r, e.GranteeUserID, e.GranteeGroupID)
	case events.SpaceShareUpdated:
		r, _ := storagespace.ParseReference(e.ID.GetOpaqueId())
		shareEv("space-share-updated", &r, e.GranteeUserID, e.GranteeGroupID)
	case events.SpaceUnshared:
		r, _ := storagespace.ParseReference(e.ID.GetOpaqueId())
		shareEv("space-member-removed", &r, e.GranteeUserID, e.GranteeGroupID)
	case events.ShareCreated:
		shareEv("share-created", &provider.Reference{ResourceId: e.ItemID}, e.GranteeUserID, e.GranteeGroupID)
	case events.ShareUpdated:
		shareEv("share-updated", &provider.Reference{ResourceId: e.ItemID}, e.GranteeUserID, e.GranteeGroupID)
	case events.ShareRemoved:
		shareEv("share-removed", &provider.Reference{ResourceId: e.ItemID}, e.GranteeUserID, e.GranteeGroupID)
	case events.LinkCreated:
		fileEv("link-created", &provider.Reference{ResourceId: e.ItemID})
	case events.LinkUpdated:
		fileEv("link-updated", &provider.Reference{ResourceId: e.ItemID})
	case events.LinkRemoved:
		fileEv("link-removed", &provider.Reference{ResourceId: e.ItemID})
	case events.BackchannelLogout:
		evType, users, data = backchannelLogoutEvent(e)
	}

	if err != nil {
		cl.log.Error().Err(err).Interface("event", event).Msg("error gathering members for event")
		return
	}

	// II) instruct sse service to send the information
	if err := cl.sendSSE(users, evType, data); err != nil {
		cl.log.Error().Err(err).Interface("userIDs", users).Str("eventid", event.ID).Msg("failed to store event for user")
		return
	}
}

func (cl *ClientlogService) sendSSE(userIDs []string, evType string, data interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return events.Publish(context.Background(), cl.publisher, events.SendSSE{
		UserIDs: userIDs,
		Type:    evType,
		Message: b,
	})
}

// process file related events
func processFileEvent(ctx context.Context, ref *provider.Reference, gwc gateway.GatewayAPIClient, initiatorid string) ([]string, FileEvent, error) {
	info, err := utils.GetResource(ctx, ref, gwc)
	if err != nil {
		return nil, FileEvent{}, err
	}

	data := FileEvent{
		ParentItemID: storagespace.FormatResourceID(info.GetParentId()),
		ItemID:       storagespace.FormatResourceID(info.GetId()),
		SpaceID:      storagespace.FormatStorageID(info.GetSpace().GetRoot().GetStorageId(), info.GetSpace().GetRoot().GetSpaceId()),
		InitiatorID:  initiatorid,
		Etag:         info.GetEtag(),
	}

	users, err := utils.GetSpaceMembers(ctx, info.GetSpace().GetId().GetOpaqueId(), gwc, utils.ViewerRole)
	return users, data, err
}

// process share related events
func processShareEvent(ctx context.Context, ref *provider.Reference, gwc gateway.GatewayAPIClient, initiatorid string, shareeID *user.UserId, shareeGroupID *group.GroupId) ([]string, FileEvent, error) {
	users, data, err := processFileEvent(ctx, ref, gwc, initiatorid)
	if err != nil {
		return users, data, err
	}

	return addShareeData(ctx, gwc, data, users, shareeID, shareeGroupID)
}

// custom logic for item trashed event
func processItemTrashedEvent(ctx context.Context, ref *provider.Reference, gwc gateway.GatewayAPIClient, initiatorid string, itemID *provider.ResourceId) ([]string, FileEvent, error) {
	data := FileEvent{
		ItemID: storagespace.FormatResourceID(itemID),
		// TODO: check with web if parentID is needed
		// ParentItemID: storagespace.FormatResourceID(*item.GetRef().GetResourceId()),
		SpaceID:     storagespace.FormatStorageID(itemID.GetStorageId(), itemID.GetSpaceId()),
		InitiatorID: initiatorid,
	}

	users, err := utils.GetSpaceMembers(ctx, itemID.GetSpaceId(), gwc, utils.ViewerRole)
	return users, data, err
}

// adds share related data to the FileEvent
func addShareeData(ctx context.Context, gwc gateway.GatewayAPIClient, fe FileEvent, users []string, shareeID *user.UserId, shareeGroupID *group.GroupId) ([]string, FileEvent, error) {
	us, err := resolveID(ctx, gwc, shareeID, shareeGroupID)
	if err != nil {
		return users, fe, err
	}

	fe.AffectedUserIDs = us

	// TODO: this list can get long. Should we add a limit? If yes, how big?
	for _, u := range us {
		users = appendUnique(users, u)
	}
	return users, fe, nil
}

// returns the user or the members of the affected group
func resolveID(ctx context.Context, gwc gateway.GatewayAPIClient, uid *user.UserId, gid *group.GroupId) ([]string, error) {
	if uid != nil {
		return []string{uid.GetOpaqueId()}, nil
	}
	return utils.GetGroupMembers(ctx, gid.GetOpaqueId(), gwc)
}

// returns users or append(users, user)
func appendUnique(users []string, user string) []string {
	for _, u := range users {
		if u == user {
			return users
		}
	}
	return append(users, user)
}

// returns true if this is just a rename
func isRename(o, n *provider.Reference) bool {
	// if resourceids are different we assume it is a move
	if !utils.ResourceIDEqual(o.GetResourceId(), n.GetResourceId()) {
		return false
	}
	return filepath.Base(o.GetPath()) != filepath.Base(n.GetPath())
}

func backchannelLogoutEvent(e events.BackchannelLogout) (string, []string, BackchannelLogout) {
	return "backchannel-logout", []string{e.Executant.GetOpaqueId()}, BackchannelLogout{
		UserID:    e.Executant.GetOpaqueId(),
		Timestamp: e.Timestamp.String(),
	}
}
