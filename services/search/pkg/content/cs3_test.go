package content_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	revactx "github.com/opencloud-eu/reva/v3/pkg/ctx"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v3/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/opencloud-eu/opencloud/pkg/log"
	conf "github.com/opencloud-eu/opencloud/services/search/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/search/pkg/content"
)

var _ = Describe("cs3 retriever", func() {
	var (
		gatewayClient  *cs3mocks.GatewayAPIClient
		tika           *content.Tika
		tikaSrv        *httptest.Server
		downloadSrv    *httptest.Server
		downloadStatus int
		ctx            = metadata.AppendToOutgoingContext(context.Background(), revactx.TokenHeader, "token")
	)

	BeforeEach(func() {
		downloadStatus = http.StatusOK
		tikaSrv = httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		downloadSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(downloadStatus)
		}))

		pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
		gatewayClient = &cs3mocks.GatewayAPIClient{}
		gatewaySelector := pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return gatewayClient
			},
		)

		cfg := conf.DefaultConfig()
		cfg.Extractor.Tika.TikaURL = tikaSrv.URL
		var err error
		tika, err = content.NewTikaExtractor(gatewaySelector, log.NewLogger(), cfg)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		tikaSrv.Close()
		downloadSrv.Close()
	})

	download := func(code rpc.Code) {
		gatewayClient.On("InitiateFileDownload", mock.Anything, mock.Anything).Return(&gateway.InitiateFileDownloadResponse{
			Status:    &rpc.Status{Code: code},
			Protocols: []*gateway.FileDownloadProtocol{{Protocol: "spaces", DownloadEndpoint: downloadSrv.URL, Token: "transfer"}},
		}, nil)
	}

	DescribeTable("tells a file out of reach from a download the storage refuses",
		func(setup func(), unreachable bool) {
			setup()

			_, err := tika.Retrieve(ctx, &provider.ResourceId{OpaqueId: "file"})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, content.ErrUnreachable)).To(Equal(unreachable))
		},
		Entry("gateway unavailable", func() {
			gatewayClient.On("InitiateFileDownload", mock.Anything, mock.Anything).Return(nil, status.Error(codes.Unavailable, "connection refused"))
		}, true),
		Entry("storage unavailable", func() { download(rpc.Code_CODE_UNAVAILABLE) }, true),
		Entry("data server error", func() { download(rpc.Code_CODE_OK); downloadStatus = http.StatusBadGateway }, true),
		Entry("data server busy", func() { download(rpc.Code_CODE_OK); downloadStatus = http.StatusTooManyRequests }, true),
		Entry("data server down", func() { download(rpc.Code_CODE_OK); downloadSrv.Close() }, true),
		Entry("file not found", func() { download(rpc.Code_CODE_NOT_FOUND) }, false),
		Entry("permission denied", func() { download(rpc.Code_CODE_PERMISSION_DENIED) }, false),
		Entry("download not found", func() { download(rpc.Code_CODE_OK); downloadStatus = http.StatusNotFound }, false),
	)
})
