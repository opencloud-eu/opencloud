package events

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ocevents "github.com/opencloud-eu/opencloud/pkg/events"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/metrics"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/prometheus/client_golang/prometheus"
	microevents "go-micro.dev/v4/events"
)

var _ = Describe("profile picture concurrency", func() {
	It("bounds active downloads and processes queued requests without delaying sign-in events", func() {
		const requests = 9
		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)
		input := make(chan microevents.Event, requests+1)
		started := make(chan string, requests)
		finished := make(chan string, requests)
		release := make(chan struct{})
		signedIn := make(chan string, 1)
		syncer := profilePictureProvider{sync: func(ctx context.Context, id, _ string) error {
			started <- id
			select {
			case <-release:
				finished <- id
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}}
		logger := log.NewLogger()
		m, err := metrics.New(prometheus.NewRegistry(), &logger, func(_ []string) (string, string) { return "", "" })
		Expect(err).To(Succeed())
		service, err := NewService(ctx, profilePictureConsumer{input}, profilePictureIdentity{signedIn: signedIn}, m, &logger, WithProfilePictureSync(syncer, nil))
		Expect(err).To(Succeed())
		DeferCleanup(service.Close)
		stopped := make(chan error, 1)
		go func() { stopped <- service.Start() }()
		send := func(event interface{}) {
			payload, err := json.Marshal(event)
			Expect(err).To(Succeed())
			input <- microevents.Event{Payload: payload, Metadata: map[string]string{events.MetadatakeyEventType: reflect.TypeOf(event).String()}}
		}
		id := &user.UserId{OpaqueId: "user123"}
		for range requests {
			send(ocevents.UserProfilePictureSyncRequested{Executant: id, PictureURL: "https://example.com/photo.png"})
		}
		send(events.UserSignedIn{Executant: id, Timestamp: utils.TimeToTS(time.Now())})
		for range maxConcurrentProfilePictureSyncs {
			Eventually(started, 5*time.Second).Should(Receive(Equal(id.OpaqueId)))
		}
		Eventually(signedIn, 5*time.Second).Should(Receive(Equal(id.OpaqueId)))
		Consistently(started, 100*time.Millisecond).ShouldNot(Receive(), "queued downloads must wait for a worker")
		close(release)
		for range requests {
			Eventually(finished, 5*time.Second).Should(Receive(Equal(id.OpaqueId)))
		}
		Expect(service.Close()).To(Succeed())
		Eventually(stopped, 5*time.Second).Should(Receive(Succeed()))
	})
})
