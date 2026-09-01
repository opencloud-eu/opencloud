package metricstest

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// copied and adapted from Prometheus testutil.ToFloat64(), since we don't import that package
func collect(c prometheus.Collector) []prometheus.Metric {
	result := []prometheus.Metric{}
	ch := make(chan prometheus.Metric)
	done := make(chan struct{})
	go func() {
		for m := range ch {
			result = append(result, m)
		}
		close(done)
	}()
	c.Collect(ch)
	close(ch)
	<-done
	return result
}

func RequireIsNotSet(t require.TestingT, c prometheus.Collector, msgAndArgs ...any) {
	if h, ok := t.(interface{ Helper() }); ok {
		h.Helper()
	}
	if !IsNotSet(t, c, msgAndArgs) {
		t.FailNow()
	}
}

func IsNotSet(t assert.TestingT, c prometheus.Collector, msgAndArgs ...any) bool {
	if h, ok := t.(interface{ Helper() }); ok {
		h.Helper()
	}

	m := collect(c)
	if len(m) > 0 {
		return assert.Fail(t, "Metric exists while expected to not exist", msgAndArgs)
	} else {
		return true
	}
}

func RequireEqual(t require.TestingT, expected float64, c prometheus.Collector, msgAndArgs ...any) {
	if h, ok := t.(interface{ Helper() }); ok {
		h.Helper()
	}
	if !Equal(t, expected, c, msgAndArgs) {
		t.FailNow()
	}
}

// copied and adapted from Prometheus testutil.ToFloat64(), since we don't import that package
func Equal(t assert.TestingT, expected float64, c prometheus.Collector, msgAndArgs ...any) bool {
	if h, ok := t.(interface{ Helper() }); ok {
		h.Helper()
	}

	m := collect(c)
	if !assert.Len(t, m, 1, msgAndArgs...) {
		return false
	}
	pb := &dto.Metric{}
	err := m[0].Write(pb)
	if !assert.NoError(t, err, msgAndArgs...) {
		return false
	}
	if pb.Gauge != nil {
		return assert.Equal(t, expected, pb.Gauge.GetValue(), msgAndArgs...)
	} else if pb.Counter != nil {
		return assert.Equal(t, expected, pb.Counter.GetValue(), msgAndArgs...)
	} else if pb.Untyped != nil {
		return assert.Equal(t, expected, pb.Untyped.GetValue(), msgAndArgs...)
	} else {
		return assert.Fail(t, fmt.Sprintf("collected a non-gauge/counter/untyped metric: %s", pb), msgAndArgs...)
	}
}

func RequireEqualWithLabels(t require.TestingT, expectedValue float64, expectedLabels map[string]string, c prometheus.Collector, msgAndArgs ...any) {
	if h, ok := t.(interface{ Helper() }); ok {
		h.Helper()
	}
	if !EqualWithLabels(t, expectedValue, expectedLabels, c, msgAndArgs) {
		t.FailNow()
	}
}

func EqualWithLabels(t assert.TestingT, expectedValue float64, expectedLabels map[string]string, c prometheus.Collector, msgAndArgs ...any) bool {
	if h, ok := t.(interface{ Helper() }); ok {
		h.Helper()
	}

	m := collect(c)
	if !assert.Len(t, m, 1, "collected %d metrics instead of exactly 1", len(m)) {
		return false
	}
	pb := &dto.Metric{}
	err := m[0].Write(pb)
	if !assert.NoError(t, err) {
		return false
	}
	if pb.Gauge != nil {
		if !assert.Equal(t, expectedValue, pb.Gauge.GetValue()) {
			return false
		}
	} else if pb.Counter != nil {
		if !assert.Equal(t, expectedValue, pb.Counter.GetValue()) {
			return false
		}
	} else if pb.Untyped != nil {
		if !assert.Equal(t, expectedValue, pb.Untyped.GetValue()) {
			return false
		}
	} else {
		return assert.Fail(t, "collected a non-gauge/counter/untyped metric: %s", pb)
	}

	if !assert.NotNil(t, pb.Label) {
		return false
	}
	actualLabels := map[string]string{}
	for _, label := range pb.Label {
		if !assert.NotNil(t, label) {
			return false
		}
		if !assert.NotNil(t, label.Name) {
			return false
		}
		if !assert.NotNil(t, label.Value) {
			return false
		}
		actualLabels[*label.Name] = *label.Value
	}
	return assert.Equal(t, expectedLabels, actualLabels, msgAndArgs)
}
