package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// The settings used for the native part of all our histograms,
// see https://prometheus.io/docs/specs/native_histograms/
const (
	// The width of a bucket grows by at most this factor from one bucket to the next,
	// which results in a relative error of at most 10%.
	NativeHistogramBucketFactor = 1.1
	// The maximum number of buckets per native histogram, in order to keep the memory
	// usage in check when the observed values are spread widely: when it is exceeded,
	// the resolution of the native histogram is reduced (until the process restarts).
	NativeHistogramMaxBucketNumber = 100
	// Never reset a native histogram that exceeds NativeHistogramMaxBucketNumber: a reset
	// would set the whole histogram back to zero, including its classic buckets, count and
	// sum, i.e. a counter reset for every scraper, also for those that only ingest the
	// classic histogram.
	NativeHistogramMinResetDuration time.Duration = 0
)

// Add the settings for the native part of a histogram to opts, so that the histogram is
// exposed in both classic and native form (dual exposition).
//
// Prometheus scrapers that don't support native histograms, or have them disabled (the
// default, see the scrape_native_histograms setting), keep ingesting the classic histogram.
// Scrapers that have native histograms enabled ingest the native histogram instead, and
// additionally the classic one if always_scrape_classic_histograms is enabled.
//
// If opts has no Buckets, prometheus.DefBuckets are used, as client_golang would otherwise
// not expose any classic buckets once the native ones are enabled.
func DualHistogramOpts(opts prometheus.HistogramOpts) prometheus.HistogramOpts {
	if len(opts.Buckets) == 0 {
		opts.Buckets = prometheus.DefBuckets
	}
	return withNativeHistogram(opts)
}

// Add the settings for the native part of a histogram to opts, and remove any classic Buckets,
// so that the histogram is only exposed in native form.
//
// This should only be used for histograms with a high cardinality, where classic buckets would
// be too expensive, and the data must then additionally be made available to Prometheus scrapers
// that don't support native histograms, or have them disabled (which is the default, see the
// scrape_native_histograms setting), through other metrics with a lower cardinality: those
// scrapers only ingest the count and the sum of a native-only histogram.
// Such fallback histograms should remain classic-only and not use DualHistogramOpts, as their
// native form would only duplicate what the native-only histogram already provides.
func NativeHistogramOpts(opts prometheus.HistogramOpts) prometheus.HistogramOpts {
	opts.Buckets = nil
	return withNativeHistogram(opts)
}

func withNativeHistogram(opts prometheus.HistogramOpts) prometheus.HistogramOpts {
	opts.NativeHistogramBucketFactor = NativeHistogramBucketFactor
	opts.NativeHistogramMaxBucketNumber = NativeHistogramMaxBucketNumber
	opts.NativeHistogramMinResetDuration = NativeHistogramMinResetDuration
	return opts
}
