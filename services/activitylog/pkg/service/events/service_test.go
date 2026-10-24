package events_test

import (
	"context"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/config"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/service/activitylog"
	eventssvc "github.com/opencloud-eu/opencloud/services/activitylog/pkg/service/events"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
)

var _ = Describe("ActivitylogService", func() {
	Describe("New", func() {
		var (
			al     *activitylog.ActivityLog
			stream events.Stream
		)

		BeforeEach(func() {
			al = &activitylog.ActivityLog{}
			stream = nil
		})

		It("creates a service with minimal options", func() {
			svc, err := eventssvc.New(al, stream)
			Expect(err).ToNot(HaveOccurred())
			Expect(svc).ToNot(BeNil())
		})

		It("creates a service with context option", func() {
			ctx := context.Background()
			svc, err := eventssvc.New(al, stream, eventssvc.Context(ctx))
			Expect(err).ToNot(HaveOccurred())
			Expect(svc).ToNot(BeNil())
		})

		It("creates a service with logger option", func() {
			logger := log.NopLogger()
			svc, err := eventssvc.New(al, stream, eventssvc.Logger(logger))
			Expect(err).ToNot(HaveOccurred())
			Expect(svc).ToNot(BeNil())
		})

		It("creates a service with service account option", func() {
			sa := config.ServiceAccount{
				ServiceAccountID:     "sa-id",
				ServiceAccountSecret: "sa-secret",
			}
			svc, err := eventssvc.New(al, stream, eventssvc.ServiceAccount(sa))
			Expect(err).ToNot(HaveOccurred())
			Expect(svc).ToNot(BeNil())
		})

		It("creates a service with registered events", func() {
			evts := []events.Unmarshaller{&events.UploadReady{}, &events.FileTouched{}}
			svc, err := eventssvc.New(al, stream, eventssvc.RegisteredEvents(evts))
			Expect(err).ToNot(HaveOccurred())
			Expect(svc).ToNot(BeNil())
		})

		It("creates a service with gateway selector", func() {
			var gs pool.Selectable[gateway.GatewayAPIClient]
			svc, err := eventssvc.New(al, stream, eventssvc.GatewaySelector(gs))
			Expect(err).ToNot(HaveOccurred())
			Expect(svc).ToNot(BeNil())
		})

		It("creates a service with num consumers option", func() {
			svc, err := eventssvc.New(al, stream, eventssvc.NumConsumers(5))
			Expect(err).ToNot(HaveOccurred())
			Expect(svc).ToNot(BeNil())
		})

		It("creates a service with all options", func() {
			ctx := context.Background()
			logger := log.NopLogger()
			sa := config.ServiceAccount{
				ServiceAccountID:     "sa-id",
				ServiceAccountSecret: "sa-secret",
			}
			evts := []events.Unmarshaller{&events.UploadReady{}, &events.ContainerCreated{}}
			var gs pool.Selectable[gateway.GatewayAPIClient]

			svc, err := eventssvc.New(
				al,
				stream,
				eventssvc.Context(ctx),
				eventssvc.Logger(logger),
				eventssvc.ServiceAccount(sa),
				eventssvc.RegisteredEvents(evts),
				eventssvc.GatewaySelector(gs),
				eventssvc.NumConsumers(3),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(svc).ToNot(BeNil())
		})
	})

	Describe("Close", func() {
		It("can be called without panic on a new service", func() {
			al := &activitylog.ActivityLog{}
			svc, err := eventssvc.New(al, nil)
			Expect(err).ToNot(HaveOccurred())

			Expect(func() { svc.Close() }).ToNot(Panic())
		})

		It("can be called multiple times without panic", func() {
			al := &activitylog.ActivityLog{}
			svc, err := eventssvc.New(al, nil)
			Expect(err).ToNot(HaveOccurred())

			svc.Close()
			svc.Close()
			svc.Close()
		})
	})
})
