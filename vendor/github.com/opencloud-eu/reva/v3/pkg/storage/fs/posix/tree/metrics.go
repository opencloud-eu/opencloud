package tree

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// constants for AssimilationCounter labels
	_labelFile    = "file"
	_labelDir     = "dir"
	_labelAdded   = "added"
	_labelUpdated = "updated"
	_labelDeleted = "deleted"
	_labelMoved   = "moved"

	// AssimilationCounter is a Prometheus counter that tracks the number of files and directories assimilated by posixfs.
	AssimilationCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "reva_assimilation_count",
		Help: "Number of files and directories assimilated by posixfs",
	},
		// type can be "file" or "dir"
		// action can be "added", "updated", "deleted", "moved"
		[]string{"type", "action"},
	)

	// AssimilationPendingTasks is a Prometheus gauge that tracks the number of active assimilation tasks.
	AssimilationPendingTasks = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "reva_assimilation_active_tasks",
		Help: "Number of active assimilation tasks in posixfs",
	})
)
