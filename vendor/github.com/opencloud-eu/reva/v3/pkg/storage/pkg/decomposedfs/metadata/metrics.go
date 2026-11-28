// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// the metrics of the backends, by the name of the backend, see backendMetrics. A read or write of the
// metadata of a node counts once, however many attributes or files it involves.
var (
	reads = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "reva_metadata_reads_total",
		Help: "Number of times the metadata of a node has been read from the filesystem",
	}, []string{"backend"})
	writes = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "reva_metadata_writes_total",
		Help: "Number of times the metadata of a node has been written to the filesystem",
	}, []string{"backend"})
	cacheLookups = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "reva_metadata_cache_lookups_total",
		Help: "Number of times the metadata of a node has been looked up in the cache, by whether it has been found (result hit) or not (result miss)",
	}, []string{"backend", "result"})
	locksWaiting = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reva_metadata_lock_waiting",
		Help: "Number of metadata locks being waited for",
	}, []string{"backend"})
	lockWait = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "reva_metadata_lock_wait_seconds",
		Help:    "Time spent waiting for metadata locks, by whether the node is a space root",
		Buckets: prometheus.ExponentialBuckets(.0001, 4, 10),
	}, []string{"backend", "space_root"})
	lockHeld = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "reva_metadata_lock_held_seconds",
		Help:    "Time metadata locks have been held, by whether the node is a space root",
		Buckets: prometheus.ExponentialBuckets(.0001, 4, 10),
	}, []string{"backend", "space_root"})
)

// backendMetrics are the metrics of a backend
type backendMetrics struct {
	reads        prometheus.Counter
	writes       prometheus.Counter
	cacheHits    prometheus.Counter
	cacheMisses  prometheus.Counter
	locksWaiting prometheus.Gauge
	// the locks of space roots are the ones every propagation in a space ends at
	spaceRootLocks lockMetrics
	otherLocks     lockMetrics
}

type lockMetrics struct {
	wait prometheus.Observer
	held prometheus.Observer
}

func newBackendMetrics(backend string) *backendMetrics {
	return &backendMetrics{
		reads:        reads.WithLabelValues(backend),
		writes:       writes.WithLabelValues(backend),
		cacheHits:    cacheLookups.WithLabelValues(backend, "hit"),
		cacheMisses:  cacheLookups.WithLabelValues(backend, "miss"),
		locksWaiting: locksWaiting.WithLabelValues(backend),
		spaceRootLocks: lockMetrics{
			wait: lockWait.WithLabelValues(backend, "true"),
			held: lockHeld.WithLabelValues(backend, "true"),
		},
		otherLocks: lockMetrics{
			wait: lockWait.WithLabelValues(backend, "false"),
			held: lockHeld.WithLabelValues(backend, "false"),
		},
	}
}

// cacheLookup counts a lookup in the cache
func (m *backendMetrics) cacheLookup(found bool) {
	if found {
		m.cacheHits.Inc()
	} else {
		m.cacheMisses.Inc()
	}
}

// locks returns the lock metrics for the locks of a node
func (m *backendMetrics) locks(n MetadataNode) lockMetrics {
	if id := n.GetID(); id != "" && id == n.GetSpaceID() {
		return m.spaceRootLocks
	}
	return m.otherLocks
}
