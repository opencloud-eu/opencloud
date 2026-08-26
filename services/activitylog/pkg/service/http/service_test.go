package http_test

import (
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/log"
	ehsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/eventhistory/v0"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/service/activitylog"
	httpsvc "github.com/opencloud-eu/opencloud/services/activitylog/pkg/service/http"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
)

var _ = Describe("Service", func() {
	Describe("New", func() {
		var (
			al *activitylog.ActivityLog
		)

		BeforeEach(func() {
			al = &activitylog.ActivityLog{}
		})

		It("creates a service with default options", func() {
			svc, err := httpsvc.New(al)
			Expect(err).ToNot(HaveOccurred())
			Expect(svc).ToNot(BeNil())
		})

		It("creates a service with logger option", func() {
			logger := log.NopLogger()
			svc, err := httpsvc.New(al, httpsvc.Logger(logger))
			Expect(err).ToNot(HaveOccurred())
			Expect(svc).ToNot(BeNil())
		})

		It("creates a service with registered events", func() {
			evts := []events.Unmarshaller{&events.UploadReady{}}
			svc, err := httpsvc.New(al, httpsvc.RegisteredEvents(evts))
			Expect(err).ToNot(HaveOccurred())
			Expect(svc).ToNot(BeNil())
		})

		It("creates a service with gateway selector", func() {
			var gs pool.Selectable[gateway.GatewayAPIClient]
			svc, err := httpsvc.New(al, httpsvc.GatewaySelector(gs))
			Expect(err).ToNot(HaveOccurred())
			Expect(svc).ToNot(BeNil())
		})

		It("creates a service with history client", func() {
			var hc ehsvc.EventHistoryService
			svc, err := httpsvc.New(al, httpsvc.HistoryClient(hc))
			Expect(err).ToNot(HaveOccurred())
			Expect(svc).ToNot(BeNil())
		})

		It("creates a service with all options", func() {
			logger := log.NopLogger()
			evts := []events.Unmarshaller{&events.UploadReady{}, &events.FileTouched{}}
			var gs pool.Selectable[gateway.GatewayAPIClient]
			var hc ehsvc.EventHistoryService

			svc, err := httpsvc.New(
				al,
				httpsvc.Logger(logger),
				httpsvc.RegisteredEvents(evts),
				httpsvc.GatewaySelector(gs),
				httpsvc.HistoryClient(hc),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(svc).ToNot(BeNil())
		})
	})
})
