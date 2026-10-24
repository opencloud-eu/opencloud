package activitylog_test

import (
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/services/activitylog/pkg/service/activitylog"
)

var _ = Describe("Debouncer", func() {
	var (
		mu          sync.Mutex
		callbacks   []activitylog.RawActivity
		newCallback func(id string, ra []activitylog.RawActivity) error
	)

	BeforeEach(func() {
		mu.Lock()
		callbacks = nil
		mu.Unlock()
		newCallback = func(id string, ra []activitylog.RawActivity) error {
			mu.Lock()
			defer mu.Unlock()
			callbacks = append(callbacks, ra...)
			return nil
		}
	})

	Context("with zero duration", func() {
		It("calls the callback immediately", func() {
			d := activitylog.NewDebouncer(0, newCallback)
			d.Debounce("space1", activitylog.RawActivity{EventID: "activity1"})
			Expect(callbacks).To(HaveLen(1))
			Expect(callbacks[0].EventID).To(Equal("activity1"))
		})

		It("calls the callback immediately for each event", func() {
			d := activitylog.NewDebouncer(0, newCallback)
			d.Debounce("space1", activitylog.RawActivity{EventID: "activity1"})
			d.Debounce("space2", activitylog.RawActivity{EventID: "activity2"})
			Expect(callbacks).To(HaveLen(2))
		})
	})

	Context("with non-zero duration", func() {
		It("batches activities with the same id", func() {
			d := activitylog.NewDebouncer(10*time.Millisecond, newCallback)
			d.Debounce("space1", activitylog.RawActivity{EventID: "activity1"})
			d.Debounce("space1", activitylog.RawActivity{EventID: "activity2"})
			d.Debounce("space1", activitylog.RawActivity{EventID: "activity3"})

			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(callbacks)
			}).Should(Equal(3))
		})

		It("handles different ids independently", func() {
			d := activitylog.NewDebouncer(10*time.Millisecond, newCallback)
			d.Debounce("space1", activitylog.RawActivity{EventID: "activity1"})
			d.Debounce("space2", activitylog.RawActivity{EventID: "activity2"})

			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(callbacks)
			}).Should(Equal(2))
		})

		It("batches activities that arrive within the debounce window", func() {
			d := activitylog.NewDebouncer(100*time.Millisecond, newCallback)
			d.Debounce("space1", activitylog.RawActivity{EventID: "activity1"})
			time.Sleep(20 * time.Millisecond)
			d.Debounce("space1", activitylog.RawActivity{EventID: "activity2"})
			time.Sleep(20 * time.Millisecond)
			d.Debounce("space1", activitylog.RawActivity{EventID: "activity3"})

			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(callbacks)
			}).Should(Equal(3))
		})

		It("processes new batch after previous completes", func() {
			d := activitylog.NewDebouncer(5*time.Millisecond, newCallback)
			d.Debounce("space1", activitylog.RawActivity{EventID: "activity1"})
			time.Sleep(20 * time.Millisecond) // let first batch complete
			d.Debounce("space1", activitylog.RawActivity{EventID: "activity2"})

			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(callbacks)
			}).Should(Equal(2))
		})

		It("skips duplicate write when timer fires during in-progress callback", func() {
			slowCallback := func(id string, ra []activitylog.RawActivity) error {
				time.Sleep(50 * time.Millisecond) // simulate slow write
				mu.Lock()
				defer mu.Unlock()
				callbacks = append(callbacks, ra...)
				return nil
			}

			d := activitylog.NewDebouncer(10*time.Millisecond, slowCallback)
			d.Debounce("space1", activitylog.RawActivity{EventID: "activity1"})
			time.Sleep(20 * time.Millisecond) // timer fires while callback is running

			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(callbacks)
			}).Should(Equal(1))
		})
	})
})
