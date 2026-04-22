package svc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	ehmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/eventhistory/v0"
	ehsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/eventhistory/v0"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

var (
	_backupFileName = "personal_data_export.json"

	// TokenTransportHeader holds the header key for the reva transfer token
	TokenTransportHeader = "X-Reva-Transfer"
)

// Marshaller is the common interface for a marshaller
type Marshaller func(any) ([]byte, error)

// ExportPersonalDataRequest is the body of the request
type ExportPersonalDataRequest struct {
	StorageLocation string `json:"storageLocation"`
}

func (g Graph) getPersonalSpace(ctx context.Context, u *user.UserId) (*provider.StorageSpace, error) {

	// we cannot assume the space id is the same as the user id, so we have to look up his personal space first
	lReq := &storageprovider.ListStorageSpacesRequest{
		Filters: []*storageprovider.ListStorageSpacesRequest_Filter{
			{
				Type: storageprovider.ListStorageSpacesRequest_Filter_TYPE_OWNER,
				Term: &storageprovider.ListStorageSpacesRequest_Filter_Owner{
					Owner: u,
				},
			},
			{
				Type: storageprovider.ListStorageSpacesRequest_Filter_TYPE_SPACE_TYPE,
				Term: &storageprovider.ListStorageSpacesRequest_Filter_SpaceType{
					SpaceType: "personal",
				},
			},
		},
	}

	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	res, err := gatewayClient.ListStorageSpaces(ctx, lReq)
	if err != nil {
		return nil, err
	}
	if res.Status.Code != rpc.Code_CODE_OK {
		return nil, errorcode.FromCS3Status(res.Status, fmt.Errorf("could not list storage spaces"))
	}
	if len(res.GetStorageSpaces()) == 0 {
		return nil, fmt.Errorf("could not find personal space for user %s", u.GetOpaqueId())
	}

	return res.GetStorageSpaces()[0], nil
}

// ExportPersonalData exports all personal data the system holds
func (g Graph) ExportPersonalData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	u := revactx.ContextMustGetUser(ctx)
	if reqUserID := chi.URLParam(r, "userID"); reqUserID != u.GetId().GetOpaqueId() {
		g.logger.Info().Msg("uid mismatch")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("personal data export for other users are not permitted"))
		return
	}
	// Get location from request
	loc := getLocation(r)

	// prepare marshaller
	var marsh Marshaller
	switch filepath.Ext(loc) {
	default:
		g.logger.Info().Str("path", loc).Msg("invalid location")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("only json format is supported for personal data export"))
		return
	case ".json":
		marsh = json.Marshal
	}

	personalSpace, err := g.getPersonalSpace(ctx, u.GetId())
	if err != nil {
		g.logger.Error().Err(err).Msg("could not get personal space")
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusInternalServerError, "could not get personal space, aborting")
		return
	}

	ref := &provider.Reference{
		ResourceId: personalSpace.GetRoot(),
		Path:       loc,
	}

	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		g.logger.Error().Err(err).Msg("could not select next gateway client")
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusInternalServerError, "could not select next gateway client, aborting")
		return
	}

	// touch file
	if err := mustTouchFile(ctx, ref, gatewayClient); err != nil {
		g.logger.Error().Err(err).Msg("error touching file")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// go start gathering
	go g.GatherPersonalData(u, ref, r.Header.Get(revactx.TokenHeader), marsh)

	w.WriteHeader(http.StatusAccepted)
}

// GatherPersonalData will all gather all personal data of the user and save it to a file in the users personal space
func (g Graph) GatherPersonalData(usr *user.User, ref *provider.Reference, token string, marsh Marshaller) {
	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		g.logger.Error().Err(err).Msg("could not select next gateway client")
		return
	}

	// the context might already be cancelled. We need to impersonate the acting user again
	ctx, err := utils.GetServiceUserContext(g.config.ServiceAccount.ServiceAccountID, gatewayClient, g.config.ServiceAccount.ServiceAccountSecret)
	if err != nil {
		g.logger.Error().Err(err).Str("userID", usr.GetId().GetOpaqueId()).Msg("cannot impersonate user")
	}
	// create data
	data := make(map[string]any)

	// reva user
	data["user"] = usr

	// Check if we have a keycloak client, and if so, get the keycloak export.
	if ctx != nil && g.keycloakClient != nil {
		kcd, err := g.keycloakClient.GetPIIReport(ctx, g.config.Keycloak.UserRealm, usr.GetUsername())
		if err != nil {
			g.logger.Error().Err(err).Str("userID", usr.GetId().GetOpaqueId()).Msg("cannot get keycloak personal data")
		}
		data["keycloak"] = kcd
	}

	// get event data
	if ctx != nil && g.historyClient != nil {
		resp, err := g.historyClient.GetEventsForUser(ctx, &ehsvc.GetEventsForUserRequest{UserID: usr.GetId().GetOpaqueId()})
		if err != nil {
			g.logger.Error().Err(err).Str("userID", usr.GetId().GetOpaqueId()).Msg("cannot get event personal data")
		}
		data["events"] = convertEvents(resp.GetEvents())
	}

	// marshal
	by, err := marsh(data)
	if err != nil {
		g.logger.Error().Err(err).Msg("cannot marshal personal user data")
		return
	}

	// upload
	var errmsg string
	if err := g.upload(usr, by, ref, token); err != nil {
		g.logger.Error().Err(err).Msg("failed uploading personal data export")
		errmsg = err.Error()
	}

	if err := events.Publish(ctx, g.eventsPublisher, events.PersonalDataExtracted{
		Executant: usr.GetId(),
		Timestamp: utils.TSNow(),
		ErrorMsg:  errmsg,
	}); err != nil {
		g.logger.Error().Err(err).Msg("cannot publish event")
	}
}

func (g Graph) upload(u *user.User, data []byte, ref *provider.Reference, th string) error {
	uReq := &provider.InitiateFileUploadRequest{
		Ref:    ref,
		Opaque: utils.AppendPlainToOpaque(nil, "Upload-Length", strconv.FormatUint(uint64(len(data)), 10)),
	}

	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		g.logger.Error().Err(err).Msg("could not select next gateway client")
		return err
	}

	ctx, err := utils.GetServiceUserContext(g.config.ServiceAccount.ServiceAccountID, gatewayClient, g.config.ServiceAccount.ServiceAccountSecret)
	if err != nil {
		return err
	}
	client, err := g.gatewaySelector.Next()
	if err != nil {
		return err
	}
	ctx = revactx.ContextSetToken(ctx, th)
	uRes, err := client.InitiateFileUpload(ctx, uReq)
	if err != nil {
		return err
	}

	if uRes.Status.Code != rpc.Code_CODE_OK {
		return fmt.Errorf("wrong status code while initiating upload: %s", uRes.GetStatus().GetMessage())
	}

	var uploadEP, uploadToken string
	for _, p := range uRes.Protocols {
		if p.Protocol == "simple" {
			uploadEP, uploadToken = p.UploadEndpoint, p.Token
		}
	}

	httpUploadReq, err := rhttp.NewRequest(ctx, "PUT", uploadEP, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	httpUploadReq.Header.Set(TokenTransportHeader, uploadToken)

	httpUploadRes, err := rhttp.GetHTTPClient(rhttp.Insecure(true)).Do(httpUploadReq)
	if err != nil {
		return err
	}
	defer httpUploadRes.Body.Close()
	if httpUploadRes.StatusCode != http.StatusOK {
		return fmt.Errorf("wrong status uploading file: %d", httpUploadRes.StatusCode)
	}

	return nil
}

// touches the file, creating folders if necessary
func mustTouchFile(ctx context.Context, ref *provider.Reference, gwc gateway.GatewayAPIClient) error {
	if err := touchFile(ctx, ref, gwc); err == nil {
		return nil
	}

	if err := createFolders(ctx, ref, gwc); err != nil {
		return err
	}

	return touchFile(ctx, ref, gwc)
}

func touchFile(ctx context.Context, ref *provider.Reference, gwc gateway.GatewayAPIClient) error {
	resp, err := gwc.TouchFile(ctx, &provider.TouchFileRequest{
		Opaque: utils.AppendPlainToOpaque(nil, "markprocessing", "true"),
		Ref:    ref,
	})
	if err != nil {
		return err
	}
	if resp.GetStatus().GetCode() != rpc.Code_CODE_OK {
		return fmt.Errorf("unexpected statuscode while touching file: %d %s", resp.GetStatus().GetCode(), resp.GetStatus().GetMessage())
	}
	return nil
}

func createFolders(ctx context.Context, ref *provider.Reference, gwc gateway.GatewayAPIClient) error {
	var paths []string
	p := filepath.Dir(ref.GetPath())
	for p != "." {
		paths = append([]string{p}, paths...)
		p = filepath.Dir(p)
	}

	for _, p := range paths {
		r := &provider.Reference{ResourceId: ref.GetResourceId(), Path: p}
		resp, err := gwc.CreateContainer(ctx, &provider.CreateContainerRequest{Ref: r})
		if err != nil {
			return err
		}

		code := resp.GetStatus().GetCode()
		if code != rpc.Code_CODE_OK && code != rpc.Code_CODE_ALREADY_EXISTS {
			return fmt.Errorf("unexpected statuscode while creating folder: %d %s", code, resp.GetStatus().GetMessage())
		}
	}
	return nil
}

func getLocation(r *http.Request) string {
	// from body
	var req ExportPersonalDataRequest
	if b, err := io.ReadAll(r.Body); err == nil {
		if err := json.Unmarshal(b, &req); err == nil && req.StorageLocation != "" {
			return req.StorageLocation
		}
	}

	// from header?

	return _backupFileName
}

// we want the events to look nice in the file, don't we?
func convertEvents(evs []*ehmsg.Event) []map[string]any {
	out := make([]map[string]any, len(evs))
	for i, e := range evs {
		var content map[string]any
		_ = json.Unmarshal(e.GetEvent(), &content)
		out[i] = map[string]any{
			"id":    e.GetId(),
			"type":  e.GetType(),
			"event": content,
		}
	}
	return out
}
