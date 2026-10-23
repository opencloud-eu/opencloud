package events

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	ocevents "github.com/opencloud-eu/opencloud/pkg/events"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/identity"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/metrics"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
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

func TestProfilePictureSyncDoesNotBlockSignInEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	input := make(chan microevents.Event, 2)
	started := make(chan struct{})
	release := make(chan struct{})
	signedIn := make(chan string, 1)
	published := make(chan interface{}, 1)
	logger := log.NewLogger()
	m, err := metrics.New(prometheus.NewRegistry(), &logger, func(_ []string) (string, string) { return "", "" })
	require.NoError(t, err)
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
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, service.Close()) })
	go func() { _ = service.Start() }()
	send := func(event interface{}) {
		payload, err := json.Marshal(event)
		require.NoError(t, err)
		input <- microevents.Event{Payload: payload, Metadata: map[string]string{
			events.MetadatakeyEventType: reflect.TypeOf(event).String(),
		}}
	}
	id := &user.UserId{OpaqueId: "user123"}
	send(ocevents.UserProfilePictureSyncRequested{Executant: id, PictureURL: "https://example.com/photo.png"})
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("profile picture sync did not start")
	}
	send(events.UserSignedIn{Executant: id, Timestamp: utils.TimeToTS(time.Now())})
	select {
	case got := <-signedIn:
		require.Equal(t, id.OpaqueId, got)
	case <-time.After(5 * time.Second):
		t.Fatal("sign-in event blocked by profile picture sync")
	}
	select {
	case <-published:
		t.Fatal("published an update before sync completed")
	default:
	}
	close(release)
	select {
	case event := <-published:
		updated, ok := event.(ocevents.UserProfilePictureUpdated)
		require.True(t, ok)
		require.Equal(t, id, updated.Executant)
		require.NotNil(t, updated.Timestamp)
	case <-time.After(5 * time.Second):
		t.Fatal("successful sync did not publish an update")
	}
}

func TestProfilePictureSyncDoesNotPublishFailedOrCancelledUpdates(t *testing.T) {
	for _, tc := range []string{"failed", "cancelled", "already cancelled"} {
		t.Run(tc, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			published := make(chan interface{}, 1)
			called := false
			publisher := profilePicturePublisher{published}
			syncer := profilePictureProvider{sync: func(gotCtx context.Context, _, _ string) error {
				called = true
				require.Equal(t, ctx, gotCtx)
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
			require.Equal(t, tc != "already cancelled", called)
			require.Empty(t, published)
		})
	}
}

func TestProfilePictureSyncCancelsDownload(t *testing.T) {
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
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("download did not start")
	}
	cancel()
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("download did not stop after cancellation")
	}
	require.Empty(t, published)
}

func TestClosingConsumerCancelsProfilePictureDownload(t *testing.T) {
	logger := log.NewLogger()
	m, err := metrics.New(prometheus.NewRegistry(), &logger, func(_ []string) (string, string) { return "", "" })
	require.NoError(t, err)
	input := make(chan microevents.Event, 1)
	started := make(chan struct{})
	cancelled := make(chan struct{})
	syncer := profilePictureProvider{sync: func(ctx context.Context, _, _ string) error {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return ctx.Err()
	}}
	service, err := NewService(t.Context(), profilePictureConsumer{input}, nil, m, &logger, WithProfilePictureSync(syncer, nil))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, service.Close()) })
	finished := make(chan error, 1)
	go func() { finished <- service.Start() }()
	event := ocevents.UserProfilePictureSyncRequested{
		Executant: &user.UserId{OpaqueId: "user123"}, PictureURL: "https://example.com/photo.png",
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)
	input <- microevents.Event{Payload: payload, Metadata: map[string]string{
		events.MetadatakeyEventType: reflect.TypeOf(event).String(),
	}}
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("download did not start")
	}
	require.NoError(t, service.Close())
	select {
	case <-cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("closing the event consumer did not cancel the download")
	}
	select {
	case err := <-finished:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("event consumer did not stop")
	}
}
