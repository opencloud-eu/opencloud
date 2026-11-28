package service

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"sync"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/postprocessing/pkg/config"
	"github.com/opencloud-eu/opencloud/services/postprocessing/pkg/metrics"
	"github.com/opencloud-eu/opencloud/services/postprocessing/pkg/postprocessing"
	"github.com/opencloud-eu/reva/v3/pkg/events"
	"github.com/opencloud-eu/reva/v3/pkg/events/raw"
	"github.com/opencloud-eu/reva/v3/pkg/store"
	"github.com/rs/zerolog"
	microevents "go-micro.dev/v4/events"
	microstore "go-micro.dev/v4/store"
	"go.opentelemetry.io/otel/trace/noop"
)

// errPublish is the error a nats publish returns when the ack of the message was not
// received in time. It is transient: the very next publish usually succeeds.
var errPublish = errors.New("nats: timeout")

// testPublisher is a publisher that fails the first `failures` publish attempts and
// records everything it accepted afterwards.
type testPublisher struct {
	mu       sync.Mutex
	failures int
	attempts int
	accepted []any
}

func (p *testPublisher) Publish(_ string, ev any, _ ...microevents.PublishOption) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.attempts++
	if p.attempts <= p.failures {
		return errPublish
	}
	p.accepted = append(p.accepted, ev)
	return nil
}

func (p *testPublisher) stats() (attempts int, accepted []any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.attempts, p.accepted
}

// errStoreRead is a store failure that is not "not found", e.g. a lost connection.
var errStoreRead = errors.New("store: connection lost")

// failingStore fails every Read with err.
type failingStore struct {
	microstore.Store
	err error
}

func (s failingStore) Read(string, ...microstore.ReadOption) ([]*microstore.Record, error) {
	return nil, s.err
}

var _ = Describe("PostprocessingService", func() {
	var (
		cfg config.Postprocessing
		pub *testPublisher
		pps *PostprocessingService
	)

	// newService builds a service that is wired to the fake publisher only. It deliberately
	// does not go through NewPostprocessingService, which would need a running nats.
	newService := func() *PostprocessingService {
		return &PostprocessingService{
			ctx:     context.Background(),
			log:     log.NopLogger(),
			pub:     pub,
			steps:   getSteps(cfg),
			store:   store.Create(),
			c:       cfg,
			tp:      noop.NewTracerProvider(),
			metrics: metrics.New(),
			stopCh:  make(chan struct{}, 1),
		}
	}

	// bytesReceived is the event that starts a postprocessing chain. Handling it makes the
	// service publish the first StartPostprocessingStep, which is the publish that used to
	// take the whole process down when nats hiccuped.
	bytesReceived := func() raw.Event {
		ev := events.BytesReceived{
			UploadID: "upload-" + uuid.New().String(),
			Filename: "test.txt",
			Filesize: 1234,
		}
		return raw.Event{
			Event: events.Event{
				ID:    uuid.New().String(),
				Type:  reflect.TypeOf(ev).String(),
				Event: ev,
			},
		}
	}

	BeforeEach(func() {
		cfg = config.Postprocessing{
			Steps:                []string{"virusscan"},
			RetryBackoffDuration: 5 * time.Millisecond,
			MaxRetries:           14,
			PublishMaxRetries:    3,
		}
		pub = &testPublisher{}
	})

	Describe("publishing the next event", func() {
		It("publishes once when the event system is healthy", func() {
			pps = newService()

			Expect(pps.processEvent(bytesReceived())).To(Succeed())

			attempts, accepted := pub.stats()
			Expect(attempts).To(Equal(1))
			Expect(accepted).To(HaveLen(1))
			Expect(accepted[0]).To(BeAssignableToTypeOf(events.StartPostprocessingStep{}))
			Expect(accepted[0].(events.StartPostprocessingStep).StepToStart).To(Equal(events.PPStepAntivirus))
		})

		It("retries a transient publish failure and succeeds on a later attempt", func() {
			pub.failures = 2
			pps = newService()

			Expect(pps.processEvent(bytesReceived())).To(Succeed())

			attempts, accepted := pub.stats()
			Expect(attempts).To(Equal(3))
			Expect(accepted).To(HaveLen(1))
			Expect(accepted[0]).To(BeAssignableToTypeOf(events.StartPostprocessingStep{}))
		})

		It("uses an exponential backoff between the attempts", func() {
			pub.failures = 3
			pps = newService()

			start := time.Now()
			Expect(pps.processEvent(bytesReceived())).To(Succeed())
			elapsed := time.Since(start)

			// 5ms + 10ms + 20ms, minus a margin so a coarse clock cannot make this flaky
			Expect(elapsed).To(BeNumerically(">=", 30*time.Millisecond))
		})

		It("caps the backoff at half the ack wait", func() {
			// Without a cap the waits would grow to 10+20+40+80=150ms, far beyond the ack
			// wait, and jetstream would redeliver the source event to a second worker while
			// this one is still retrying. Capped at AckWait/2 they are 10+10+10+10=40ms.
			cfg.RetryBackoffDuration = 10 * time.Millisecond
			cfg.PublishMaxRetries = 4
			cfg.Events.AckWait = 20 * time.Millisecond
			pub.failures = 4
			pps = newService()

			start := time.Now()
			Expect(pps.processEvent(bytesReceived())).To(Succeed())
			elapsed := time.Since(start)

			Expect(elapsed).To(BeNumerically("<", 100*time.Millisecond))

			attempts, _ := pub.stats()
			Expect(attempts).To(Equal(5))
		})

		It("is fatal once the retries are genuinely exhausted", func() {
			pub.failures = 1000
			pps = newService()

			err := pps.processEvent(bytesReceived())
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrFatal)).To(BeTrue())

			// the initial attempt plus PublishMaxRetries retries
			attempts, accepted := pub.stats()
			Expect(attempts).To(Equal(4))
			Expect(accepted).To(BeEmpty())
		})

		It("does not retry when retrying is disabled", func() {
			cfg.PublishMaxRetries = 0
			pub.failures = 1000
			pps = newService()

			err := pps.processEvent(bytesReceived())
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrFatal)).To(BeTrue())

			attempts, _ := pub.stats()
			Expect(attempts).To(Equal(1))
		})

		It("does not take the process down when the service is stopping", func() {
			pub.failures = 1000
			pps = newService()
			pps.Close()

			err := pps.processEvent(bytesReceived())
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrEvent)).To(BeTrue())
			Expect(errors.Is(err, ErrFatal)).To(BeFalse())
		})
	})

	Describe("handling UploadReady", func() {
		uploadReady := func(uploadID string) raw.Event {
			ev := events.UploadReady{
				UploadID: uploadID,
				Filename: "test.txt",
			}
			return raw.Event{
				Event: events.Event{
					ID:    uuid.New().String(),
					Type:  reflect.TypeOf(ev).String(),
					Event: ev,
				},
			}
		}

		It("ignores an event that does not belong to an upload", func() {
			pps = newService()

			Expect(pps.processEvent(uploadReady(""))).To(Succeed())
		})

		It("fails for an upload it does not know", func() {
			pps = newService()

			err := pps.processEvent(uploadReady("upload-" + uuid.New().String()))
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrEvent)).To(BeTrue())
		})
	})

	Describe("acking events when the store read fails", func() {
		var logs *bytes.Buffer

		// The events in these tests carry no jetstream message, so Ack() fails and the
		// service logs "unable to ack event". That log line is how we see an ack attempt.
		acked := func() bool { return bytes.Contains(logs.Bytes(), []byte("unable to ack event")) }

		withStore := func(sto microstore.Store) {
			// the global level can be raised by other packages and would hide the log line
			prev := zerolog.GlobalLevel()
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
			DeferCleanup(zerolog.SetGlobalLevel, prev)

			logs = &bytes.Buffer{}
			pps = newService()
			pps.log = log.Logger{Logger: zerolog.New(logs)}
			pps.store = sto
		}

		mk := func(ev any) raw.Event {
			return raw.Event{Event: events.Event{
				ID:    uuid.New().String(),
				Type:  reflect.TypeOf(ev).String(),
				Event: ev,
			}}
		}

		cases := map[string]func() raw.Event{
			"PostprocessingStepFinished": func() raw.Event {
				return mk(events.PostprocessingStepFinished{UploadID: "up-1", FinishedStep: events.PPStepAntivirus, Outcome: events.PPOutcomeContinue})
			},
			"StartPostprocessingStep": func() raw.Event {
				return mk(events.StartPostprocessingStep{UploadID: "up-1", StepToStart: events.PPStepDelay})
			},
			"UploadReady": func() raw.Event {
				return mk(events.UploadReady{UploadID: "up-1", Filename: "test.txt"})
			},
		}

		for name, newEvent := range cases {
			It("does not ack "+name+" when the store read fails", func() {
				withStore(failingStore{Store: store.Create(), err: errStoreRead})

				err := pps.processEvent(newEvent())
				Expect(errors.Is(err, ErrEvent)).To(BeTrue())
				Expect(acked()).To(BeFalse())
				_, accepted := pub.stats()
				Expect(accepted).To(BeEmpty())
			})

			It("still acks "+name+" when the upload is not in the store", func() {
				withStore(store.Create())

				err := pps.processEvent(newEvent())
				Expect(errors.Is(err, ErrEvent)).To(BeTrue())
				Expect(acked()).To(BeTrue())
			})
		}

		It("acks UploadReady when the upload is read normally", func() {
			withStore(store.Create())
			Expect(storePP(pps.store, &postprocessing.Postprocessing{ID: "up-1"})).To(Succeed())

			Expect(pps.processEvent(mk(events.UploadReady{UploadID: "up-1", Filename: "test.txt"}))).To(Succeed())
			Expect(acked()).To(BeTrue())
		})
	})
})
