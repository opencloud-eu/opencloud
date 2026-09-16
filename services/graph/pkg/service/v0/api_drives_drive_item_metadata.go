package svc

import (
	"encoding/json"
	"io"
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

// SetItemMetadata sets custom metadata (user.oc.md.*) on a drive item.
// Accepts a JSON object with key-value pairs to set.
//
// PUT /drives/{driveID}/items/{itemID}/metadata
//
// Request body: { "note": "some text", ... }
func (g Graph) SetItemMetadata(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	itemID, err := parseIDParam(r, "itemID")
	if err != nil {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid itemID")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "could not read request body")
		return
	}
	defer r.Body.Close()

	var metadata map[string]string
	if err := json.Unmarshal(body, &metadata); err != nil {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if len(metadata) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusServiceUnavailable, "gateway not available")
		return
	}

	ref := &provider.Reference{ResourceId: &itemID}
	res, err := gatewayClient.SetArbitraryMetadata(ctx, &provider.SetArbitraryMetadataRequest{
		Ref: ref,
		ArbitraryMetadata: &provider.ArbitraryMetadata{
			Metadata: metadata,
		},
	})
	if err != nil {
		g.logger.Error().Err(err).Msg("metadata: SetArbitraryMetadata error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not set metadata")
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
