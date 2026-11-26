package service_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	merrors "go-micro.dev/v4/errors"
	"google.golang.org/grpc"

	"github.com/opencloud-eu/opencloud/pkg/log"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search/mocks"
	service "github.com/opencloud-eu/opencloud/services/search/pkg/service/grpc/v0"
)

var _ = Describe("OptimizeIndex", func() {
	var (
		searcher *mocks.Searcher
		handler  searchsvc.SearchProviderHandler
		logs     *bytes.Buffer
	)

	BeforeEach(func() {
		searcher = mocks.NewSearcher(GinkgoT())
		logs = &bytes.Buffer{}
		gws := pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(grpc.ClientConnInterface) gateway.GatewayAPIClient { return nil },
		)

		var err error
		handler, err = service.NewHandler(
			service.Config(&config.Config{}),
			service.Logger(log.Logger{Logger: zerolog.New(logs)}),
			// the handler builds a token manager, which refuses an empty secret
			service.JWTSecret("not-a-secret"),
			service.GatewaySelector(gws),
			service.Searcher(searcher),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	It("reports how long the optimization took", func() {
		searcher.EXPECT().OptimizeIndex(mock.Anything).Return(nil)

		out := &searchsvc.OptimizeIndexResponse{}
		Expect(handler.OptimizeIndex(context.Background(), &searchsvc.OptimizeIndexRequest{}, out)).To(Succeed())
		Expect(out.GetDuration()).ToNot(BeNil())
	})

	DescribeTable("maps the failure to a status",
		func(cause error, code int32) {
			searcher.EXPECT().OptimizeIndex(mock.Anything).Return(cause)

			err := handler.OptimizeIndex(context.Background(), &searchsvc.OptimizeIndexRequest{}, &searchsvc.OptimizeIndexResponse{})
			Expect(merrors.FromError(err).GetCode()).To(Equal(code))
		},
		Entry("an engine that merges on its own", search.ErrOptimizeUnsupported, int32(http.StatusNotImplemented)),
		Entry("a merge already running", search.ErrOptimizeInProgress, int32(http.StatusPreconditionFailed)),
		Entry("any other error", errors.New("disk full"), int32(http.StatusInternalServerError)),
	)

	It("keeps the cause of an unexpected error out of the response", func() {
		searcher.EXPECT().OptimizeIndex(mock.Anything).Return(errors.New("/var/lib/index: disk full"))

		err := handler.OptimizeIndex(context.Background(), &searchsvc.OptimizeIndexRequest{}, &searchsvc.OptimizeIndexResponse{})
		Expect(merrors.FromError(err).GetDetail()).ToNot(ContainSubstring("disk full"))
	})

	It("keeps the index type out of the unsupported answer", func() {
		searcher.EXPECT().OptimizeIndex(mock.Anything).Return(fmt.Errorf("%w: %s", search.ErrOptimizeUnsupported, "*upsidedown.UpsideDownCouch"))

		err := handler.OptimizeIndex(context.Background(), &searchsvc.OptimizeIndexRequest{}, &searchsvc.OptimizeIndexResponse{})
		Expect(merrors.FromError(err).GetDetail()).To(Equal(search.ErrOptimizeUnsupported.Error()))
	})

	It("logs no error when the client cancels", func() {
		searcher.EXPECT().OptimizeIndex(mock.Anything).Return(context.Canceled)

		err := handler.OptimizeIndex(context.Background(), &searchsvc.OptimizeIndexRequest{}, &searchsvc.OptimizeIndexResponse{})
		Expect(err).To(MatchError(context.Canceled))
		Expect(logs.String()).ToNot(ContainSubstring(`"level":"error"`))
	})
})
