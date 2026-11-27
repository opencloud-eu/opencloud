package helpers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	registryv1beta1 "github.com/cs3org/go-cs3apis/cs3/app/registry/v1beta1"
	gatewayv1beta1 "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/collaboration/mocks"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/config"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/helpers"
)

const minimalDiscovery = `<?xml version="1.0" encoding="utf-8"?>
<wopi-discovery>
  <net-zone name="external-http">
    <app name="Word">
      <action name="edit" ext="docx" urlsrc="https://office.test/edit"/>
    </app>
  </net-zone>
</wopi-discovery>
`

var _ = Describe("RefreshAppProvider", func() {
	var (
		// answers is called with the number of the request, starting at 1,
		// and tells whether the discovery answers it
		answers    func(n int64) bool
		requests   atomic.Int64
		registered chan []string
		srv        *httptest.Server
		cfg        *config.Config
		appURLs    *helpers.AppURLs
		gws        *mocks.Selectable[gatewayv1beta1.GatewayAPIClient]
		cancel     context.CancelFunc
		ctx        context.Context
	)

	BeforeEach(func() {
		requests.Store(0)
		registered = make(chan []string, 100)
		srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if !answers(requests.Add(1)) {
				w.WriteHeader(http.StatusBadGateway)
				return
			}
			_, _ = w.Write([]byte(minimalDiscovery))
		}))

		cfg = &config.Config{
			App:    config.App{Addr: srv.URL, Name: "Office"},
			CS3Api: config.CS3Api{APPRegistrationInterval: time.Hour},
		}
		appURLs = helpers.NewAppURLs()
		DeferCleanup(helpers.SetFirstRetryDelayForTest(10 * time.Millisecond))

		gwc := cs3mocks.NewGatewayAPIClient(GinkgoT())
		gwc.On("AddAppProvider", mock.Anything, mock.Anything).
			Return(func(_ context.Context, req *registryv1beta1.AddAppProviderRequest, _ ...grpc.CallOption) *registryv1beta1.AddAppProviderResponse {
				registered <- req.GetProvider().GetMimeTypes()
				return &registryv1beta1.AddAppProviderResponse{Status: &rpcv1beta1.Status{Code: rpcv1beta1.Code_CODE_OK}}
			}, nil).Maybe()
		gws = mocks.NewSelectable[gatewayv1beta1.GatewayAPIClient](GinkgoT())
		gws.On("Next").Return(gwc, nil).Maybe()

		ctx, cancel = context.WithCancel(context.Background())
	})

	AfterEach(func() {
		cancel()
		srv.Close()
	})

	// start runs RefreshAppProvider until the end of the test, and checks that
	// it returns when the context is done
	start := func() {
		done := make(chan struct{})
		go func() {
			defer close(done)
			helpers.RefreshAppProvider(ctx, cfg, log.NopLogger(), gws, appURLs)
		}()
		DeferCleanup(func() {
			cancel()
			Eventually(done).Should(BeClosed())
		})
	}

	It("registers the mimetypes of the discovery right away", func() {
		answers = func(int64) bool { return true }
		start()

		Eventually(registered).Should(Receive(ConsistOf(
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		)))
		Expect(appURLs.Fetched()).To(BeTrue())
		Expect(appURLs.GetAppURLFor("edit", ".docx")).To(Equal("https://office.test/edit"))
	})

	It("retries a failed fetch long before the registration interval", func() {
		// the first fetch fails, as when the WOPI app isn't up yet
		answers = func(n int64) bool { return n > 1 }
		start()

		Eventually(registered).Should(Receive(BeEmpty()))
		Expect(appURLs.Fetched()).To(BeFalse())

		// the interval is an hour, the retry comes right away
		Eventually(appURLs.Fetched).Should(BeTrue())
		Eventually(registered).Should(Receive(Not(BeEmpty())))
	})

	It("keeps the app URLs when a fetch fails once", func() {
		// the second fetch fails, as when the WOPI app restarts
		answers = func(n int64) bool { return n != 2 }
		cfg.CS3Api.APPRegistrationInterval = 100 * time.Millisecond
		start()

		// registered after the first fetch, the failed one and the retry
		for range 3 {
			Eventually(registered).Should(Receive(Not(BeEmpty())))
		}
		Expect(requests.Load()).To(BeNumerically(">=", 3))
	})

	It("clears the app URLs when the WOPI app stays unreachable for about an interval", func() {
		// only the first fetch succeeds, as when the WOPI app goes away
		answers = func(n int64) bool { return n == 1 }
		cfg.CS3Api.APPRegistrationInterval = 100 * time.Millisecond
		start()

		Eventually(registered).Should(Receive(Not(BeEmpty())))
		// retries after 10, 20, 40 and 80 ms keep the app URLs
		for range 4 {
			Eventually(registered).Should(Receive(Not(BeEmpty())))
		}
		// the next one is after the full interval: the app URLs are cleared
		Eventually(registered).Should(Receive(BeEmpty()))
		Expect(appURLs.GetAppURLFor("edit", ".docx")).To(BeEmpty())
		// it stays ready, as it has fetched the app URLs before
		Expect(appURLs.Fetched()).To(BeTrue())
	})

	It("brings the app URLs back when the WOPI app answers again", func() {
		answers = func(n int64) bool { return n == 1 || n > 6 }
		cfg.CS3Api.APPRegistrationInterval = 50 * time.Millisecond
		start()

		Eventually(registered).Should(Receive(BeEmpty()))
		Eventually(registered).Should(Receive(Not(BeEmpty())))
		Expect(appURLs.GetAppURLFor("edit", ".docx")).To(Equal("https://office.test/edit"))
	})
})
