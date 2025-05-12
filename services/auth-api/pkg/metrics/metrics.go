package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	// Namespace defines the namespace for the defines metrics.
	Namespace = "opencloud"

	// Subsystem defines the subsystem for the defines metrics.
	Subsystem = "authapi"
)

// Metrics defines the available metrics of this service.
type Metrics struct {
	BuildInfo *prometheus.GaugeVec
	Duration  *prometheus.HistogramVec
	Attempts  *prometheus.CounterVec
}

const (
	TypeLabel             = "type"
	BasicType             = "basic"
	BearerType            = "bearer"
	UnsupportedType       = "unsupported"
	OutcomeLabel          = "outcome"
	AttemptSuccessOutcome = "success"
	AttemptFailureOutcome = "failure"
)

// New initializes the available metrics.
func New(opts ...Option) *Metrics {
	options := newOptions(opts...)

	m := &Metrics{
		BuildInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "build_info",
			Help:      "Build information",
		}, []string{"version"}),
		Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "authentication_duration_seconds",
			Help:      "Authentication processing time in seconds",
		}, []string{"type"}),
		Attempts: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "athentication_attempts_total",
			Help:      "How many authentication attempts were processed",
		}, []string{"outcome"}),
	}

	if err := prometheus.Register(m.BuildInfo); err != nil {
		options.Logger.Error().
			Err(err).
			Str("metric", "BuildInfo").
			Msg("Failed to register prometheus metric")
	}
	if err := prometheus.Register(m.Duration); err != nil {
		options.Logger.Error().
			Err(err).
			Str("metric", "Duration").
			Msg("Failed to register prometheus metric")
	}
	if err := prometheus.Register(m.Attempts); err != nil {
		options.Logger.Error().
			Err(err).
			Str("metric", "Attempts").
			Msg("Failed to register prometheus metric")
	}
	return m
}
