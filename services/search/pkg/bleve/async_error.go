package bleve

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blevesearch/bleve/v2/index/scorch"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/metrics"
)

const asyncErrorCallbackName = "opencloud-search"

var asyncErrors = newAsyncErrorReporter(metrics.New().BleveAsyncErrors)

// scorch reads its callback registry without a lock whenever an index opens,
// so the callback goes in once here and NewIndex only swaps the logger.
func init() {
	scorch.RegistryAsyncErrorCallbacks[asyncErrorCallbackName] = asyncErrors.report
}

// asyncErrorReporter logs and counts the errors of scorch's merger, persister
// and introducer, which bleve drops otherwise. A failing merge retries several
// times a second, so each source and error type logs at most once a minute.
// Panics always log, and the counter sees every error.
type asyncErrorReporter struct {
	logger  atomic.Pointer[log.Logger]
	counter *prometheus.CounterVec

	mu       sync.Mutex
	samplers map[asyncErrorKey]zerolog.Sampler
}

type asyncErrorKey struct {
	source  string
	errType string
}

func newAsyncErrorReporter(counter *prometheus.CounterVec) *asyncErrorReporter {
	r := &asyncErrorReporter{counter: counter, samplers: map[asyncErrorKey]zerolog.Sampler{}}
	// an index created by NewIndex keeps the callback name in its metadata, so
	// a plain bleve.Open of it can report here before NewIndex set a logger
	nop := log.NopLogger()
	r.logger.Store(&nop)
	return r
}

func (r *asyncErrorReporter) report(err error, path string) {
	var key asyncErrorKey
	var scorchErr *scorch.ScorchError
	if errors.As(err, &scorchErr) {
		key = asyncErrorKey{source: scorchErr.Source, errType: string(scorchErr.ErrType)}
	}
	r.counter.WithLabelValues(key.source, key.errType).Inc()

	if !errors.Is(err, scorch.ErrAsyncPanic) && !r.sampler(key).Sample(zerolog.ErrorLevel) {
		return
	}
	r.logger.Load().Error().Err(err).
		Str("source", key.source).
		Str("err_type", key.errType).
		Str("path", path).
		Msg("bleve index background error")
}

func (r *asyncErrorReporter) sampler(key asyncErrorKey) zerolog.Sampler {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.samplers[key]
	if !ok {
		s = &zerolog.BurstSampler{Burst: 1, Period: time.Minute}
		r.samplers[key] = s
	}
	return s
}
