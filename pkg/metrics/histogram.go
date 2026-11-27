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

// The upper bounds (in seconds) of the classic buckets of our duration histograms, shared
// by the services so that their histograms can be compared.
//
// Every bucket adds a time series per label combination for the Prometheus scrapers that
// only ingest the classic histogram, so each set is kept to about a dozen buckets.
var (
	// For requests and API calls, which take milliseconds up to tens of seconds.
	RequestDurationBuckets = []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60}
	// For fast operations, such as LDAP operations, which mostly take a few milliseconds.
	FastOperationDurationBuckets = []float64{0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	// For requests that include the transfer of their request or response body, such as the
	// ones going through the proxy, including uploads, downloads and long-lived connections.
	TransferDurationBuckets = []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 300, 1800}
	// For long-running processing, such as postprocessing or indexing, which take from a
	// fraction of a second up to many minutes.
	ProcessingDurationBuckets = []float64{0.1, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300, 600, 1200}
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
