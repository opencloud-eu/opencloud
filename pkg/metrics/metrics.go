package metrics

import (
	"errors"
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

func postProcess(c prometheus.Collector) error {
	// special post-treatment for the BuildInfo metric, as we have that one pretty much
	// everywhere: set its value with the current version so we don't need to do that every time
	switch buildInfo := c.(type) {
	case BuildInfoMetric:
		if name, err := describe(buildInfo, func() error { buildInfo.WithLabelValues("0").Set(0.0); return nil }); err != nil {
			return err
		} else if strings.HasSuffix(name, "_build_info") {
			buildInfo.Reset()
			buildInfo.WithLabelValues(version.GetString()).Set(1)
		}
	}
	return nil
}

// Take a struct that contains metrics as attributes and register all of them
// with the specified Registerer.
//
// Metrics that are already registered are reused when m is a pointer to the
// struct, and replaced otherwise, see Register.
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
			if err := register(registerer, c, f); err != nil {
				failed[n] = err
			} else {
				succeeded = append(succeeded, n)
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

// Register all the metrics that are contained as public attributes in the struct, as
// well as the additional metrics that are passed individually, such as the ones that
// are not exported, and log any errors that might occur while doing so.
//
// When the suture supervisor restarts a service, which it does within the same process,
// the service creates and registers its metrics again, while those of its previous run
// are still registered. Ignoring that would leave the service updating its new metrics,
// while only the old ones are exported, and never change anymore. Instead, the metrics
// that are already registered are handed back and reused, so that their values continue
// across such restarts: they replace the attributes of the struct when m is a pointer to
// it, and the variables or attributes that additional metrics refer to, see WithRef.
//
// Metrics whose value is computed by a function, such as GaugeFuncs, can't be reused,
// as the function of the registered one reads the state of the previous run of the
// service: they replace the ones that are registered instead, see WithFunc. That is also
// why they must not be public attributes of the struct.
func Register[M any](reg prometheus.Registerer, m M, logger *log.Logger, metrics ...Metric) (M, error) {
	lr := NewLoggingPrometheusRegisterer(reg, logger)
	err := errors.Join(RegisterAll(lr, m, logger), registerMetrics(lr, metrics...))
	return m, err
}

// Register individual metrics, see Register.
func RegisterMetrics(reg prometheus.Registerer, logger *log.Logger, metrics ...Metric) error {
	return registerMetrics(NewLoggingPrometheusRegisterer(reg, logger), metrics...)
}

func registerMetrics(registerer prometheus.Registerer, metrics ...Metric) error {
	errs := []error{}
	for _, m := range metrics {
		errs = append(errs, m.register(registerer))
	}
	return errors.Join(errs...)
}

// A metric that is passed individually to Register or RegisterMetrics, created with
// WithRef, WithVecRef or WithFunc.
type Metric interface {
	register(registerer prometheus.Registerer) error
}

// Refer to the variable or attribute that holds a metric, such as a prometheus.Counter,
// so that it can be set to the metric that is already registered, which is then
// reused, see Register. Use WithVecRef for metric vectors, such as *prometheus.CounterVec.
//
// The constraints let the compiler reject a metric that is passed without its address,
// which could not be handed back.
func WithRef[C interface {
	prometheus.Metric
	prometheus.Collector
}](c *C) Metric {
	return ref[C]{c: c}
}

// Same as WithRef, for metric vectors, such as *prometheus.CounterVec.
//
// Those need their own function, as the vector types embed a pointer, which makes their
// values metrics as well: with a single function, the compiler would accept a vector
// that is passed without its address.
func WithVecRef[V any, C interface {
	*V
	prometheus.Collector
}](c *C) Metric {
	return ref[C]{c: c}
}

type ref[C prometheus.Collector] struct {
	c *C
}

func (r ref[C]) register(registerer prometheus.Registerer) error {
	if r.c == nil {
		return fmt.Errorf("cannot register a nil reference to a metric of type %T", r.c)
	}
	target := reflect.ValueOf(r.c).Elem()
	if k := target.Kind(); (k == reflect.Pointer || k == reflect.Interface) && target.IsNil() {
		return fmt.Errorf("cannot register a nil metric of type %T", *r.c)
	}
	return register(registerer, *r.c, target)
}

// A metric whose value is computed by a function, such as a GaugeFunc, which replaces
// the metric that is already registered, as that one can't be reused, see Register.
func WithFunc(c prometheus.Collector) Metric {
	return fn{c: c}
}

type fn struct {
	c prometheus.Collector
}

func (f fn) register(registerer prometheus.Registerer) error {
	if f.c == nil {
		return fmt.Errorf("cannot register a nil metric")
	}
	return register(registerer, f.c, reflect.Value{})
}

// Register a metric, and if a metric with the same descriptors is already registered,
// reuse it by setting target to it when target can be set, or replace it otherwise,
// see Register.
func register(registerer prometheus.Registerer, c prometheus.Collector, target reflect.Value) error {
	err := registerer.Register(c)
	var are prometheus.AlreadyRegisteredError
	switch {
	case err == nil:
		return postProcess(c)
	case !errors.As(err, &are):
		return err
	case target.CanSet():
		existing := reflect.ValueOf(are.ExistingCollector)
		if !existing.Type().AssignableTo(target.Type()) {
			return fmt.Errorf("cannot reuse the already registered metric of type %T: %w", are.ExistingCollector, err)
		}
		target.Set(existing)
		return nil
	default:
		registerer.Unregister(are.ExistingCollector)
		if err := registerer.Register(c); err != nil {
			return err
		}
		return postProcess(c)
	}
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

// Register a metric with the delegate, and log the errors, except when the metric is
// already registered, as that case is handled by the caller, see Register.
func (r *LoggingPrometheusRegisterer) Register(c prometheus.Collector) error {
	err := r.delegate.Register(c)
	var are prometheus.AlreadyRegisteredError
	if err != nil && !errors.As(err, &are) && r.logger != nil {
		r.logger.Warn().Err(err).Msgf("failed to register metric")
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
