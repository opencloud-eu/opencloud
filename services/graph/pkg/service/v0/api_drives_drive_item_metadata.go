package svc

import (
	"encoding/json"
	"net/http"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
)

// GetItemMetadata returns all custom metadata (user.oc.md.*) for a drive item
// as a JSON object. Read-only endpoint.
//
// GET /drives/{driveID}/items/{itemID}/metadata
//
// Response: { "oy.subject": "...", "oy.created": "...", ... }
func (g Graph) GetItemMetadata(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	itemID, err := parseIDParam(r, "itemID")
	if err != nil {
		g.logger.Debug().Err(err).Msg("could not parse itemID")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid itemID")
		return
	}

	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusServiceUnavailable, "gateway not available")
		return
	}

	ref := &provider.Reference{ResourceId: &itemID}

	statRes, err := gatewayClient.Stat(ctx, &provider.StatRequest{
		Ref:                   ref,
		ArbitraryMetadataKeys: []string{"*"},
	})
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not stat resource")
		return
	}
	switch statRes.GetStatus().GetCode() {
	case rpc.Code_CODE_OK:
		// continue
	case rpc.Code_CODE_NOT_FOUND:
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "resource not found")
		return
	case rpc.Code_CODE_PERMISSION_DENIED:
		errorcode.AccessDenied.Render(w, r, http.StatusForbidden, "permission denied")
		return
	default:
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, statRes.GetStatus().GetMessage())
		return
	}

	metadata := make(map[string]string)
	if am := statRes.GetInfo().GetArbitraryMetadata(); am != nil {
		for k, v := range am.GetMetadata() {
			metadata[k] = v
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		g.logger.Error().Err(err).Msg("could not encode metadata")
	}
}
