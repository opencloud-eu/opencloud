package activitylog_test

import (
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/data"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/service/activitylog"
)

// ackTracker tracks per-name ack invocations so we can assert each event's ack fires.
type ackTracker struct {
	mu     sync.Mutex
	calls  map[string]int
	errors map[string]error
}

func newAckTracker() *ackTracker {
	return &ackTracker{
		calls:  make(map[string]int),
		errors: make(map[string]error),
	}
}

func (t *ackTracker) track(name string, fn func() error) func() error {
	return func() error {
		t.mu.Lock()
		t.calls[name]++
		err := fn()
		t.errors[name] = err
		t.mu.Unlock()
		return err
	}
}

func (t *ackTracker) count(name string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.calls[name]
}

var _ = Describe("Debouncer", func() {
	var (
		testlog   = log.NopLogger()
		callbacks []data.RawActivity
		mu        sync.Mutex
		cbErr     error
		ackCalled int
		ackMu     sync.Mutex
	)

	newCallback := func(_ string, ras []data.RawActivity) error {
		mu.Lock()
		defer mu.Unlock()
		callbacks = append(callbacks, ras...)
		return cbErr
	}

	newAck := func() error {
		ackMu.Lock()
		defer ackMu.Unlock()
		ackCalled++
		return nil
	}

	BeforeEach(func() {
		callbacks = nil
		cbErr = nil
		ackMu.Lock()
		ackCalled = 0
		ackMu.Unlock()
	})

	Context("with zero duration", func() {
		It("calls the function immediately", func() {
			d := activitylog.NewDebouncer(testlog, 0, newCallback)

			ra := data.RawActivity{EventID: "immediate"}
			d.Debounce("space1", ra, nil)

			mu.Lock()
			defer mu.Unlock()
			Expect(callbacks).To(HaveLen(1))
			Expect(callbacks[0].EventID).To(Equal("immediate"))
		})

		It("calls the function immediately for each call without batching", func() {
			d := activitylog.NewDebouncer(testlog, 0, newCallback)

			ra1 := data.RawActivity{EventID: "act1"}
			ra2 := data.RawActivity{EventID: "act2"}
			d.Debounce("space1", ra1, nil)
			d.Debounce("space1", ra2, nil)

			mu.Lock()
			defer mu.Unlock()
			Expect(callbacks).To(HaveLen(2))
		})
	})

	Context("with non-zero duration", func() {
		It("batches activities with the same id", func() {
			d := activitylog.NewDebouncer(testlog, 50*time.Millisecond, newCallback)

			ra1 := data.RawActivity{EventID: "act1"}
			ra2 := data.RawActivity{EventID: "act2"}
			ra3 := data.RawActivity{EventID: "act3"}

			d.Debounce("space1", ra1, nil)
			d.Debounce("space1", ra2, nil)
			d.Debounce("space1", ra3, nil)

			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(callbacks)
			}).Should(Equal(3))

			mu.Lock()
			defer mu.Unlock()
			Expect(callbacks[0].EventID).To(Equal("act1"))
			Expect(callbacks[1].EventID).To(Equal("act2"))
			Expect(callbacks[2].EventID).To(Equal("act3"))
		})

		It("handles different ids independently", func() {
			d := activitylog.NewDebouncer(testlog, 50*time.Millisecond, newCallback)

			ra1 := data.RawActivity{EventID: "space1-act"}
			ra2 := data.RawActivity{EventID: "space2-act"}

			d.Debounce("space1", ra1, nil)
			d.Debounce("space2", ra2, nil)

			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(callbacks)
			}).Should(Equal(2))
		})

		It("calls the ack function after callback completes", func() {
			d := activitylog.NewDebouncer(testlog, 50*time.Millisecond, newCallback)

			ra := data.RawActivity{EventID: "act1"}
			d.Debounce("space1", ra, newAck)

			Eventually(func() int {
				ackMu.Lock()
				defer ackMu.Unlock()
				return ackCalled
			}).Should(Equal(1))
		})

		It("does not panic when ack is nil", func() {
			d := activitylog.NewDebouncer(testlog, 50*time.Millisecond, newCallback)

			ra := data.RawActivity{EventID: "act1"}
			Expect(func() { d.Debounce("space1", ra, nil) }).ToNot(Panic())

			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(callbacks)
			}).Should(Equal(1))
		})

		It("does not panic when ack returns an error", func() {
			d := activitylog.NewDebouncer(testlog, 50*time.Millisecond, newCallback)

			errAck := func() error { return fmt.Errorf("ack error") }
			ra := data.RawActivity{EventID: "act1"}
			Expect(func() { d.Debounce("space1", ra, errAck) }).ToNot(Panic())

			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(callbacks)
			}).Should(Equal(1))
		})

		It("batches activities that arrive within the debounce window", func() {
			d := activitylog.NewDebouncer(testlog, 200*time.Millisecond, newCallback)

			ra1 := data.RawActivity{EventID: "act1"}
			d.Debounce("space1", ra1, nil)

			time.Sleep(50 * time.Millisecond)

			ra2 := data.RawActivity{EventID: "act2"}
			ra3 := data.RawActivity{EventID: "act3"}
			d.Debounce("space1", ra2, nil)
			d.Debounce("space1", ra3, nil)

			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(callbacks)
			}).Should(Equal(3))
		})

		It("processes new batch after previous completes", func() {
			d := activitylog.NewDebouncer(testlog, 50*time.Millisecond, newCallback)

			// First batch
			ra1 := data.RawActivity{EventID: "batch1-act1"}
			ra2 := data.RawActivity{EventID: "batch1-act2"}
			d.Debounce("space1", ra1, nil)
			d.Debounce("space1", ra2, nil)

			// Wait for first batch to be processed
			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(callbacks)
			}).Should(Equal(2))

			// Second batch
			callbacks = nil
			ra3 := data.RawActivity{EventID: "batch2-act1"}
			d.Debounce("space1", ra3, nil)

			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(callbacks)
			}).Should(Equal(1))
			mu.Lock()
			Expect(callbacks[0].EventID).To(Equal("batch2-act1"))
			mu.Unlock()
		})

		It("reschedules when timer fires during in-progress callback", func() {
			unblockCh := make(chan struct{})

			blockingCallback := func(_ string, ras []data.RawActivity) error {
				mu.Lock()
				callbacks = append(callbacks, ras...)
				mu.Unlock()

				<-unblockCh
				return nil
			}

			d := activitylog.NewDebouncer(testlog, 50*time.Millisecond, blockingCallback)

			// First activity starts the timer
			ra1 := data.RawActivity{EventID: "act1"}
			d.Debounce("space1", ra1, nil)

			// Second activity gets queued while first is being processed
			time.Sleep(60 * time.Millisecond)
			// Timer fires for act1, callback starts (and blocks)
			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(callbacks)
			}, 1*time.Second).Should(Equal(1))

			// Queue another activity while the first callback is still running
			ra2 := data.RawActivity{EventID: "act2"}
			d.Debounce("space1", ra2, nil)

			// Unblock the first callback
			close(unblockCh)

			// The second activity should eventually be processed
			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(callbacks)
			}, 2*time.Second).Should(Equal(2))

			mu.Lock()
			defer mu.Unlock()
			Expect(callbacks[0].EventID).To(Equal("act1"))
			Expect(callbacks[1].EventID).To(Equal("act2"))
		})

		It("calls ack even when callback returns an error", func() {
			cbErr = fmt.Errorf("store failed")
			d := activitylog.NewDebouncer(testlog, 50*time.Millisecond, newCallback)

			ra := data.RawActivity{EventID: "act1"}
			d.Debounce("space1", ra, newAck)

			Eventually(func() int {
				ackMu.Lock()
				defer ackMu.Unlock()
				return ackCalled
			}).Should(Equal(1))
		})

		It("calls each ack when multiple events target the same id within one batch", func() {
			tracker := newAckTracker()
			d := activitylog.NewDebouncer(testlog, 200*time.Millisecond, newCallback)

			ra1 := data.RawActivity{EventID: "act1"}
			ra2 := data.RawActivity{EventID: "act2"}
			ra3 := data.RawActivity{EventID: "act3"}

			d.Debounce("space1", ra1, tracker.track("ack1", func() error { return nil }))
			d.Debounce("space1", ra2, tracker.track("ack2", func() error { return nil }))
			d.Debounce("space1", ra3, tracker.track("ack3", func() error { return nil }))

			// All three activities should be stored ...
			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(callbacks)
			}).Should(Equal(3))

			// ... and all three acks should have been called.
			Eventually(func() int { return tracker.count("ack1") }).Should(Equal(1))
			Eventually(func() int { return tracker.count("ack2") }).Should(Equal(1))
			Eventually(func() int { return tracker.count("ack3") }).Should(Equal(1))
		})

		It("calls ack on the re-schedule path when timer fires during in-progress callback", func() {
			tracker := newAckTracker()
			unblockCh := make(chan struct{})

			blockingCallback := func(_ string, ras []data.RawActivity) error {
				mu.Lock()
				callbacks = append(callbacks, ras...)
				mu.Unlock()

				<-unblockCh
				return nil
			}

			d := activitylog.NewDebouncer(testlog, 50*time.Millisecond, blockingCallback)

			// First activity starts the timer.
			ra1 := data.RawActivity{EventID: "act1"}
			d.Debounce("space1", ra1, tracker.track("ack1", func() error { return nil }))

			// Wait for timer to fire so the blocking callback begins.
			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(callbacks)
			}, 1*time.Second).Should(Equal(1))

			// Queue a second activity on a *different* id so its own timer fires
			// while the first callback is still blocked. When that timer fires,
			// it will complete immediately (no blocking), then we'll queue onto
			// space1 again to hit the re-schedule path.
			ra2 := data.RawActivity{EventID: "act2"}
			d.Debounce("space2", ra2, tracker.track("ack2", func() error { return nil }))

			// Wait for space2 batch to complete (non-blocking callback).
			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(callbacks)
			}, 1*time.Second).Should(Equal(2))

			// Now queue onto space1 while its callback is still in-progress.
			// The timer for space1 has already fired once; new Debounce calls
			// will append to the existing pending item and reset the timer,
			// which triggers the re-schedule path when it fires again.
			ra3 := data.RawActivity{EventID: "act3"}
			d.Debounce("space1", ra3, tracker.track("ack3", func() error { return nil }))

			// Unblock the first callback so space1 can process act3.
			close(unblockCh)

			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(callbacks)
			}, 2*time.Second).Should(Equal(3))

			// All three acks should have been called.
			Eventually(func() int { return tracker.count("ack1") }).Should(Equal(1))
			Eventually(func() int { return tracker.count("ack2") }).Should(Equal(1))
			Eventually(func() int { return tracker.count("ack3") }).Should(Equal(1))
		})

		It("acks event even when store callback fails (data-loss risk)", func() {
			// This test documents current behaviour: ack fires regardless of
			// whether StoreActivity succeeded.  If storage fails but the event
			// is still acked, it will never be replayed by JetStream.
			cbErr = fmt.Errorf("nats kv put failed")
			d := activitylog.NewDebouncer(testlog, 50*time.Millisecond, newCallback)

			ra := data.RawActivity{EventID: "act1"}
			d.Debounce("space1", ra, newAck)

			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(callbacks)
			}).Should(Equal(1))

			// ack was called despite the store error -- event is lost.
			Eventually(func() int {
				ackMu.Lock()
				defer ackMu.Unlock()
				return ackCalled
			}).Should(Equal(1))
		})
	})

	Context("zero-duration ack behaviour", func() {
		It("calls ack when duration is zero (immediate path)", func() {
			d := activitylog.NewDebouncer(testlog, 0, newCallback)

			ra := data.RawActivity{EventID: "immediate"}
			d.Debounce("space1", ra, newAck)

			ackMu.Lock()
			defer ackMu.Unlock()
			Expect(ackCalled).To(Equal(1))
		})

		It("calls each ack when multiple immediate events target the same id", func() {
			tracker := newAckTracker()
			d := activitylog.NewDebouncer(testlog, 0, newCallback)

			ra1 := data.RawActivity{EventID: "act1"}
			ra2 := data.RawActivity{EventID: "act2"}

			d.Debounce("space1", ra1, tracker.track("ack1", func() error { return nil }))
			d.Debounce("space1", ra2, tracker.track("ack2", func() error { return nil }))

			Expect(tracker.count("ack1")).To(Equal(1))
			Expect(tracker.count("ack2")).To(Equal(1))
		})
	})
})


