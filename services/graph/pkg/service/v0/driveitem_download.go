package svc

import (
	"errors"
	"net/http"
	"path"
	"strings"
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

func shouldSelect(r *http.Request, property string) bool {
	for _, v := range strings.Split(r.URL.Query().Get("$select"), ",") {
		if strings.TrimSpace(v) == property {
			return true
		}
	}
	return false
}

func (g Graph) setDriveItemsDownloadURL(r *http.Request, items []*libregraph.DriveItem, infos []*storageprovider.ResourceInfo) {
	if g.downloadSigner == nil || !shouldSelect(r, "@microsoft.graph.downloadUrl") {
		return
	}
	user, ok := revactx.ContextGetUser(r.Context())
	if !ok {
		return
	}
	for i := range items {
		if items[i].File == nil {
			continue
		}
		u, err := g.signedDownloadURL(infos[i].GetId(), user.GetId().GetOpaqueId())
		if err != nil {
			continue
		}
		items[i].MicrosoftGraphDownloadUrl = &u
	}
}
