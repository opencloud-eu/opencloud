package bleve

import (
	"bytes"
	"strings"

	"github.com/blevesearch/bleve/v2/index/scorch"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

var _ = Describe("asyncErrorReporter", func() {
	var (
		out      *bytes.Buffer
		counter  *prometheus.CounterVec
		reporter *asyncErrorReporter
	)

	mergeErr := scorch.NewScorchError("merger", "merging err: corrupt segment", scorch.ErrPersist)
	persistErr := scorch.NewScorchError("persister", "got err persisting snapshot: disk full", scorch.ErrPersist)
	cleanupErr := scorch.NewScorchError("persister", "got err removing old bolt snapshots", scorch.ErrCleanup)
	panicErr := scorch.NewScorchError("merger", "panic: boom", scorch.ErrAsyncPanic)

	BeforeEach(func() {
		// match log.NewLogger's global level
		DeferCleanup(zerolog.SetGlobalLevel, zerolog.GlobalLevel())
		zerolog.SetGlobalLevel(zerolog.TraceLevel)

		out = &bytes.Buffer{}
		counter = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "bleve_async_errors_total"}, []string{"source", "err_type"})
		reporter = newAsyncErrorReporter(counter)
		reporter.logger.Store(&log.Logger{Logger: zerolog.New(out)})
	})

	lines := func() int { return strings.Count(out.String(), "\n") }

	count := func(source string, errType scorch.ScorchErrorType) float64 {
		var m dto.Metric
		Expect(counter.WithLabelValues(source, string(errType)).Write(&m)).To(Succeed())
		return m.GetCounter().GetValue()
	}

	It("logs a repeating error once", func() {
		for range 3 {
			reporter.report(mergeErr, "/index")
		}
		Expect(lines()).To(Equal(1))
	})

	It("logs a different source or error type right away", func() {
		reporter.report(mergeErr, "/index")
		reporter.report(mergeErr, "/index")
		reporter.report(persistErr, "/index")
		reporter.report(cleanupErr, "/index")

		Expect(lines()).To(Equal(3))
		Expect(out.String()).To(ContainSubstring("disk full"))
		Expect(out.String()).To(ContainSubstring("removing old bolt snapshots"))
	})

	It("logs every panic", func() {
		reporter.report(mergeErr, "/index")
		for range 3 {
			reporter.report(panicErr, "/index")
		}
		Expect(lines()).To(Equal(4))
	})

	It("logs the source, error type and index path", func() {
		reporter.report(persistErr, "/index")

		Expect(out.String()).To(ContainSubstring(`"source":"persister"`))
		Expect(out.String()).To(ContainSubstring(`"err_type":"persist error"`))
		Expect(out.String()).To(ContainSubstring(`"path":"/index"`))
	})

	It("counts every error by source and error type", func() {
		for range 3 {
			reporter.report(mergeErr, "/index")
		}
		reporter.report(persistErr, "/index")
		reporter.report(panicErr, "/index")

		Expect(count("merger", scorch.ErrPersist)).To(Equal(3.0))
		Expect(count("persister", scorch.ErrPersist)).To(Equal(1.0))
		Expect(count("merger", scorch.ErrAsyncPanic)).To(Equal(1.0))
		Expect(count("persister", scorch.ErrCleanup)).To(BeZero())
	})
})
