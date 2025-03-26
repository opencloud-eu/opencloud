package service

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/l10n"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

//go:embed l10n/locale
var _translationFS embed.FS

var (
	_resourceTypeResource = "resource"
	_resourceTypeSpace    = "storagespace"
	_resourceTypeShare    = "share"
	_resourceTypeGlobal   = "global"

	_domain = "userlog"
)

// OC10Notification is the oc10 style representation of an event
// some fields are left out for simplicity
type OC10Notification struct {
	EventID        string                 `json:"notification_id"`
	Service        string                 `json:"app"`
	UserName       string                 `json:"user"`
	Timestamp      string                 `json:"datetime"`
	ResourceID     string                 `json:"object_id"`
	ResourceType   string                 `json:"object_type"`
	Subject        string                 `json:"subject"`
	SubjectRaw     string                 `json:"subjectRich"`
	Message        string                 `json:"message"`
	MessageRaw     string                 `json:"messageRich"`
	MessageDetails map[string]interface{} `json:"messageRichParameters"`
}

// Converter is responsible for converting eventhistory events to OC10Notifications
type Converter struct {
	locale                string
	gatewaySelector       pool.Selectable[gateway.GatewayAPIClient]
	serviceName           string
	translationPath       string
	defaultLanguage       string
	serviceAccountContext context.Context

	// cached within one request not to query other service too much
	spaces    map[string]*storageprovider.StorageSpace
	users     map[string]*user.User
	resources map[string]*storageprovider.ResourceInfo
}

// NewConverter returns a new Converter
func NewConverter(ctx context.Context, loc string, gatewaySelector pool.Selectable[gateway.GatewayAPIClient], name, translationPath, defaultLanguage string) *Converter {
	return &Converter{
		locale:                loc,
		gatewaySelector:       gatewaySelector,
		serviceName:           name,
		translationPath:       translationPath,
		defaultLanguage:       defaultLanguage,
		serviceAccountContext: ctx,
		spaces:                make(map[string]*storageprovider.StorageSpace),
		users:                 make(map[string]*user.User),
		resources:             make(map[string]*storageprovider.ResourceInfo),
	}
}

// ConvertEvent converts an eventhistory event to an OC10Notification
func (c *Converter) ConvertEvent(eventid string, event interface{}) (OC10Notification, error) {
	switch ev := event.(type) {
	default:
		return OC10Notification{}, fmt.Errorf("unknown event type: %T", ev)
	// file related
	case events.PostprocessingStepFinished:
		switch ev.FinishedStep {
		case events.PPStepAntivirus:
			res := ev.Result.(events.VirusscanResult)
			return c.virusMessage(eventid, VirusFound, ev.ExecutingUser, res.ResourceID, ev.Filename, res.Description, res.Scandate)
		case events.PPStepPolicies:
			return c.policiesMessage(eventid, PoliciesEnforced, ev.ExecutingUser, ev.Filename, time.Now())
		default:
			return OC10Notification{}, fmt.Errorf("unknown postprocessing step: %s", ev.FinishedStep)
		}

	// space related
	case events.SpaceDisabled:
		return c.spaceMessage(eventid, SpaceDisabled, ev.Executant, ev.ID.GetOpaqueId(), ev.Timestamp)
	case events.SpaceDeleted:
		return c.spaceDeletedMessage(eventid, ev.Executant, ev.ID.GetOpaqueId(), ev.SpaceName, ev.Timestamp)
	case events.SpaceShared:
		return c.spaceMessage(eventid, SpaceShared, ev.Executant, ev.ID.GetOpaqueId(), ev.Timestamp)
	case events.SpaceUnshared:
		return c.spaceMessage(eventid, SpaceUnshared, ev.Executant, ev.ID.GetOpaqueId(), ev.Timestamp)
	case events.SpaceMembershipExpired:
		return c.spaceMessage(eventid, SpaceMembershipExpired, ev.SpaceOwner, ev.SpaceID.GetOpaqueId(), ev.ExpiredAt)

	// share related
	case events.ShareCreated:
		return c.shareMessage(eventid, ShareCreated, ev.Executant, ev.ItemID, ev.ShareID, utils.TSToTime(ev.CTime))
	case events.ShareExpired:
		return c.shareMessage(eventid, ShareExpired, ev.ShareOwner, ev.ItemID, ev.ShareID, ev.ExpiredAt)
	case events.ShareRemoved:
		return c.shareMessage(eventid, ShareRemoved, ev.Executant, ev.ItemID, ev.ShareID, ev.Timestamp)
	}
}

// ConvertGlobalEvent converts a global event to an OC10Notification
func (c *Converter) ConvertGlobalEvent(typ string, data json.RawMessage) (OC10Notification, error) {
	switch typ {
	default:
		return OC10Notification{}, fmt.Errorf("unknown global event type: %s", typ)
	case "deprovision":
		var dd DeprovisionData
		if err := json.Unmarshal(data, &dd); err != nil {
			return OC10Notification{}, err
		}

		return c.deprovisionMessage(PlatformDeprovision, dd.DeprovisionDate.Format(dd.DeprovisionFormat))
	}

}

func (c *Converter) spaceDeletedMessage(eventid string, executant *user.UserId, spaceid string, spacename string, ts time.Time) (OC10Notification, error) {
	usr, err := c.getUser(context.Background(), executant)
	if err != nil {
		return OC10Notification{}, err
	}

	subj, subjraw, msg, msgraw, err := composeMessage(SpaceDeleted, c.locale, c.defaultLanguage, c.translationPath, map[string]interface{}{
		"username":  usr.GetDisplayName(),
		"spacename": spacename,
	})
	if err != nil {
		return OC10Notification{}, err
	}

	space := &storageprovider.StorageSpace{Id: &storageprovider.StorageSpaceId{OpaqueId: spaceid}, Name: spacename}

	return OC10Notification{
		EventID:        eventid,
		Service:        c.serviceName,
		UserName:       usr.GetUsername(),
		Timestamp:      ts.Format(time.RFC3339Nano),
		ResourceID:     spaceid,
		ResourceType:   _resourceTypeSpace,
		Subject:        subj,
		SubjectRaw:     subjraw,
		Message:        msg,
		MessageRaw:     msgraw,
		MessageDetails: generateDetails(usr, space, nil, nil),
	}, nil
}

func (c *Converter) spaceMessage(eventid string, nt NotificationTemplate, executant *user.UserId, spaceid string, ts time.Time) (OC10Notification, error) {
	usr, err := c.getUser(context.Background(), executant)
	if err != nil {
		return OC10Notification{}, err
	}

	space, err := c.getSpace(c.serviceAccountContext, spaceid)
	if err != nil {
		return OC10Notification{}, err
	}

	subj, subjraw, msg, msgraw, err := composeMessage(nt, c.locale, c.defaultLanguage, c.translationPath, map[string]interface{}{
		"username":  usr.GetDisplayName(),
		"spacename": space.GetName(),
	})
	if err != nil {
		return OC10Notification{}, err
	}

	return OC10Notification{
		EventID:        eventid,
		Service:        c.serviceName,
		UserName:       usr.GetUsername(),
		Timestamp:      ts.Format(time.RFC3339Nano),
		ResourceID:     spaceid,
		ResourceType:   _resourceTypeSpace,
		Subject:        subj,
		SubjectRaw:     subjraw,
		Message:        msg,
		MessageRaw:     msgraw,
		MessageDetails: generateDetails(usr, space, nil, nil),
	}, nil
}

func (c *Converter) shareMessage(eventid string, nt NotificationTemplate, executant *user.UserId, resourceid *storageprovider.ResourceId, shareid *collaboration.ShareId, ts time.Time) (OC10Notification, error) {
	usr, err := c.getUser(context.Background(), executant)
	if err != nil {
		return OC10Notification{}, err
	}

	info, err := c.getResource(c.serviceAccountContext, resourceid)
	if err != nil {
		return OC10Notification{}, err
	}

	subj, subjraw, msg, msgraw, err := composeMessage(nt, c.locale, c.defaultLanguage, c.translationPath, map[string]interface{}{
		"username":     usr.GetDisplayName(),
		"resourcename": info.GetName(),
	})
	if err != nil {
		return OC10Notification{}, err
	}

	return OC10Notification{
		EventID:        eventid,
		Service:        c.serviceName,
		UserName:       usr.GetUsername(),
		Timestamp:      ts.Format(time.RFC3339Nano),
		ResourceID:     storagespace.FormatResourceID(info.GetId()),
		ResourceType:   _resourceTypeShare,
		Subject:        subj,
		SubjectRaw:     subjraw,
		Message:        msg,
		MessageRaw:     msgraw,
		MessageDetails: generateDetails(usr, nil, info, shareid),
	}, nil
}

func (c *Converter) virusMessage(eventid string, nt NotificationTemplate, executant *user.User, rid *storageprovider.ResourceId, filename string, virus string, ts time.Time) (OC10Notification, error) {
	subj, subjraw, msg, msgraw, err := composeMessage(nt, c.locale, c.defaultLanguage, c.translationPath, map[string]interface{}{
		"resourcename":     filename,
		"virusdescription": virus,
	})
	if err != nil {
		return OC10Notification{}, err
	}

	dets := map[string]interface{}{
		"resource": map[string]string{
			"name": filename,
		},
		"virus": map[string]interface{}{
			"name":     virus,
			"scandate": ts,
		},
	}

	return OC10Notification{
		EventID:        eventid,
		Service:        c.serviceName,
		UserName:       executant.GetUsername(),
		Timestamp:      ts.Format(time.RFC3339Nano),
		ResourceID:     storagespace.FormatResourceID(rid),
		ResourceType:   _resourceTypeResource,
		Subject:        subj,
		SubjectRaw:     subjraw,
		Message:        msg,
		MessageRaw:     msgraw,
		MessageDetails: dets,
	}, nil
}

func (c *Converter) policiesMessage(eventid string, nt NotificationTemplate, executant *user.User, filename string, ts time.Time) (OC10Notification, error) {
	subj, subjraw, msg, msgraw, err := composeMessage(nt, c.locale, c.defaultLanguage, c.translationPath, map[string]interface{}{
		"resourcename": filename,
	})
	if err != nil {
		return OC10Notification{}, err
	}

	dets := map[string]interface{}{
		"resource": map[string]string{
			"name": filename,
		},
	}

	return OC10Notification{
		EventID:        eventid,
		Service:        c.serviceName,
		UserName:       executant.GetUsername(),
		Timestamp:      ts.Format(time.RFC3339Nano),
		ResourceType:   _resourceTypeResource,
		Subject:        subj,
		SubjectRaw:     subjraw,
		Message:        msg,
		MessageRaw:     msgraw,
		MessageDetails: dets,
	}, nil
}

func (c *Converter) deprovisionMessage(nt NotificationTemplate, deproDate string) (OC10Notification, error) {
	subj, subjraw, msg, msgraw, err := composeMessage(nt, c.locale, c.defaultLanguage, c.translationPath, map[string]interface{}{
		"date": deproDate,
	})
	if err != nil {
		return OC10Notification{}, err
	}

	return OC10Notification{
		EventID: "deprovision",
		Service: c.serviceName,
		// UserName:       executant.GetUsername(), // TODO: do we need the deprovisioner?
		Timestamp:      time.Now().Format(time.RFC3339Nano), // Fake timestamp? Or we store one with the event?
		ResourceType:   _resourceTypeResource,
		Subject:        subj,
		SubjectRaw:     subjraw,
		Message:        msg,
		MessageRaw:     msgraw,
		MessageDetails: map[string]interface{}{},
	}, nil
}

func (c *Converter) getSpace(ctx context.Context, spaceID string) (*storageprovider.StorageSpace, error) {
	if space, ok := c.spaces[spaceID]; ok {
		return space, nil
	}
	gwc, err := c.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	space, err := utils.GetSpace(ctx, spaceID, gwc)
	if err == nil {
		c.spaces[spaceID] = space
	}
	return space, err
}

func (c *Converter) getResource(ctx context.Context, resourceID *storageprovider.ResourceId) (*storageprovider.ResourceInfo, error) {
	if r, ok := c.resources[resourceID.GetOpaqueId()]; ok {
		return r, nil
	}
	gwc, err := c.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	resource, err := utils.GetResourceByID(ctx, resourceID, gwc)
	if err == nil {
		c.resources[resourceID.GetOpaqueId()] = resource
	}
	return resource, err
}

func (c *Converter) getUser(_ context.Context, userID *user.UserId) (*user.User, error) {
	if u, ok := c.users[userID.GetOpaqueId()]; ok {
		return u, nil
	}
	gwc, err := c.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	usr, err := utils.GetUser(userID, gwc)
	if err == nil {
		c.users[userID.GetOpaqueId()] = usr
	}
	return usr, err
}

func composeMessage(nt NotificationTemplate, locale, defaultLocale, path string, vars map[string]interface{}) (string, string, string, string, error) {
	subjectraw, messageraw := loadTemplates(nt, locale, defaultLocale, path)

	subject, err := executeTemplate(subjectraw, vars)
	if err != nil {
		return "", "", "", "", err
	}

	message, err := executeTemplate(messageraw, vars)
	return subject, subjectraw, message, messageraw, err
}

func loadTemplates(nt NotificationTemplate, locale, defaultLocale, path string) (string, string) {
	t := l10n.NewTranslatorFromCommonConfig(defaultLocale, _domain, path, _translationFS, "l10n/locale").Locale(locale)
	return t.Get(nt.Subject, []interface{}{}...), t.Get(nt.Message, []interface{}{}...)
}

func executeTemplate(raw string, vars map[string]interface{}) (string, error) {
	for o, n := range _placeholders {
		raw = strings.ReplaceAll(raw, o, n)
	}
	tpl, err := template.New("").Parse(raw)
	if err != nil {
		return "", err
	}
	var writer bytes.Buffer
	if err := tpl.Execute(&writer, vars); err != nil {
		return "", err
	}

	return writer.String(), nil
}

func generateDetails(user *user.User, space *storageprovider.StorageSpace, item *storageprovider.ResourceInfo, shareid *collaboration.ShareId) map[string]interface{} {
	details := make(map[string]interface{})

	if user != nil {
		details["user"] = map[string]string{
			"id":          user.GetId().GetOpaqueId(),
			"name":        user.GetUsername(),
			"displayname": user.GetDisplayName(),
		}
	}

	if space != nil {
		details["space"] = map[string]string{
			"id":   space.GetId().GetOpaqueId(),
			"name": space.GetName(),
		}
	}

	if item != nil {
		details["resource"] = map[string]string{
			"id":   storagespace.FormatResourceID(item.GetId()),
			"name": item.GetName(),
		}
	}

	if shareid != nil {
		details["share"] = map[string]string{
			"id": shareid.GetOpaqueId(),
		}
	}

	return details
}
