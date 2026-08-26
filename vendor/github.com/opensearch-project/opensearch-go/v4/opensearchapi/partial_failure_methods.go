// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"github.com/opensearch-project/opensearch-go/v4/errmask"
)

// This file pairs every operation that can return a partial failure
// with per-Resp helper methods that detect each category. The
// per-category methods (e.g. SearchShardFailures) absorb every
// shape-specific concern: pointer guards on optional `_shards`
// envelopes, per-sub-response iteration, union-branch dispatch (in
// v5preview).
//
// Two surfaces are exposed for each op:
//
//   - Per-category public method, returning *<TypedError> or nil:
//     `func (r *<Op>Resp) <Category>Failures() *<TypedError>`
//
//   - Aggregator method, consulting the caller-supplied mask:
//     `func (r *<Op>Resp) PartialFailures(mask errmask.ErrorMask) []error`
//
// The dispatch handler in api_*.go calls the aggregator + collapses
// via [collapsePerOpErrors] to produce the error returned to callers.
//
// The per-Resp helpers exposed by this file are not the recommended
// way for callers to inspect partial failures. They answer the narrow
// question "did this category happen?" -- a category added in a
// future release is silently missed by call sites that only check the
// existing helpers. The idiomatic pattern is a for/switch over
// [Errors] applied to the dispatch error; see guides/error_handling.md
// (Recommended pattern). The methods are kept here as engine
// machinery for the dispatch and remain available for code that needs
// focused inspection of a known category.

// ---------------------------------------------------------------------------
// Bulk: BulkItems
// ---------------------------------------------------------------------------

// BulkItemFailures detects partial failures on a Bulk response by
// scanning every per-item op for a non-nil Error. Returns nil when no
// items failed.
func (r *BulkResp) BulkItemFailures() *PartialBulkError {
	if r == nil || !r.Errors {
		return nil
	}
	var failed []BulkRespItem
	succeeded := 0
	for _, item := range r.Items {
		for _, v := range item {
			if v.Error != nil {
				failed = append(failed, v)
			} else {
				succeeded++
			}
		}
	}
	if len(failed) == 0 {
		return nil
	}
	return &PartialBulkError{
		FailedItems:    failed,
		SucceededCount: succeeded,
	}
}

// PartialFailures returns the partial-failure sub-errors detected on
// the Bulk response, gated by mask.
func (r *BulkResp) PartialFailures(mask errmask.ErrorMask) []error {
	var errs []error
	if !mask.Has(errmask.BulkItems) {
		if e := r.BulkItemFailures(); e != nil {
			errs = append(errs, e)
		}
	}
	return errs
}

// ---------------------------------------------------------------------------
// Search-family: SearchShards
// ---------------------------------------------------------------------------

// SearchShardFailures detects partial failures on a Search response by
// inspecting the top-level _shards envelope. Returns nil when no
// shards failed.
func (r *SearchResp) SearchShardFailures() *PartialSearchError {
	if r == nil || r.Shards.Failed == 0 {
		return nil
	}
	return &PartialSearchError{
		FailedShards: r.Shards.Failed,
		TotalShards:  r.Shards.Total,
		Failures:     r.Shards.Failures,
	}
}

// PartialFailures returns the partial-failure sub-errors detected on
// the Search response, gated by mask.
func (r *SearchResp) PartialFailures(mask errmask.ErrorMask) []error {
	var errs []error
	if !mask.Has(errmask.SearchShards) {
		if e := r.SearchShardFailures(); e != nil {
			errs = append(errs, e)
		}
	}
	return errs
}

// SearchShardFailures detects partial failures on a Scroll.Get response.
func (r *ScrollGetResp) SearchShardFailures() *PartialSearchError {
	if r == nil || r.Shards.Failed == 0 {
		return nil
	}
	return &PartialSearchError{
		FailedShards: r.Shards.Failed,
		TotalShards:  r.Shards.Total,
		Failures:     r.Shards.Failures,
	}
}

// PartialFailures returns the partial-failure sub-errors detected on
// the Scroll.Get response, gated by mask.
func (r *ScrollGetResp) PartialFailures(mask errmask.ErrorMask) []error {
	var errs []error
	if !mask.Has(errmask.SearchShards) {
		if e := r.SearchShardFailures(); e != nil {
			errs = append(errs, e)
		}
	}
	return errs
}

// SearchShardFailures detects partial failures on a SearchTemplate response.
func (r *SearchTemplateResp) SearchShardFailures() *PartialSearchError {
	if r == nil || r.Shards.Failed == 0 {
		return nil
	}
	return &PartialSearchError{
		FailedShards: r.Shards.Failed,
		TotalShards:  r.Shards.Total,
		Failures:     r.Shards.Failures,
	}
}

// PartialFailures returns the partial-failure sub-errors detected on
// the SearchTemplate response, gated by mask.
func (r *SearchTemplateResp) PartialFailures(mask errmask.ErrorMask) []error {
	var errs []error
	if !mask.Has(errmask.SearchShards) {
		if e := r.SearchShardFailures(); e != nil {
			errs = append(errs, e)
		}
	}
	return errs
}

// ---------------------------------------------------------------------------
// MSearch / MSearchTemplate: SearchShards + MultiSearchItems
//
// These responses carry per-sub-response shard envelopes plus an Error
// field on each sub-response. SearchShards aggregates shard failures
// across success-shaped sub-responses; MultiSearchItems collects
// sub-responses that failed outright (Error != nil).
// ---------------------------------------------------------------------------

// SearchShardFailures detects partial failures by aggregating the
// per-sub-response _shards envelopes on an MSearch result.
func (r *MSearchResp) SearchShardFailures() *PartialSearchError {
	if r == nil {
		return nil
	}
	var totalShards, failedShards int
	var failures []ResponseShardsFailure
	for _, resp := range r.Responses {
		if resp.Error != nil {
			continue
		}
		totalShards += resp.Shards.Total
		failedShards += resp.Shards.Failed
		failures = append(failures, resp.Shards.Failures...)
	}
	if failedShards == 0 {
		return nil
	}
	return &PartialSearchError{
		FailedShards: failedShards,
		TotalShards:  totalShards,
		Failures:     failures,
	}
}

// MultiSearchItemFailures detects per-sub-response Error objects on an
// MSearch result. Returns nil when every sub-response succeeded.
func (r *MSearchResp) MultiSearchItemFailures() *MultiSearchItemError {
	if r == nil {
		return nil
	}
	var failed []MultiSearchItemFailure
	succeeded := 0
	for i, resp := range r.Responses {
		if resp.Error != nil {
			failed = append(failed, MultiSearchItemFailure{
				Index:  i,
				Status: resp.Status,
				Error:  resp.Error,
			})
		} else {
			succeeded++
		}
	}
	if len(failed) == 0 {
		return nil
	}
	return &MultiSearchItemError{
		Items:          failed,
		SucceededCount: succeeded,
	}
}

// PartialFailures returns the partial-failure sub-errors detected on
// the MSearch response, gated by mask. Both wrappers can fire on the
// same response.
func (r *MSearchResp) PartialFailures(mask errmask.ErrorMask) []error {
	var errs []error
	if !mask.Has(errmask.SearchShards) {
		if e := r.SearchShardFailures(); e != nil {
			errs = append(errs, e)
		}
	}
	if !mask.Has(errmask.MultiSearchItems) {
		if e := r.MultiSearchItemFailures(); e != nil {
			errs = append(errs, e)
		}
	}
	return errs
}

// SearchShardFailures detects partial failures on an MSearchTemplate
// response by aggregating per-sub-response shard envelopes.
func (r *MSearchTemplateResp) SearchShardFailures() *PartialSearchError {
	if r == nil {
		return nil
	}
	var totalShards, failedShards int
	var failures []ResponseShardsFailure
	for _, resp := range r.Responses {
		if resp.Error != nil {
			continue
		}
		totalShards += resp.Shards.Total
		failedShards += resp.Shards.Failed
		failures = append(failures, resp.Shards.Failures...)
	}
	if failedShards == 0 {
		return nil
	}
	return &PartialSearchError{
		FailedShards: failedShards,
		TotalShards:  totalShards,
		Failures:     failures,
	}
}

// MultiSearchItemFailures detects per-sub-response Error objects on an
// MSearchTemplate result.
func (r *MSearchTemplateResp) MultiSearchItemFailures() *MultiSearchItemError {
	if r == nil {
		return nil
	}
	var failed []MultiSearchItemFailure
	succeeded := 0
	for i, resp := range r.Responses {
		if resp.Error != nil {
			failed = append(failed, MultiSearchItemFailure{
				Index:  i,
				Status: resp.Status,
				Error:  resp.Error,
			})
		} else {
			succeeded++
		}
	}
	if len(failed) == 0 {
		return nil
	}
	return &MultiSearchItemError{
		Items:          failed,
		SucceededCount: succeeded,
	}
}

// PartialFailures returns the partial-failure sub-errors detected on
// the MSearchTemplate response, gated by mask.
func (r *MSearchTemplateResp) PartialFailures(mask errmask.ErrorMask) []error {
	var errs []error
	if !mask.Has(errmask.SearchShards) {
		if e := r.SearchShardFailures(); e != nil {
			errs = append(errs, e)
		}
	}
	if !mask.Has(errmask.MultiSearchItems) {
		if e := r.MultiSearchItemFailures(); e != nil {
			errs = append(errs, e)
		}
	}
	return errs
}

// ---------------------------------------------------------------------------
// Single-doc writes: WriteShards
// ---------------------------------------------------------------------------

// WriteShardFailures detects replica-shard failures on an Index response.
func (r *IndexResp) WriteShardFailures() *ShardFailureError {
	if r == nil || r.Shards.Failed == 0 {
		return nil
	}
	return &ShardFailureError{
		Operation:    OperationIndex,
		FailedShards: r.Shards.Failed,
		TotalShards:  r.Shards.Total,
	}
}

// PartialFailures returns the partial-failure sub-errors detected on
// the Index response, gated by mask.
func (r *IndexResp) PartialFailures(mask errmask.ErrorMask) []error {
	var errs []error
	if !mask.Has(errmask.WriteShards) {
		if e := r.WriteShardFailures(); e != nil {
			errs = append(errs, e)
		}
	}
	return errs
}

// WriteShardFailures detects replica-shard failures on a
// Document.Create response.
func (r *DocumentCreateResp) WriteShardFailures() *ShardFailureError {
	if r == nil || r.Shards.Failed == 0 {
		return nil
	}
	return &ShardFailureError{
		Operation:    OperationCreate,
		FailedShards: r.Shards.Failed,
		TotalShards:  r.Shards.Total,
	}
}

// PartialFailures returns the partial-failure sub-errors detected on
// the Document.Create response, gated by mask.
func (r *DocumentCreateResp) PartialFailures(mask errmask.ErrorMask) []error {
	var errs []error
	if !mask.Has(errmask.WriteShards) {
		if e := r.WriteShardFailures(); e != nil {
			errs = append(errs, e)
		}
	}
	return errs
}

// WriteShardFailures detects replica-shard failures on a
// Document.Delete response.
func (r *DocumentDeleteResp) WriteShardFailures() *ShardFailureError {
	if r == nil || r.Shards.Failed == 0 {
		return nil
	}
	return &ShardFailureError{
		Operation:    OperationDelete,
		FailedShards: r.Shards.Failed,
		TotalShards:  r.Shards.Total,
	}
}

// PartialFailures returns the partial-failure sub-errors detected on
// the Document.Delete response, gated by mask.
func (r *DocumentDeleteResp) PartialFailures(mask errmask.ErrorMask) []error {
	var errs []error
	if !mask.Has(errmask.WriteShards) {
		if e := r.WriteShardFailures(); e != nil {
			errs = append(errs, e)
		}
	}
	return errs
}

// WriteShardFailures detects replica-shard failures on an Update response.
func (r *UpdateResp) WriteShardFailures() *ShardFailureError {
	if r == nil || r.Shards.Failed == 0 {
		return nil
	}
	return &ShardFailureError{
		Operation:    OperationUpdate,
		FailedShards: r.Shards.Failed,
		TotalShards:  r.Shards.Total,
	}
}

// PartialFailures returns the partial-failure sub-errors detected on
// the Update response, gated by mask.
func (r *UpdateResp) PartialFailures(mask errmask.ErrorMask) []error {
	var errs []error
	if !mask.Has(errmask.WriteShards) {
		if e := r.WriteShardFailures(); e != nil {
			errs = append(errs, e)
		}
	}
	return errs
}
