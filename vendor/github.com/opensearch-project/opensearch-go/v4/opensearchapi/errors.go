// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"errors"
	"fmt"
	"strings"
)

// PartialFailureError is implemented by error types that represent partial
// operation success. When a caller receives a PartialFailureError, the
// accompanying response is fully populated — use errors.As to inspect
// the failure details while still processing the partial results.
//
// Use [IsPartialFailure] to test, [ToleratePartialFailures] to suppress,
// or [RequireSuccessRate] for threshold-based tolerance. For exact counts,
// use errors.As with the concrete type ([PartialBulkError],
// [PartialSearchError], or [ShardFailureError]).
type PartialFailureError interface {
	error
	IsPartial() bool
}

// ---------------------------------------------------------------------------
// PartialBulkError
// ---------------------------------------------------------------------------

// PartialBulkError indicates that a bulk operation completed with HTTP 200
// but some items failed. The accompanying BulkResp is fully populated.
type PartialBulkError struct {
	FailedItems    []BulkRespItem
	SucceededCount int
}

//nolint:errcheck // false positive: check-blank flags error-embedding interface assertions
var _ PartialFailureError = (*PartialBulkError)(nil)

func (e *PartialBulkError) Error() string {
	total := e.SucceededCount + len(e.FailedItems)
	return fmt.Sprintf("bulk operation partially failed: %d/%d items failed", len(e.FailedItems), total)
}

// IsPartial implements [PartialFailureError].
func (e *PartialBulkError) IsPartial() bool { return true }

// ---------------------------------------------------------------------------
// PartialSearchError
// ---------------------------------------------------------------------------

// PartialSearchError indicates that a search-family operation returned
// results but some shards failed. The accompanying response is fully
// populated with whatever hits came back from the successful shards.
type PartialSearchError struct {
	FailedShards int
	TotalShards  int
	Failures     []ResponseShardsFailure
}

//nolint:errcheck // false positive: check-blank flags error-embedding interface assertions
var _ PartialFailureError = (*PartialSearchError)(nil)

func (e *PartialSearchError) Error() string {
	return fmt.Sprintf("search partially failed: %d/%d shards failed", e.FailedShards, e.TotalShards)
}

// IsPartial implements [PartialFailureError].
func (e *PartialSearchError) IsPartial() bool { return true }

// ---------------------------------------------------------------------------
// MultiSearchItemError
// ---------------------------------------------------------------------------

// MultiSearchItemFailure captures one failed sub-response inside an
// MSearch / MSearchTemplate result: the position within the Responses
// slice, the per-sub-response HTTP-like status, and the server-side
// error detail.
type MultiSearchItemFailure struct {
	Index  int
	Status int
	Error  *DocumentError
}

// MultiSearchItemError indicates that an MSearch / MSearchTemplate
// response carries one or more sub-queries whose top-level Error field
// is set (a fully-failed sub-query within an otherwise-successful
// envelope). The accompanying response is fully populated; callers can
// inspect successful sub-queries directly while iterating Items for the
// failures.
//
// This is distinct from [PartialSearchError]: shard-level failures are
// masked by the SearchShards bit, while the per-sub-response Error
// surface is masked by the MultiSearchItems bit.
type MultiSearchItemError struct {
	Items          []MultiSearchItemFailure
	SucceededCount int
}

//nolint:errcheck // false positive: check-blank flags error-embedding interface assertions
var _ PartialFailureError = (*MultiSearchItemError)(nil)

func (e *MultiSearchItemError) Error() string {
	total := e.SucceededCount + len(e.Items)
	return fmt.Sprintf("multi-search partially failed: %d/%d sub-queries failed", len(e.Items), total)
}

// IsPartial implements [PartialFailureError].
func (e *MultiSearchItemError) IsPartial() bool { return true }

// ---------------------------------------------------------------------------
// Per-operation error containers
// ---------------------------------------------------------------------------
//
// Operations declared in the OpenAPI spec's x-error-responses extension
// can carry more than one wrapper-shape failure on a single response
// (e.g. MSearch surfaces both per-sub-response shard failures AND
// per-sub-response Error envelopes). The per-op error type aggregates
// every sub-error that fired into a Go 1.20 multi-error, so callers can
// either:
//
//   - use [errors.As] against a concrete sub-error type
//     (works whether the response had one or many sub-errors), or
//   - use [errors.As] against the per-op type to enumerate every sub-error
//     via [errors.Unwrap]'s []error contract.
//
// The dispatch handler applies a runtime-collapse rule:
//
//	0 sub-errors  -> returns nil
//	1 sub-error   -> returns the sub-error directly (no wrapper allocated)
//	2+ sub-errors -> returns the per-op type wrapping the slice
//
// Callers writing for known sub-error types are unaffected by the
// collapse rule -- errors.As walks Unwrap() in the multi case and
// matches directly in the single case.

// MSearchErrors aggregates partial-failure sub-errors from an MSearch
// response. Sub-error types include [PartialSearchError] (per-sub-
// response shard envelope) and [MultiSearchItemError] (per-sub-response
// Error object).
//
//nolint:errname // plural form is intentional: a single response can carry multiple sub-errors
type MSearchErrors struct {
	errs []error
}

func (e *MSearchErrors) Error() string   { return formatPerOpErrors("msearch", e.errs) }
func (e *MSearchErrors) Unwrap() []error { return e.errs }

// IsPartial implements [PartialFailureError].
func (e *MSearchErrors) IsPartial() bool { return true }

//nolint:errcheck // false positive on error-embedding interface assertion
var _ PartialFailureError = (*MSearchErrors)(nil)

// MSearchTemplateErrors aggregates partial-failure sub-errors from an
// MSearchTemplate response. Same shape as [MSearchErrors] (see that
// type's documentation for the collapse rule and caller patterns).
//
//nolint:errname // plural form is intentional; see MSearchErrors
type MSearchTemplateErrors struct {
	errs []error
}

func (e *MSearchTemplateErrors) Error() string   { return formatPerOpErrors("msearch_template", e.errs) }
func (e *MSearchTemplateErrors) Unwrap() []error { return e.errs }

// IsPartial implements [PartialFailureError].
func (e *MSearchTemplateErrors) IsPartial() bool { return true }

//nolint:errcheck // false positive on error-embedding interface assertion
var _ PartialFailureError = (*MSearchTemplateErrors)(nil)

// formatPerOpErrors renders a stable, deterministic error string for a
// per-op aggregate. Used by all <Op>Errors types so their Error()
// outputs share a recognizable prefix and a "; "-separated list of
// sub-error messages.
func formatPerOpErrors(op string, errs []error) string {
	parts := make([]string, len(errs))
	for i, e := range errs {
		parts[i] = e.Error()
	}
	return fmt.Sprintf("%s partial failures: %s", op, strings.Join(parts, "; "))
}

// collapsePerOpErrors implements the runtime-collapse rule documented
// on the per-op error types: returns nil for an empty slice, the lone
// sub-error for a single-element slice, or wrap(errs) otherwise. The
// wrap closure constructs the per-op type from the aggregated errors.
func collapsePerOpErrors(errs []error, wrap func([]error) error) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		if wrap != nil {
			return wrap(errs)
		}
		return errors.Join(errs...)
	}
}

// Errors returns the partial-failure sub-errors carried by err,
// flattening any per-op multi-error wrapper (e.g. [MSearchErrors]) into
// a flat slice.
//
// Use this from caller code so a single switch/default block handles
// both the single-sub-error case (collapse rule returned the bare
// sub-error) and the multi-sub-error case (collapse returned a per-op
// wrapper). Adding new wrapper categories later becomes purely
// additive: a new case in the caller's switch picks up the new type;
// the default keeps catching everything else.
//
//	resp, err := c.MSearch(ctx, req)
//	for _, sub := range opensearchapi.Errors(err) {
//	    switch e := sub.(type) {
//	    case *opensearchapi.PartialSearchError:
//	        // shard aggregation
//	    case *opensearchapi.MultiSearchItemError:
//	        // per-sub-response Error envelope
//	    default:
//	        // transport / HTTP / decoding error
//	    }
//	}
//
// A nil err returns nil. A non-partial err (transport, HTTP, decode)
// returns a single-element slice containing err.
func Errors(err error) []error {
	if err == nil {
		return nil
	}
	return partialSubErrors(err)
}

// partialSubErrors returns the partial-failure sub-errors carried by err.
// For a recognized partial-failure wrapper that implements Unwrap() []error
// (e.g. [MSearchErrors]) it returns the unwrapped slice; for any other
// non-nil err it returns err as a single element. Gating on
// [IsPartialFailure] keeps a joined or third-party multi-error that is not
// itself a partial failure from being silently exploded into its
// sub-errors (which would lose its top-level identity).
func partialSubErrors(err error) []error {
	if IsPartialFailure(err) {
		var multi interface{ Unwrap() []error }
		if errors.As(err, &multi) {
			return multi.Unwrap()
		}
	}
	return []error{err}
}

// ---------------------------------------------------------------------------
// ShardFailureError
// ---------------------------------------------------------------------------

// Write operation names used in [ShardFailureError.Operation].
const (
	OperationIndex  = "index"
	OperationCreate = "create"
	OperationUpdate = "update"
	OperationDelete = "delete"
)

// ShardFailureError indicates that a single-document write operation
// succeeded on the primary shard but one or more replica shards failed.
// The accompanying response is fully populated.
type ShardFailureError struct {
	Operation    string // one of OperationIndex, OperationCreate, OperationUpdate, OperationDelete
	FailedShards int
	TotalShards  int
}

//nolint:errcheck // false positive: check-blank flags error-embedding interface assertions
var _ PartialFailureError = (*ShardFailureError)(nil)

func (e *ShardFailureError) Error() string {
	return fmt.Sprintf("%s had shard failures: %d/%d shards failed", e.Operation, e.FailedShards, e.TotalShards)
}

// IsPartial implements [PartialFailureError].
func (e *ShardFailureError) IsPartial() bool { return true }

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

// IsPartialFailure reports whether err is a partial failure.
func IsPartialFailure(err error) bool {
	var partial PartialFailureError
	return errors.As(err, &partial)
}

// ToleratePartialFailures returns nil for partial failure errors,
// passes through all other errors unchanged.
func ToleratePartialFailures(err error) error {
	if IsPartialFailure(err) {
		return nil
	}
	return err
}

// RequireSuccessRate returns nil if err is a partial failure whose success
// rate is at or above threshold (0.0 to 1.0) in every failure category it
// carries. Non-partial errors pass through unchanged. A nil err returns nil.
//
// A single response can surface more than one partial-failure category at
// once (e.g. an MSearch carrying both shard failures and fully-failed
// sub-queries, wrapped in [MSearchErrors]). Each category is evaluated
// against threshold independently and the first one below threshold is
// returned, so the call succeeds only when every category clears the bar --
// a first-match check would let a healthy shard rate mask a failed-sub-query
// rate (or vice versa).
//
// The success rate is computed from the integer counts on the concrete error
// type: succeeded/total for [PartialBulkError] and [MultiSearchItemError],
// (total-failed)/total for [PartialSearchError] and [ShardFailureError].
func RequireSuccessRate(err error, threshold float64) error {
	if err == nil {
		return nil
	}

	// Evaluate every partial-failure sub-error, not just the first match: a
	// multi-category wrapper (e.g. *MSearchErrors) would otherwise have its
	// later categories ignored by a single errors.As.
	matched := false
	for _, sub := range partialSubErrors(err) {
		succeeded, total, ok := partialSuccessCounts(sub)
		if !ok {
			continue
		}
		matched = true
		if total == 0 {
			return err // degenerate category: cannot verify a rate, stay conservative
		}
		if rate := float64(succeeded) / float64(total); rate < threshold {
			return fmt.Errorf("%d/%d succeeded (%.1f%%) below threshold %.1f%%: %w",
				succeeded, total, rate*100, threshold*100, err)
		}
	}
	if !matched {
		return err
	}
	return nil
}

// partialSuccessCounts returns the succeeded/total counts for a single
// concrete partial-failure error, or ok=false if err is not one. Call it on
// an individual sub-error (not a multi-category wrapper) so errors.As matches
// the concrete type directly rather than the first sub-error of a wrapper.
func partialSuccessCounts(err error) (int, int, bool) {
	var (
		bulkErr    *PartialBulkError
		searchErr  *PartialSearchError
		shardErr   *ShardFailureError
		msearchErr *MultiSearchItemError
	)
	switch {
	case errors.As(err, &bulkErr):
		return bulkErr.SucceededCount, bulkErr.SucceededCount + len(bulkErr.FailedItems), true
	case errors.As(err, &searchErr):
		return searchErr.TotalShards - searchErr.FailedShards, searchErr.TotalShards, true
	case errors.As(err, &shardErr):
		return shardErr.TotalShards - shardErr.FailedShards, shardErr.TotalShards, true
	case errors.As(err, &msearchErr):
		return msearchErr.SucceededCount, msearchErr.SucceededCount + len(msearchErr.Items), true
	}
	return 0, 0, false
}
