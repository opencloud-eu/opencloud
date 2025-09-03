package metrics

import (
	"reflect"
	"strconv"

	"github.com/opencloud-eu/opencloud/pkg/log"
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
	SessionCacheDesc        *prometheus.Desc
	EventBufferSizeDesc     *prometheus.Desc
	EventBufferQueuedOpts   prometheus.GaugeOpts
	SSESubscribersOpts      prometheus.GaugeOpts
	WorkersBufferSizeDesc   *prometheus.Desc
	WorkersBufferQueuedOpts prometheus.GaugeOpts
	TotalWorkersDesc        *prometheus.Desc
	BusyWorkersOpts         prometheus.GaugeOpts

	JmapErrorCounter             *prometheus.CounterVec
	ParameterErrorCounter        *prometheus.CounterVec
	AuthenticationFailureCounter prometheus.Counter
	SessionFailureCounter        prometheus.Counter
	SSEEventsCounter             *prometheus.CounterVec
	OutdatedSessionsCounter      prometheus.Counter

	SuccessfulRequestPerEndpointCounter              *prometheus.CounterVec
	FailedRequestPerEndpointCounter                  *prometheus.CounterVec
	FailedRequestStatusPerEndpointCounter            *prometheus.CounterVec
	ResponseBodyReadingErrorPerEndpointCounter       *prometheus.CounterVec
	ResponseBodyUnmarshallingErrorPerEndpointCounter *prometheus.CounterVec

	EmailByIdDuration       *prometheus.HistogramVec
	EmailSameSenderDuration *prometheus.HistogramVec
	EmailSameThreadDuration *prometheus.HistogramVec
}

var Labels = struct {
	Endpoint         string
	Result           string
	SessionCacheType string
	RequestId        string
	TraceId          string
	SSEType          string
	ErrorCode        string
	HttpStatusCode   string
}{
	Endpoint:         "endpoint",
	Result:           "result",
	SessionCacheType: "type",
	RequestId:        "requestID",
	TraceId:          "traceID",
	SSEType:          "type",
	ErrorCode:        "code",
	HttpStatusCode:   "statusCode",
}

var Values = struct {
	Result struct {
		Found    string
		NotFound string
		Success  string
		Failure  string
	}
	SessionCache struct {
		Insertions string
		Hits       string
		Misses     string
		Evictions  string
	}
}{
	Result: struct {
		Found    string
		NotFound string
		Success  string
		Failure  string
	}{
		Found:    "found",
		NotFound: "not-found",
		Success:  "success",
		Failure:  "failure",
	},
	SessionCache: struct {
		Insertions string
		Hits       string
		Misses     string
		Evictions  string
	}{
		Insertions: "insertions",
		Hits:       "hits",
		Misses:     "misses",
		Evictions:  "evictions",
	},
}

// New initializes the available metrics.
func New(registerer prometheus.Registerer, logger *log.Logger) *Metrics {
	m := &Metrics{
		SessionCacheDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, Subsystem, "session_cache"),
			"Session cache statistics",
			[]string{Labels.SessionCacheType},
			nil,
		),
		EventBufferSizeDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, Subsystem, "event_buffer_size"),
			"Size of the buffer channel for server-sent events to process",
			nil,
			nil,
		),
		EventBufferQueuedOpts: prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "event_buffer_queued",
			Help:      "Number of queued server-sent events",
		},
		SSESubscribersOpts: prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "sse_subscribers",
			Help:      "Number of subscribers for server-sent event streams",
		},
		WorkersBufferSizeDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, Subsystem, "workers_buffer_size"),
			"Size of the buffer channel for background worker jobs",
			nil,
			nil,
		),
		WorkersBufferQueuedOpts: prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "workers_buffer_queued",
			Help:      "Number of queued background jobs",
		},
		TotalWorkersDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, Subsystem, "workers_total"),
			"Total amount of background job workers",
			nil,
			nil,
		),
		BusyWorkersOpts: prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "workers_busy",
			Help:      "Number of background job workers that are currently busy executing jobs",
		},
		AuthenticationFailureCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "auth_failures_count",
			Help:      "Number of failed authentications",
		}),
		SessionFailureCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "session_failures_count",
			Help:      "Number of session retrieval failures",
		}),
		ParameterErrorCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "param_errors_count",
			Help:      "Number of invalid request parameter errors that occured",
		}, []string{Labels.ErrorCode}),
		JmapErrorCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "jmap_errors_count",
			Help:      "Number of JMAP errors that occured",
		}, []string{Labels.Endpoint, Labels.ErrorCode}),
		SuccessfulRequestPerEndpointCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "jmap_requests_count",
			Help:      "Number of JMAP requests",
			ConstLabels: prometheus.Labels{
				Labels.Result: Values.Result.Success,
			},
		}, []string{Labels.Endpoint}),
		FailedRequestPerEndpointCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "jmap_requests_count",
			Help:      "Number of JMAP requests",
			ConstLabels: prometheus.Labels{
				Labels.Result: Values.Result.Failure,
			},
		}, []string{Labels.Endpoint}),
		FailedRequestStatusPerEndpointCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "jmap_requests_failures_status_count",
			Help:      "Number of JMAP requests",
		}, []string{Labels.Endpoint, Labels.HttpStatusCode}),
		ResponseBodyReadingErrorPerEndpointCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "jmap_requests_body_reading_errors_count",
			Help:      "Number of JMAP body reading errors",
		}, []string{Labels.Endpoint}),
		ResponseBodyUnmarshallingErrorPerEndpointCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "jmap_requests_body_unmarshalling_errors_count",
			Help:      "Number of JMAP body unmarshalling errors",
		}, []string{Labels.Endpoint}),
		SSEEventsCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "sse_events_count",
			Help:      "Number of Server-Side Events that have been sent",
		}, []string{Labels.SSEType}),
		OutdatedSessionsCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "outdated_sessions_count",
			Help:      "Counts outdated session events",
		}),
		EmailByIdDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace:                   Namespace,
			Subsystem:                   Subsystem,
			NativeHistogramBucketFactor: 1.1,
			Name:                        "email_by_id_duration_seconds",
			Help:                        "Duration in seconds for retrieving an Email by its id",
		}, []string{Labels.Endpoint, Labels.Result}),
		EmailSameSenderDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace:                   Namespace,
			Subsystem:                   Subsystem,
			NativeHistogramBucketFactor: 1.1,
			Name:                        "email_same_sender_duration_seconds",
			Help:                        "Duration in seconds for searching for related same-sender Emails",
		}, []string{Labels.Endpoint}),
		EmailSameThreadDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace:                   Namespace,
			Subsystem:                   Subsystem,
			NativeHistogramBucketFactor: 1.1,
			Name:                        "email_same_thread_duration_seconds",
			Help:                        "Duration in seconds for searching for related same-thread Emails",
		}, []string{Labels.Endpoint}),
	}

	registerAll(registerer, m, logger)

	return m
}

func WithExemplar(obs prometheus.Observer, value float64, requestId string, traceId string) {
	obs.(prometheus.ExemplarObserver).ObserveWithExemplar(value, prometheus.Labels{Labels.RequestId: requestId, Labels.TraceId: traceId})
}

func registerAll(registerer prometheus.Registerer, m any, logger *log.Logger) {
	r := reflect.ValueOf(m)
	if r.Kind() == reflect.Pointer {
		r = r.Elem()
	}
	total := 0
	succeeded := 0
	failed := 0
	for i := 0; i < r.NumField(); i++ {
		n := r.Type().Field(i).Name
		f := r.Field(i)
		v := f.Interface()
		c, ok := v.(prometheus.Collector)
		if ok {
			total++
			err := registerer.Register(c)
			if err != nil {
				failed++
				logger.Warn().Err(err).Msgf("failed to register metric '%s' (%T)", n, c)
			} else {
				succeeded++
			}
		}
	}
	logger.Debug().Msgf("registered %d/%d metrics successfully (%d failed)", succeeded, total, failed)
}

type ConstMetricCollector struct {
	Metric prometheus.Metric
}

func (c ConstMetricCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.Metric.Desc()
}
func (c ConstMetricCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- c.Metric
}

type LoggingPrometheusRegisterer struct {
	delegate prometheus.Registerer
	logger   *log.Logger
}

func NewLoggingPrometheusRegisterer(delegate prometheus.Registerer, logger *log.Logger) *LoggingPrometheusRegisterer {
	return &LoggingPrometheusRegisterer{
		delegate: delegate,
		logger:   logger,
	}
}

func (r *LoggingPrometheusRegisterer) Register(c prometheus.Collector) error {
	err := r.delegate.Register(c)
	if err != nil {
		switch err.(type) {
		case prometheus.AlreadyRegisteredError:
			// silently ignore this error, as this case can happen when the suture service decides to restart
			err = nil
		default:
			r.logger.Warn().Err(err).Msgf("failed to register metric")
		}
	}
	return err
}

func (r *LoggingPrometheusRegisterer) MustRegister(collectors ...prometheus.Collector) {
	for _, c := range collectors {
		r.Register(c)
	}
}

func (r *LoggingPrometheusRegisterer) Unregister(c prometheus.Collector) bool {
	return r.delegate.Unregister(c)
}

var _ prometheus.Registerer = &LoggingPrometheusRegisterer{}

func Endpoint(endpoint string) prometheus.Labels {
	return prometheus.Labels{Labels.Endpoint: endpoint}
}

func EndpointAndStatus(endpoint string, status int) prometheus.Labels {
	return prometheus.Labels{Labels.Endpoint: endpoint, Labels.HttpStatusCode: strconv.Itoa(status)}
}
