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
