package service

import (
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
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/raw"
	"github.com/opencloud-eu/reva/v2/pkg/store"
	microevents "go-micro.dev/v4/events"
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

	// stepFinishedForUnknownUpload reports a finished step for an upload the store does
	// not know about. Handling it fails with ErrEvent, which concerns only that event.
	stepFinishedForUnknownUpload := func() raw.Event {
		ev := events.PostprocessingStepFinished{
			UploadID:     "unknown-" + uuid.New().String(),
			FinishedStep: events.PPStepAntivirus,
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

	Describe("running the event loop", func() {
		// newRunningService is newService plus the worker count and the event channel
		// Run needs. The channel is handed back so a test can feed the workers.
		newRunningService := func(workers int) (*PostprocessingService, chan raw.Event) {
			cfg.Workers = workers
			evs := make(chan raw.Event)
			svc := newService()
			svc.events = evs
			return svc, evs
		}

		It("returns without an error when it is stopped", func() {
			svc, _ := newRunningService(2)

			done := make(chan error, 1)
			go func() { done <- svc.Run() }()

			svc.Close()

			Eventually(done, 5*time.Second).Should(Receive(BeNil()))
		})

		It("returns the fatal error instead of killing the process", func() {
			// every publish fails, so the first event exhausts the retries and comes
			// back as ErrFatal. That used to be a log.Fatal, which exits the process
			// and takes every other service in the same binary with it.
			pub.failures = 1000
			svc, evs := newRunningService(2)

			done := make(chan error, 1)
			go func() { done <- svc.Run() }()

			evs <- bytesReceived()

			var err error
			Eventually(done, 5*time.Second).Should(Receive(&err))
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrFatal)).To(BeTrue())
		})

		It("keeps going after an error that only concerns a single event", func() {
			// a step finished for an upload the store knows nothing about fails with
			// ErrEvent, which the loop logs and moves on from
			svc, evs := newRunningService(1)

			done := make(chan error, 1)
			go func() { done <- svc.Run() }()

			evs <- stepFinishedForUnknownUpload()

			Consistently(done, 200*time.Millisecond).ShouldNot(Receive())

			svc.Close()
			Eventually(done, 5*time.Second).Should(Receive(BeNil()))
		})
	})
})
