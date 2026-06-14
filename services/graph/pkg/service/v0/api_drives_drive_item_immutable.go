package svc

import (
	"net/http"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
)

// FreezeItem sets the immutable attribute on a file (irreversible).
// The client should confirm this action before calling, since freezing
// a file cannot be undone.
func (api DrivesDriveItemApi) FreezeItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, itemID, err := GetDriveAndItemIDParam(r, &api.logger)
	if err != nil {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid driveID or itemID")
		return
	}

	gatewayClient, err := api.baseGraphService.gatewaySelector.Next()
	if err != nil {
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusServiceUnavailable, "gateway not available")
		return
	}

	ref := &provider.Reference{ResourceId: &itemID}

	statRes, err := gatewayClient.Stat(ctx, &provider.StatRequest{Ref: ref})
	if err != nil || statRes.GetStatus().GetCode() != rpc.Code_CODE_OK {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not stat resource")
		return
	}
	if statRes.GetInfo().GetType() == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "cannot freeze a directory, use protect instead")
		return
	}

	res, err := gatewayClient.SetImmutable(ctx, &provider.SetImmutableRequest{Ref: ref})
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not freeze item")
		return
	}
	switch res.GetStatus().GetCode() {
	case rpc.Code_CODE_OK:
		w.WriteHeader(http.StatusNoContent)
	case rpc.Code_CODE_NOT_FOUND:
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "resource not found")
	case rpc.Code_CODE_PERMISSION_DENIED:
		errorcode.AccessDenied.Render(w, r, http.StatusForbidden, "permission denied")
	default:
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, res.GetStatus().GetMessage())
	}
}

// ProtectItem sets the immutable attribute on a directory (reversible by managers).
func (api DrivesDriveItemApi) ProtectItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, itemID, err := GetDriveAndItemIDParam(r, &api.logger)
	if err != nil {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid driveID or itemID")
		return
	}

	gatewayClient, err := api.baseGraphService.gatewaySelector.Next()
	if err != nil {
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusServiceUnavailable, "gateway not available")
		return
	}

	ref := &provider.Reference{ResourceId: &itemID}

	statRes, err := gatewayClient.Stat(ctx, &provider.StatRequest{Ref: ref})
	if err != nil || statRes.GetStatus().GetCode() != rpc.Code_CODE_OK {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not stat resource")
		return
	}
	if statRes.GetInfo().GetType() != provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "only directories can be protected, use freeze for files")
		return
	}

	res, err := gatewayClient.SetImmutable(ctx, &provider.SetImmutableRequest{Ref: ref})
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not protect item")
		return
	}
	switch res.GetStatus().GetCode() {
	case rpc.Code_CODE_OK:
		w.WriteHeader(http.StatusNoContent)
	case rpc.Code_CODE_NOT_FOUND:
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "resource not found")
	case rpc.Code_CODE_PERMISSION_DENIED:
		errorcode.AccessDenied.Render(w, r, http.StatusForbidden, "permission denied")
	default:
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, res.GetStatus().GetMessage())
	}
}

// UnprotectItem removes the immutable attribute from a directory.
func (api DrivesDriveItemApi) UnprotectItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, itemID, err := GetDriveAndItemIDParam(r, &api.logger)
	if err != nil {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid driveID or itemID")
		return
	}

	gatewayClient, err := api.baseGraphService.gatewaySelector.Next()
	if err != nil {
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusServiceUnavailable, "gateway not available")
		return
	}

	ref := &provider.Reference{ResourceId: &itemID}

	statRes, err := gatewayClient.Stat(ctx, &provider.StatRequest{Ref: ref})
	if err != nil || statRes.GetStatus().GetCode() != rpc.Code_CODE_OK {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not stat resource")
		return
	}
	if statRes.GetInfo().GetType() != provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "only directories can be unprotected")
		return
	}

	res, err := gatewayClient.UnsetImmutable(ctx, &provider.UnsetImmutableRequest{Ref: ref})
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not unprotect item")
		return
	}
	switch res.GetStatus().GetCode() {
	case rpc.Code_CODE_OK:
		w.WriteHeader(http.StatusNoContent)
	case rpc.Code_CODE_NOT_FOUND:
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "resource not found")
	case rpc.Code_CODE_PERMISSION_DENIED:
		errorcode.AccessDenied.Render(w, r, http.StatusForbidden, "permission denied")
	default:
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, res.GetStatus().GetMessage())
	}
}
