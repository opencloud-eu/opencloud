// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

// Package errmask defines the bitfield used by opensearchapi to mask
// (ignore) specific categories of partial-failure errors.
//
// Each bit corresponds to one wrapper schema in the
// "x-error-responses" extension proposed for the OpenAPI spec
// (see issue-x-partial-failure-mode.md in opensearch-api-specification).
// Until that extension lands upstream, cmd/osgen carries a hardcoded
// per-operation map of which wrappers each operation can return.
//
// Bit semantics: a set bit masks (suppresses) that category; an unset
// bit reports it as a typed Go error. [Empty] (the zero value) reports
// every category; [All] is the union of every defined wrapper bit and
// masks every category. [None] and [Unknown] are doc-friendly aliases
// for [Empty] -- both equal 0.
//
// Callers wire ErrorMask through Config.Errors as a *ErrorMask pointer.
// The pointer's nil/non-nil state is what disambiguates "use the
// version's default" from "caller chose this value":
//
//	cfg.Errors == nil               use the version's default
//	                                  (v4: errmask.All; v5+: errmask.Empty)
//	cfg.Errors == errmask.New()     caller wants every category reported
//	cfg.Errors == errmask.New(All)  caller wants every category masked
//
// Because [Empty] and [All] are constants they are not addressable; use
// [New] to obtain the *ErrorMask the Config field expects (New() with no
// arguments yields a pointer to [Empty]).
//
// The same ErrorMask value is consumed by both the v4 opensearchapi
// package and the generated v5preview/opensearchapi package, so callers
// can reason about error behavior uniformly across the two surfaces.
package errmask

import (
	"strings"
)

// ErrorMask is a bitfield of error categories to mask. A set bit causes
// the API method to return (resp, nil) for that class of partial failure
// instead of the typed error. The zero value [Empty] masks nothing.
type ErrorMask uint32

// New returns a pointer to the mask formed by OR-ing bits together, ready
// to assign to Config.Errors. With no arguments it returns a pointer to
// [Empty] (report every partial-failure category). Constants such as
// [Empty] and [All] are not addressable, so this is the supported way to
// build the *ErrorMask the Config field expects:
//
//	Config{Errors: errmask.New()}                 // report everything
//	Config{Errors: errmask.New(errmask.All)}      // mask everything
//	Config{Errors: errmask.New(                   // composite
//	    errmask.SearchShards | errmask.MultiSearchItems)}
func New(bits ...ErrorMask) *ErrorMask {
	var m ErrorMask
	for _, b := range bits {
		m |= b
	}
	return &m
}

// ErrorMask bits. One bit per wrapper schema in the proposed
// x-error-responses catalog. Append new wrappers at the end so existing
// values remain stable across releases. The set of bits assigned here
// matches the catalog in
// opensearch-api-specification/issue-x-partial-failure-mode.md.
const (
	// BulkItems masks PartialBulkError on bulk-style responses with
	// errors=true and per-item error objects (Bulk, BulkStream).
	BulkItems ErrorMask = 1 << iota

	// SearchShards masks PartialSearchError on the standard _shards
	// envelope of search-family responses (Search, Scroll, SearchTemplate,
	// CreatePIT, Count, AsyncSearch.*).
	SearchShards

	// WriteShards masks ShardFailureError on the _shards envelope of
	// single-document write responses (Index, Update, Delete).
	WriteShards

	// BroadcastShards masks shard failures on the _shards envelope of
	// broadcast responses (Indices.Refresh/Flush/ForceMerge/etc.).
	BroadcastShards

	// NodeFailures masks node-level failures emitted under the _nodes
	// envelope (Cluster.Stats, Nodes.Info/Stats/Usage, etc.).
	NodeFailures

	// BulkByScrollFailures masks the top-level failures[] array in
	// Reindex / UpdateByQuery / DeleteByQuery responses.
	BulkByScrollFailures

	// TaskFailures masks the parallel task_failures[] / node_failures[]
	// arrays returned by Tasks.List and Tasks.Cancel.
	TaskFailures

	// MultiSearchItems masks per-response error objects within
	// responses[] of MSearch / MSearchTemplate.
	MultiSearchItems

	// MultiDocItems masks per-document error objects within docs[]
	// of MGet / MTermvectors.
	MultiDocItems

	// SnapshotCreateShardFailures masks snapshot.failures[] on a
	// Snapshot.Create response (when wait_for_completion=true).
	SnapshotCreateShardFailures

	// SnapshotGetShardFailures masks snapshots[].failures[] on a
	// Snapshot.Get response (per snapshot in the list).
	SnapshotGetShardFailures

	// SimulateDocFailures masks docs[].error and per-processor errors
	// on Ingest.Simulate responses.
	SimulateDocFailures

	// RankEvalFailures masks the top-level failures: { queryId: error }
	// map on _rank_eval responses.
	RankEvalFailures

	// IngestionShardFailures masks the failures: { indexName: [...] }
	// map on Ingestion.Pause / Ingestion.Resume responses.
	IngestionShardFailures

	// PitNodeFailures masks the top-level failures[] array on
	// _core.get_all_pits responses (server-side server quirk: not
	// wrapped in _nodes).
	PitNodeFailures
)

// Empty is the zero value of [ErrorMask]: no categories masked, every
// partial-failure category is reported as a typed Go error.
//
// [Unknown] and [None] are doc-friendly aliases. They all equal 0; the
// nil-vs-non-nil pointer state of Config.Errors is what disambiguates
// "use the version's default" from "caller explicitly set Empty".
const Empty ErrorMask = 0

// Unknown is an alias for [Empty] used at API boundaries where a
// nil-vs-non-nil pointer disambiguates "the caller did not configure
// a mask" from "the caller chose Empty". The resolver substitutes a
// version-specific default for nil pointers (v4 -> [All]; v5 ->
// [Empty]); a non-nil pointer to Unknown is honored verbatim.
const Unknown = Empty

// None is an alias for [Empty]. Reads naturally at call sites that say
// "no categories should be suppressed" without invoking the
// nil-pointer-versus-zero-value subtlety.
const None = Empty

// All is the union of every defined wrapper bit; every category is
// masked.
const All = BulkItems | SearchShards | WriteShards | BroadcastShards |
	NodeFailures | BulkByScrollFailures | TaskFailures |
	MultiSearchItems | MultiDocItems |
	SnapshotCreateShardFailures | SnapshotGetShardFailures |
	SimulateDocFailures | RankEvalFailures | IngestionShardFailures |
	PitNodeFailures

// Token names accepted by [Parse] and emitted by [ErrorMask.String].
//
//nolint:gosec // G101 false positive: these are wire-format error category tokens, not credentials
const (
	TokenBulkItems                   = "bulk_items"
	TokenSearchShards                = "search_shards"
	TokenWriteShards                 = "write_shards"
	TokenBroadcastShards             = "broadcast_shards"
	TokenNodeFailures                = "node_failures"
	TokenBulkByScrollFailures        = "bulk_by_scroll_failures"
	TokenTaskFailures                = "task_failures"
	TokenMultiSearchItems            = "multi_search_items"
	TokenMultiDocItems               = "multi_doc_items"
	TokenSnapshotCreateShardFailures = "snapshot_create_shard_failures"
	TokenSnapshotGetShardFailures    = "snapshot_get_shard_failures"
	TokenSimulateDocFailures         = "simulate_doc_failures"
	TokenRankEvalFailures            = "rank_eval_failures"
	TokenIngestionShardFailures      = "ingestion_shard_failures"
	TokenPitNodeFailures             = "pit_node_failures"

	// TokenAll selects every wrapper bit at once.
	TokenAll = "all"

	// TokenEmpty is the canonical "mask nothing" token. TokenNone and
	// TokenUnknown are accepted as aliases on input; [String] always
	// emits TokenEmpty for a zero mask.
	TokenEmpty   = "empty"
	TokenNone    = "none"
	TokenUnknown = "unknown"
)

// Token prefixes used by [Parse]. A bare token (no prefix) is treated
// as PrefixSet.
const (
	PrefixSet   = '+'
	PrefixClear = '-'
)

// Has reports whether every bit in mask is set on m. At API call sites
// the natural form is `if !m.Has(errmask.BulkItems) { ... }` -- negate
// to mean "this category is reported (not masked)".
func (m ErrorMask) Has(mask ErrorMask) bool {
	return m&mask == mask
}

// String returns a canonical comma-separated token list (snake_case,
// sorted by bit position) of the masked categories. The zero value
// renders as [TokenEmpty]. Useful for logging and the round-trip Parse
// contract.
func (m ErrorMask) String() string {
	if m == Empty {
		return TokenEmpty
	}
	var parts []string
	for _, e := range tokenOrder {
		if m.Has(e.bit) {
			parts = append(parts, e.token)
		}
	}
	return strings.Join(parts, ",")
}

// Parse applies a comma-separated token list to base and returns the
// resulting mask plus a slice of any unknown tokens that were ignored.
// Each token is one of:
//
//	<name>   shorthand for +<name>
//	+<name>  set the bit (mask the category)
//	-<name>  clear the bit (unmask the category)
//
// Recognized names are the snake_case tokens above plus the special
// "all" token (set every wrapper bit) and the "empty"/"none"/"unknown"
// aliases (clear every bit). Tokens are applied left-to-right, so
// callers can express composite changes such as "+all,-bulk_items".
//
// Parse is liberal: unknown tokens are skipped (forward-compatible) and
// returned in the second slice so callers can debug-log them. An empty
// input returns base unchanged with a nil unknown slice.
func Parse(s string, base ErrorMask) (ErrorMask, []string) {
	out := base
	if s == "" {
		return out, nil
	}
	var unknown []string
	for raw := range strings.SplitSeq(s, ",") {
		tok := strings.TrimSpace(raw)
		if tok == "" {
			continue
		}
		set := true
		switch tok[0] {
		case PrefixSet:
			tok = tok[1:]
		case PrefixClear:
			set = false
			tok = tok[1:]
		}
		bit, isReset, ok := nameToBit(tok)
		if !ok {
			unknown = append(unknown, raw)
			continue
		}
		switch {
		case isReset:
			// "empty" / "none" / "unknown" reset to the zero value
			// regardless of +/- prefix.
			out = Empty
		case set:
			out |= bit
		default:
			out &^= bit
		}
	}
	return out, unknown
}

// tokenEntry pairs a bit with its canonical snake_case token. The order
// of this slice defines the deterministic String() output.
type tokenEntry struct {
	bit   ErrorMask
	token string
}

//nolint:gochecknoglobals // canonical bit-order lookup table shared by String and nameToBit
var tokenOrder = []tokenEntry{
	{BulkItems, TokenBulkItems},
	{SearchShards, TokenSearchShards},
	{WriteShards, TokenWriteShards},
	{BroadcastShards, TokenBroadcastShards},
	{NodeFailures, TokenNodeFailures},
	{BulkByScrollFailures, TokenBulkByScrollFailures},
	{TaskFailures, TokenTaskFailures},
	{MultiSearchItems, TokenMultiSearchItems},
	{MultiDocItems, TokenMultiDocItems},
	{SnapshotCreateShardFailures, TokenSnapshotCreateShardFailures},
	{SnapshotGetShardFailures, TokenSnapshotGetShardFailures},
	{SimulateDocFailures, TokenSimulateDocFailures},
	{RankEvalFailures, TokenRankEvalFailures},
	{IngestionShardFailures, TokenIngestionShardFailures},
	{PitNodeFailures, TokenPitNodeFailures},
}

// nameToBit maps a snake_case token to its bit pattern. The second
// return is true for the empty/none/unknown aliases, which always
// reset the mask to zero regardless of +/- prefix. Tokens are matched
// exactly as written -- the canonical lowercase snake_case form is
// the only accepted spelling.
func nameToBit(name string) (ErrorMask, bool, bool) {
	switch name {
	case TokenAll:
		return All, false, true
	case TokenEmpty, TokenNone, TokenUnknown:
		return 0, true, true
	}
	for _, e := range tokenOrder {
		if name == e.token {
			return e.bit, false, true
		}
	}
	return 0, false, false
}
