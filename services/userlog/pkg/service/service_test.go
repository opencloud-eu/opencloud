package service_test

import (
	"context"
	"reflect"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/store"
	"github.com/stretchr/testify/mock"
	microstore "go-micro.dev/v4/store"
	"go.opentelemetry.io/otel/trace"

	"github.com/opencloud-eu/opencloud/pkg/log"
	ehmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/eventhistory/v0"
	ehsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/eventhistory/v0"
	"github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/eventhistory/v0/mocks"
	"github.com/opencloud-eu/opencloud/services/userlog/pkg/service"
)

var _ = Describe("UserlogService", func() {
	var (
		ul  *service.UserlogService
		sto microstore.Store
		ehc mocks.EventHistoryService
	)

	BeforeEach(func() {
		var err error
		sto = store.Create()
		ehc = mocks.EventHistoryService{}

		ul, err = service.NewUserlogService(
			service.Store(sto),
			service.Logger(log.NewLogger()),
			service.HistoryClient(&ehc),
			service.TraceProvider(trace.NewNoopTracerProvider()),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	newEvent := func() events.Event {
		return events.Event{
			ID:    uuid.New().String(),
			Type:  reflect.TypeOf(events.SpaceDisabled{}).String(),
			Event: events.SpaceDisabled{},
		}
	}

	It("it stores, returns and deletes a couple of events", func() {
		ids := make(map[string]struct{})
		for i := 0; i < 3; i++ {
			ev := newEvent()
			Expect(ul.AddEventToUser("userid", ev)).To(Succeed())
			ids[ev.ID] = struct{}{}
		}

		var events []*ehmsg.Event
		for id := range ids {
			events = append(events, &ehmsg.Event{Id: id})
		}

		ehc.On("GetEvents", mock.Anything, mock.Anything).Return(&ehsvc.GetEventsResponse{Events: events}, nil)

		evs, err := ul.GetEvents(context.Background(), "userid")
		Expect(err).ToNot(HaveOccurred())
		Expect(len(evs)).To(Equal(len(ids)))

		var evids []string
		for _, e := range evs {
			_, exists := ids[e.Id]
			Expect(exists).To(BeTrue())
			delete(ids, e.Id)
			evids = append(evids, e.Id)
		}

		Expect(len(ids)).To(Equal(0))
		err = ul.DeleteEvents("userid", evids)
		Expect(err).ToNot(HaveOccurred())

		evs, err = ul.GetEvents(context.Background(), "userid")
		Expect(err).ToNot(HaveOccurred())
		Expect(len(evs)).To(Equal(0))
	})

	It("works without event consumer (HTTP-only mode)", func() {
		ev := newEvent()
		Expect(ul.AddEventToUser("userid", ev)).To(Succeed())

		ehc.On("GetEvents", mock.Anything, mock.Anything).Return(&ehsvc.GetEventsResponse{
			Events: []*ehmsg.Event{{Id: ev.ID}},
		}, nil)

		evs, err := ul.GetEvents(context.Background(), "userid")
		Expect(err).ToNot(HaveOccurred())
		Expect(len(evs)).To(Equal(1))
		Expect(evs[0].Id).To(Equal(ev.ID))
	})

	It("stores and deletes global events", func() {
		Expect(ul.StoreGlobalEvent(context.Background(), "deprovision", map[string]string{
			"deprovision_date": time.Now().Add(time.Hour).Format(time.RFC3339),
		})).To(Succeed())

		evs, err := ul.GetGlobalEvents(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(evs).To(HaveLen(1))

		Expect(ul.DeleteGlobalEvents(context.Background(), []string{"deprovision"})).To(Succeed())
		evs, err = ul.GetGlobalEvents(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(evs).To(HaveLen(0))
	})
})
