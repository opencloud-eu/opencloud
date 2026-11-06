package writer

import (
	"errors"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	recordsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "opencloud",
		Subsystem: "audit",
		Name:      "records_total",
		Help:      "Audit records handled by the audit service, by result (written, invalid).",
	}, []string{"result"})

	sinkErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "opencloud",
		Subsystem: "audit",
		Name:      "sink_errors_total",
		Help:      "Failed writes to an output; the record is delivered again.",
	}, []string{"sink"})

	consumerLag = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "opencloud",
		Subsystem: "audit",
		Name:      "consumer_lag",
		Help:      "Audit records in the stream that the audit service has not read yet.",
	})

	metricsOnce sync.Once
)

func registerMetrics() {
	metricsOnce.Do(func() {
		for _, c := range []prometheus.Collector{recordsTotal, sinkErrorsTotal, consumerLag} {
			if err := prometheus.Register(c); err != nil {
				var are prometheus.AlreadyRegisteredError
				if !errors.As(err, &are) {
					panic(err)
				}
			}
		}
	})
}
