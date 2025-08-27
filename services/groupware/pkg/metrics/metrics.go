package metrics

import (
	"reflect"

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
	SessionCacheDesc             *prometheus.Desc
	JmapErrorCounter             *prometheus.CounterVec
	ParameterErrorCounter        *prometheus.CounterVec
	AuthenticationFailureCounter prometheus.Counter
	SessionFailureCounter        prometheus.Counter
	SSEEventsCounter             *prometheus.CounterVec

	SuccessfulRequestPerEndpointCounter   *prometheus.CounterVec
	FailedRequestPerEndpointCounter       *prometheus.CounterVec
	FailedRequestStatusPerEndpointCounter *prometheus.CounterVec

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
func New(logger *log.Logger) *Metrics {
	m := &Metrics{
		SessionCacheDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, Subsystem, "session_cache"),
			"Session cache statistics",
			[]string{Labels.SessionCacheType},
			nil,
		),
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
		SSEEventsCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "sse_events_count",
			Help:      "Number of Server-Side Events that have been sent",
		}, []string{Labels.SSEType}),
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

	r := reflect.ValueOf(*m)
	for i := 0; i < r.NumField(); i++ {
		n := r.Type().Field(i).Name
		f := r.Field(i)
		v := f.Interface()
		c, ok := v.(prometheus.Collector)
		if ok {
			err := prometheus.Register(c)
			if err != nil {
				logger.Warn().Err(err).Msgf("failed to register metric '%s' (%T)", n, c)
			}
		}
	}
	return m
}

func WithExemplar(obs prometheus.Observer, value float64, requestId string, traceId string) {
	obs.(prometheus.ExemplarObserver).ObserveWithExemplar(value, prometheus.Labels{Labels.RequestId: requestId, Labels.TraceId: traceId})
}
