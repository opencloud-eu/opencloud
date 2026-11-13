package svc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/graph/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/metrics"
	service "github.com/opencloud-eu/opencloud/services/graph/pkg/service/v0"
)

var _ = Describe("ExportPersonalData", func() {
	var (
		cfg *config.Config
		ctx context.Context
	)

	newService := func() service.Service {
		pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
		gatewayClient := &cs3mocks.GatewayAPIClient{}
		gatewaySelector := pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return gatewayClient
			},
		)
		logger := log.NewLogger()
		metrics, _ := metrics.New(prometheus.NewRegistry(), &logger, func([]string) (string, string) { return "", "" })
		svc, err := service.NewService(
			service.Config(cfg),
			service.Metrics(metrics),
			service.WithGatewaySelector(gatewaySelector),
			service.EventsPublisher(&mocks.Publisher{}),
			service.PermissionService(&mocks.Permissions{}),
		)
		Expect(err).ToNot(HaveOccurred())
		return svc
	}

	// export sends an export request for the user with the given id. The
	// storage location has an unsupported file extension, so that a request
	// that reaches the handler fails right away, with a response that differs
	// from those of the middlewares in front of it.
	export := func(svc service.Service, path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"storageLocation": "/export.txt"}`))
		r = r.WithContext(ctx)
		rr := httptest.NewRecorder()
		svc.ServeHTTP(rr, r)
		return rr
	}
	reachedHandler := func(rr *httptest.ResponseRecorder) bool {
		return rr.Code == http.StatusBadRequest && strings.Contains(rr.Body.String(), "only json format")
	}

	BeforeEach(func() {
		ctx = revactx.ContextSetUser(context.Background(), &userv1beta1.User{
			Id: &userv1beta1.UserId{OpaqueId: "alice"},
		})

		cfg = defaults.FullDefaultConfig()
		cfg.Identity.LDAP.CACert = ""
		cfg.TokenManager.JWTSecret = "loremipsum"
		cfg.Commons = &shared.Commons{}
		cfg.GRPCClientTLS = &shared.GRPCClientTLS{}
		cfg.API.ExportPersonalDataRateLimit = 2
	})

	It("rejects exports for other users", func() {
		svc := newService()
		rr := export(svc, "/graph/v1.0/users/bob/exportPersonalData")
		Expect(rr.Code).To(Equal(http.StatusBadRequest))
		Expect(reachedHandler(rr)).To(BeFalse())
	})

	It("limits the exports of a user, also when the path varies", func() {
		svc := newService()
		Expect(reachedHandler(export(svc, "/graph/v1.0/users/alice/exportPersonalData"))).To(BeTrue())
		Expect(reachedHandler(export(svc, "/graph/v1.0/users/alice/exportPersonalData/"))).To(BeTrue())
		Expect(export(svc, "/graph/v1.0/users/al%69ce/exportPersonalData").Code).To(Equal(http.StatusTooManyRequests))
	})

	It("answers beyond the limit in the error format of graph, with Retry-After", func() {
		svc := newService()
		export(svc, "/graph/v1.0/users/alice/exportPersonalData")
		export(svc, "/graph/v1.0/users/alice/exportPersonalData")
		rr := export(svc, "/graph/v1.0/users/alice/exportPersonalData")
		Expect(rr.Code).To(Equal(http.StatusTooManyRequests))
		Expect(rr.Header().Get("Retry-After")).ToNot(BeEmpty())

		var body struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		Expect(json.Unmarshal(rr.Body.Bytes(), &body)).To(Succeed())
		Expect(body.Error.Code).To(Equal("activityLimitReached"))
	})

	It("rejects requests for other users before they count against any limit", func() {
		svc := newService()
		for range 5 {
			Expect(export(svc, "/graph/v1.0/users/bob/exportPersonalData").Code).To(Equal(http.StatusBadRequest))
		}
		// alice still has her whole budget
		Expect(reachedHandler(export(svc, "/graph/v1.0/users/alice/exportPersonalData"))).To(BeTrue())
		Expect(reachedHandler(export(svc, "/graph/v1.0/users/alice/exportPersonalData"))).To(BeTrue())
		// and so has bob
		ctx = revactx.ContextSetUser(context.Background(), &userv1beta1.User{
			Id: &userv1beta1.UserId{OpaqueId: "bob"},
		})
		Expect(reachedHandler(export(svc, "/graph/v1.0/users/bob/exportPersonalData"))).To(BeTrue())
		Expect(reachedHandler(export(svc, "/graph/v1.0/users/bob/exportPersonalData"))).To(BeTrue())
	})

	It("doesn't limit the exports when the limit is 0", func() {
		cfg.API.ExportPersonalDataRateLimit = 0
		svc := newService()
		for range 5 {
			Expect(reachedHandler(export(svc, "/graph/v1.0/users/alice/exportPersonalData"))).To(BeTrue())
		}
	})
})
