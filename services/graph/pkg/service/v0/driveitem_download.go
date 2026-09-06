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

	downloadURL, err := g.signedDownloadURL(&itemID, user.GetId().GetOpaqueId())
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	http.Redirect(w, r, downloadURL, http.StatusFound)
}

func (g BaseGraphService) signedDownloadURL(id *storageprovider.ResourceId, userID string) (string, error) {
	if g.downloadSigner == nil {
		return "", errors.New("download url signing is not configured")
	}
	base, err := g.getWebDavBaseURL()
	if err != nil {
		return "", err
	}
	base.Path = path.Join(base.Path, storagespace.FormatResourceID(id))
	return g.downloadSigner.Sign(base.String(), userID, 30*time.Minute)
}

func (g Graph) setDriveItemsDownloadURL(r *http.Request, items []libregraph.DriveItem, infos []*storageprovider.ResourceInfo) {
	if !g.downloadURLRequested(r) {
		return
	}
	for i := range items {
		g.setDriveItemDownloadURL(r, &items[i], infos[i])
	}
}

func (g Graph) setDriveItemDownloadURL(r *http.Request, item *libregraph.DriveItem, info *storageprovider.ResourceInfo) {
	if item.File == nil || !g.downloadURLRequested(r) {
		return
	}
	user, ok := revactx.ContextGetUser(r.Context())
	if !ok {
		return
	}
	u, err := g.signedDownloadURL(info.GetId(), user.GetId().GetOpaqueId())
	if err != nil {
		return
	}
	item.MicrosoftGraphDownloadUrl = &u
}

// downloadURLRequested reports whether a signed download url can and should be built
func (g Graph) downloadURLRequested(r *http.Request) bool {
	return g.downloadSigner != nil && driveItemPropertySelected(r, _selectDownloadURL)
}
