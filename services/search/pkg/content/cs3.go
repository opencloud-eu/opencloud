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
