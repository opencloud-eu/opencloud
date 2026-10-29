package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

// Create a histogram from opts, observe a value and return what it would expose.
func observe(t *testing.T, opts prometheus.HistogramOpts) *dto.Histogram {
	h := prometheus.NewHistogram(opts)
	h.Observe(0.42)
	m := &dto.Metric{}
	require.NoError(t, h.Write(m))
	require.NotNil(t, m.GetHistogram())
	return m.GetHistogram()
}

func TestDualHistogramOpts(t *testing.T) {
	tests := []struct {
		name            string
		buckets         []float64
		expectedBuckets []float64
	}{
		{"with buckets", []float64{0.1, 0.5, 1}, []float64{0.1, 0.5, 1}},
		{"without buckets", nil, prometheus.DefBuckets},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)

			opts := DualHistogramOpts(prometheus.HistogramOpts{
				Name:    "test_" + randName(),
				Help:    "test",
				Buckets: tt.buckets,
			})
			require.Equal(NativeHistogramBucketFactor, opts.NativeHistogramBucketFactor)
			require.Equal(uint32(NativeHistogramMaxBucketNumber), opts.NativeHistogramMaxBucketNumber)
			require.Equal(NativeHistogramMinResetDuration, opts.NativeHistogramMinResetDuration)
			// a reset would also set the classic buckets, count and sum back to zero
			require.Zero(opts.NativeHistogramMinResetDuration)

			h := observe(t, opts)

			// classic part
			bounds := []float64{}
			for _, b := range h.GetBucket() {
				bounds = append(bounds, b.GetUpperBound())
			}
			require.Equal(tt.expectedBuckets, bounds)

			// native part
			require.NotNil(h.Schema)
			require.NotEmpty(h.GetPositiveSpan())
			require.EqualValues(1, h.GetSampleCount())
		})
	}
}

func TestNativeHistogramOpts(t *testing.T) {
	require := require.New(t)

	opts := NativeHistogramOpts(prometheus.HistogramOpts{
		Name:    "test_" + randName(),
		Help:    "test",
		Buckets: []float64{0.1, 0.5, 1},
	})
	require.Nil(opts.Buckets)

	h := observe(t, opts)

	// no classic part
	require.Empty(h.GetBucket())

	// native part
	require.NotNil(h.Schema)
	require.NotEmpty(h.GetPositiveSpan())
	require.EqualValues(1, h.GetSampleCount())
}
