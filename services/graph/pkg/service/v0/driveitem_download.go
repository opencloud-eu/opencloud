package svc

import (
	"errors"
	"net/http"
	"path"
	"time"

	cs3rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"

	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
)

const downloadURLTTL = 30 * time.Minute

// ErrDownloadURLSigningNotConfigured is returned when no url signing secret is configured
var ErrDownloadURLSigningNotConfigured = errors.New("download url signing is not configured")

// GetDriveItemContent redirects to a signed download url for a file
func (g Graph) GetDriveItemContent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	driveID, err := parseIDParam(r, "driveID")
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}
	itemID, err := parseIDParam(r, "itemID")
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}
	if driveID.GetStorageId() != itemID.GetStorageId() || driveID.GetSpaceId() != itemID.GetSpaceId() {
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "Item does not exist")
		return
	}

	user, ok := revactx.ContextGetUser(ctx)
	if !ok {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "user not in context")
		return
	}

	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	stat, err := gatewayClient.Stat(ctx, &storageprovider.StatRequest{Ref: &storageprovider.Reference{ResourceId: &itemID}})
	switch {
	case err != nil:
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	case stat.GetStatus().GetCode() == cs3rpc.Code_CODE_OK:
	case stat.GetStatus().GetCode() == cs3rpc.Code_CODE_NOT_FOUND:
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, stat.GetStatus().GetMessage())
		return
	case stat.GetStatus().GetCode() == cs3rpc.Code_CODE_PERMISSION_DENIED:
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, stat.GetStatus().GetMessage())
		return
	case stat.GetStatus().GetCode() == cs3rpc.Code_CODE_UNAUTHENTICATED:
		errorcode.Unauthenticated.Render(w, r, http.StatusUnauthorized, stat.GetStatus().GetMessage())
		return
	default:
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, stat.GetStatus().GetMessage())
		return
	}
	if stat.GetInfo().GetType() != storageprovider.ResourceType_RESOURCE_TYPE_FILE {
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "Item is not a file")
		return
	}

	downloadURL, err := g.signedDownloadURL(&itemID, user.GetId().GetOpaqueId())
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	http.Redirect(w, r, downloadURL, http.StatusFound)
}

// SetDriveItemsDownloadURL adds a signed download url to every file in items when requested via $select
func (g BaseGraphService) SetDriveItemsDownloadURL(r *http.Request, items []libregraph.DriveItem) {
	if !g.downloadURLRequested(r) {
		return
	}
	user, ok := revactx.ContextGetUser(r.Context())
	if !ok {
		return
	}
	for i := range items {
		g.signDriveItemDownloadURL(&items[i], user.GetId().GetOpaqueId())
	}
}

func (g BaseGraphService) setDriveItemDownloadURL(r *http.Request, item *libregraph.DriveItem) {
	if !g.downloadURLRequested(r) {
		return
	}
	user, ok := revactx.ContextGetUser(r.Context())
	if !ok {
		return
	}
	g.signDriveItemDownloadURL(item, user.GetId().GetOpaqueId())
}

func (g BaseGraphService) signDriveItemDownloadURL(item *libregraph.DriveItem, userID string) {
	if item.File == nil {
		return
	}
	id, err := storagespace.ParseID(item.GetId())
	if err != nil {
		g.logger.Debug().Err(err).Str("id", item.GetId()).Msg("could not parse drive item id for the download url")
		return
	}
	u, err := g.signedDownloadURL(&id, userID)
	if err != nil {
		g.logger.Debug().Err(err).Str("id", item.GetId()).Msg("could not sign the download url")
		return
	}
	item.MicrosoftGraphDownloadUrl = &u
}

func (g BaseGraphService) signedDownloadURL(id *storageprovider.ResourceId, userID string) (string, error) {
	if g.downloadSigner == nil {
		return "", ErrDownloadURLSigningNotConfigured
	}
	base, err := g.getWebDavBaseURL()
	if err != nil {
		return "", err
	}
	base.Path = path.Join(base.Path, storagespace.FormatResourceID(id))
	return g.downloadSigner.Sign(base.String(), userID, downloadURLTTL)
}

func (g BaseGraphService) downloadURLRequested(r *http.Request) bool {
	return g.downloadSigner != nil && driveItemPropertySelected(r, _selectDownloadURL)
}
