package connector

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	gatewayv1beta1 "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/config"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/connector/utf7"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/locks"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/rs/zerolog"
	"go-micro.dev/v4/selector"
	microstore "go-micro.dev/v4/store"
)

const (
	HeaderWopiLock              string = "X-WOPI-Lock"
	HeaderWopiOldLock           string = "X-WOPI-OldLock"
	HeaderWopiLockFailureReason string = "X-WOPI-LockFailureReason"
	HeaderWopiST                string = "X-WOPI-SuggestedTarget"
	HeaderWopiRT                string = "X-WOPI-RelativeTarget"
	HeaderWopiOverwriteRT       string = "X-WOPI-OverwriteRelativeTarget"
	HeaderWopiSize              string = "X-WOPI-Size"
	HeaderWopiValidRT           string = "X-WOPI-ValidRelativeTarget"
	HeaderWopiRequestedName     string = "X-WOPI-RequestedName"
	HeaderContentLength         string = "Content-Length"
	HeaderContentType           string = "Content-Type"
	HeaderWopiVersion           string = "X-WOPI-ItemVersion"
)

// HttpAdapter will adapt the responses from the connector to HTTP.
//
// The adapter will use the request's context for the connector operations,
// this means that the request MUST have a valid WOPI context and a
// pre-configured logger. This should have been prepared in the routing.
//
// All operations are expected to follow the definitions found in
// https://learn.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/endpoints
type HttpAdapter struct {
	con   ConnectorService
	locks locks.LockParser
}

// NewHttpAdapter will create a new HTTP adapter. A new connector using the
// provided gateway API client and configuration will be used in the adapter
func NewHttpAdapter(gws pool.Selectable[gatewayv1beta1.GatewayAPIClient], cfg *config.Config, st microstore.Store, graphSelector selector.Selector) *HttpAdapter {
	httpAdapter := &HttpAdapter{
		con: NewConnector(
			NewFileConnector(gws, cfg, st, graphSelector),
			NewContentConnector(gws, cfg),
		),
	}

	// TODO: check if we can get rid of custom log parsing completely
	httpAdapter.locks = &locks.NoopLockParser{}
	return httpAdapter
}

// NewHttpAdapterWithConnector will create a new HTTP adapter that will use
// the provided connector service
func NewHttpAdapterWithConnector(con ConnectorService, l locks.LockParser) *HttpAdapter {
	return &HttpAdapter{
		con:   con,
		locks: l,
	}
}

// GetLock adapts the "GetLock" operation for WOPI.
// Only the request's context is needed in order to extract the WOPI context.
// The operation's response will be sent through the response writer and
// the headers according to the spec
func (h *HttpAdapter) GetLock(w http.ResponseWriter, r *http.Request) {
	fileCon := h.con.GetFileConnector()
	response, err := fileCon.GetLock(r.Context())

	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h.writeConnectorResponse(w, r, response)
}

// Lock adapts the "Lock" and "UnlockAndRelock" operations for WOPI.
// The request's context is needed in order to extract the WOPI context. In
// addition, the "X-WOPI-Lock" and "X-WOPI-OldLock" headers might be needed"
// (check spec)
// The operation's response will be sent through the response writer and
// the headers according to the spec
func (h *HttpAdapter) Lock(w http.ResponseWriter, r *http.Request) {
	oldLockID := h.locks.ParseLock(r.Header.Get(HeaderWopiOldLock))
	lockID := h.locks.ParseLock(r.Header.Get(HeaderWopiLock))

	fileCon := h.con.GetFileConnector()
	response, err := fileCon.Lock(r.Context(), lockID, oldLockID)

	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h.writeConnectorResponse(w, r, response)
}

// RefreshLock adapts the "RefreshLock" operation for WOPI
// The request's context is needed in order to extract the WOPI context. In
// addition, the "X-WOPI-Lock" header is needed (check spec).
// The lock will be refreshed to last another 30 minutes. The value is
// hardcoded
// The operation's response will be sent through the response writer and
// the headers according to the spec
func (h *HttpAdapter) RefreshLock(w http.ResponseWriter, r *http.Request) {
	lockID := h.locks.ParseLock(r.Header.Get(HeaderWopiLock))

	fileCon := h.con.GetFileConnector()
	response, err := fileCon.RefreshLock(r.Context(), lockID)

	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h.writeConnectorResponse(w, r, response)
}

// UnLock adapts the "Unlock" operation for WOPI
// The request's context is needed in order to extract the WOPI context. In
// addition, the "X-WOPI-Lock" header is needed (check spec).
// The operation's response will be sent through the response writer and
// the headers according to the spec
func (h *HttpAdapter) UnLock(w http.ResponseWriter, r *http.Request) {
	lockID := h.locks.ParseLock(r.Header.Get(HeaderWopiLock))

	fileCon := h.con.GetFileConnector()
	response, err := fileCon.UnLock(r.Context(), lockID)

	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h.writeConnectorResponse(w, r, response)
}

// CheckFileInfo will retrieve the information of the file in json format
// Only the request's context is needed in order to extract the WOPI context.
// The operation's response will be sent through the response writer and
// the headers according to the spec
func (h *HttpAdapter) CheckFileInfo(w http.ResponseWriter, r *http.Request) {
	fileCon := h.con.GetFileConnector()
	response, err := fileCon.CheckFileInfo(r.Context())

	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h.writeConnectorResponse(w, r, response)
}

// GetFile will download the file
// Only the request's context is needed in order to extract the WOPI context.
// The file's content will be written in the response writer
func (h *HttpAdapter) GetFile(w http.ResponseWriter, r *http.Request) {
	contentCon := h.con.GetContentConnector()
	err := contentCon.GetFile(r.Context(), w)
	if err != nil {
		var conError *ConnectorError
		if errors.As(err, &conError) {
			http.Error(w, http.StatusText(conError.HttpCodeOut), conError.HttpCodeOut)
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
	// status might have been already sent if the file is big enough, but for
	// small files, the content might still be buffered.
	w.WriteHeader(http.StatusOK)
}

// PutFile will upload the file
// The request's context and its body are needed (content length is also
// needed)
// The operation's response will be sent through the response writer and
// the headers according to the spec
func (h *HttpAdapter) PutFile(w http.ResponseWriter, r *http.Request) {
	lockID := h.locks.ParseLock(r.Header.Get(HeaderWopiLock))

	contentCon := h.con.GetContentConnector()
	response, err := contentCon.PutFile(r.Context(), r.Body, r.ContentLength, lockID)

	if err != nil {
		var connErr *ConnectorError
		if errors.As(err, &connErr) && connErr.HttpCodeOut != 0 {
			response = NewResponse(connErr.HttpCodeOut)
		} else {
			response = NewResponse(http.StatusInternalServerError)
		}
	}

	h.writeConnectorResponse(w, r, response)
}

// PutRelativeFile will upload the file with a specific name. The name might be
// automatically adjusted depending on the request headers.
// Note that this method will also send a json body in the response.
// It has 2 mutually exclusive operation methods that are used based on the
// provided headers in the request.
// Note that this method won't used locks (not documented).
//
// The file name must be encoded in utf7. This method will decode the utf7 name
// into utf8. The utf8 (not utf7) name must have less than 512 bytes, otherwise
// the request will fail.
func (h *HttpAdapter) PutRelativeFile(w http.ResponseWriter, r *http.Request) {
	relativeTarget := r.Header.Get(HeaderWopiRT)
	suggestedTarget := r.Header.Get(HeaderWopiST)

	if relativeTarget != "" && suggestedTarget != "" {
		// headers are mutually exclusive
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var response *ConnectorResponse
	var putErr error
	fileCon := h.con.GetFileConnector()

	if suggestedTarget != "" {
		utf8Target, decErr := utf7.DecodeString(suggestedTarget)
		if decErr != nil || len(utf8Target) > 512 {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		response, putErr = fileCon.PutRelativeFileSuggested(r.Context(), h.con.GetContentConnector(), r.Body, r.ContentLength, utf8Target)
	}

	if relativeTarget != "" {
		utf8Target, decErr := utf7.DecodeString(relativeTarget)
		if decErr != nil || len(utf8Target) > 512 {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		response, putErr = fileCon.PutRelativeFileRelative(r.Context(), h.con.GetContentConnector(), r.Body, r.ContentLength, utf8Target)
	}

	if putErr != nil {
		var connErr *ConnectorError
		if errors.As(putErr, &connErr) && connErr.HttpCodeOut != 0 {
			response = NewResponse(connErr.HttpCodeOut)
		} else {
			response = NewResponse(http.StatusInternalServerError)
		}
	}

	h.writeConnectorResponse(w, r, response)
}

// DeleteFile will delete the provided file. If the file is locked and can't
// be deleted, a 409 conflict error will be returned with its corresponding
// lock.
func (h *HttpAdapter) DeleteFile(w http.ResponseWriter, r *http.Request) {
	lockID := r.Header.Get(HeaderWopiLock)

	fileCon := h.con.GetFileConnector()
	response, err := fileCon.DeleteFile(r.Context(), lockID)

	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h.writeConnectorResponse(w, r, response)
}

// RenameFile will rename the file. The name might be automatically adjusted.
// Note that this method will also send a json body in the response. The
// adjusted file name will be returned in the body.
//
// The file name must be encoded in utf7. This method will decode the utf7 name
// into utf8. The utf8 (not utf7) name must have less than 495 bytes, otherwise
// the request will fail.
func (h *HttpAdapter) RenameFile(w http.ResponseWriter, r *http.Request) {
	lockID := r.Header.Get(HeaderWopiLock)
	requestedName := r.Header.Get(HeaderWopiRequestedName)

	utf8Target, decErr := utf7.DecodeString(requestedName)
	if decErr != nil || len(utf8Target) > 495 { // need space for the possible prefix and the extension
		w.Header().Set("X-WOPI-InvalidFileNameError", "Filename too long")
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	fileCon := h.con.GetFileConnector()
	response, err := fileCon.RenameFile(r.Context(), lockID, utf8Target)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h.writeConnectorResponse(w, r, response)
}

// GetAvatar proxies the user's avatar from the Graph API.
// The WOPI token in the query string provides authentication (validated by
// the WopiContextAuthMiddleware). Collabora loads avatars via img.src which
// is a plain browser GET — it cannot send auth headers.
func (h *HttpAdapter) GetAvatar(w http.ResponseWriter, r *http.Request) {
	logger := zerolog.Ctx(r.Context())
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	fileCon := h.con.GetFileConnector()
	response, err := fileCon.GetAvatar(r.Context(), userID)
	if err != nil {
		var connErr *ConnectorError
		if errors.As(err, &connErr) {
			http.Error(w, http.StatusText(connErr.HttpCodeOut), connErr.HttpCodeOut)
		} else {
			logger.Error().Err(err).Msg("GetAvatar: failed to fetch avatar")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}
	h.writeConnectorAvatarResponse(w, r, response)
}

func (h *HttpAdapter) writeConnectorAvatarResponse(w http.ResponseWriter, r *http.Request, response *ConnectorResponse) {
	data, _ := response.Body.([]byte)
	for key, value := range response.Headers {
		w.Header().Set(key, value)
	}
	w.WriteHeader(response.Status)
	written, err := w.Write(data)
	if err != nil {
		logger := zerolog.Ctx(r.Context())
		logger.Error().
			Err(err).
			Int("TotalBytes", len(data)).
			Int("WrittenBytes", written).
			Msg("failed to write avatar contents in the HTTP response")
	}
}

func (h *HttpAdapter) writeConnectorResponse(w http.ResponseWriter, r *http.Request, response *ConnectorResponse) {
	jsonBody := []byte{}
	if response.Body != nil {
		var err error
		jsonBody, err = json.Marshal(response.Body)
		if err != nil {
			logger := zerolog.Ctx(r.Context())
			logger.Error().Err(err).Msg("failed to marshal response")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set(HeaderContentType, "application/json")
		w.Header().Set(HeaderContentLength, strconv.Itoa(len(jsonBody)))
	}

	for key, value := range response.Headers {
		w.Header().Set(key, value)
	}
	w.WriteHeader(response.Status)

	bytes, err := w.Write(jsonBody)
	if err != nil {
		logger := zerolog.Ctx(r.Context())
		logger.Error().
			Err(err).
			Int("TotalBytes", len(jsonBody)).
			Int("WrittenBytes", bytes).
			Msg("failed to write contents in the HTTP response")
	}
}
