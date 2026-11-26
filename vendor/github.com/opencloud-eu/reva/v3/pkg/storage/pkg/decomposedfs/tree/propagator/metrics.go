// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package propagator

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// the metrics of the propagation, by the internal root of the storage, see metrics
var (
	queuedChanges = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "reva_propagation_changes_total",
		Help: "Number of changes to be propagated",
	}, []string{"root"})
	directoryUpdates = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "reva_propagation_directory_updates_total",
		Help: "Number of directory updates written by the propagation",
	}, []string{"root"})
	failures = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "reva_propagation_failures_total",
		Help: "Number of failed attempts to apply changes to a directory",
	}, []string{"root"})
)

// metrics are the metrics of a propagator
type metrics struct {
	changes  prometheus.Counter
	updates  prometheus.Counter
	failures prometheus.Counter
}

func newMetrics(internalRoot string) metrics {
	return metrics{
		changes:  queuedChanges.WithLabelValues(internalRoot),
		updates:  directoryUpdates.WithLabelValues(internalRoot),
		failures: failures.WithLabelValues(internalRoot),
	}
}
