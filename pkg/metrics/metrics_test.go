package metrics

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
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

func gatherByName(t *testing.T, reg *prometheus.Registry) map[string]*dto.MetricFamily {
	t.Helper()
	mfs, err := reg.Gather()
	require.NoError(t, err)
	out := map[string]*dto.MetricFamily{}
	for _, mf := range mfs {
		out[mf.GetName()] = mf
	}
	return out
}

func TestServiceMetrics(t *testing.T) {
	require := require.New(t)

	reg := prometheus.NewRegistry()
	logger := log.NewLogger()
	namespace := "name-" + randName()
	subsystem := "sub-" + randName()

	m, err := New(reg, &logger, namespace, subsystem,
		WithEnabled(APIHTTP, APIEvents),
		WithEventsProcessed(),
	)
	require.NoError(err)
	require.NotNil(m)

	buildInfoName := namespace + "_" + subsystem + "_build_info"
	httpEnabledName := namespace + "_" + subsystem + "_http_enabled"
	eventsEnabledName := namespace + "_" + subsystem + "_events_enabled"
	grpcEnabledName := namespace + "_" + subsystem + "_grpc_enabled"
	eventsName := namespace + "_" + subsystem + "_events"

	// the build info metric is set to 1 with the current version
	{
		mf := gatherByName(t, reg)[buildInfoName]
		require.NotNil(mf, "expected metric %q", buildInfoName)
		ms := mf.GetMetric()
		require.Len(ms, 1)
		labels := ms[0].GetLabel()
		require.Len(labels, 1)
		require.Equal("version", labels[0].GetName())
		require.Equal(version.GetString(), labels[0].GetValue())
		require.Equal(1.0, ms[0].GetGauge().GetValue())
	}

	// only requested APIs have an enabled gauge, enabled by default
	{
		mfs := gatherByName(t, reg)
		require.Nil(mfs[grpcEnabledName], "did not expect metric %q", grpcEnabledName)
		require.NotNil(mfs[httpEnabledName])
		require.NotNil(mfs[eventsEnabledName])
		require.Equal(1.0, mfs[httpEnabledName].GetMetric()[0].GetGauge().GetValue())
		require.Equal(1.0, mfs[eventsEnabledName].GetMetric()[0].GetGauge().GetValue())
	}

	// SetDisabled flips the gauges, and is a no-op for unrequested APIs
	m.SetDisabled(APIEvents)
	m.SetDisabled(APIGRPC)
	{
		mfs := gatherByName(t, reg)
		require.Equal(1.0, mfs[httpEnabledName].GetMetric()[0].GetGauge().GetValue())
		require.Equal(0.0, mfs[eventsEnabledName].GetMetric()[0].GetGauge().GetValue())
		require.Nil(mfs[grpcEnabledName])
	}

	// the events counter is only present once there is a series
	require.Nil(gatherByName(t, reg)[eventsName])
	m.RecordSuccess("Foo")
	m.RecordFailure("Bar")
	m.RecordSuccess("Baz")
	{
		mf := gatherByName(t, reg)[eventsName]
		require.NotNil(mf, "expected metric %q", eventsName)
		type key struct{ event, result string }
		counts := map[key]float64{}
		for _, ms := range mf.GetMetric() {
			labels := ms.GetLabel()
			require.Len(labels, 2)
			counts[key{labels[0].GetValue(), labels[1].GetValue()}] = ms.GetCounter().GetValue()
		}
		require.Equal(1.0, counts[key{"Foo", ResultSuccess}])
		require.Equal(1.0, counts[key{"Bar", ResultFailure}])
		require.Equal(1.0, counts[key{"Baz", ResultSuccess}])
	}
}

func TestServiceMetricsNoEventsCounter(t *testing.T) {
	require := require.New(t)

	reg := prometheus.NewRegistry()
	logger := log.NewLogger()
	namespace := "name-" + randName()
	subsystem := "sub-" + randName()

	m, err := New(reg, &logger, namespace, subsystem, WithEnabled(APIHTTP))
	require.NoError(err)

	// recording is a no-op without the events counter
	require.NotPanics(func() { m.RecordSuccess("Foo") })
	require.NotPanics(func() { m.RecordFailure("Foo") })

	eventsName := namespace + "_" + subsystem + "_events"
	require.Nil(gatherByName(t, reg)[eventsName])
}

func TestServiceMetricsAlreadyRegistered(t *testing.T) {
	require := require.New(t)

	reg := prometheus.NewRegistry()
	logger := log.NewLogger()
	namespace := "name-" + randName()
	subsystem := "sub-" + randName()

	_, err := New(reg, &logger, namespace, subsystem,
		WithEnabled(APIHTTP, APIEvents),
		WithEventsProcessed(),
	)
	require.NoError(err)

	// a second construction on the same registry must not fail
	_, err = New(reg, &logger, namespace, subsystem,
		WithEnabled(APIHTTP, APIEvents),
		WithEventsProcessed(),
	)
	require.NoError(err)
}

func TestHTTPMetrics(t *testing.T) {
	require := require.New(t)

	reg := prometheus.NewRegistry()
	logger := log.NewLogger()
	namespace := "name-" + randName()
	subsystem := "sub-" + randName()

	m, err := New(reg, &logger, namespace, subsystem, WithHTTPMetrics())
	require.NoError(err)

	durationName := namespace + "_" + subsystem + "_http_request_duration_seconds"
	requestsName := namespace + "_" + subsystem + "_http_requests_total"
	inFlightName := namespace + "_" + subsystem + "_http_requests"

	// nothing is exposed until the first request is observed
	{
		mfs := gatherByName(t, reg)
		require.Nil(mfs[durationName])
		require.Nil(mfs[requestsName])
		require.NotNil(mfs[inFlightName], "in-flight gauge should always be exposed")
		require.Equal(0.0, mfs[inFlightName].GetMetric()[0].GetGauge().GetValue())
	}

	router := chi.NewRouter()
	router.Use(m.HTTPMiddleware)
	router.Get("/foo/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	router.Get("/bar", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/foo/123")
	require.NoError(err)
	require.Equal(http.StatusOK, resp.StatusCode)
	require.NoError(resp.Body.Close())

	resp, err = http.Get(srv.URL + "/bar")
	require.NoError(err)
	require.Equal(http.StatusInternalServerError, resp.StatusCode)
	require.NoError(resp.Body.Close())

	mfs := gatherByName(t, reg)
	require.NotNil(mfs[durationName])
	require.NotNil(mfs[requestsName])

	type key struct{ method, path, code, result string }
	counts := map[key]float64{}
	for _, ms := range mfs[requestsName].GetMetric() {
		labels := map[string]string{}
		for _, l := range ms.GetLabel() {
			labels[l.GetName()] = l.GetValue()
		}
		counts[key{labels[LabelMethod], labels[LabelPath], labels[LabelCode], labels[LabelResult]}] = ms.GetCounter().GetValue()
	}
	require.Equal(1.0, counts[key{"GET", "/foo/{id}", "200", ResultSuccess}])
	require.Equal(1.0, counts[key{"GET", "/bar", "500", ResultServerError}])
}

func TestHTTPMetricsMiddlewareWithoutMetrics(t *testing.T) {
	require := require.New(t)

	reg := prometheus.NewRegistry()
	logger := log.NewLogger()
	namespace := "name-" + randName()
	subsystem := "sub-" + randName()

	m, err := New(reg, &logger, namespace, subsystem)
	require.NoError(err)

	// without WithHTTPMetrics the middleware must be a pass-through
	router := chi.NewRouter()
	router.Use(m.HTTPMiddleware)
	router.Get("/foo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/foo")
	require.NoError(err)
	require.Equal(http.StatusOK, resp.StatusCode)
	require.NoError(resp.Body.Close())

	require.Nil(gatherByName(t, reg)[namespace+"_"+subsystem+"_http_requests"])
}
