// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ErrNilTaskStatus is returned when a nil json.RawMessage is passed to a
// Parse*TaskStatus function.
var ErrNilTaskStatus = errors.New("task status is nil")

// BulkByScrollTaskStatus represents the status of reindex, delete_by_query,
// and update_by_query tasks as defined by the OpenSearch API specification.
//
// All fields are present since OpenSearch 1.0.0 (inherited from Elasticsearch 5.x).
// Fields marked omitempty are conditionally present based on runtime state, not
// server version.
type BulkByScrollTaskStatus struct {
	// SliceID identifies the sub-slice when the request is parallelized.
	// Only present on sub-slice worker tasks; absent on the parent task.
	// Since OpenSearch 1.0.0.
	SliceID *int32 `json:"slice_id,omitempty"`

	// Total is the number of documents that were successfully processed.
	// Since OpenSearch 1.0.0.
	Total int64 `json:"total"`

	// Updated is the number of documents successfully updated.
	// Since OpenSearch 1.0.0.
	Updated int64 `json:"updated"`

	// Created is the number of documents successfully created.
	// Since OpenSearch 1.0.0.
	Created int64 `json:"created"`

	// Deleted is the number of documents successfully deleted.
	// Since OpenSearch 1.0.0.
	Deleted int64 `json:"deleted"`

	// Batches is the number of scroll responses pulled back by the operation.
	// Since OpenSearch 1.0.0.
	Batches int32 `json:"batches"`

	// VersionConflicts is the number of version conflicts encountered.
	// Since OpenSearch 1.0.0.
	VersionConflicts int64 `json:"version_conflicts"`

	// Noops is the number of documents that were ignored.
	// Since OpenSearch 1.0.0.
	Noops int64 `json:"noops"`

	// Retries contains the retry counts for bulk and search sub-operations.
	// Since OpenSearch 1.0.0.
	Retries BulkByScrollTaskStatusRetries `json:"retries"`

	// ThrottledMillis is the total time the request was throttled, in milliseconds.
	// Since OpenSearch 1.0.0.
	ThrottledMillis int64 `json:"throttled_millis"`

	// Throttled is the human-readable throttle duration string.
	// Only present when the request includes ?human=true.
	// Since OpenSearch 1.0.0.
	Throttled string `json:"throttled,omitempty"`

	// RequestsPerSecond is the effective request rate. A value of -1 means unlimited.
	// Since OpenSearch 1.0.0.
	RequestsPerSecond float32 `json:"requests_per_second"`

	// Canceled is the reason the task was cancelled.
	// Only present when the task has been cancelled.
	// Since OpenSearch 1.0.0.
	Canceled string `json:"canceled,omitempty"`

	// ThrottledUntilMillis is when the next throttle window opens, in milliseconds.
	// Since OpenSearch 1.0.0.
	ThrottledUntilMillis int64 `json:"throttled_until_millis"`

	// ThrottledUntil is the human-readable duration until the next throttle window.
	// Only present when the request includes ?human=true.
	// Since OpenSearch 1.0.0.
	ThrottledUntil string `json:"throttled_until,omitempty"`

	// Slices contains per-slice status when the request is parallelized.
	// Each element is either a BulkByScrollTaskStatus (success) or a
	// BulkByScrollTaskException (failure).
	// Only present when the request was sliced.
	// Since OpenSearch 1.0.0.
	Slices []BulkByScrollTaskStatusOrException `json:"slices,omitempty"`
}

// BulkByScrollTaskStatusRetries contains retry statistics for bulk and search
// sub-operations. Since OpenSearch 1.0.0.
type BulkByScrollTaskStatusRetries struct {
	Bulk   int64 `json:"bulk"`
	Search int64 `json:"search"`
}

// BulkByScrollTaskStatusOrException represents either a BulkByScrollTaskStatus
// or an error cause, used in the Slices array. When a slice succeeds, Status is
// populated. When a slice fails, Exception is populated.
// Since OpenSearch 1.0.0.
type BulkByScrollTaskStatusOrException struct {
	Status    *BulkByScrollTaskStatus
	Exception *BulkByScrollTaskException
}

// BulkByScrollTaskException represents an error from a failed bulk-by-scroll slice.
// Since OpenSearch 1.0.0.
type BulkByScrollTaskException struct {
	Type     string          `json:"type"`
	Reason   string          `json:"reason"`
	CausedBy json.RawMessage `json:"caused_by,omitempty"`
}

// UnmarshalJSON implements custom unmarshaling for BulkByScrollTaskStatusOrException.
// The server returns either a status object (with "total" field) or an error cause
// (with "type" field but no "total").
func (s *BulkByScrollTaskStatusOrException) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	var probe struct {
		Type  *string `json:"type"`
		Total *int64  `json:"total"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return fmt.Errorf("failed to probe BulkByScrollTaskStatusOrException: %w", err)
	}

	if probe.Type != nil && probe.Total == nil {
		var exc BulkByScrollTaskException
		if err := json.Unmarshal(data, &exc); err != nil {
			return fmt.Errorf("failed to unmarshal BulkByScrollTaskException: %w", err)
		}
		s.Exception = &exc
		return nil
	}

	var status BulkByScrollTaskStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return fmt.Errorf("failed to unmarshal BulkByScrollTaskStatus: %w", err)
	}
	s.Status = &status
	return nil
}

// MarshalJSON implements custom marshaling for BulkByScrollTaskStatusOrException.
func (s BulkByScrollTaskStatusOrException) MarshalJSON() ([]byte, error) {
	if s.Exception != nil {
		return json.Marshal(s.Exception)
	}
	if s.Status != nil {
		return json.Marshal(s.Status)
	}
	return []byte("null"), nil
}

// ReplicationTaskStatus represents the status of a replication task.
// The phase field indicates the current replication phase (e.g. "starting",
// "indexing", "translog", "finalizing", "done").
// Since OpenSearch 1.0.0.
type ReplicationTaskStatus struct {
	Phase string `json:"phase"`
}

// ResyncTaskStatus represents the status of a primary-replica resync task.
// Since OpenSearch 1.0.0.
type ResyncTaskStatus struct {
	// Phase is the current resync phase.
	// Since OpenSearch 1.0.0.
	Phase string `json:"phase"`

	// TotalOperations is the total number of operations to resync.
	// Since OpenSearch 1.0.0.
	TotalOperations int `json:"totalOperations"`

	// ResyncedOperations is the number of operations already resynced.
	// Since OpenSearch 1.0.0.
	ResyncedOperations int `json:"resyncedOperations"`

	// SkippedOperations is the number of operations skipped during resync.
	// Since OpenSearch 1.0.0.
	SkippedOperations int `json:"skippedOperations"`
}

// PersistentTaskStatus represents the status of a persistent task executor.
// The state field contains the executor state (e.g. "STARTED", "COMPLETED",
// "PENDING_CANCEL").
// Since OpenSearch 1.0.0.
type PersistentTaskStatus struct {
	State string `json:"state"`
}

// ParseTaskStatus unmarshals a task's raw Status field into a concrete type.
// The Status field is polymorphic; its shape depends on the task action.
//
// Supported types: BulkByScrollTaskStatus (reindex, delete_by_query,
// update_by_query), ReplicationTaskStatus, ResyncTaskStatus,
// PersistentTaskStatus, or any struct that can be unmarshaled from JSON.
//
// Returns ErrNilTaskStatus if raw is nil.
func ParseTaskStatus[T any](raw json.RawMessage) (*T, error) {
	if raw == nil {
		return nil, ErrNilTaskStatus
	}
	var s T
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("unmarshaling %T: %w", s, err)
	}
	return &s, nil
}
