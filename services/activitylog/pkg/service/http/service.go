package http

import (
	"context"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/olekukonko/errors"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/pkg/kql"
	"github.com/opencloud-eu/opencloud/pkg/l10n"
	"github.com/opencloud-eu/opencloud/pkg/log"
	ehmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/eventhistory/v0"
	ehsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/eventhistory/v0"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/apierrors"
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
	tracer = otel.Tracer("github.com/opencloud-eu/opencloud/services/activitylog/pkg/service/http")
}

// New returns a new instance of Service
func New(al *activitylog.ActivityLog, opts ...Option) (*svc, error) {
	o := newOptions(opts...)

	registeredEvents := make(map[string]events.Unmarshaller)
	for _, e := range o.RegisteredEvents {
		typ := reflect.TypeOf(e)
		registeredEvents[typ.String()] = e
	}

	return &svc{
		log:              o.Logger,
		evHistory:        o.HistoryClient,
		al:               al,
		registeredEvents: registeredEvents,
		gws:              o.GatewaySelector,
	}, nil
}

type svc struct {
	log              log.Logger
	evHistory        ehsvc.EventHistoryService
	gws              pool.Selectable[gateway.GatewayAPIClient]
	al               *activitylog.ActivityLog
	registeredEvents map[string]events.Unmarshaller
}

func (s *svc) GetItemActivities(ctx context.Context, query, loc string, t l10n.Translator) ([]libregraph.Activity, error) {
	gwc, err := s.gws.Next()
	if err != nil {
		return nil, err
	}

	rid, limit, rawActivityAccepted, activityAccepted, sort, err := s.getFilters(query)
	if err != nil {
		s.log.Info().Str("query", query).Err(err).Msg("error getting filters")
		return nil, apierrors.ErrBadRequest
	}

	info, err := utils.GetResourceByID(ctx, rid, gwc)
	if err != nil {
		return nil, apierrors.ErrForbidden
	}

	// you need ListGrants to see activities
	if !info.GetPermissionSet().GetListGrants() {
		return nil, apierrors.ErrForbidden
	}

	raw, err := s.al.Activities(rid)
	if err != nil {
		s.log.Error().Err(err).Msg("error getting activities")
		return nil, err
	}

	ids := make([]string, 0, len(raw))
	toDelete := make(map[string]struct{}, len(raw))
	for _, a := range raw {
		if !rawActivityAccepted(a) {
			continue
		}
		ids = append(ids, a.EventID)
		toDelete[a.EventID] = struct{}{}
	}

	evRes, err := s.evHistory.GetEvents(ctx, &ehsvc.GetEventsRequest{Ids: ids})
	if err != nil {
		s.log.Error().Err(err).Msg("error getting events")
		return nil, err
	}

	evs := evRes.GetEvents()
	sort(evs)

	// TODO cut the interface here?
	activities := make([]libregraph.Activity, 0, len(evRes.GetEvents()))
	for _, e := range evs {
		delete(toDelete, e.GetId())

		if limit > 0 && limit <= len(activities) {
			continue
		}

		if !activityAccepted(e) {
			continue
		}

		var (
			message string
			ts      time.Time
			vars    map[string]any
		)

		switch ev := s.unwrapEvent(e).(type) {
		case nil:
			// error already logged in unwrapEvent
			continue
		case events.UploadReady:
			message = MessageResourceCreated
			if ev.IsVersion {
				message = MessageResourceUpdated
			}
			ts = utils.TSToTime(ev.Timestamp)
			vars, err = s.GetVars(ctx, WithResource(ev.FileRef, false, ""), WithUser(nil, ev.ExecutingUser, ev.ImpersonatingUser))
		case events.FileTouched:
			message = MessageResourceCreated
			ts = utils.TSToTime(ev.Timestamp)
			vars, err = s.GetVars(ctx, WithResource(ev.Ref, false, ""), WithUser(ev.Executant, nil, ev.ImpersonatingUser))
		case events.FileDownloaded:
			message = MessageResourceDownloaded
			ts = utils.TSToTime(ev.Timestamp)
			vars, err = s.GetVars(ctx, WithResource(ev.Ref, false, ""), WithUser(ev.Executant, nil, ev.ImpersonatingUser), WithVar("token", "", ev.ImpersonatingUser.GetId().GetOpaqueId()))
		case events.ContainerCreated:
			message = MessageResourceCreated
			ts = utils.TSToTime(ev.Timestamp)
			vars, err = s.GetVars(ctx, WithResource(ev.Ref, false, ""), WithUser(ev.Executant, nil, ev.ImpersonatingUser))
		case events.ItemTrashed:
			message = MessageResourceTrashed
			ts = utils.TSToTime(ev.Timestamp)
			vars, err = s.GetVars(ctx, WithTrashedResource(ev.Ref, ev.ID), WithUser(ev.Executant, nil, ev.ImpersonatingUser))
		case events.ItemMoved:
			switch isRename(ev.OldReference, ev.Ref) {
			case true:
				message = MessageResourceRenamed
				vars, err = s.GetVars(ctx, WithResource(ev.Ref, false, ""), WithOldResource(ev.OldReference), WithUser(ev.Executant, nil, ev.ImpersonatingUser))
			case false:
				message = MessageResourceMoved
				vars, err = s.GetVars(ctx, WithResource(ev.Ref, false, ""), WithUser(ev.Executant, nil, ev.ImpersonatingUser))
			}
			ts = utils.TSToTime(ev.Timestamp)
		case events.ShareCreated:
			message = MessageShareCreated
			ts = utils.TSToTime(ev.CTime)
			vars, err = s.GetVars(ctx,
				WithResource(toRef(ev.ItemID), false, ev.ResourceName),
				WithUser(ev.Executant, nil, nil),
				WithSharee(ev.GranteeUserID, ev.GranteeGroupID))
		case events.ShareUpdated:
			if ev.Sharer != nil && ev.ItemID != nil && ev.Sharer.GetOpaqueId() == ev.ItemID.GetSpaceId() {
				continue
			}
			message = MessageShareUpdated
			ts = utils.TSToTime(ev.MTime)
			vars, err = s.GetVars(ctx,
				WithResource(toRef(ev.ItemID), false, ev.ResourceName),
				WithUser(ev.Executant, nil, nil),
				WithTranslation(&t, loc, "field", ev.UpdateMask))
		case events.ShareRemoved:
			message = MessageShareDeleted
			ts = ev.Timestamp
			vars, err = s.GetVars(ctx,
				WithResource(toRef(ev.ItemID), false, ev.ResourceName),
				WithUser(ev.Executant, nil, nil),
				WithSharee(ev.GranteeUserID, ev.GranteeGroupID))
		case events.LinkCreated:
			message = MessageLinkCreated
			ts = utils.TSToTime(ev.CTime)
			vars, err = s.GetVars(ctx,
				WithResource(toRef(ev.ItemID), false, ev.ResourceName),
				WithUser(ev.Executant, nil, nil))
		case events.LinkUpdated:
			if ev.Sharer != nil && ev.ItemID != nil && ev.Sharer.GetOpaqueId() == ev.ItemID.GetSpaceId() {
				continue
			}
			message = MessageLinkUpdated
			ts = utils.TSToTime(ev.MTime)
			vars, err = s.GetVars(ctx,
				WithVar("resource", storagespace.FormatResourceID(ev.ItemID), ev.ResourceName),
				WithUser(ev.Executant, nil, nil),
				WithTranslation(&t, loc, "field", []string{ev.FieldUpdated}),
				WithVar("token", ev.ItemID.GetOpaqueId(), ev.DisplayName))
		case events.LinkRemoved:
			message = MessageLinkDeleted
			ts = utils.TSToTime(ev.Timestamp)
			vars, err = s.GetVars(ctx, WithResource(toRef(ev.ItemID), false, ""), WithUser(ev.Executant, nil, nil))
		case events.SpaceShared:
			message = MessageSpaceShared
			ts = ev.Timestamp
			vars, err = s.GetVars(ctx, WithSpace(ev.ID), WithUser(ev.Executant, nil, nil), WithSharee(ev.GranteeUserID, ev.GranteeGroupID))
		case events.SpaceUnshared:
			message = MessageSpaceUnshared
			ts = ev.Timestamp
			vars, err = s.GetVars(ctx, WithSpace(ev.ID), WithUser(ev.Executant, nil, nil), WithSharee(ev.GranteeUserID, ev.GranteeGroupID))
		}

		if err != nil {
			s.log.Error().Err(err).Msg("error getting response data")
			continue
		}

		activities = append(activities, NewActivity(t.Translate(message, loc), ts, e.GetId(), vars))
	}

	// delete activities in separate go routine
	if len(toDelete) > 0 {
		go func() {
			err := s.al.RemoveActivities(rid, toDelete)
			if err != nil {
				s.log.Error().Err(err).Msg("error removing activities")
			}
		}()
	}
	return activities, nil

}

func toRef(r *provider.ResourceId) *provider.Reference {
	return &provider.Reference{
		ResourceId: r,
	}
}

func (s *svc) unwrapEvent(e *ehmsg.Event) any {
	etype, ok := s.registeredEvents[e.GetType()]
	if !ok {
		s.log.Error().Str("eventid", e.GetId()).Str("eventtype", e.GetType()).Msg("event not registered")
		return nil
	}

	einterface, err := etype.Unmarshal(e.GetEvent())
	if err != nil {
		s.log.Error().Str("eventid", e.GetId()).Str("eventtype", e.GetType()).Msg("failed to umarshal event")
		return nil
	}

	return einterface
}

func (s *svc) getFilters(query string) (*provider.ResourceId, int, func(data.RawActivity) bool, func(*ehmsg.Event) bool, func([]*ehmsg.Event), error) {
	qast, err := kql.Builder{}.Build(query)
	if err != nil {
		return nil, 0, nil, nil, nil, err
	}

	prefilters := make([]func(data.RawActivity) bool, 0)
	postfilters := make([]func(*ehmsg.Event) bool, 0)

	sortby := func(_ []*ehmsg.Event) {}

	var (
		itemID string
		limit  int
	)

	for _, n := range qast.Nodes {
		switch v := n.(type) {
		case *ast.StringNode:
			switch strings.ToLower(v.Key) {
			case "itemid":
				itemID = v.Value
			case "depth":
				depth, err := strconv.Atoi(v.Value)
				if err != nil {
					return nil, limit, nil, nil, sortby, err
				}
				if depth == -1 {
					break
				}

				prefilters = append(prefilters, func(a data.RawActivity) bool {
					return a.Depth <= depth
				})
			case "limit":
				l, err := strconv.Atoi(v.Value)
				if err != nil {
					return nil, limit, nil, nil, sortby, err
				}

				limit = l
			case "sort":
				switch v.Value {
				case "asc":
					// nothing to do - already ascending
				case "desc":
					sortby = func(activities []*ehmsg.Event) {
						slices.Reverse(activities)
					}
				}
			}
		case *ast.DateTimeNode:
			switch v.Operator.Value {
			case "<", "<=":
				prefilters = append(prefilters, func(a data.RawActivity) bool {
					return a.Timestamp.Before(v.Value)
				})
			case ">", ">=":
				prefilters = append(prefilters, func(a data.RawActivity) bool {
					return a.Timestamp.After(v.Value)
				})
			}
		case *ast.OperatorNode:
			if v.Value != "AND" {
				return nil, limit, nil, nil, sortby, errors.New("only AND operator is supported")
			}
		}
	}

	rid, err := storagespace.ParseID(itemID)
	if err != nil {
		return nil, limit, nil, nil, sortby, err
	}
	if rid.GetOpaqueId() == "" {
		// space root requested - fix format
		rid.OpaqueId = rid.GetSpaceId()
	}
	pref := func(a data.RawActivity) bool {
		for _, f := range prefilters {
			if !f(a) {
				return false
			}
		}
		return true
	}
	postf := func(e *ehmsg.Event) bool {
		for _, f := range postfilters {
			if !f(e) {
				return false
			}
		}
		return true
	}
	return &rid, limit, pref, postf, sortby, nil
}

// returns true if this is just a rename
func isRename(o, n *provider.Reference) bool {
	// if resourceids are different we assume it is a move
	if !utils.ResourceIDEqual(o.GetResourceId(), n.GetResourceId()) {
		return false
	}
	return filepath.Base(o.GetPath()) != filepath.Base(n.GetPath())
}
