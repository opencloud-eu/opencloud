package svc

import (
	"net/http"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
)

func (g Graph) FreezeItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	itemID, err := parseIDParam(r, "itemID")
	if err != nil {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid itemID")
		return
	}
	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusServiceUnavailable, "gateway not available")
		return
	}
	ref := &provider.Reference{ResourceId: &itemID}
	statRes, err := gatewayClient.Stat(ctx, &provider.StatRequest{Ref: ref})
	if err != nil {
		g.logger.Error().Err(err).Msg("freeze: stat error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not stat resource: "+err.Error())
		return
	}
	if statRes.GetStatus().GetCode() != rpc.Code_CODE_OK {
		g.logger.Error().Str("msg", statRes.GetStatus().GetMessage()).Msg("freeze: stat non-OK")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "stat: "+statRes.GetStatus().GetMessage())
		return
	}
	if statRes.GetInfo().GetType() == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "cannot freeze a directory, use protect instead")
		return
	}
	res, err := gatewayClient.SetImmutable(ctx, &provider.SetImmutableRequest{Ref: ref})
	if err != nil {
		g.logger.Error().Err(err).Msg("freeze: SetImmutable error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not freeze item: "+err.Error())
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

func (g Graph) ProtectItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	itemID, err := parseIDParam(r, "itemID")
	if err != nil {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid itemID")
		return
	}
	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusServiceUnavailable, "gateway not available")
		return
	}
	ref := &provider.Reference{ResourceId: &itemID}
	statRes, err := gatewayClient.Stat(ctx, &provider.StatRequest{Ref: ref})
	if err != nil {
		g.logger.Error().Err(err).Msg("protect: stat error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not stat resource: "+err.Error())
		return
	}
	if statRes.GetStatus().GetCode() != rpc.Code_CODE_OK {
		g.logger.Error().Str("msg", statRes.GetStatus().GetMessage()).Msg("protect: stat non-OK")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "stat: "+statRes.GetStatus().GetMessage())
		return
	}
	if statRes.GetInfo().GetType() != provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "only directories can be protected")
		return
	}
	res, err := gatewayClient.SetImmutable(ctx, &provider.SetImmutableRequest{Ref: ref})
	if err != nil {
		g.logger.Error().Err(err).Msg("protect: SetImmutable error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not protect item: "+err.Error())
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

func (g Graph) UnprotectItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	itemID, err := parseIDParam(r, "itemID")
	if err != nil {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid itemID")
		return
	}
	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusServiceUnavailable, "gateway not available")
		return
	}
	ref := &provider.Reference{ResourceId: &itemID}
	statRes, err := gatewayClient.Stat(ctx, &provider.StatRequest{Ref: ref})
	if err != nil {
		g.logger.Error().Err(err).Msg("unprotect: stat error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not stat resource: "+err.Error())
		return
	}
	if statRes.GetStatus().GetCode() != rpc.Code_CODE_OK {
		g.logger.Error().Str("msg", statRes.GetStatus().GetMessage()).Msg("unprotect: stat non-OK")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "stat: "+statRes.GetStatus().GetMessage())
		return
	}
	if statRes.GetInfo().GetType() != provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "only directories can be unprotected")
		return
	}
	res, err := gatewayClient.UnsetImmutable(ctx, &provider.UnsetImmutableRequest{Ref: ref})
	if err != nil {
		g.logger.Error().Err(err).Msg("unprotect: UnsetImmutable error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not unprotect item: "+err.Error())
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
