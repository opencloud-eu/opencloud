package metrics

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	merrors "go-micro.dev/v4/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func TestRegisterAllEmbeddedContainer(t *testing.T) {
	require := require.New(t)

	reg := prometheus.NewRegistry()
	logger := log.NewLogger()
	service := "sub-" + randName()

	m, err := New(reg, &logger, service, WithHTTP(true), WithHTTPRequestMetrics(true), WithEvents(true))
	require.NoError(err)
	require.NotNil(m)

	// a service can embed the container to register its own metrics with it
	type svcMetrics struct {
		*Metrics
		Foo *prometheus.CounterVec
	}
	svc := svcMetrics{
		Metrics: m,
		Foo: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: service,
			Name:      "foo_total",
			Help:      "A service-specific metric.",
		}, []string{"label"}),
	}

	// New already registered the container into reg, so RegisterAll has to run
	// against a second registry; otherwise the container's collectors would only
	// take the AlreadyRegistered path and the embedded-container case would go
	// untested.
	svcReg := prometheus.NewRegistry()
	require.NoError(RegisterAll(svcReg, svc, &logger))

	svc.Foo.WithLabelValues("a").Inc()

	fooName := Namespace + "_" + service + "_foo_total"
	{
		mfs := gatherByName(t, svcReg)
		mf := mfs[fooName]
		require.NotNil(mf, "expected metric %q", fooName)
		ms := mf.GetMetric()
		require.Len(ms, 1)
		labels := ms[0].GetLabel()
		require.Len(labels, 1)
		require.Equal("label", labels[0].GetName())
		require.Equal("a", labels[0].GetValue())
		require.Equal(1.0, ms[0].GetCounter().GetValue())
	}

	// the embedded container's own metrics are still exposed exactly once
	{
		mfs := gatherByName(t, svcReg)
		for _, name := range []string{
			Namespace + "_" + service + "_build_info",
			Namespace + "_" + service + "_http_enabled",
			Namespace + "_" + service + "_events_enabled",
		} {
			mf := mfs[name]
			require.NotNil(mf, "expected metric %q", name)
			require.Len(mf.GetMetric(), 1, "expected exactly one series for %q", name)
		}
	}

	// the middleware promoted from the embedded container is usable
	router := chi.NewRouter()
	router.Use(svc.HTTP.Middleware)
	router.Get("/foo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/foo")
	require.NoError(err)
	require.Equal(http.StatusOK, resp.StatusCode)
	require.NoError(resp.Body.Close())

	require.NotNil(gatherByName(t, svcReg)[Namespace+"_"+service+"_http_requests_total"])
}

func TestRegisterAllEmbeddedNilContainer(t *testing.T) {
	require := require.New(t)

	reg := prometheus.NewRegistry()
	logger := log.NewLogger()
	namespace := "name-" + randName()
	subsystem := "sub-" + randName()

	type svcMetrics struct {
		*Metrics
		Bar prometheus.Counter
	}
	svc := svcMetrics{
		Bar: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "bar",
			Help:      "A service-specific metric.",
		}),
	}

	// a nil embedded container is skipped, the struct's own fields still register
	require.NoError(RegisterAll(reg, svc, &logger))

	barName := namespace + "_" + subsystem + "_bar"
	require.NotNil(gatherByName(t, reg)[barName], "expected metric %q", barName)
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
	service := "sub-" + randName()

	m, err := New(reg, &logger, service, WithHTTP(true), WithEvents(true))
	require.NoError(err)
	require.NotNil(m)

	buildInfoName := Namespace + "_" + service + "_build_info"
	httpEnabledName := Namespace + "_" + service + "_http_enabled"
	eventsEnabledName := Namespace + "_" + service + "_events_enabled"
	eventsName := Namespace + "_" + service + "_events"

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

	// the enabled gauges of the declared APIs are exposed and enabled
	{
		mfs := gatherByName(t, reg)
		for _, name := range []string{httpEnabledName, eventsEnabledName} {
			mf := mfs[name]
			require.NotNil(mf, "expected metric %q", name)
			require.Equal(1.0, mf.GetMetric()[0].GetGauge().GetValue())
		}
	}

	// the events counter is only present once there is a series
	require.Nil(gatherByName(t, reg)[eventsName])
	m.Events.RecordSuccess("Foo")
	m.Events.RecordFailure("Bar")
	m.Events.RecordSuccess("Baz")
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

func TestServiceMetricsUndeclaredAPIs(t *testing.T) {
	require := require.New(t)

	reg := prometheus.NewRegistry()
	logger := log.NewLogger()
	service := "sub-" + randName()

	m, err := New(reg, &logger, service, WithHTTP(true))
	require.NoError(err)

	require.NotNil(m.HTTP)
	require.Nil(m.GRPC)
	require.Nil(m.Events)

	// only the declared API is exposed
	mfs := gatherByName(t, reg)
	require.NotNil(mfs[Namespace+"_"+service+"_http_enabled"])
	require.Nil(mfs[Namespace+"_"+service+"_grpc_enabled"])
	require.Nil(mfs[Namespace+"_"+service+"_events_enabled"])

	// recording on an undeclared API is a no-op
	m.Events.RecordSuccess("Foo")
	m.GRPC.RecordDuration("/svc/Method", nil, time.Millisecond)
}

func TestServiceMetricsWithGRPC(t *testing.T) {
	require := require.New(t)

	logger := log.NewLogger()

	for _, enabled := range []bool{true, false} {
		reg := prometheus.NewRegistry()
		service := "sub-" + randName()

		m, err := New(reg, &logger, service, WithGRPC(enabled))
		require.NoError(err)

		require.Equal(boolToFloat64(enabled), gaugeValue(t, m.GRPC.enabled))
		require.NotNil(gatherByName(t, reg)[Namespace+"_"+service+"_grpc_enabled"])
	}
}

func TestServiceMetricsWithHTTP(t *testing.T) {
	require := require.New(t)

	logger := log.NewLogger()

	for _, enabled := range []bool{true, false} {
		reg := prometheus.NewRegistry()
		service := "sub-" + randName()

		m, err := New(reg, &logger, service, WithHTTP(enabled))
		require.NoError(err)

		httpEnabledName := Namespace + "_" + service + "_http_enabled"
		mf := gatherByName(t, reg)[httpEnabledName]
		require.NotNil(mf, "expected metric %q", httpEnabledName)
		require.Equal(boolToFloat64(enabled), mf.GetMetric()[0].GetGauge().GetValue())
		require.Equal(boolToFloat64(enabled), gaugeValue(t, m.HTTP.enabled))
	}
}

func TestServiceMetricsHTTPRequestMetricsOptionOrder(t *testing.T) {
	logger := log.NewLogger()

	tests := []struct {
		name     string
		enabled  bool
		opts     func(enabled bool) []Option
		expected float64
	}{
		{
			name:    "http first, enabled",
			enabled: true,
			opts: func(enabled bool) []Option {
				return []Option{WithHTTP(enabled), WithHTTPRequestMetrics(true)}
			},
			expected: 1,
		},
		{
			name:    "request metrics first, enabled",
			enabled: true,
			opts: func(enabled bool) []Option {
				return []Option{WithHTTPRequestMetrics(true), WithHTTP(enabled)}
			},
			expected: 1,
		},
		{
			name:    "http first, disabled",
			enabled: false,
			opts: func(enabled bool) []Option {
				return []Option{WithHTTP(enabled), WithHTTPRequestMetrics(true)}
			},
			expected: 0,
		},
		{
			name:    "request metrics first, disabled",
			enabled: false,
			opts: func(enabled bool) []Option {
				return []Option{WithHTTPRequestMetrics(true), WithHTTP(enabled)}
			},
			expected: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)

			reg := prometheus.NewRegistry()
			service := "sub-" + randName()

			m, err := New(reg, &logger, service, tt.opts(tt.enabled)...)
			require.NoError(err)

			require.NotNil(m.HTTP)
			require.NotNil(m.HTTP.requests)
			require.Equal(tt.expected, gaugeValue(t, m.HTTP.enabled))

			mfs := gatherByName(t, reg)
			require.NotNil(mfs[Namespace+"_"+service+"_http_requests"])
			require.Equal(tt.expected, mfs[Namespace+"_"+service+"_http_enabled"].GetMetric()[0].GetGauge().GetValue())
		})
	}
}

func TestServiceMetricsRequestMetricsDisabled(t *testing.T) {
	require := require.New(t)

	reg := prometheus.NewRegistry()
	logger := log.NewLogger()
	service := "sub-" + randName()

	m, err := New(reg, &logger, service, WithHTTPRequestMetrics(false), WithGRPCRequestMetrics(false))
	require.NoError(err)

	require.Nil(m.HTTP)
	require.Nil(m.GRPC)
}

func gaugeValue(t *testing.T, g prometheus.Gauge) float64 {
	t.Helper()
	var dm dto.Metric
	require.NoError(t, g.Write(&dm))
	return dm.GetGauge().GetValue()
}

func TestServiceMetricsAlreadyRegistered(t *testing.T) {
	require := require.New(t)

	reg := prometheus.NewRegistry()
	logger := log.NewLogger()
	service := "sub-" + randName()

	_, err := New(reg, &logger, service)
	require.NoError(err)

	// a second construction on the same registry must not fail
	_, err = New(reg, &logger, service)
	require.NoError(err)
}

func TestHTTPMetrics(t *testing.T) {
	require := require.New(t)

	reg := prometheus.NewRegistry()
	logger := log.NewLogger()
	service := "sub-" + randName()

	m, err := New(reg, &logger, service, WithHTTP(true), WithHTTPRequestMetrics(true))
	require.NoError(err)

	durationName := Namespace + "_" + service + "_http_request_duration_seconds"
	requestsName := Namespace + "_" + service + "_http_requests_total"
	inFlightName := Namespace + "_" + service + "_http_requests"

	// nothing is exposed until the first request is observed
	{
		mfs := gatherByName(t, reg)
		require.Nil(mfs[durationName])
		require.Nil(mfs[requestsName])
		require.NotNil(mfs[inFlightName], "in-flight gauge should always be exposed")
		require.Equal(0.0, mfs[inFlightName].GetMetric()[0].GetGauge().GetValue())
	}

	router := chi.NewRouter()
	router.Use(m.HTTP.Middleware)
	router.Get("/foo/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	router.Get("/bar", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	router.Get("/empty", func(w http.ResponseWriter, r *http.Request) {})

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

	resp, err = http.Get(srv.URL + "/empty")
	require.NoError(err)
	require.Equal(http.StatusOK, resp.StatusCode)
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
	require.Equal(1.0, counts[key{"GET", "/empty", "200", ResultSuccess}])
}

func TestHTTPMetricsMiddlewareWithoutRequestMetrics(t *testing.T) {
	require := require.New(t)

	reg := prometheus.NewRegistry()
	logger := log.NewLogger()
	service := "sub-" + randName()

	m, err := New(reg, &logger, service, WithHTTP(true))
	require.NoError(err)

	// without the request metrics enabled, the middleware must be a pass-through
	require.NotNil(m.HTTP)
	router := chi.NewRouter()
	router.Use(m.HTTP.Middleware)
	router.Get("/foo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/foo")
	require.NoError(err)
	require.Equal(http.StatusOK, resp.StatusCode)
	require.NoError(resp.Body.Close())

	require.Nil(gatherByName(t, reg)[Namespace+"_"+service+"_http_requests"])
}

func TestHTTPMetricsMiddlewareNilReceiver(t *testing.T) {
	require := require.New(t)

	var h *httpMetrics

	router := chi.NewRouter()
	router.Use(h.Middleware)
	router.Get("/foo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/foo")
	require.NoError(err)
	require.Equal(http.StatusOK, resp.StatusCode)
	require.NoError(resp.Body.Close())
}

func TestGRPCMetrics(t *testing.T) {
	require := require.New(t)

	reg := prometheus.NewRegistry()
	logger := log.NewLogger()
	service := "sub-" + randName()

	m, err := New(reg, &logger, service, WithGRPC(true), WithGRPCRequestMetrics(true))
	require.NoError(err)

	durationName := Namespace + "_" + service + "_grpc_request_duration_seconds"
	requestsName := Namespace + "_" + service + "_grpc_requests_total"
	inFlightName := Namespace + "_" + service + "_grpc_requests"

	// the in-flight gauge is always exposed, the others only after a call
	{
		mfs := gatherByName(t, reg)
		require.Nil(mfs[durationName])
		require.Nil(mfs[requestsName])
		require.NotNil(mfs[inFlightName])
	}

	m.GRPC.RecordDuration("/svc/Method", nil, 100*time.Millisecond)
	m.GRPC.RecordDuration("/svc/Method", status.Error(codes.NotFound, "nope"), time.Millisecond)
	m.GRPC.RecordDuration("/svc/MicroMethod", merrors.NotFound("svc", "nope"), time.Millisecond)

	mfs := gatherByName(t, reg)
	require.NotNil(mfs[durationName])
	require.NotNil(mfs[requestsName])

	type key struct{ method, code, result string }
	counts := map[key]float64{}
	for _, ms := range mfs[requestsName].GetMetric() {
		labels := map[string]string{}
		for _, l := range ms.GetLabel() {
			labels[l.GetName()] = l.GetValue()
		}
		counts[key{labels[LabelMethod], labels[LabelCode], labels[LabelResult]}] = ms.GetCounter().GetValue()
	}
	require.Equal(1.0, counts[key{"/svc/Method", "200", ResultSuccess}])
	require.Equal(1.0, counts[key{"/svc/Method", "404", ResultClientError}])
	require.Equal(1.0, counts[key{"/svc/MicroMethod", "404", ResultClientError}])
}

func TestGRPCStatusCode(t *testing.T) {
	merr := merrors.NotFound("svc", "x")

	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{name: "nil", err: nil, expected: http.StatusOK},
		{name: "go-micro not found", err: merr, expected: http.StatusNotFound},
		{
			name:     "go-micro internal server error",
			err:      merrors.InternalServerError("svc", "x"),
			expected: http.StatusInternalServerError,
		},
		{
			name:     "wrapped go-micro error",
			err:      fmt.Errorf("lookup failed: %w", merr),
			expected: http.StatusNotFound,
		},
		{name: "grpc not found", err: status.Error(codes.NotFound, "x"), expected: http.StatusNotFound},
		{name: "grpc already exists", err: status.Error(codes.AlreadyExists, "x"), expected: http.StatusConflict},
		{
			name:     "grpc deadline exceeded",
			err:      status.Error(codes.DeadlineExceeded, "x"),
			expected: http.StatusGatewayTimeout,
		},
		{name: "plain error", err: errors.New("plain"), expected: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, grpcStatusCode(tt.err))
		})
	}
}
