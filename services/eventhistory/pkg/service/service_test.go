package service_test

import (
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"time"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/log"
	ehsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/eventhistory/v0"
	"github.com/opencloud-eu/opencloud/services/eventhistory/pkg/config"
	"github.com/opencloud-eu/opencloud/services/eventhistory/pkg/service"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/store"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	microevents "go-micro.dev/v4/events"
	microstore "go-micro.dev/v4/store"
)

var _ = Describe("EventHistoryService", func() {
	var (
		cfg = &config.Config{}

		eh  *service.EventHistoryService
		bus testBus
		sto microstore.Store
	)

	BeforeEach(func() {
		var err error
		sto = store.Create()
		bus = testBus(make(chan events.Event))
		eh, err = service.NewEventHistoryService(cfg, bus, sto, log.Logger{})
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		close(bus)
	})

	It("Records events, stores them and allows them to be retrieved", func() {
		id := bus.Publish(events.UploadReady{})

		// service will store eventually
		time.Sleep(500 * time.Millisecond)

		resp := &ehsvc.GetEventsResponse{}
		err := eh.GetEvents(context.Background(), &ehsvc.GetEventsRequest{Ids: []string{id}}, resp)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp).ToNot(BeNil())

		Expect(len(resp.Events)).To(Equal(1))
		Expect(resp.Events[0].Id).To(Equal(id))
	})

	It("Gets all events", func() {
		ids := make([]string, 3)
		ids[0] = bus.Publish(events.UploadReady{
			ExecutingUser: &userv1beta1.User{
				Id: &userv1beta1.UserId{
					OpaqueId: "test-id",
				},
			},
			Failed:    false,
			Timestamp: utils.TimeToTS(time.Time{}),
		})
		ids[1] = bus.Publish(events.UserCreated{
			UserID: "another-id",
		})
		ids[2] = bus.Publish(events.UserDeleted{
			Executant: &userv1beta1.UserId{
				OpaqueId: "another-id",
			},
			UserID: "test-id",
		})

		time.Sleep(500 * time.Millisecond)

		resp := &ehsvc.GetEventsResponse{}
		err := eh.GetEventsForUser(context.Background(), &ehsvc.GetEventsForUserRequest{UserID: "test-id"}, resp)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp).ToNot(BeNil())

		// Events don't always come back in the same order as they were sent, so we need to sort them and
		// do the same for the expected IDs as well.
		expectedIDs := []string{ids[0], ids[2]}
		sort.Strings(expectedIDs)
		var gotIDs []string
		for _, ev := range resp.Events {
			gotIDs = append(gotIDs, ev.Id)
		}
		sort.Strings(gotIDs)

		Expect(len(gotIDs)).To(Equal(len(expectedIDs)))
		Expect(gotIDs[0]).To(Equal(expectedIDs[0]))
		Expect(gotIDs[1]).To(Equal(expectedIDs[1]))
	})
})

type testBus chan events.Event

func (tb testBus) Consume(_ string, _ ...microevents.ConsumeOption) (<-chan microevents.Event, error) {
	ch := make(chan microevents.Event)
	go func() {
		for ev := range tb {
			b, _ := json.Marshal(ev.Event)
			ch <- microevents.Event{
				Payload: b,
				Metadata: map[string]string{
					events.MetadatakeyEventID:   ev.ID,
					events.MetadatakeyEventType: ev.Type,
				},
			}
		}
	}()
	return ch, nil
}

func (tb testBus) Publish(e any) string {
	ev := events.Event{
		ID:    uuid.New().String(),
		Type:  reflect.TypeOf(e).String(),
		Event: e,
	}

	tb <- ev
	return ev.ID
}
