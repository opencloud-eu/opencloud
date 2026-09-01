package metrics

import (
	"github.com/opencloud-eu/opencloud/pkg/log"
	ocmetrics "github.com/opencloud-eu/opencloud/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Namespace defines the namespace for the defines metrics.
	Namespace = "opencloud"

	// Subsystem defines the subsystem for the defines metrics.
	Subsystem = "policies"
)

// Metrics defines the available metrics of this service.
type Metrics struct {
	BuildInfo                     *prometheus.GaugeVec
	EventsEnabled                 prometheus.Gauge
	GrpcEnabled                   prometheus.Gauge
	GrpcEvaluationsThatAllow      *prometheus.CounterVec
	GrpcEvaluationsThatDontAllow  *prometheus.CounterVec
	FailedGrpcEvaluations         *prometheus.CounterVec
	EventEvaluationsThatAllow     *prometheus.CounterVec
	EventEvaluationsThatDontAllow *prometheus.CounterVec
	FailedEventEvaluations        *prometheus.CounterVec
	EventsReceived                *prometheus.CounterVec
	GrpcCallsReceived             *prometheus.CounterVec
}

var Labels = struct {
	Origin string
	Result string
}{
	Origin: "origin",
	Result: "result",
}

var Values = struct {
	Origin struct {
		GRPC  string
		Event string
	}
	Result struct {
		Allowed    string
		NotAllowed string
	}
}{
	Origin: struct {
		GRPC  string
		Event string
	}{
		GRPC:  "grpc",
		Event: "event",
	},
	Result: struct {
		Allowed    string
		NotAllowed string
	}{
		Allowed:    "allowed",
		NotAllowed: "not-allowed",
	},
}

func New(registerer prometheus.Registerer, logger *log.Logger) (*Metrics, error) {
	return ocmetrics.Register(registerer, &Metrics{
		BuildInfo: ocmetrics.BuildInfo(Namespace, Subsystem),
		EventsEnabled: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "events_enabled",
			Help:      "Whether this instance processes events (1) or not (0)",
		}),
		GrpcEnabled: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "grpc_enabled",
			Help:      "Whether this instance processes gRPC API calls (1) or not (0)",
		}),
		GrpcEvaluationsThatAllow: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "policies_processed",
			Help:      "Number of evaluations of policies",
			ConstLabels: prometheus.Labels{
				Labels.Origin: Values.Origin.GRPC,
				Labels.Result: Values.Result.Allowed,
			},
		}, []string{}),
		GrpcEvaluationsThatDontAllow: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "policies_processed",
			Help:      "Number of evaluations of policies",
			ConstLabels: prometheus.Labels{
				Labels.Origin: Values.Origin.GRPC,
				Labels.Result: Values.Result.NotAllowed,
			},
		}, []string{}),
		FailedGrpcEvaluations: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "policies_failures",
			Help:      "Number of failed policy evaluations",
			ConstLabels: prometheus.Labels{
				Labels.Origin: Values.Origin.GRPC,
			},
		}, []string{}),
		EventEvaluationsThatAllow: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "policies_processed",
			Help:      "Number of evaluations of policies",
			ConstLabels: prometheus.Labels{
				Labels.Origin: Values.Origin.Event,
				Labels.Result: Values.Result.Allowed,
			},
		}, []string{}),
		EventEvaluationsThatDontAllow: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "policies_processed",
			Help:      "Number of evaluations of policies",
			ConstLabels: prometheus.Labels{
				Labels.Origin: Values.Origin.Event,
				Labels.Result: Values.Result.NotAllowed,
			},
		}, []string{}),
		FailedEventEvaluations: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "policies_failures",
			Help:      "Number of failed policy evaluations",
			ConstLabels: prometheus.Labels{
				Labels.Origin: Values.Origin.Event,
			},
		}, []string{}),
		EventsReceived: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "policies_requests",
			Help:      "Number of inbound policies requests, by origin",
			ConstLabels: prometheus.Labels{
				Labels.Origin: Values.Origin.Event,
			},
		}, []string{}),
		GrpcCallsReceived: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "policies_requests",
			Help:      "Number of inbound policies requests, by origin",
			ConstLabels: prometheus.Labels{
				Labels.Origin: Values.Origin.GRPC,
			},
		}, []string{}),
	}, logger)
}
