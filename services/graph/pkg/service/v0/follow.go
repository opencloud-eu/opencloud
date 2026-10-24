package svc

import (
	"net/http"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/render"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	revaCtx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"
)

const (
	_favoriteLabel = "favorite"
	// deliberately not using the existing node.FavoriteKey constant to avoid importing a specific storage driver package
	_favoriteMetadataKey = "http://owncloud.org/ns/favorite"
)

// FollowDriveItem marks a drive item as favorite.
func (g Graph) FollowDriveItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	itemID, err := parseIDParam(r, "itemID")
	if err != nil {
		g.logger.Debug().Err(err).Msg("could not parse itemID")
		return
	}

	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		g.logger.Error().Err(err).Msg("could not select next gateway client")
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusServiceUnavailable, "could not select next gateway client")
		return
	}

	ref := &provider.Reference{
		ResourceId: &itemID,
	}

	u, ok := revactx.ContextGetUser(ctx)
	if !ok {
		errorcode.GeneralException.Render(w, r, http.StatusUnauthorized, "User not found in context")
		return
	}

	statReq := &provider.StatRequest{
		Ref: ref,
	}
	statRes, err := gatewayClient.Stat(ctx, statReq)
	if err != nil {
		g.logger.Error().Err(err).Msg("could not stat resource")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not stat resource")
		return
	}
	switch statRes.GetStatus().GetCode() {
	case rpc.Code_CODE_OK:
		// continue
	case rpc.Code_CODE_NOT_FOUND:
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "resource not found")
		return
	default:
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not stat resource")
		return
	}

	req := &provider.AddLabelRequest{
		Ref:    ref,
		UserId: u.Id,
		Label:  _favoriteLabel,
	}

	res, err := gatewayClient.AddLabel(ctx, req)
	if err != nil {
		g.logger.Error().Err(err).Msg("could not add label")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not add label")
		return
	}

	if res.Status.Code != rpc.Code_CODE_OK {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not add label")
		return
	}

	if g.eventsPublisher != nil {
		ev := events.LabelAdded{
			Ref: &provider.Reference{
				ResourceId: &itemID,
				Path:       ".",
			},
			Label:     _favoriteLabel,
			UserID:    u.Id,
			Executant: revaCtx.ContextMustGetUser(r.Context()).Id,
		}
		if err := events.Publish(r.Context(), g.eventsPublisher, ev); err != nil {
			g.logger.Error().Err(err).Msg("Failed to publish LabelAdded event")
		}
	}

	driveItem, err := cs3ResourceToDriveItem(g.logger, g.publicBaseURL, statRes.GetInfo())
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	driveItem.SetLibreGraphMeFollowing(true)

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, &driveItem)
}

// UnfollowDriveItem unmarks a drive item as favorite.
func (g Graph) UnfollowDriveItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	itemID, err := parseIDParam(r, "itemID")
	if err != nil {
		g.logger.Debug().Err(err).Msg("could not parse itemID")
		return
	}

	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		g.logger.Error().Err(err).Msg("could not select next gateway client")
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusServiceUnavailable, "could not select next gateway client")
		return
	}

	ref := &provider.Reference{
		ResourceId: &itemID,
	}

	u, ok := revactx.ContextGetUser(ctx)
	if !ok {
		errorcode.GeneralException.Render(w, r, http.StatusUnauthorized, "User not found in context")
		return
	}

	req := &provider.RemoveLabelRequest{
		Ref:    ref,
		UserId: u.Id,
		Label:  _favoriteLabel,
	}

	res, err := gatewayClient.RemoveLabel(ctx, req)
	if err != nil {
		g.logger.Error().Err(err).Msg("could not remove label")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not remove label")
		return
	}

	switch res.Status.Code {
	case rpc.Code_CODE_OK:
		// continue
	case rpc.Code_CODE_NOT_FOUND:
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "favorite not found")
		return
	default:
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not remove favorite")
		return
	}

	if g.eventsPublisher != nil {
		ev := events.LabelRemoved{
			Ref: &provider.Reference{
				ResourceId: &itemID,
				Path:       ".",
			},
			Label:     _favoriteLabel,
			UserID:    u.Id,
			Executant: revaCtx.ContextMustGetUser(r.Context()).Id,
		}
		if err := events.Publish(r.Context(), g.eventsPublisher, ev); err != nil {
			g.logger.Error().Err(err).Msg("Failed to publish LabelRemoved event")
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
