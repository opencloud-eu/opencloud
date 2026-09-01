package metrics

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/prometheus/client_golang/prometheus"
)

type BuildInfoMetric = *prometheus.GaugeVec

// Create a BuildInfo metric for the specified namespace and subsystem.
func BuildInfo(namespace, subsystem string) BuildInfoMetric {
	return prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "build_info",
		Help:      "Build information",
	}, []string{"version"})
}

// Determine the fully qualified name of a metric.
//
// Beware that this requires storing a value into the metric in order to make it
// visible in a temporary registry.
// If the metric is a MetricVec, it will be Reset().
func describe(metric prometheus.Collector, initialize func() error) (string, error) {
	reg := prometheus.NewRegistry()
	if err := reg.Register(metric); err != nil {
		return "", err
	}
	if err := initialize(); err != nil {
		return "", err
	}
	if resettable, ok := metric.(*prometheus.MetricVec); ok {
		defer resettable.Reset()
	}
	fams, err := reg.Gather()
	if err != nil {
		return "", err
	}
	if len(fams) == 0 {
		return "", fmt.Errorf("no metric families gathered")
	}
	return fams[0].GetName(), nil
}

// Take a struct that contains metrics as attributes and register all of them
// with the specified Registerer.
func RegisterAll(registerer prometheus.Registerer, m any, logger *log.Logger) error {
	// we go over all of them, use this to keep track of succeesses and failures
	total := 0
	succeeded := []string{}
	failed := map[string]error{}

	// we need to use reflection here to iterate over the public metric attributes
	// that are contained in it
	r := reflect.ValueOf(m)
	if r.Kind() == reflect.Pointer {
		r = r.Elem()
	}

	for i := 0; i < r.NumField(); i++ {
		t := r.Type().Field(i)
		n := t.Name // the name of the attribute (not the name of the metric)
		f := r.Field(i)
		if !f.CanInterface() {
			continue // we won't be able to process that one, most probably because it's not exported
		}
		v := f.Interface()
		switch c := v.(type) {
		case prometheus.Collector:
			total++
			if err := registerer.Register(c); err != nil {
				switch err.(type) {
				case prometheus.AlreadyRegisteredError:
					// silently ignore this error, as this case can happen when the suture service decides to restart
					err = nil
					succeeded = append(succeeded, n)
				default:
					failed[n] = err
				}
			} else {
				succeeded = append(succeeded, n)

				// special post-treatment for the BuildInfo metric, as we have that one pretty much
				// everywhere: set its value with the current version so we don't need to do that every time
				switch buildInfo := c.(type) {
				case BuildInfoMetric:
					if name, err := describe(buildInfo, func() error { buildInfo.WithLabelValues("0").Set(0.0); return nil }); err != nil {
						failed[n] = err
					} else if strings.HasSuffix(name, "_build_info") {
						buildInfo.Reset()
						buildInfo.WithLabelValues(version.GetString()).Set(1)
					}
				}
			}
		case *prometheus.Desc,
			prometheus.GaugeOpts,
			prometheus.CounterOpts,
			prometheus.HistogramOpts,
			prometheus.SummaryOpts,
			prometheus.UntypedOpts:
			// skip these
		default:
			failed[n] = fmt.Errorf("unsupported metric '%s' of type %T", n, c)
		}
	}
	if len(failed) > 0 {
		failedMsgs := []string{}
		for name, err := range failed {
			failedMsgs = append(failedMsgs, fmt.Sprintf("'%s' (%v)", name, err))
		}
		msg := strings.Join(failedMsgs, ", ")
		if logger != nil {
			logger.Warn().Msgf("registered %d/%d metrics successfully (%d failed): %s", len(succeeded), total, len(failed), msg)
		}
		return fmt.Errorf("failed to register metrics: %s", msg)
	} else {
		if logger != nil {
			logger.Debug().Msgf("registered %d/%d metrics successfully (%d failed)", len(succeeded), total, len(failed))
		}
		return nil
	}
}

// Register all the metrics that are contained as public attributes in the struct,
// and log any errors that might occur while doing so.
func Register[M any](reg prometheus.Registerer, m M, logger *log.Logger) (M, error) {
	lr := NewLoggingPrometheusRegisterer(reg, logger)
	err := RegisterAll(lr, m, logger)
	return m, err
}

// Register a single metric.
func RegisterMetric[M prometheus.Collector](reg prometheus.Registerer, m M, logger *log.Logger) error {
	return NewLoggingPrometheusRegisterer(reg, logger).Register(m)
}

// Prometheus Registerer wrapper that logs every error that occurs when registering
// a metric, and delegates to an actual Registerer.
type LoggingPrometheusRegisterer struct {
	delegate prometheus.Registerer
	logger   *log.Logger
}

// Instantiate a Prometheus Registerer wrapper that logs every error that occurs when registering
// a metric, and that delegates to an actual Registerer specified here.
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
			if r.logger != nil {
				r.logger.Warn().Err(err).Msgf("failed to register metric")
			}
		}
	}
	return err
}

func (r *LoggingPrometheusRegisterer) MustRegister(collectors ...prometheus.Collector) {
	for _, c := range collectors {
		if err := r.Register(c); err != nil {
			if r.logger != nil {
				r.logger.Error().Err(err).Msg("failed to register metrics collector")
			}
		}
	}
}

func (r *LoggingPrometheusRegisterer) Unregister(c prometheus.Collector) bool {
	return r.delegate.Unregister(c)
}

var _ prometheus.Registerer = &LoggingPrometheusRegisterer{}
