package events

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ocevents "github.com/opencloud-eu/opencloud/pkg/events"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/identity"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/metrics"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/prometheus/client_golang/prometheus"
	microevents "go-micro.dev/v4/events"
)

type profilePictureConsumer struct {
	input chan microevents.Event
}

func (c profilePictureConsumer) Consume(string, ...microevents.ConsumeOption) (<-chan microevents.Event, error) {
	return c.input, nil
}

type profilePicturePublisher struct {
	published chan interface{}
}

func (p profilePicturePublisher) Publish(_ string, event interface{}, _ ...microevents.PublishOption) error {
	p.published <- event
	return nil
}

type profilePictureProvider struct {
	sync func(context.Context, string, string) error
}

func (p profilePictureProvider) SyncPhotoFromURL(ctx context.Context, id, url string) error {
	return p.sync(ctx, id, url)
}

type profilePictureIdentity struct {
	identity.Backend
	signedIn chan string
}

func (i profilePictureIdentity) UpdateLastSignInDate(_ context.Context, id string, _ time.Time) error {
	i.signedIn <- id
	return nil
}

var _ = It("does not block sign-in events while syncing a profile picture", func() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	input := make(chan microevents.Event, 2)
	started := make(chan struct{})
	release := make(chan struct{})
	signedIn := make(chan string, 1)
	published := make(chan interface{}, 1)
	logger := log.NewLogger()
	m, err := metrics.New(prometheus.NewRegistry(), &logger, func(_ []string) (string, string) { return "", "" })
	Expect(err).To(Succeed())
	syncer := profilePictureProvider{sync: func(ctx context.Context, id, url string) error {
		close(started)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}}
	service, err := NewService(ctx, profilePictureConsumer{input}, profilePictureIdentity{signedIn: signedIn}, m, &logger,
		WithProfilePictureSync(syncer, profilePicturePublisher{published}))
	Expect(err).To(Succeed())
	DeferCleanup(service.Close)
	go func() { _ = service.Start() }()
	send := func(event interface{}) {
		payload, err := json.Marshal(event)
		Expect(err).To(Succeed())
		input <- microevents.Event{Payload: payload, Metadata: map[string]string{
			events.MetadatakeyEventType: reflect.TypeOf(event).String(),
		}}
	}
	id := &user.UserId{OpaqueId: "user123"}
	send(ocevents.UserProfilePictureSyncRequested{Executant: id, PictureURL: "https://example.com/photo.png"})
	Eventually(started, 5*time.Second).Should(BeClosed(), "profile picture sync did not start")
	send(events.UserSignedIn{Executant: id, Timestamp: utils.TimeToTS(time.Now())})
	Eventually(signedIn, 5*time.Second).Should(Receive(Equal(id.OpaqueId)), "sign-in event blocked by profile picture sync")
	Expect(published).NotTo(Receive(), "published an update before sync completed")
	close(release)
	var event interface{}
	Eventually(published, 5*time.Second).Should(Receive(&event), "successful sync did not publish an update")
	updated, ok := event.(ocevents.UserProfilePictureUpdated)
	Expect(ok).To(BeTrue())
	Expect(updated.Executant).To(Equal(id))
	Expect(updated.Timestamp).NotTo(BeNil())
})

var _ = Describe("failed or cancelled profile picture sync", func() {
	for _, tc := range []string{"failed", "cancelled", "already cancelled"} {
		It(tc, func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			published := make(chan interface{}, 1)
			called := false
			publisher := profilePicturePublisher{published}
			syncer := profilePictureProvider{sync: func(gotCtx context.Context, _, _ string) error {
				called = true
				Expect(gotCtx).To(Equal(ctx))
				if tc == "cancelled" {
					cancel()
					return nil
				}
				return errors.New("download failed")
			}}
			if tc == "already cancelled" {
				cancel()
			}
			logger := log.NewLogger()
			syncProfilePicture(ctx, &logger, syncer, publisher, ocevents.UserProfilePictureSyncRequested{
				Executant: &user.UserId{OpaqueId: "user123"}, PictureURL: "https://example.com/photo.png",
			})
			Expect(called).To(Equal(tc != "already cancelled"))
			Expect(published).To(BeEmpty())
		})
	}
})

var _ = It("cancels a profile picture download with its context", func() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	finished := make(chan struct{})
	published := make(chan interface{}, 1)
	logger := log.NewLogger()
	publisher := profilePicturePublisher{published}
	syncer := profilePictureProvider{sync: func(ctx context.Context, _, _ string) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}}
	go func() {
		defer close(finished)
		syncProfilePicture(ctx, &logger, syncer, publisher, ocevents.UserProfilePictureSyncRequested{
			Executant: &user.UserId{OpaqueId: "user123"}, PictureURL: "https://example.com/photo.png",
		})
	}()
	Eventually(started, 5*time.Second).Should(BeClosed(), "download did not start")
	cancel()
	Eventually(finished, 5*time.Second).Should(BeClosed(), "download did not stop after cancellation")
	Expect(published).To(BeEmpty())
})

var _ = It("cancels profile picture downloads when the consumer closes", func() {
	logger := log.NewLogger()
	m, err := metrics.New(prometheus.NewRegistry(), &logger, func(_ []string) (string, string) { return "", "" })
	Expect(err).To(Succeed())
	input := make(chan microevents.Event, 1)
	started := make(chan struct{})
	cancelled := make(chan struct{})
	syncer := profilePictureProvider{sync: func(ctx context.Context, _, _ string) error {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return ctx.Err()
	}}
	service, err := NewService(GinkgoT().Context(), profilePictureConsumer{input}, nil, m, &logger, WithProfilePictureSync(syncer, nil))
	Expect(err).To(Succeed())
	DeferCleanup(service.Close)
	finished := make(chan error, 1)
	go func() { finished <- service.Start() }()
	event := ocevents.UserProfilePictureSyncRequested{
		Executant: &user.UserId{OpaqueId: "user123"}, PictureURL: "https://example.com/photo.png",
	}
	payload, err := json.Marshal(event)
	Expect(err).To(Succeed())
	input <- microevents.Event{Payload: payload, Metadata: map[string]string{
		events.MetadatakeyEventType: reflect.TypeOf(event).String(),
	}}
	Eventually(started, 5*time.Second).Should(BeClosed(), "download did not start")
	Expect(service.Close()).To(Succeed())
	Eventually(cancelled, 5*time.Second).Should(BeClosed(), "closing the event consumer did not cancel the download")
	Eventually(finished, 5*time.Second).Should(Receive(Succeed()), "event consumer did not stop")
})
