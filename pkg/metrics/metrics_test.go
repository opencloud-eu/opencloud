package metrics

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func randName() string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	n := 8 + rand.IntN(33)
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.IntN(len(letterBytes))]
	}
	return string(b)
}

func TestBuildInfo(t *testing.T) {
	require := require.New(t)

	namespace := "name-" + randName()
	subsystem := "sub-" + randName()
	expectedName := fmt.Sprintf("%s_%s_build_info", namespace, subsystem)
	version := fmt.Sprintf("%d.%d.%d", rand.IntN(10), rand.IntN(10), rand.IntN(10))

	g := BuildInfo(namespace, subsystem)
	reg := prometheus.NewRegistry()
	require.NoError(reg.Register(g))

	{
		mfs, err := reg.Gather()
		require.NoError(err)
		require.Len(mfs, 0)
	}

	g.WithLabelValues(version).Set(1)
	{
		mfs, err := reg.Gather()
		require.NoError(err)
		found := false
		for _, mf := range mfs {
			if mf.GetName() == expectedName {
				found = true
				ms := mf.GetMetric()
				require.Len(ms, 1)
				labels := ms[0].GetLabel()
				require.Len(labels, 1)
				require.NotNil(labels[0].Name)
				require.Equal("version", *labels[0].Name)
				require.NotNil(labels[0].Value)
				require.Equal(version, *labels[0].Value)
				require.Equal(1.0, ms[0].GetGauge().GetValue())
			} else {
				t.Fatalf("unexpected metric family %q", mf.GetName())
			}
		}
		require.True(found, "failed to find metric %q", expectedName)
	}
}

func TestRegisterAll(t *testing.T) {
	require := require.New(t)

	reg := prometheus.NewRegistry()
	logger := log.NewLogger()
	namespace := "name-" + randName()
	subsystem := "sub-" + randName()

	m := struct {
		BuildInfo *prometheus.GaugeVec
		Foo       *prometheus.GaugeVec
		Bar       prometheus.Counter
	}{
		BuildInfo: BuildInfo(namespace, subsystem),
		Foo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "foo",
			ConstLabels: prometheus.Labels{
				"f":  "oo",
				"fo": "o",
			},
		}, []string{"oof"}),
		Bar: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "bar",
		}),
	}

	expectedNameForBuildInfo := namespace + "_" + subsystem + "_build_info"
	expectedNameForFoo := namespace + "_" + subsystem + "_foo"
	expectedNameForBar := namespace + "_" + subsystem + "_bar"

	{
		mfs, err := reg.Gather()
		require.NoError(err)
		require.Len(mfs, 0)
	}
	require.NoError(RegisterAll(reg, m, &logger))
	{
		mfs, err := reg.Gather()
		require.NoError(err)
		require.Len(mfs, 3)
		found := 0
		for _, mf := range mfs {
			switch mf.GetName() {
			case expectedNameForBuildInfo:
				found++
				ms := mf.GetMetric()
				require.Len(ms, 1)
				labels := ms[0].GetLabel()
				require.Len(labels, 1)
				require.NotNil(labels[0].Name)
				require.Equal("version", *labels[0].Name)
				require.NotNil(labels[0].Value)
				require.Equal(version.GetString(), *labels[0].Value)
				require.Equal(1.0, ms[0].GetGauge().GetValue())
			case expectedNameForFoo:
				found++
				ms := mf.GetMetric()
				require.Len(ms, 1)
				labels := ms[0].GetLabel()
				require.Len(labels, 3)
				require.NotNil(labels[0].Name)
				require.Equal("f", *labels[0].Name)
				require.NotNil(labels[0].Value)
				require.Equal("oo", *labels[0].Value)
				require.NotNil(labels[1].Name)
				require.Equal("fo", *labels[1].Name)
				require.NotNil(labels[1].Value)
				require.Equal("o", *labels[1].Value)
				require.Equal(0.0, ms[0].GetGauge().GetValue())
			case expectedNameForBar:
				found++
				ms := mf.GetMetric()
				require.Len(ms, 1)
				labels := ms[0].GetLabel()
				require.Len(labels, 0)
				require.Equal(0.0, ms[0].GetGauge().GetValue())
			default:
				t.Fatalf("unexpected metric family %q", mf.GetName())
			}
		}
		require.Equal(3, found, "failed to find expected metrics")
	}
}

// Returns the value of the only sample of the metric family with the given
// name, failing if there is no such family, or if it doesn't have exactly one.
func gatherValue(t *testing.T, reg *prometheus.Registry, name string) float64 {
	t.Helper()
	mfs, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() == name {
			ms := mf.GetMetric()
			require.Len(t, ms, 1)
			return ms[0].GetGauge().GetValue() + ms[0].GetCounter().GetValue()
		}
	}
	t.Fatalf("failed to find metric %q", name)
	return 0
}

type restartMetrics struct {
	Enabled prometheus.Gauge
	Events  prometheus.Counter
}

func newRestartMetrics(namespace, subsystem string) *restartMetrics {
	return &restartMetrics{
		Enabled: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "enabled",
		}),
		Events: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "events_total",
		}),
	}
}

// When the suture supervisor restarts a service, which happens within the same
// process, the service creates and registers its metrics again: those that are
// already registered must then be reused, so that the service updates the ones
// that are exported, and their values continue.
func TestRegisterAfterRestart(t *testing.T) {
	require := require.New(t)

	reg := prometheus.NewRegistry()
	logger := log.NewLogger()
	namespace := "name_" + randName()
	subsystem := "sub_" + randName()

	before, err := Register(reg, newRestartMetrics(namespace, subsystem), &logger)
	require.NoError(err)
	before.Enabled.Set(1)
	before.Events.Add(3)

	after, err := Register(reg, newRestartMetrics(namespace, subsystem), &logger)
	require.NoError(err)
	require.Same(before.Events, after.Events)
	after.Enabled.Set(0)
	after.Events.Inc()

	require.Equal(0.0, gatherValue(t, reg, namespace+"_"+subsystem+"_enabled"))
	require.Equal(4.0, gatherValue(t, reg, namespace+"_"+subsystem+"_events_total"))
}

// Same as TestRegisterAfterRestart, but with a plain Registerer instead of one
// that logs, as some services call RegisterAll directly.
func TestRegisterAllAfterRestart(t *testing.T) {
	require := require.New(t)

	reg := prometheus.NewRegistry()
	logger := log.NewLogger()
	namespace := "name_" + randName()
	subsystem := "sub_" + randName()

	before := newRestartMetrics(namespace, subsystem)
	require.NoError(RegisterAll(reg, before, &logger))
	before.Events.Add(3)

	after := newRestartMetrics(namespace, subsystem)
	require.NoError(RegisterAll(reg, after, &logger))
	after.Events.Inc()

	require.Equal(4.0, gatherValue(t, reg, namespace+"_"+subsystem+"_events_total"))
}

// When the struct is passed as a copy, its attributes can't be replaced with the
// metrics that are already registered: those are replaced in the registry instead.
func TestRegisterAllAfterRestartWithoutPointer(t *testing.T) {
	require := require.New(t)

	reg := prometheus.NewRegistry()
	logger := log.NewLogger()
	namespace := "name_" + randName()
	subsystem := "sub_" + randName()

	before := newRestartMetrics(namespace, subsystem)
	require.NoError(RegisterAll(reg, *before, &logger))
	before.Events.Add(3)

	after := newRestartMetrics(namespace, subsystem)
	require.NoError(RegisterAll(reg, *after, &logger))
	after.Events.Inc()

	require.Equal(1.0, gatherValue(t, reg, namespace+"_"+subsystem+"_events_total"))
}

// Same as TestRegisterAfterRestart, for the additional metrics that are passed
// individually, such as unexported ones: those that are referred to are reused,
// and those whose value is computed by a function, which reads the state of the
// previous run of the service, are replaced.
func TestRegisterAdditionalMetricsAfterRestart(t *testing.T) {
	require := require.New(t)

	reg := prometheus.NewRegistry()
	logger := log.NewLogger()
	namespace := "name_" + randName()
	subsystem := "sub_" + randName()

	newEvents := func() prometheus.Counter {
		return prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "unexported_events_total",
		})
	}
	newInFlight := func(value float64) prometheus.GaugeFunc {
		return prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "in_flight",
		}, func() float64 { return value })
	}

	beforeEvents := newEvents()
	before, err := Register(reg, newRestartMetrics(namespace, subsystem), &logger, WithRef(&beforeEvents), WithFunc(newInFlight(5)))
	require.NoError(err)
	before.Events.Add(3)
	beforeEvents.Add(3)

	afterEvents := newEvents()
	after, err := Register(reg, newRestartMetrics(namespace, subsystem), &logger, WithRef(&afterEvents), WithFunc(newInFlight(2)))
	require.NoError(err)
	require.Same(beforeEvents, afterEvents)
	after.Events.Inc()
	afterEvents.Inc()

	require.Equal(4.0, gatherValue(t, reg, namespace+"_"+subsystem+"_events_total"))
	require.Equal(4.0, gatherValue(t, reg, namespace+"_"+subsystem+"_unexported_events_total"))
	require.Equal(2.0, gatherValue(t, reg, namespace+"_"+subsystem+"_in_flight"))
}

// A metric of another type with the same descriptors can't be reused.
func TestRegisterMetricsWithAnotherType(t *testing.T) {
	require := require.New(t)

	reg := prometheus.NewRegistry()
	opts := prometheus.CounterOpts{Namespace: "name_" + randName(), Name: "events_total"}

	counter := prometheus.NewCounter(opts)
	require.NoError(RegisterMetrics(reg, nil, WithRef(&counter)))

	vec := prometheus.NewCounterVec(opts, []string{})
	require.ErrorContains(RegisterMetrics(reg, nil, WithVecRef(&vec)), "cannot reuse")
}

// Nil metrics and references are rejected instead of panicking.
func TestRegisterMetricsNil(t *testing.T) {
	require := require.New(t)

	reg := prometheus.NewRegistry()
	var missing prometheus.Counter
	var missingVec *prometheus.CounterVec
	require.ErrorContains(RegisterMetrics(reg, nil, WithRef(&missing)), "nil metric")
	require.ErrorContains(RegisterMetrics(reg, nil, WithVecRef(&missingVec)), "nil metric")
	require.ErrorContains(RegisterMetrics(reg, nil, WithRef[prometheus.Counter](nil)), "nil reference")
	require.ErrorContains(RegisterMetrics(reg, nil, WithFunc(nil)), "nil metric")
}
