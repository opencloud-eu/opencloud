package svc

import (
	"net/http"
	"net/url"
	"path"
	"sort"
	"time"

	cs3rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"

	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
)

const _versionIDCurrent = "current"

// ListDriveItemVersions lists the versions of a file
func (g Graph) ListDriveItemVersions(w http.ResponseWriter, r *http.Request) {
	g.logger.Info().Msg("Calling ListDriveItemVersions")

	itemID, ok := g.fileDriveItemFromRequest(w, r)
	if !ok {
		return
	}

	versions, ok := g.listDriveItemVersions(w, r, itemID)
	if !ok {
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &ListResponse{Value: versions})
}

// GetDriveItemVersion returns a single version of a file
func (g Graph) GetDriveItemVersion(w http.ResponseWriter, r *http.Request) {
	g.logger.Info().Msg("Calling GetDriveItemVersion")

	itemID, ok := g.fileDriveItemFromRequest(w, r)
	if !ok {
		return
	}
	versionID := chi.URLParam(r, "versionID")

	if versionID == _versionIDCurrent {
		version, ok := g.currentDriveItemVersion(w, r, itemID)
		if !ok {
			return
		}
		render.Status(r, http.StatusOK)
		render.JSON(w, r, version)
		return
	}

	version, ok := g.getDriveItemVersion(w, r, itemID, versionID)
	if !ok {
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, version)
}

// GetDriveItemVersionContent redirects to a signed download url for a version
func (g Graph) GetDriveItemVersionContent(w http.ResponseWriter, r *http.Request) {
	g.logger.Info().Msg("Calling GetDriveItemVersionContent")

	itemID, ok := g.fileDriveItemFromRequest(w, r)
	if !ok {
		return
	}
	versionID := chi.URLParam(r, "versionID")

	user, ok := revactx.ContextGetUser(r.Context())
	if !ok {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "user not in context")
		return
	}

	var downloadURL string
	var err error
	switch versionID {
	case _versionIDCurrent:
		downloadURL, err = g.signedDownloadURL(itemID, user.GetId().GetOpaqueId())
	default:
		if _, ok := g.getDriveItemVersion(w, r, itemID, versionID); !ok {
			return
		}
		downloadURL, err = g.signedVersionDownloadURL(itemID, versionID, user.GetId().GetOpaqueId())
	}
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	http.Redirect(w, r, downloadURL, http.StatusFound)
}

// RestoreDriveItemVersion restores a version of a file
func (g Graph) RestoreDriveItemVersion(w http.ResponseWriter, r *http.Request) {
	g.logger.Info().Msg("Calling RestoreDriveItemVersion")

	itemID, ok := g.fileDriveItemFromRequest(w, r)
	if !ok {
		return
	}
	versionID := chi.URLParam(r, "versionID")

	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	res, err := gatewayClient.RestoreFileVersion(r.Context(), &storageprovider.RestoreFileVersionRequest{
		Ref: &storageprovider.Reference{ResourceId: itemID},
		Key: versionID,
	})
	if err := errorcode.FromCS3Status(res.GetStatus(), err); err != nil {
		errorcode.RenderError(w, r, err)
		return
	}

	render.Status(r, http.StatusNoContent)
	render.NoContent(w, r)
}

// fileDriveItemFromRequest parses and stats the item, rendering 404 unless it is a file
func (g Graph) fileDriveItemFromRequest(w http.ResponseWriter, r *http.Request) (*storageprovider.ResourceId, bool) {
	driveID, err := parseIDParam(r, "driveID")
	if err != nil {
		errorcode.RenderError(w, r, err)
		return nil, false
	}
	itemID, err := parseIDParam(r, "driveItemID")
	if err != nil {
		errorcode.RenderError(w, r, err)
		return nil, false
	}
	if driveID.GetStorageId() != itemID.GetStorageId() || driveID.GetSpaceId() != itemID.GetSpaceId() {
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "Item does not exist")
		return nil, false
	}

	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return nil, false
	}
	res, err := gatewayClient.Stat(r.Context(), &storageprovider.StatRequest{Ref: &storageprovider.Reference{ResourceId: &itemID}})
	if !renderStatStatus(w, r, res, err) {
		return nil, false
	}
	if res.GetInfo().GetType() != storageprovider.ResourceType_RESOURCE_TYPE_FILE {
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "Item is not a file")
		return nil, false
	}
	return &itemID, true
}

// renderStatStatus renders the error of a failed stat
func renderStatStatus(w http.ResponseWriter, r *http.Request, res *storageprovider.StatResponse, err error) bool {
	switch {
	case err != nil:
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return false
	case res.GetStatus().GetCode() == cs3rpc.Code_CODE_OK:
		return true
	case res.GetStatus().GetCode() == cs3rpc.Code_CODE_NOT_FOUND:
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, res.GetStatus().GetMessage())
		return false
	case res.GetStatus().GetCode() == cs3rpc.Code_CODE_PERMISSION_DENIED:
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, res.GetStatus().GetMessage())
		return false
	case res.GetStatus().GetCode() == cs3rpc.Code_CODE_UNAUTHENTICATED:
		errorcode.Unauthenticated.Render(w, r, http.StatusUnauthorized, res.GetStatus().GetMessage())
		return false
	default:
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, res.GetStatus().GetMessage())
		return false
	}
}

func (g Graph) listDriveItemVersions(w http.ResponseWriter, r *http.Request, itemID *storageprovider.ResourceId) ([]libregraph.DriveItemVersion, bool) {
	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return nil, false
	}
	res, err := gatewayClient.ListFileVersions(r.Context(), &storageprovider.ListFileVersionsRequest{
		Ref: &storageprovider.Reference{ResourceId: itemID},
	})
	if err := errorcode.FromCS3Status(res.GetStatus(), err); err != nil {
		errorcode.RenderError(w, r, err)
		return nil, false
	}

	fileVersions := res.GetVersions()
	// newest first, the storage does not guarantee an order
	sort.SliceStable(fileVersions, func(i, j int) bool {
		if fileVersions[i].GetMtime() != fileVersions[j].GetMtime() {
			return fileVersions[i].GetMtime() > fileVersions[j].GetMtime()
		}
		return fileVersions[i].GetKey() > fileVersions[j].GetKey()
	})

	versions := make([]libregraph.DriveItemVersion, 0, len(fileVersions))
	for _, fv := range fileVersions {
		version := libregraph.NewDriveItemVersion()
		version.SetId(fv.GetKey())
		version.SetLastModifiedDateTime(time.Unix(int64(fv.GetMtime()), 0).UTC())
		version.SetSize(int64(fv.GetSize()))
		g.setDriveItemVersionDownloadURL(r, version, itemID, fv.GetKey())
		versions = append(versions, *version)
	}
	return versions, true
}

func (g Graph) getDriveItemVersion(w http.ResponseWriter, r *http.Request, itemID *storageprovider.ResourceId, versionID string) (*libregraph.DriveItemVersion, bool) {
	versions, ok := g.listDriveItemVersions(w, r, itemID)
	if !ok {
		return nil, false
	}
	for i := range versions {
		if versions[i].GetId() == versionID {
			return &versions[i], true
		}
	}
	errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "Version does not exist")
	return nil, false
}

// currentDriveItemVersion describes the file itself as a version
func (g Graph) currentDriveItemVersion(w http.ResponseWriter, r *http.Request, itemID *storageprovider.ResourceId) (*libregraph.DriveItemVersion, bool) {
	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return nil, false
	}
	res, err := gatewayClient.Stat(r.Context(), &storageprovider.StatRequest{Ref: &storageprovider.Reference{ResourceId: itemID}})
	if !renderStatStatus(w, r, res, err) {
		return nil, false
	}

	version := libregraph.NewDriveItemVersion()
	version.SetId(_versionIDCurrent)
	version.SetLastModifiedDateTime(cs3TimestampToTime(res.GetInfo().GetMtime()).UTC())
	version.SetSize(int64(res.GetInfo().GetSize()))
	if user, ok := revactx.ContextGetUser(r.Context()); ok && g.downloadURLRequested(r) {
		if u, err := g.signedDownloadURL(itemID, user.GetId().GetOpaqueId()); err == nil {
			version.SetMicrosoftGraphDownloadUrl(u)
		}
	}
	return version, true
}

func (g Graph) setDriveItemVersionDownloadURL(r *http.Request, version *libregraph.DriveItemVersion, itemID *storageprovider.ResourceId, key string) {
	if !g.downloadURLRequested(r) {
		return
	}
	user, ok := revactx.ContextGetUser(r.Context())
	if !ok {
		return
	}
	u, err := g.signedVersionDownloadURL(itemID, key, user.GetId().GetOpaqueId())
	if err != nil {
		return
	}
	version.SetMicrosoftGraphDownloadUrl(u)
}

// signedVersionDownloadURL signs a download url for the WebDAV meta endpoint of a version
func (g BaseGraphService) signedVersionDownloadURL(itemID *storageprovider.ResourceId, key, userID string) (string, error) {
	if g.downloadSigner == nil {
		return "", ErrDownloadURLSigningNotConfigured
	}
	u, err := g.getWebDavMetaURL()
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, storagespace.FormatResourceID(itemID), "v", key)
	return g.downloadSigner.Sign(u.String(), userID, downloadURLTTL)
}

func (g BaseGraphService) getWebDavMetaURL() (*url.URL, error) {
	u := *g.publicBaseURL
	u.Path = path.Join(u.Path, path.Dir(path.Clean(g.config.Spaces.WebDavPath)), "meta")
	return &u, nil
}
