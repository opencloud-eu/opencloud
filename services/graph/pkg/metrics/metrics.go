package metrics

import (
	"errors"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
	ocmetrics "github.com/opencloud-eu/opencloud/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Namespace defines the namespace for the defines metrics.
	Namespace = "opencloud"

	// Subsystem defines the subsystem for the defines metrics.
	Subsystem = "graph"
)

// Metrics defines the available metrics of this service.
type Metrics struct {
	BuildInfo           *prometheus.GaugeVec
	EventsEnabled       prometheus.Gauge
	HttpEnabled         prometheus.Gauge
	EventsProcessed     *prometheus.CounterVec
	InvalidEvents       prometheus.Counter
	UnsupportedEvents   prometheus.Counter
	UserPasswordChanges *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpPathSplitter    func(pieces []string) (string, string)
}

const (
	ResultSuccess     = "success"
	ResultFailure     = "failure"
	ResultNotFound    = "not-found"
	ResultReadOnly    = "read-only"
	ResultClientError = "client-error"
	ResultServerError = "server-error"
)

const (
	LabelMethod    = "method"
	LabelPath      = "path"
	LabelVersion   = "version"
	LabelResource  = "resource"
	LabelCode      = "code"
	LabelResult    = "result"
	LabelReason    = "reason"
	LabelEvent     = "event"
	LabelOperation = "operation"
)

const (
	ReasonInvalid       = "invalid"
	ReasonError         = "error"
	ReasonWrongPassword = "wrong-password"
)

const (
	UnmatchedRoutePattern = "unknown"
)

// New initializes the available metrics.
func New(registerer prometheus.Registerer, logger *log.Logger, httpPathSplitter func(pieces []string) (string, string)) (*Metrics, error) {
	m := &Metrics{
		BuildInfo: ocmetrics.BuildInfo(Namespace, Subsystem),
		EventsEnabled: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "events_enabled",
			Help:      "Whether this instance consumes events (1) or not (0)",
		}),
		HttpEnabled: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "http_enabled",
			Help:      "Whether this instance processes HTTP API calls (1) or not (0)",
		}),
		EventsProcessed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "events",
			Help:      "Number of consumed events",
		}, []string{LabelEvent, LabelResult}),
		InvalidEvents: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "events_invalid",
			Help:      "Number of supported events with invalid data",
		}),
		UnsupportedEvents: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "events_unsupported",
			Help:      "Number of unsupported events that were consumed and ignored",
		}),
		UserPasswordChanges: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "user_password_changes",
			Help:      "Counts occurences of users changing their password",
		}, []string{LabelResult, LabelReason}),
		// keeping this one private as it should only be used via the RecordHTTPDuration() method below,
		// as its number of labels is too fragile to keep in check if they ever change
		httpRequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "http_request_duration_seconds",
			Help:      "Duration of HTTP operations in seconds.",
			Buckets:   prometheus.DefBuckets,
		}, []string{LabelMethod, LabelPath, LabelVersion, LabelResource, LabelCode, LabelResult}), // when changing these, make sure to also modify the methods below accordingly
		httpPathSplitter: httpPathSplitter,
	}

	m, err := ocmetrics.Register(registerer, m, logger)
	// must additionally register unexported metrics:
	err = errors.Join(err, ocmetrics.RegisterMetric(registerer, m.httpRequestDuration, logger))
	return m, err
}

func (m Metrics) InitHttpInFlightGauge(inFlight *atomic.Int64) {
	_ = prometheus.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: Subsystem,
		Unit:      "requests",
		Name:      "http_requests",
		Help:      "Concurrent inbound HTTP requests.",
	}, func() float64 {
		return float64(inFlight.Load())
	}))
}

func (m Metrics) RecordHTTPDuration(method string, pattern string, statusCode int, duration time.Duration) {
	result := ""
	if statusCode < 400 {
		result = ResultSuccess
	} else if statusCode < 500 {
		result = ResultClientError
	} else {
		result = ResultServerError
	}
	// all the HTTP routes for the Graph API start with a version (v1.0 or v1beta1), followed by a top level
	// resource "module", which might be useful to extract and include as a label, to aggregate metrics and
	// statistics before drilling down further
	pieces := strings.FieldsFunc(pattern, func(r rune) bool {
		return r == '/'
	})
	version, resource := m.httpPathSplitter(pieces)
	m.httpRequestDuration.WithLabelValues(method, pattern, version, resource, strconv.Itoa(statusCode), result).Observe(duration.Seconds())
}
