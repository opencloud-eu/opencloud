package content

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
)

type cs3 struct {
	httpClient      http.Client
	gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
	logger          log.Logger
}

func newCS3Retriever(gatewaySelector pool.Selectable[gateway.GatewayAPIClient], logger log.Logger, insecure bool) cs3 {
	return cs3{
		httpClient: http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure}, //nolint:gosec
			},
		},
		gatewaySelector: gatewaySelector,
		logger:          logger,
	}
}

// initiateDownload resolves the download endpoint, transfer token and auth token
// for rID through the cs3 gateway.
func (s cs3) initiateDownload(ctx context.Context, rID *provider.ResourceId) (endpoint, transferToken, authToken string, err error) {
	authToken, ok := contextGet(ctx, revactx.TokenHeader)
	if !ok {
		return "", "", "", fmt.Errorf("context without %s", revactx.TokenHeader)
	}

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		s.logger.Error().Err(err).Msg("could not get reva gatewayClient")
		return "", "", "", err
	}

	res, err := gatewayClient.InitiateFileDownload(ctx, &provider.InitiateFileDownloadRequest{Ref: &provider.Reference{ResourceId: rID, Path: "."}})
	if err != nil {
		return "", "", "", err
	}
	if res.Status.Code != rpc.Code_CODE_OK {
		return "", "", "", fmt.Errorf("could not load resoure: %s", res.Status.Message)
	}

	for _, p := range res.Protocols {
		if p.Protocol == "spaces" {
			return p.DownloadEndpoint, p.Token, authToken, nil
		}
	}
	if len(res.Protocols) > 0 {
		return res.Protocols[0].DownloadEndpoint, res.Protocols[0].Token, authToken, nil
	}
	return "", "", "", fmt.Errorf("no download protocol found")
}

// Retrieve downloads the file from a cs3 service
// The caller MUST make sure to close the returned ReadCloser
func (s cs3) Retrieve(ctx context.Context, rID *provider.ResourceId) (io.ReadCloser, error) {
	ep, tt, at, err := s.initiateDownload(ctx, rID)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, ep, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set(revactx.TokenHeader, at)
	req.Header.Set("X-Reva-Transfer", tt)

	cres, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if cres.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not download resource. Request returned with statuscode %d ", cres.StatusCode)
	}

	return cres.Body, nil
}

// RetrieveRange downloads length bytes starting at offset from a cs3 service.
// The caller MUST make sure to close the returned ReadCloser.
// It relies on HTTP range support of the download endpoint. If the endpoint
// ignores the Range header and returns the full file (200 instead of 206), the
// leading offset bytes are discarded so the returned reader is always positioned
// at offset.
func (s cs3) RetrieveRange(ctx context.Context, rID *provider.ResourceId, offset, length int64) (io.ReadCloser, error) {
	if offset < 0 || length <= 0 {
		return nil, fmt.Errorf("invalid range: offset %d, length %d", offset, length)
	}

	ep, tt, at, err := s.initiateDownload(ctx, rID)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, ep, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set(revactx.TokenHeader, at)
	req.Header.Set("X-Reva-Transfer", tt)
	// A single range keeps the response a plain 206 with a Content-Range header,
	// never a multipart/byteranges body.
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))

	cres, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	switch cres.StatusCode {
	case http.StatusPartialContent:
		// Range honored: the body already starts at offset.
		return capped(cres.Body, length), nil
	case http.StatusOK:
		// Range ignored: the body is the whole file. Skip to offset so the
		// caller always reads from the requested position.
		if _, err := io.CopyN(io.Discard, cres.Body, offset); err != nil {
			_ = cres.Body.Close()
			return nil, fmt.Errorf("could not skip to offset %d: %w", offset, err)
		}
		return capped(cres.Body, length), nil
	default:
		_ = cres.Body.Close()
		return nil, fmt.Errorf("could not download range. Request returned with statuscode %d ", cres.StatusCode)
	}
}

type cappedReadCloser struct {
	io.Reader
	io.Closer
}

// capped limits rc to length bytes, keeping the contract even when the server
// returns more than the requested range.
func capped(rc io.ReadCloser, length int64) io.ReadCloser {
	return cappedReadCloser{io.LimitReader(rc, length), rc}
}
