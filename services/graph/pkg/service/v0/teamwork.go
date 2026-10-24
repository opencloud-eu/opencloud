package svc

import (
	"context"
	"net/http"
	"net/url"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"google.golang.org/grpc/metadata"

	ocEvents "github.com/opencloud-eu/opencloud/pkg/events"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	settingsServiceExt "github.com/opencloud-eu/opencloud/services/settings/pkg/store/defaults"
)

const (
	// WebOfficeAppID identifies the app a notification comes from
	WebOfficeAppID = "8d1c9c88-9e2c-4d0b-9a1e-6a9de1cb9d3c"

	_activityTypeMentioned = "mentioned"
	_topicSourceText       = "text"
)

type activityTopic struct {
	Source string `json:"source"`
	Value  string `json:"value"`
}

type activityNotification struct {
	Topic        activityTopic `json:"topic"`
	ActivityType string        `json:"activityType"`
	TeamsAppID   string        `json:"teamsAppId"`
}

// SendActivityNotification tells a user that they were named on a resource.
func (g Graph) SendActivityNotification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if g.eventsPublisher == nil {
		g.logger.Error().Msg("no events publisher configured, activity notifications cannot be delivered")
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusServiceUnavailable, "activity notifications are not available")
		return
	}

	executant, ok := revactx.ContextGetUser(ctx)
	if !ok {
		errorcode.GeneralException.Render(w, r, http.StatusUnauthorized, "user not found in context")
		return
	}

	userID, err := url.PathUnescape(chi.URLParam(r, "userID"))
	if err != nil || userID == "" {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid user id")
		return
	}

	notification := activityNotification{}
	if err := StrictJSONUnmarshal(r.Body, &notification); err != nil {
		g.logger.Debug().Err(err).Msg("could not parse the activity notification")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	switch {
	case notification.ActivityType != _activityTypeMentioned:
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unsupported activityType")
		return
	case notification.TeamsAppID != WebOfficeAppID:
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unknown teamsAppId")
		return
	case notification.Topic.Source != _topicSourceText:
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unsupported topic source")
		return
	}

	itemID, err := storagespace.ParseID(notification.Topic.Value)
	if err != nil || itemID.GetStorageId() == "" || itemID.GetSpaceId() == "" || itemID.GetOpaqueId() == "" {
		g.logger.Debug().Err(err).Msg("could not parse the topic value")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "the topic value is no resource id")
		return
	}

	if _, err := g.permissionsService.GetPermissionByID(ctx, &settingssvc.GetPermissionByIDRequest{
		PermissionId: settingsServiceExt.CollaborationPublishNotificationPermission(0).Id,
	}); err != nil {
		g.logger.Debug().Err(err).Msg("user is not allowed to publish activity notifications")
		errorcode.AccessDenied.Render(w, r, http.StatusForbidden, "not allowed to publish activity notifications")
		return
	}

	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		g.logger.Error().Err(err).Msg("could not select next gateway client")
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusServiceUnavailable, "could not select next gateway client")
		return
	}

	ref := &storageprovider.Reference{ResourceId: &itemID}

	statResponse, err := gatewayClient.Stat(ctx, &storageprovider.StatRequest{Ref: ref})
	switch {
	case err != nil:
		g.logger.Error().Err(err).Msg("could not stat item")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not stat item")
		return
	case statResponse.GetStatus().GetCode() == rpc.Code_CODE_NOT_FOUND,
		statResponse.GetStatus().GetCode() == rpc.Code_CODE_PERMISSION_DENIED:
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "item not found")
		return
	case statResponse.GetStatus().GetCode() != rpc.Code_CODE_OK:
		g.logger.Error().
			Str("code", statResponse.GetStatus().GetCode().String()).
			Str("message", statResponse.GetStatus().GetMessage()).
			Msg("could not stat item")
		errorcode.RenderError(w, r, errorcode.FromCS3Status(statResponse.GetStatus(), nil))
		return
	}

	authResponse, err := gatewayClient.Authenticate(ctx, &gateway.AuthenticateRequest{
		Type:         "machine",
		ClientId:     "userid:" + userID,
		ClientSecret: g.config.MachineAuthAPIKey,
	})
	switch {
	case err != nil:
		g.logger.Error().Err(err).Msg("could not authenticate the recipient")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not authenticate the recipient")
		return
	case authResponse.GetStatus().GetCode() == rpc.Code_CODE_NOT_FOUND:
		g.logger.Debug().Str("userID", userID).Msg("the recipient does not exist")
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "recipient not found")
		return
	case authResponse.GetStatus().GetCode() != rpc.Code_CODE_OK:
		g.logger.Error().
			Str("userID", userID).
			Str("code", authResponse.GetStatus().GetCode().String()).
			Str("message", authResponse.GetStatus().GetMessage()).
			Msg("could not authenticate the recipient")
		errorcode.RenderError(w, r, errorcode.FromCS3Status(authResponse.GetStatus(), nil))
		return
	}

	recipientStat, err := gatewayClient.Stat(
		func() context.Context {
			md, _ := metadata.FromOutgoingContext(ctx)
			md = md.Copy()
			md.Set(revactx.TokenHeader, authResponse.GetToken())

			return metadata.NewOutgoingContext(ctx, md)
		}(),
		&storageprovider.StatRequest{Ref: ref},
	)

	switch {
	case err != nil:
		g.logger.Error().Err(err).Msg("could not stat the item as the recipient")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not stat item")
		return
	case recipientStat.GetStatus().GetCode() == rpc.Code_CODE_NOT_FOUND,
		recipientStat.GetStatus().GetCode() == rpc.Code_CODE_PERMISSION_DENIED:
		g.logger.Debug().Str("userID", userID).Msg("mention dropped, the recipient has no access to the item")
		w.WriteHeader(http.StatusAccepted)
		return
	case recipientStat.GetStatus().GetCode() != rpc.Code_CODE_OK:
		g.logger.Error().
			Str("userID", userID).
			Str("code", recipientStat.GetStatus().GetCode().String()).
			Str("message", recipientStat.GetStatus().GetMessage()).
			Msg("could not stat the item as the recipient")
		errorcode.RenderError(w, r, errorcode.FromCS3Status(recipientStat.GetStatus(), nil))
		return
	}

	event := ocEvents.ResourceMention{
		Executant: executant.GetId(),
		UserIDs:   []*userpb.UserId{authResponse.GetUser().GetId()},
		Ref:       &storageprovider.Reference{ResourceId: statResponse.GetInfo().GetId()},
		Timestamp: time.Now(),
	}

	if err := events.Publish(ctx, g.eventsPublisher, event); err != nil {
		g.logger.Error().Err(err).Msg("could not publish the activity notification")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not publish the activity notification")
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
