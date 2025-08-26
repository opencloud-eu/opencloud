package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Namespace defines the namespace for the defines metrics.
	Namespace = "opencloud"

	// Subsystem defines the subsystem for the defines metrics.
	Subsystem = "groupware"
)

// Metrics defines the available metrics of this service.
type Metrics struct {
	SessionCacheDesc *prometheus.Desc
	/*SSessionCache     *prometheus.GaugeVec*/
	EmailByIdDuration *prometheus.HistogramVec
}

const (
	ResultFound    = "found"
	ResultNotFound = "not-found"

	SessionCacheTypeInsertions = "insertions"
	SessionCacheTypeHits       = "hits"
	SessionCacheTypeMisses     = "misses"
	SessionCacheTypeEvictions  = "evictions"
)

var Labels = struct {
	Endpoint         string
	Result           string
	SessionCacheType string
	RequestId        string
}{
	Endpoint:         "endpoint",
	Result:           "result",
	SessionCacheType: "type",
	RequestId:        "requestId",
}

// New initializes the available metrics.
func New() *Metrics {
	m := &Metrics{
		SessionCacheDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, Subsystem, "session_cache"),
			"Session cache statistics",
			[]string{Labels.SessionCacheType},
			nil,
		),
		EmailByIdDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			//Buckets:   []float64{0.1, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300, 600, 1200},
			Name: "email_by_id_duration_seconds",
			Help: "Duration in seconds for retrieving an Email by its id",
		}, []string{Labels.Endpoint, Labels.Result}),
	}

	return m
}
