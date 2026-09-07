package svc

import (
	"errors"
	"io"
	"net/http"
	"path"

	cs3rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
)

// restoreRequest is the optional body of driveItem: restore
type restoreRequest struct {
	ParentReference *libregraph.ItemReference `json:"parentReference,omitempty"`
	Name            *string                   `json:"name,omitempty"`
}

// RestoreDriveItem restores a trashed item, by default to its original location.
// The item id carries the recycle key, see recycleItemToDriveItem.
func (g Graph) RestoreDriveItem(w http.ResponseWriter, r *http.Request) {
	g.logger.Debug().Msg("Calling RestoreDriveItem")
	ctx := r.Context()

	itemID, ok := parseTrashItemID(w, r)
	if !ok {
		return
	}

	var body restoreRequest
	if err := StrictJSONUnmarshal(r.Body, &body); err != nil && !errors.Is(err, io.EOF) {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}

	// the listing gives us the original location, needed for the default target and the response
	key := itemID.GetOpaqueId()
	items, ok := g.listRecycle(w, r, itemID, key)
	if !ok {
		return
	}
	var trashed *storageprovider.RecycleItem
	for _, item := range items {
		if item.GetKey() == key {
			trashed = item
		}
	}
	if trashed == nil {
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "item not found")
		return
	}

	target, ok := restoreTarget(w, r, itemID, trashed, body)
	if !ok {
		return
	}

	gatewayClient, ok := g.GetGatewayClient(w, r)
	if !ok {
		return
	}
	res, err := gatewayClient.RestoreRecycleItem(ctx, &storageprovider.RestoreRecycleItemRequest{
		Ref:        &storageprovider.Reference{ResourceId: spaceRootID(itemID)},
		Key:        key,
		RestoreRef: target,
	})
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if !renderTrashStatus(w, r, res.GetStatus()) {
		return
	}

	statRes, err := gatewayClient.Stat(ctx, &storageprovider.StatRequest{Ref: target})
	switch {
	case err != nil:
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	case statRes.GetStatus().GetCode() != cs3rpc.Code_CODE_OK:
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "restored, but could not stat the item: "+statRes.GetStatus().GetMessage())
		return
	}
	driveItem, err := cs3ResourceToDriveItem(g.logger, g.publicBaseURL, statRes.GetInfo())
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, driveItem)
}

// restoreTarget builds the restore reference: the original location unless the body moves the item
func restoreTarget(w http.ResponseWriter, r *http.Request, itemID *storageprovider.ResourceId, trashed *storageprovider.RecycleItem, body restoreRequest) (*storageprovider.Reference, bool) {
	origin := trashed.GetRef().GetPath()
	name := path.Base(origin)
	if body.Name != nil && *body.Name != "" {
		name = *body.Name
	}
	if name != path.Base(name) || name == "." || name == "/" {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid name")
		return nil, false
	}

	parent := body.ParentReference
	if parent != nil && parent.DriveId != nil {
		parentDrive, err := storagespace.ParseID(parent.GetDriveId())
		if err != nil || parentDrive.GetStorageId() != itemID.GetStorageId() || parentDrive.GetSpaceId() != itemID.GetSpaceId() {
			errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "restore into another drive is not supported")
			return nil, false
		}
	}

	switch {
	case parent != nil && parent.Id != nil:
		parentID, err := storagespace.ParseID(parent.GetId())
		if err != nil || parentID.GetStorageId() != itemID.GetStorageId() || parentID.GetSpaceId() != itemID.GetSpaceId() {
			errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid parentReference.id")
			return nil, false
		}
		return &storageprovider.Reference{ResourceId: &parentID, Path: utils.MakeRelativePath(name)}, true
	case parent != nil && parent.Path != nil:
		return &storageprovider.Reference{
			ResourceId: spaceRootID(itemID),
			Path:       utils.MakeRelativePath(path.Join(parent.GetPath(), name)),
		}, true
	default:
		return &storageprovider.Reference{
			ResourceId: spaceRootID(itemID),
			Path:       utils.MakeRelativePath(path.Join(path.Dir(origin), name)),
		}, true
	}
}

// PermanentDeleteDriveItem deletes a live item and purges it from the trash right away.
// reva has no delete that bypasses the trash, so this is two steps; the trash key of a
// freshly deleted item is its node id in both decomposedfs and posixfs.
func (g Graph) PermanentDeleteDriveItem(w http.ResponseWriter, r *http.Request) {
	g.logger.Debug().Msg("Calling PermanentDeleteDriveItem")
	ctx := r.Context()

	itemID, ok := parseTrashItemID(w, r)
	if !ok {
		return
	}
	if IsSpaceRoot(itemID) {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "cannot delete the drive root")
		return
	}
	if IsShareJail(itemID) {
		// the trash of a shared item lives in the owner's space, which the share jail id does not tell us
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "not supported for shared items, use the item id of the owning drive")
		return
	}

	gatewayClient, ok := g.GetGatewayClient(w, r)
	if !ok {
		return
	}
	delRes, err := gatewayClient.Delete(ctx, &storageprovider.DeleteRequest{
		Ref: &storageprovider.Reference{ResourceId: itemID},
	})
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if !renderTrashStatus(w, r, delRes.GetStatus()) {
		return
	}

	purgeRes, err := gatewayClient.PurgeRecycle(ctx, &storageprovider.PurgeRecycleRequest{
		Ref: &storageprovider.Reference{ResourceId: spaceRootID(itemID)},
		Key: itemID.GetOpaqueId(),
	})
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "deleted, but could not purge from the trash: "+err.Error())
		return
	}
	if purgeRes.GetStatus().GetCode() != cs3rpc.Code_CODE_OK {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "deleted, but could not purge from the trash: "+purgeRes.GetStatus().GetMessage())
		return
	}

	render.Status(r, http.StatusNoContent)
	render.NoContent(w, r)
}

// DeleteDriveSpecialItem purges one trashed item
func (g Graph) DeleteDriveSpecialItem(w http.ResponseWriter, r *http.Request) {
	g.logger.Debug().Msg("Calling DeleteDriveSpecialItem")

	driveID, ok := parseSpecialParams(w, r)
	if !ok {
		return
	}
	itemID, ok := parseTrashItemID(w, r)
	if !ok {
		return
	}

	g.purgeRecycle(w, r, &driveID, itemID.GetOpaqueId())
}

// EmptyDriveSpecial purges the whole trash. In the colon path form
// (special/recyclebin:/{key}) it purges only that item.
func (g Graph) EmptyDriveSpecial(w http.ResponseWriter, r *http.Request) {
	g.logger.Debug().Msg("Calling EmptyDriveSpecial")

	driveID, ok := parseSpecialParams(w, r)
	if !ok {
		return
	}

	g.purgeRecycle(w, r, &driveID, specialFolderKey(r))
}

// purgeRecycle purges the key, or the whole trash for an empty key
func (g Graph) purgeRecycle(w http.ResponseWriter, r *http.Request, driveID *storageprovider.ResourceId, key string) {
	gatewayClient, ok := g.GetGatewayClient(w, r)
	if !ok {
		return
	}
	res, err := gatewayClient.PurgeRecycle(r.Context(), &storageprovider.PurgeRecycleRequest{
		Ref: &storageprovider.Reference{ResourceId: spaceRootID(driveID)},
		Key: key,
	})
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if !renderTrashStatus(w, r, res.GetStatus()) {
		return
	}

	render.Status(r, http.StatusNoContent)
	render.NoContent(w, r)
}

// parseTrashItemID reads the item id, which the v1.0 and v1beta1 routes bind under different names,
// and checks it against the drive id when the route has one
func parseTrashItemID(w http.ResponseWriter, r *http.Request) (*storageprovider.ResourceId, bool) {
	param := "itemID"
	if chi.URLParam(r, param) == "" {
		param = "driveItemID"
	}
	itemID, err := parseIDParam(r, param)
	if err != nil {
		errorcode.RenderError(w, r, err)
		return nil, false
	}
	if itemID.GetOpaqueId() == "" {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid itemID")
		return nil, false
	}

	if chi.URLParam(r, "driveID") != "" {
		driveID, err := parseIDParam(r, "driveID")
		if err != nil {
			errorcode.RenderError(w, r, err)
			return nil, false
		}
		if driveID.GetStorageId() != itemID.GetStorageId() || driveID.GetSpaceId() != itemID.GetSpaceId() {
			errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "item not found")
			return nil, false
		}
	}
	return &itemID, true
}

// renderTrashStatus maps a mutating trash call's status; true means OK. Unlike the listing,
// PERMISSION_DENIED is a 403 here: the caller could already see the item.
func renderTrashStatus(w http.ResponseWriter, r *http.Request, st *cs3rpc.Status) bool {
	switch st.GetCode() {
	case cs3rpc.Code_CODE_OK:
		return true
	case cs3rpc.Code_CODE_NOT_FOUND:
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, st.GetMessage())
	case cs3rpc.Code_CODE_PERMISSION_DENIED:
		errorcode.AccessDenied.Render(w, r, http.StatusForbidden, st.GetMessage())
	case cs3rpc.Code_CODE_UNAUTHENTICATED:
		errorcode.Unauthenticated.Render(w, r, http.StatusUnauthorized, st.GetMessage())
	case cs3rpc.Code_CODE_ALREADY_EXISTS:
		errorcode.NameAlreadyExists.Render(w, r, http.StatusConflict, st.GetMessage())
	case cs3rpc.Code_CODE_LOCKED, cs3rpc.Code_CODE_FAILED_PRECONDITION:
		errorcode.ItemIsLocked.Render(w, r, http.StatusLocked, st.GetMessage())
	default:
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, st.GetMessage())
	}
	return false
}
