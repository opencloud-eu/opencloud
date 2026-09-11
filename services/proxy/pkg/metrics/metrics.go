package metrics

import (
	"errors"
	"iter"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
	ocmetrics "github.com/opencloud-eu/opencloud/pkg/metrics"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/router"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Namespace defines the namespace for the defines metrics.
	Namespace = "opencloud"

	// Subsystem defines the subsystem for the defines metrics.
	Subsystem = "proxy"
)

// Metrics defines the available metrics of this service.
type Metrics struct {
	routingFailures   prometheus.Counter
	legacyCount       *prometheus.CounterVec
	duration          *prometheus.HistogramVec
	legacyDuration    *prometheus.HistogramVec
	inflightByService map[string]*atomic.Int64
}

const (
	LabelMethod  = "method"
	LabelService = "service"
	LabelResult  = "result"
)

const (
	ResultSuccess     = "success"
	ResultClientError = "client-error"
	ResultServerError = "server-error"
)

func resultFromStatusCode(statusCode int) string {
	if statusCode < 300 {
		return ResultSuccess
	}
	if statusCode < 500 {
		return ResultClientError
	}
	return ResultServerError
}

// New initializes the available metrics.
func New(routes iter.Seq[config.Route], logger *log.Logger) (*Metrics, error) {
	m := &Metrics{
		routingFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "routing_failure_count",
			Help:      "number of inbound requests that could not be routed",
		}),

		// Since we have a higher cardinality with this one, as we have three labels,
		// we really should use Prometheus native histograms, as those are still recorded
		// as singular samples, instead of a matrix of
		//  method ⨯ service ⨯ result ⨯ bucket
		// where bucket is going to have about a dozen values.
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace:                   Namespace,
			Subsystem:                   Subsystem,
			Name:                        "request_duration_seconds",
			Help:                        "request duration in seconds",
			NativeHistogramBucketFactor: 1.1, // native exponential histograms with a 10% maximum bucket width
		}, []string{LabelMethod, LabelService, LabelResult}),

		// In order to still make those measures available to Prometheus scrapers that are
		// not configured to support native histograms (https://prometheus.io/docs/specs/native_histograms/),
		// we also keep these two metrics.
		// Once native histograms become the default in Prometheus scrapers, we could remove those
		// two metrics below, as the one above provides all that data already (including the
		// counter).

		// First, a counter which has the higher cardinality of
		//   method ⨯ service ⨯ result
		// but without buckets, since it's just a counter.
		legacyCount: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "request_total",
			Help:      "total number of requests",
		}, []string{LabelMethod, LabelService, LabelResult}),

		// Secondly, a histogram that buckets the duration, but since this is not a native histogram,
		// we want to keep the cardinality in check by only using the service as label.
		legacyDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "duration_seconds",
			Help:      "request duration in seconds (legacy)",
		}, []string{LabelService}),
	}

	// Initialize the metrics with 0 so that they immediately show up in the list of
	// scraped metrics, instead of only showing up on-demand when they first collect
	// a value later on, and possibly never
	inflightByService := map[string]*atomic.Int64{}
	inflightByServiceGaugeFuncs := map[string]prometheus.GaugeFunc{}
	for route := range routes {
		method := route.Method
		if method == "" {
			method = http.MethodGet
		}
		m.legacyDuration.WithLabelValues(route.Service) // initializes a Histogram as empty

		var counter atomic.Int64
		inflightByService[route.Service] = &counter
		inflightByServiceGaugeFuncs[route.Service] = prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Namespace:   Namespace,
			Subsystem:   Subsystem,
			Name:        "concurrent_service_requests",
			Help:        "number of concurrent requests being processed for a given service",
			ConstLabels: prometheus.Labels{LabelService: route.Service},
		}, func() float64 {
			return float64(counter.Load())
		})

		for _, result := range []string{ResultSuccess, ResultClientError, ResultServerError} {
			m.duration.WithLabelValues(method, route.Service, result)           // initializes a Histogram as empty
			m.legacyCount.WithLabelValues(method, route.Service, result).Add(0) // initializes a Counter as empty
		}
	}

	m.inflightByService = inflightByService

	buildInfo := ocmetrics.BuildInfo(Namespace, Subsystem)

	errs := []error{}
	// registers all the exported metrics:
	errs = append(errs, ocmetrics.RegisterAll(prometheus.DefaultRegisterer, m, logger))
	// need an additional call for the unexported ones:
	errs = append(errs, ocmetrics.RegisterMetrics(prometheus.DefaultRegisterer, logger,
		// need to list unexported metrics here:
		buildInfo,
		m.routingFailures,
		m.duration,
		m.legacyCount,
		m.legacyDuration,
	))
	// need to iterate over these as the number of entries is dynamic:
	{
		collectors := make([]prometheus.Collector, 0, len(inflightByServiceGaugeFuncs))
		for _, c := range inflightByServiceGaugeFuncs {
			collectors = append(collectors, c)
		}
		errs = append(errs, ocmetrics.RegisterMetrics(prometheus.DefaultRegisterer, logger, collectors...))
	}
	return m, errors.Join(errs...)
}

func (m *Metrics) Duration(r *http.Request, statusCode int, duration time.Duration) {
	ri := router.ContextRoutingInfo(r.Context())
	service := ri.Service()
	d := float64(duration.Seconds())
	result := resultFromStatusCode(statusCode)

	m.duration.WithLabelValues(r.Method, service, result).Observe(d)
	m.legacyDuration.WithLabelValues(service).Observe(d)
	m.legacyCount.WithLabelValues(r.Method, service, result).Inc()
}

func (m *Metrics) RoutingFailed(r *http.Request) {
	m.routingFailures.Inc()
}

func (m *Metrics) InFlightInc(r *http.Request) {
	ri := router.ContextRoutingInfo(r.Context())
	service := ri.Service()
	if counter, ok := m.inflightByService[service]; ok {
		counter.Add(1)
	}
}

func (m *Metrics) InFlightDec(r *http.Request) {
	ri := router.ContextRoutingInfo(r.Context())
	service := ri.Service()
	if counter, ok := m.inflightByService[service]; ok {
		counter.Add(-1)
	}
}
