// Package metrics provides prometheus metrics for the data managers.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// DownloadsActive is the number of active downloads
	DownloadsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "reva_download_active",
		Help: "Number of active downloads",
	})
	// UploadsActive is the number of active uploads
	UploadsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "reva_upload_active",
		Help: "Number of active uploads",
	})
	// UploadProcessing is the number of uploads in processing
	UploadProcessing = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "reva_upload_processing",
		Help: "Number of uploads in processing",
	})
	// UploadSessionsInitiated is the number of upload sessions that have been initiated
	UploadSessionsInitiated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "reva_upload_sessions_initiated",
		Help: "Number of upload sessions that were initiated",
	})
	// UploadSessionsBytesReceived counts upload sessions whose data has been fully received and is
	// safely stored, before postprocessing decides their outcome; it counts sessions, not bytes,
	// despite its name (kept for compatibility, see https://github.com/opencloud-eu/reva/issues/930)
	UploadSessionsBytesReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "reva_upload_sessions_bytes_received",
		Help: "Number of upload sessions whose data has been fully received, before postprocessing (counts sessions, not bytes)",
	})
	// UploadSessionsFinalized is the number of upload sessions that have successfully completed
	UploadSessionsFinalized = promauto.NewCounter(prometheus.CounterOpts{
		Name: "reva_upload_sessions_finalized",
		Help: "Number of upload sessions that have successfully completed",
	})
	// UploadSessionsAborted is the number of upload sessions that have been aborted
	UploadSessionsAborted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "reva_upload_sessions_aborted",
		Help: "Number of upload sessions that have been aborted by postprocessing",
	})
	// UploadSessionsDeleted is the number of upload sessions that have been deleted
	UploadSessionsDeleted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "reva_upload_sessions_deleted",
		Help: "Number of upload sessions that have been deleted by postprocessing",
	})
	// UploadSessionsRestarted is the number of upload sessions that have been restarted
	UploadSessionsRestarted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "reva_upload_sessions_restarted",
		Help: "Number of upload sessions that have been restarted by postprocessing",
	})
	// UploadSessionsScanned is the number of upload sessions that have been scanned by antivirus
	UploadSessionsScanned = promauto.NewCounter(prometheus.CounterOpts{
		Name: "reva_upload_sessions_scanned",
		Help: "Number of upload sessions that have been scanned by antivirus",
	})
)
