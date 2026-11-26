package connector

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	appproviderv1beta1 "github.com/cs3org/go-cs3apis/cs3/app/provider/v1beta1"
	gatewayv1beta1 "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	providerv1beta1 "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/config"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/middleware"
	revactx "github.com/opencloud-eu/reva/v3/pkg/ctx"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/propagation"
)

// ContentConnectorService is the interface to implement the "File contents"
// endpoint. Basically upload and download contents.
// All operations need a context containing a WOPI context and, optionally,
// a zerolog logger.
// Target file is within the WOPI context
type ContentConnectorService interface {
	// GetFile downloads the file and write its contents in the provider writer
	GetFile(ctx context.Context, w http.ResponseWriter) error
	// PutFile uploads the stream up to the stream length. The file should be
	// locked beforehand, so the lockID needs to be provided.
	// The current lockID will be returned ONLY if a conflict happens (the file is
	// locked with a different lockID)
	PutFile(ctx context.Context, stream io.Reader, streamLength int64, lockID string) (*ConnectorResponse, error)
}

// ContentConnector implements the "File contents" endpoint.
// Basically, the ContentConnector handles downloads (GetFile) and
// uploads (PutFile)
// Note that operations might return any kind of error, not just ConnectorError
type ContentConnector struct {
	gws pool.Selectable[gatewayv1beta1.GatewayAPIClient]
	cfg *config.Config
}

// NewContentConnector creates a new content connector
func NewContentConnector(gws pool.Selectable[gatewayv1beta1.GatewayAPIClient], cfg *config.Config) *ContentConnector {
	return &ContentConnector{
		gws: gws,
		cfg: cfg,
	}
}

func newHttpRequest(ctx context.Context, wopiContext middleware.WopiContext, method, url, transferToken string, body io.Reader) (*http.Request, error) {
	httpReq, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	if url == "" {
		return nil, NewConnectorError(500, "url is missing")
	}
	if transferToken != "" {
		httpReq.Header.Add("X-Reva-Transfer", transferToken)
	}
	if wopiContext.ViewMode == appproviderv1beta1.ViewMode_VIEW_MODE_VIEW_ONLY && wopiContext.ViewOnlyToken != "" {
		httpReq.Header.Add("X-Access-Token", wopiContext.ViewOnlyToken)
	} else {
		httpReq.Header.Add("X-Access-Token", wopiContext.AccessToken)
	}
	tracingProp := tracing.GetPropagator()
	tracingProp.Inject(ctx, propagation.HeaderCarrier(httpReq.Header))
	return httpReq, nil
}

// GetFile downloads the file from the storage
// https://docs.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/files/getfile
//
// The context MUST have a WOPI context, otherwise an error will be returned.
// You can pass a pre-configured zerologger instance through the context that
// will be used to log messages.
//
// The contents of the file will be written directly into the http Response writer passed as
// parameter.
// Be aware that the body of the response will be written during the execution of this method.
// Any further modifications to the response headers or body will be ignored.
func (c *ContentConnector) GetFile(ctx context.Context, w http.ResponseWriter) error {
	wopiContext, err := middleware.WopiContextFromCtx(ctx)
	if err != nil {
		return err
	}

	logger := zerolog.Ctx(ctx).With().
		Interface("FileReference", wopiContext.FileReference).
		Logger()
	logger.Debug().Msg("GetFile: start")

	gwc, err := c.gws.Next()
	if err != nil {
		return err
	}
	sResp, err := gwc.Stat(ctx, &providerv1beta1.StatRequest{
		Ref: wopiContext.FileReference,
	})
	if err := requestFailed(logger, sResp.GetStatus(), false, err, "GetFile: Stat Request failed"); err != nil {
		return err
	}

	// Initiate download request
	req := &providerv1beta1.InitiateFileDownloadRequest{
		Ref: wopiContext.FileReference,
	}

	if wopiContext.ViewMode == appproviderv1beta1.ViewMode_VIEW_MODE_VIEW_ONLY && wopiContext.ViewOnlyToken != "" {
		ctx = revactx.ContextSetToken(ctx, wopiContext.ViewOnlyToken)
	}
	gwc, err = c.gws.Next()
	if err != nil {
		return err
	}
	resp, err := gwc.InitiateFileDownload(ctx, req)
	if err := requestFailed(logger, resp.GetStatus(), false, err, "GetFile: InitiateFileDownload failed"); err != nil {
		return err
	}

	// Figure out the download endpoint and download token
	downloadEndpoint := ""
	downloadToken := ""
	hasDownloadToken := false

	for _, proto := range resp.GetProtocols() {
		if proto.GetProtocol() == "simple" || proto.GetProtocol() == "spaces" {
			downloadEndpoint = proto.GetDownloadEndpoint()
			downloadToken = proto.GetToken()
			hasDownloadToken = proto.GetToken() != ""
			break
		}
	}

	logger = logger.With().
		Str("Endpoint", downloadEndpoint).
		Bool("HasDownloadToken", hasDownloadToken).Logger()

	httpClient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: c.cfg.CS3Api.DataGateway.Insecure,
			},
		},
	}

	// Prepare the request to download the file
	// public link downloads have the token in the download endpoint
	httpReq, err := newHttpRequest(ctx, wopiContext, http.MethodGet, downloadEndpoint, downloadToken, bytes.NewReader([]byte("")))
	if err != nil {
		logger.Error().Err(err).Msg("GetFile: Could not create the request to the endpoint")
		return err
	}

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		logger.Error().Err(err).Msg("GetFile: Get request to the download endpoint failed")
		return err
	}

	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		logger.Error().
			Int("HttpCode", httpResp.StatusCode).
			Msg("GetFile: downloading the file failed")
		return NewConnectorError(500, "GetFile: Downloading the file failed")
	}

	w.Header().Set(HeaderWopiVersion, getVersion(sResp.GetInfo().GetMtime()))

	// Copy the download into the writer
	_, err = io.Copy(w, httpResp.Body)
	if err != nil {
		logger.Error().Msg("GetFile: copying the file content to the response body failed")
		return err
	}
	logger.Debug().Msg("GetFile: success")
	return nil
}

// PutFile uploads the file to the storage
// https://docs.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/files/putfile
//
// The context MUST have a WOPI context, otherwise an error will be returned.
// You can pass a pre-configured zerologger instance through the context that
// will be used to log messages.
//
// The contents of the file will be read from the stream. The full stream
// length must be provided in order to upload the file.
//
// A lock ID must be provided for the upload (which must match the lock in the
// file). The only case where an empty lock ID can be used is if the target
// file has 0 size.
//
// This method will return the lock ID that should be returned in case of a
// conflict, otherwise it will return an empty string. This means that if the
// method returns a ConnectorError with code 409, the returned string is the
// lock ID that should be used in the X-WOPI-Lock header. In other error
// cases or if the method is successful, an empty string will be returned
// (check for err != nil to know if something went wrong)
//
// On success, the method will return the new mtime of the file
func (c *ContentConnector) PutFile(ctx context.Context, stream io.Reader, streamLength int64, lockID string) (*ConnectorResponse, error) {
	wopiContext, err := middleware.WopiContextFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	logger := zerolog.Ctx(ctx).With().
		Str("RequestedLockID", lockID).
		Int64("UploadLength", streamLength).
		Interface("FileReference", wopiContext.FileReference).
		Logger()
	logger.Debug().Msg("PutFile: start")

	gwc, err := c.gws.Next()
	if err != nil {
		return nil, err
	}
	// We need a stat call on the target file in order to get both the lock
	// (if any) and the current size of the file
	statRes, err := gwc.Stat(ctx, &providerv1beta1.StatRequest{
		Ref: wopiContext.FileReference,
	})
	// we can ignore a not found error here, as we're going to create the file
	if err := requestFailed(logger, statRes.GetStatus(), true, err, "PutFile: stat failed"); err != nil {
		return nil, err
	}

	mtime := statRes.GetInfo().GetMtime()
	// If there is a lock and it mismatches, return 409
	if statRes.GetInfo().GetLock() != nil && statRes.GetInfo().GetLock().GetLockId() != lockID {
		logger.Error().
			Str("LockID", statRes.GetInfo().GetLock().GetLockId()).
			Msg("PutFile: wrong lock")
		// onlyoffice says it's required to send the current lockId, MS doesn't say anything
		return NewResponseLockConflict(statRes.GetInfo().GetLock().GetLockId(), "Lock Mismatch"), nil
	}

	// only unlocked uploads can go through if the target file is empty,
	// otherwise the X-WOPI-Lock header is required even if there is no lock on the file
	// This is part of the onlyoffice documentation (https://api.onlyoffice.com/editors/wopi/restapi/putfile)
	// Wopivalidator fails some tests if we don't also check for the X-WOPI-Lock header.
	if lockID == "" && statRes.GetInfo().GetLock() == nil && statRes.GetInfo().GetSize() > 0 {
		logger.Error().Msg("PutFile: file must be locked first")
		// onlyoffice says to send an empty string if the file is unlocked, MS doesn't say anything
		return NewResponseLockConflict("", "Cannot PutFile on unlocked file"), nil
	}

	// Prepare the data to initiate the upload
	opaque := &types.Opaque{
		Map: make(map[string]*types.OpaqueEntry),
	}

	if streamLength >= 0 {
		opaque.Map["Upload-Length"] = &types.OpaqueEntry{
			Decoder: "plain",
			Value:   []byte(strconv.FormatInt(streamLength, 10)),
		}
	}

	req := &providerv1beta1.InitiateFileUploadRequest{
		Opaque: opaque,
		Ref:    wopiContext.FileReference,
		LockId: lockID,
		Options: &providerv1beta1.InitiateFileUploadRequest_IfMatch{
			IfMatch: statRes.GetInfo().GetEtag(),
		},
	}

	gwc, err = c.gws.Next()
	if err != nil {
		return nil, err
	}
	// Initiate the upload request
	resp, err := gwc.InitiateFileUpload(ctx, req)
	if err := requestFailed(logger, resp.GetStatus(), false, err, "PutFile: InitiateFileUpload failed"); err != nil {
		return nil, err
	}

	// if the content length is greater than 0, we need to upload the content to the
	// target endpoint, otherwise we're done
	if streamLength > 0 {

		uploadEndpoint := ""
		uploadToken := ""
		hasUploadToken := false

		for _, proto := range resp.GetProtocols() {
			if proto.GetProtocol() == "simple" || proto.GetProtocol() == "spaces" {
				uploadEndpoint = proto.GetUploadEndpoint()
				uploadToken = proto.GetToken()
				hasUploadToken = proto.GetToken() != ""
				break
			}
		}

		logger = logger.With().
			Str("Endpoint", uploadEndpoint).
			Bool("HasUploadToken", hasUploadToken).Logger()

		httpClient := http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion:         tls.VersionTLS12,
					InsecureSkipVerify: c.cfg.CS3Api.DataGateway.Insecure,
				},
			},
			Timeout: 10 * time.Second,
		}

		// prepare the request to upload the contents to the upload endpoint
		// public link uploads have the token in the upload endpoint
		httpReq, err := newHttpRequest(ctx, wopiContext, http.MethodPut, uploadEndpoint, uploadToken, stream)
		if err != nil {
			logger.Error().Err(err).Msg("UploadHelper: Could not create the request to the endpoint")
			return nil, err
		}
		// "stream" is an *http.body and doesn't fill the httpReq.ContentLength automatically
		// we need to fill the ContentLength ourselves, and must match the stream length in order
		// to prevent issues
		httpReq.ContentLength = streamLength

		httpReq.Header.Add("X-Lock-Id", lockID)
		// TODO: better mechanism for the upload while locked, relies on patch in REVA
		//if lockID, ok := ctxpkg.ContextGetLockID(ctx); ok {
		//	httpReq.Header.Add("X-Lock-Id", lockID)
		//}

		httpResp, err := httpClient.Do(httpReq)
		if err != nil {
			logger.Error().Err(err).Msg("UploadHelper: Put request to the upload endpoint failed")
			return nil, err
		}
		defer httpResp.Body.Close()

		if httpResp.StatusCode != http.StatusOK {
			logger.Error().
				Int("HttpCode", httpResp.StatusCode).
				Msg("UploadHelper: Put request to the upload endpoint failed with unexpected status")
			return nil, NewConnectorError(500, fmt.Sprintf("unexpected status code %d from the upload endpoint", httpResp.StatusCode))
		}
		gwc, err = c.gws.Next()
		if err != nil {
			return nil, err
		}
		// We need a stat call on the target file after the upload to get the
		// new mtime
		statResAfter, err := gwc.Stat(ctx, &providerv1beta1.StatRequest{
			Ref: wopiContext.FileReference,
		})
		if err := requestFailed(logger, statResAfter.GetStatus(), false, err, "PutFile: stat after upload failed"); err != nil {
			return nil, err
		}
		mtime = statResAfter.GetInfo().GetMtime()
	}

	logger.Debug().Msg("PutFile: success")
	return NewResponseWithVersion(mtime), nil
}

func requestFailed(logger zerolog.Logger, s *rpcv1beta1.Status, allowNotFound bool, err error, msg string) error {
	switch {
	case err != nil: // a connection error
		logger.Error().Err(err).Msg(msg)
		return err
	case s == nil: // we need a status
		logger.Error().Msg(msg + ": nil status")
		return NewConnectorError(500, msg+": nil status")
	case s.GetCode() == rpcv1beta1.Code_CODE_OK: // ok is fine
		return nil
	case allowNotFound && s.GetCode() == rpcv1beta1.Code_CODE_NOT_FOUND: // not found might be ok
		return nil
	default: // any other status is an error
		logger.Error().
			Str("StatusCode", s.GetCode().String()).
			Str("StatusMsg", s.GetMessage()).
			Msg(msg)
		return NewConnectorError(500, s.GetCode().String()+" "+s.GetMessage())
	}
}
