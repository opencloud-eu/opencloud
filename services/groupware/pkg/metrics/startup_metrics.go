package metrics

import (
	"sync/atomic"

	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/prometheus/client_golang/prometheus"
)

var registered atomic.Bool

func StartupMetrics() {
	// use an atomic boolean to make the operation idempotent,
	// instead of causing a panic in case this function is
	// called twice
	if registered.CompareAndSwap(false, true) {
		// https://github.com/prometheus/common/blob/8558a5b7db3c84fa38b4766966059a7bd5bfa2ee/version/info.go#L36-L56
		prometheus.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "build_info",
			Help:      "Build information",
			ConstLabels: prometheus.Labels{
				"version": version.GetString(),
			},
		}, func() float64 { return 1 }))
	}
}
