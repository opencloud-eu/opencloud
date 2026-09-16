package svc

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/reva/v2/pkg/openextension"

	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
)

const _expandOpenExtensions = "extensions"

const maxOpenExtensionBody = 2 * openextension.MaxProperties * openextension.MaxValueSize

type openExtensionCollection struct {
	Value []openextension.OpenExtension `json:"value"`
}

// ListOpenExtensions lists the open extensions of a driveItem.
func (g Graph) ListOpenExtensions(w http.ResponseWriter, r *http.Request) {
	info, ok := g.statOpenExtensionItem(w, r)
	if !ok {
		return
	}
	exts := openextension.FromMetadata(info.GetArbitraryMetadata().GetMetadata())
	if exts == nil {
		exts = []openextension.OpenExtension{}
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, openExtensionCollection{Value: exts})
}

// GetOpenExtension returns one open extension of a driveItem.
func (g Graph) GetOpenExtension(w http.ResponseWriter, r *http.Request) {
	name, ok := openExtensionNameParam(w, r)
	if !ok {
		return
	}
	info, ok := g.statOpenExtensionItem(w, r)
	if !ok {
		return
	}
	ext, found := openextension.Lookup(info.GetArbitraryMetadata().GetMetadata(), name)
	if !found {
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "extension not found")
		return
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, ext)
}

// UpsertOpenExtension creates an open extension or merges into an existing one:
// members set, null removes, omitted stay.
func (g Graph) UpsertOpenExtension(w http.ResponseWriter, r *http.Request) {
	name, ok := openExtensionNameParam(w, r)
	if !ok {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxOpenExtensionBody+1))
	if err != nil {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "could not read the request body")
		return
	}
	if len(body) > maxOpenExtensionBody {
		errorcode.InvalidRequest.Render(w, r, http.StatusRequestEntityTooLarge, "the request body is too large")
		return
	}
	patch, err := openextension.Parse(body)
	if err != nil {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}

	info, ok := g.statOpenExtensionItem(w, r)
	if !ok {
		return
	}
	if !canWriteOpenExtensions(info) {
		errorcode.AccessDenied.Render(w, r, http.StatusForbidden, "no permission to write extensions")
		return
	}

	metadata := info.GetArbitraryMetadata().GetMetadata()
	ext, existed := openextension.Lookup(metadata, name)
	ext.Apply(patch)
	if len(ext.Values) > openextension.MaxProperties {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest,
			fmt.Sprintf("extension %s would have %d properties, at most %d are allowed", name, len(ext.Values), openextension.MaxProperties))
		return
	}
	set, unset := patch.Metadata(name)
	unset = storedOnly(unset, metadata)

	client, err := g.gatewaySelector.Next()
	if err != nil {
		g.logger.Error().Err(err).Msg("error selecting next gateway client")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	ref := &provider.Reference{ResourceId: info.GetId()}
	if len(set) > 0 {
		res, err := client.SetArbitraryMetadata(r.Context(), &provider.SetArbitraryMetadataRequest{
			Ref:               ref,
			ArbitraryMetadata: &provider.ArbitraryMetadata{Metadata: set},
		})
		if !g.renderMetadataStatus(w, r, res.GetStatus(), err) {
			return
		}
	}
	if len(unset) > 0 {
		res, err := client.UnsetArbitraryMetadata(r.Context(), &provider.UnsetArbitraryMetadataRequest{Ref: ref, ArbitraryMetadataKeys: unset})
		if !g.renderMetadataStatus(w, r, res.GetStatus(), err) {
			return
		}
	}

	if len(ext.Values) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	status := http.StatusOK
	if !existed {
		status = http.StatusCreated
	}
	render.Status(r, status)
	render.JSON(w, r, ext)
}

// DeleteOpenExtension removes an open extension from a driveItem.
func (g Graph) DeleteOpenExtension(w http.ResponseWriter, r *http.Request) {
	name, ok := openExtensionNameParam(w, r)
	if !ok {
		return
	}
	info, ok := g.statOpenExtensionItem(w, r)
	if !ok {
		return
	}
	if !canWriteOpenExtensions(info) {
		errorcode.AccessDenied.Render(w, r, http.StatusForbidden, "no permission to write extensions")
		return
	}
	keys := openextension.MetadataKeys(info.GetArbitraryMetadata().GetMetadata(), name)
	if len(keys) == 0 {
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "extension not found")
		return
	}

	client, err := g.gatewaySelector.Next()
	if err != nil {
		g.logger.Error().Err(err).Msg("error selecting next gateway client")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	res, err := client.UnsetArbitraryMetadata(r.Context(), &provider.UnsetArbitraryMetadataRequest{
		Ref:                   &provider.Reference{ResourceId: info.GetId()},
		ArbitraryMetadataKeys: keys,
	})
	if !g.renderMetadataStatus(w, r, res.GetStatus(), err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func openExtensionNameParam(w http.ResponseWriter, r *http.Request) (string, bool) {
	name, err := url.PathUnescape(chi.URLParam(r, "extensionName"))
	if err != nil {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid extension name")
		return "", false
	}
	if err := openextension.ValidateName(name); err != nil {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return "", false
	}
	return name, true
}

// statOpenExtensionItem stats the item of the request; false means the error was rendered.
func (g Graph) statOpenExtensionItem(w http.ResponseWriter, r *http.Request) (*provider.ResourceInfo, bool) {
	_, itemID, err := GetDriveAndItemIDParam(r, g.logger)
	if err != nil {
		errorcode.RenderError(w, r, err)
		return nil, false
	}
	client, err := g.gatewaySelector.Next()
	if err != nil {
		g.logger.Error().Err(err).Msg("error selecting next gateway client")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return nil, false
	}
	res, err := client.Stat(r.Context(), &provider.StatRequest{Ref: &provider.Reference{ResourceId: itemID}})
	switch {
	case err != nil:
		g.logger.Error().Err(err).Msg("error statting the item")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return nil, false
	case res.GetStatus().GetCode() == rpc.Code_CODE_OK:
		return res.GetInfo(), true
	case res.GetStatus().GetCode() == rpc.Code_CODE_NOT_FOUND, res.GetStatus().GetCode() == rpc.Code_CODE_PERMISSION_DENIED:
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, res.GetStatus().GetMessage())
		return nil, false
	case res.GetStatus().GetCode() == rpc.Code_CODE_UNAUTHENTICATED:
		errorcode.Unauthenticated.Render(w, r, http.StatusUnauthorized, res.GetStatus().GetMessage())
		return nil, false
	default:
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, res.GetStatus().GetMessage())
		return nil, false
	}
}

// renderMetadataStatus renders a failed write and reports whether it succeeded.
func (g Graph) renderMetadataStatus(w http.ResponseWriter, r *http.Request, status *rpc.Status, err error) bool {
	switch {
	case err != nil:
		g.logger.Error().Err(err).Msg("error writing arbitrary metadata")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return false
	case status.GetCode() == rpc.Code_CODE_OK:
		return true
	case status.GetCode() == rpc.Code_CODE_LOCKED:
		errorcode.ItemIsLocked.Render(w, r, http.StatusLocked, "the item is locked")
		return false
	case status.GetCode() == rpc.Code_CODE_PERMISSION_DENIED:
		errorcode.AccessDenied.Render(w, r, http.StatusForbidden, status.GetMessage())
		return false
	case status.GetCode() == rpc.Code_CODE_NOT_FOUND:
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, status.GetMessage())
		return false
	default:
		g.logger.Error().Interface("status", status).Msg("error writing arbitrary metadata")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, status.GetMessage())
		return false
	}
}

// canWriteOpenExtensions is the tag rule: writing metadata needs write access.
func canWriteOpenExtensions(info *provider.ResourceInfo) bool {
	pm := info.GetPermissionSet()
	return pm != nil && (pm.GetInitiateFileUpload() || pm.GetCreateContainer())
}

func storedOnly(keys []string, metadata map[string]string) []string {
	out := keys[:0]
	for _, key := range keys {
		if _, ok := metadata[key]; ok {
			out = append(out, key)
		}
	}
	return out
}

// driveItemWithOpenExtensions adds the extensions relation; the generated model
// has no field for it yet.
func driveItemWithOpenExtensions(item *libregraph.DriveItem, info *provider.ResourceInfo) (map[string]any, error) {
	m, err := item.ToMap()
	if err != nil {
		return nil, err
	}
	exts := openextension.FromMetadata(info.GetArbitraryMetadata().GetMetadata())
	if exts == nil {
		exts = []openextension.OpenExtension{}
	}
	m[_expandOpenExtensions] = exts
	return m, nil
}
