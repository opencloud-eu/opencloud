package event_test

import (
	"sync/atomic"
	"time"

	sprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/service/event"
)

var _ = Describe("SpaceDebouncer", func() {
	var (
		debouncer *event.SpaceDebouncer

		callCount atomic.Int32

		spaceid = &sprovider.StorageSpaceId{
			OpaqueId: "spaceid",
		}
	)

	BeforeEach(func() {
		callCount = atomic.Int32{}
		debouncer = event.NewSpaceDebouncer(50*time.Millisecond, 10*time.Second, func(id *sprovider.StorageSpaceId) {
			if id.OpaqueId == "spaceid" {
				callCount.Add(1)
			}
		}, log.NewLogger())
	})

	It("debounces", func() {
		debouncer.Debounce(spaceid, nil)
		debouncer.Debounce(spaceid, nil)
		debouncer.Debounce(spaceid, nil)
		Eventually(func() int {
			return int(callCount.Load())
		}, "200ms").Should(Equal(1))
	})

	It("works multiple times", func() {
		debouncer.Debounce(spaceid, nil)
		debouncer.Debounce(spaceid, nil)
		debouncer.Debounce(spaceid, nil)
		time.Sleep(100 * time.Millisecond)

		debouncer.Debounce(spaceid, nil)
		debouncer.Debounce(spaceid, nil)

		Eventually(func() int {
			return int(callCount.Load())
		}, "200ms").Should(Equal(2))
	})

	It("doesn't trigger twice simultaneously", func() {
		debouncer = event.NewSpaceDebouncer(50*time.Millisecond, 5*time.Second, func(id *sprovider.StorageSpaceId) {
			if id.OpaqueId == "spaceid" {
				callCount.Add(1)
			}
			time.Sleep(300 * time.Millisecond)
		}, log.NewLogger())
		debouncer.Debounce(spaceid, nil)
		time.Sleep(100 * time.Millisecond) // Let it trigger once

		debouncer.Debounce(spaceid, nil)
		time.Sleep(100 * time.Millisecond) // shouldn't trigger as the other run is still in progress
		Expect(int(callCount.Load())).To(Equal(1))

		Eventually(func() int {
			return int(callCount.Load())
		}, "2000ms").Should(Equal(2))
	})

	It("fires at the timeout even when continuously debounced", func() {
		debouncer = event.NewSpaceDebouncer(100*time.Millisecond, 250*time.Millisecond, func(id *sprovider.StorageSpaceId) {
			if id.OpaqueId == "spaceid" {
				callCount.Add(1)
			}
		}, log.NewLogger())

		// Initial call to start the timers
		debouncer.Debounce(spaceid, nil)

		// Continuously reset the debounce timer using a ticker, at an interval
		// shorter than the debounce time.
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		done := make(chan bool)
		go func() {
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					debouncer.Debounce(spaceid, nil)
				}
			}
		}()

		// The debounce timer (100ms) should be reset every 50ms and thus never fire.
		// The timeout timer (250ms) should fire regardless.
		Eventually(func() int {
			return int(callCount.Load())
		}, "300ms").Should(Equal(1))

		// Stop the ticker goroutine
		close(done)

		// And it should not fire again
		Consistently(func() int {
			return int(callCount.Load())
		}, "300ms").Should(Equal(1))
	})

	It("doesn't run the timeout function if the work function has been called", func() {
		debouncer = event.NewSpaceDebouncer(100*time.Millisecond, 250*time.Millisecond, func(id *sprovider.StorageSpaceId) {
			if id.OpaqueId == "spaceid" {
				callCount.Add(1)
			}
		}, log.NewLogger())

		// Initial call to start the timers
		debouncer.Debounce(spaceid, nil)

		// Wait for the debounce timer to fire
		Eventually(func() int {
			return int(callCount.Load())
		}, "200ms").Should(Equal(1))

		// The timeout function should not be called
		time.Sleep(300 * time.Millisecond)
		Expect(int(callCount.Load())).To(Equal(1))
	})

	It("calls the ack function when the debounce fires", func() {
		var ackCalled atomic.Bool
		ackFunc := func() error {
			ackCalled.Store(true)
			return nil
		}

		debouncer.Debounce(spaceid, ackFunc)

		Eventually(func() int {
			return int(callCount.Load())
		}, "200ms").Should(Equal(1))
		Eventually(func() bool {
			return ackCalled.Load()
		}, "200ms").Should(BeTrue())
	})

	It("calls the ack function immediately for subsequent calls", func() {
		var firstAckCalled atomic.Bool
		firstAckFunc := func() error {
			firstAckCalled.Store(true)
			return nil
		}

		var secondAckCalled atomic.Bool
		secondAckFunc := func() error {
			secondAckCalled.Store(true)
			return nil
		}

		// First call, sets up the trigger
		debouncer.Debounce(spaceid, firstAckFunc)
		Expect(firstAckCalled.Load()).To(BeFalse())
		Expect(secondAckCalled.Load()).To(BeFalse())

		// Second call, should call its ack immediately
		debouncer.Debounce(spaceid, secondAckFunc)
		Eventually(func() bool {
			return secondAckCalled.Load()
		}, "50ms").Should(BeTrue())
		// The first ack is not yet called.
		Expect(firstAckCalled.Load()).To(BeFalse())

		// After the debounce period, the trigger fires, calling the main function and the first ack.
		Eventually(func() int {
			return int(callCount.Load())
		}, "200ms").Should(Equal(1))
		Eventually(func() bool {
			return firstAckCalled.Load()
		}, "200ms").Should(BeTrue())
	})
})
