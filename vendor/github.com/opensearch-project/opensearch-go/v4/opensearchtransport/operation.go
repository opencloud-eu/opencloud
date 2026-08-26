// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.
//
// Modifications Copyright OpenSearch Contributors. See
// GitHub history for details.

// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package opensearchtransport

import "strconv"

// OperationID is a bit-packed int64 that identifies an OpenSearch operation.
//
// Layout:
//
//	 63  62       56  55       40  39       32  31                0
//	┌───┬──────────┬────────────┬────────────┬─────────────────────┐
//	│R/W│ Category │  Reserved  │  Reserved  │   Minor operation   │
//	│1b │   7b     │   16b      │    8b      │       32b           │
//	└───┴──────────┴────────────┴────────────┴─────────────────────┘
//	     ◄── 24 bits: major ──►  ◄── gap ──► ◄──── 32b: minor ───►
//
// Bit 63:    R/W flag (0=read, 1=write)
// Bits 62-56: Category (7 bits, 128 per R/W, 256 total)
// Bits 55-40: Reserved for future sub-categories or flags
// Bits 39-32: Reserved gap
// Bits 31-0:  Minor operation (sequential per category)
type OperationID int64

// Bit layout constants.
const (
	opRWBit                = 63
	opRWMask   OperationID = -1 << opRWBit // 0x8000_0000_0000_0000 (sign bit)
	opCatShift             = 56
	opCatMask  OperationID = 0x7F << opCatShift    // 0x7F00_0000_0000_0000
	opMinMask  OperationID = 0x0000_0000_FFFF_FFFF // low 32 bits
	opMajMask  OperationID = opRWMask | opCatMask  // top 8 bits (R/W + category)
)

// IsWrite reports whether the operation is a write (mutating) operation.
func (op OperationID) IsWrite() bool { return op&opRWMask != 0 }

// IsRead reports whether the operation is a read-only operation.
func (op OperationID) IsRead() bool { return op&opRWMask == 0 }

// Category returns the major category bits (R/W + category slot).
func (op OperationID) Category() OperationID { return op & opMajMask }

// Minor returns the minor operation identifier within the category.
func (op OperationID) Minor() OperationID { return op & opMinMask }

// ---------------------------------------------------------------------------
// Read categories (bit 63 = 0)
// ---------------------------------------------------------------------------

// CatCluster and the following constants define read-side operation categories.
const (
	CatCluster  OperationID = iota << opCatShift //  0: Cluster info and management reads
	CatSearch                                    //  1: Search, msearch, count, etc.
	CatScroll                                    //  2: Scroll reads
	CatPIT                                       //  3: Point-in-time reads
	CatDocRead                                   //  4: Document retrieval
	CatIndex                                     //  5: Index metadata reads
	CatMapping                                   //  6: Mapping reads
	CatAlias                                     //  7: Alias reads
	CatTemplate                                  //  8: Template reads
	CatMaint                                     //  9: Maintenance reads (segments, recovery, stats)
	CatIngest                                    // 10: Ingest pipeline reads
	CatAdmin                                     // 11: Administrative reads (cat, nodes, tasks, etc.)
	CatPing                                      // 12: Ping

	catBulkSlot OperationID = 13 << opCatShift // 13: Bulk (write-only, no read equivalent)
)

// ---------------------------------------------------------------------------
// Write categories (bit 63 = 1)
// ---------------------------------------------------------------------------

// CatClusterWrite and the following constants define write-side operation categories.
const (
	CatClusterWrite  = opRWMask | CatCluster
	CatScrollWrite   = opRWMask | CatScroll
	CatPITWrite      = opRWMask | CatPIT
	CatDocWrite      = opRWMask | CatDocRead // reuses slot 4; R/W bit distinguishes
	CatBulk          = opRWMask | catBulkSlot
	CatIndexWrite    = opRWMask | CatIndex
	CatMappingWrite  = opRWMask | CatMapping
	CatAliasWrite    = opRWMask | CatAlias
	CatTemplateWrite = opRWMask | CatTemplate
	CatMaintWrite    = opRWMask | CatMaint
	CatIngestWrite   = opRWMask | CatIngest
	CatAdminWrite    = opRWMask | CatAdmin
)

// ---------------------------------------------------------------------------
// Minor operations — Search (CatSearch)
// ---------------------------------------------------------------------------

const (
	minSearch OperationID = iota + 1
	minMSearch
	minCount
	minSearchTemplate
	minMSearchTemplate
	minValidate
	minRankEval
	minExplain
	minSearchShards
	minFieldCaps
)

// ---------------------------------------------------------------------------
// Minor operations — Scroll (CatScroll / CatScrollWrite)
// ---------------------------------------------------------------------------

const (
	minScrollGet OperationID = iota + 1
	minScrollDelete
)

// ---------------------------------------------------------------------------
// Minor operations — PIT (CatPIT / CatPITWrite)
// ---------------------------------------------------------------------------

const (
	minPITList OperationID = iota + 1
	minPITCreate
	minPITDelete
)

// ---------------------------------------------------------------------------
// Minor operations — Document read (CatDocRead)
// ---------------------------------------------------------------------------

const (
	minDocGet OperationID = iota + 1
	minDocExists
	minDocSourceGet
	minDocSourceExists
	minMGet
	minTermVectors
	minMTermVectors
)

// ---------------------------------------------------------------------------
// Minor operations — Document write (CatDocWrite)
// ---------------------------------------------------------------------------

const (
	minDocIndex OperationID = iota + 1
	minDocCreate
	minDocUpdate
	minDocDelete
)

// ---------------------------------------------------------------------------
// Minor operations — Bulk (CatBulk)
// ---------------------------------------------------------------------------

const (
	minBulk OperationID = iota + 1
	minBulkStream
	minReindex
	minDeleteByQuery
	minUpdateByQuery
	minReindexRethrottle
	minDBQRethrottle
	minUBQRethrottle
)

// ---------------------------------------------------------------------------
// Minor operations — Index management (CatIndex / CatIndexWrite)
// ---------------------------------------------------------------------------

const (
	minIndexGet OperationID = iota + 1
	minIndexExists
	minIndexCreate
	minIndexDelete
	minIndexOpen
	minIndexClose
	minIndexClone
	minIndexShrink
	minIndexSplit
	minIndexRollover
	minIndexBlock
	minIndexResolve
	minIndexAnalyze
)

// ---------------------------------------------------------------------------
// Minor operations — Mapping (CatMapping / CatMappingWrite)
// ---------------------------------------------------------------------------

const (
	minMappingGet OperationID = iota + 1
	minMappingPut
)

// ---------------------------------------------------------------------------
// Minor operations — Alias (CatAlias / CatAliasWrite)
// ---------------------------------------------------------------------------

const (
	minAliasGet OperationID = iota + 1
	minAliasPut
	minAliasDelete
	minCatAliases
)

// ---------------------------------------------------------------------------
// Minor operations — Template (CatTemplate / CatTemplateWrite)
// ---------------------------------------------------------------------------

const (
	minIndexTemplateGet OperationID = iota + 1
	minIndexTemplateCreate
	minIndexTemplateDelete
	minIndexTemplateExists
	minIndexTemplateSimulate
	minIndexTemplateSimulateIndex
	minComponentTemplateGet
	minComponentTemplateCreate
	minComponentTemplateDelete
	minComponentTemplateExists
	minLegacyTemplateGet
	minLegacyTemplateCreate
	minLegacyTemplateDelete
	minLegacyTemplateExists
)

// ---------------------------------------------------------------------------
// Minor operations — Maintenance (CatMaint / CatMaintWrite)
// ---------------------------------------------------------------------------

const (
	minRefresh OperationID = iota + 1
	minFlush
	minForceMerge
	minCacheClear
	minSegments
	minRecovery
	minShardStores
	minStats
)

// ---------------------------------------------------------------------------
// Minor operations — Ingest (CatIngest / CatIngestWrite)
// ---------------------------------------------------------------------------

const (
	minIngestGet OperationID = iota + 1
	minIngestCreate
	minIngestDelete
	minIngestSimulate
	minIngestGrok
)

// ---------------------------------------------------------------------------
// Minor operations — Cluster (CatCluster / CatClusterWrite)
// ---------------------------------------------------------------------------

const (
	minClusterInfo OperationID = iota + 1
	minClusterHealth
	minClusterStats
	minClusterState
	minClusterSettingsGet
	minClusterSettingsPut
	minClusterReroute
	minClusterPendingTasks
	minClusterAllocExplain
	minClusterRemoteInfo
	minClusterVotingConfigGet
	minClusterVotingConfigPut
	minClusterDecommission
)

// ---------------------------------------------------------------------------
// Minor operations — Admin (CatAdmin / CatAdminWrite)
// ---------------------------------------------------------------------------

const (
	// Cat operations
	minCatAllocation OperationID = iota + 1
	minCatClusterMgr
	minCatCount
	minCatFielddata
	minCatHealth
	minCatIndices
	minCatMaster
	minCatNodeAttrs
	minCatNodes
	minCatPendingTasks
	minCatPlugins
	minCatRecovery
	minCatRepositories
	minCatSegments
	minCatShards
	minCatSnapshots
	minCatTasks
	minCatTemplates
	minCatThreadPool

	// Nodes
	minNodesInfo
	minNodesStats
	minNodesUsage
	minNodesHotThreads
	minNodesReloadSecurity

	// Tasks
	minTasksList
	minTasksGet
	minTasksCancel

	// Snapshots
	minSnapshotCreate
	minSnapshotGet
	minSnapshotDelete
	minSnapshotClone
	minSnapshotRestore
	minSnapshotStatus
	minSnapshotRepoCreate
	minSnapshotRepoGet
	minSnapshotRepoDelete
	minSnapshotRepoVerify
	minSnapshotRepoCleanup

	// Scripts
	minScriptGet
	minScriptPut
	minScriptDelete
	minScriptContext
	minScriptLanguage
	minScriptPainlessExec

	// Dangling
	minDanglingGet
	minDanglingDelete
	minDanglingImport

	// DataStream
	minDataStreamGet
	minDataStreamCreate
	minDataStreamDelete
	minDataStreamStats

	// Render search template
	minRenderSearchTemplate
)

// ---------------------------------------------------------------------------
// Minor operations — Ping (CatPing)
// ---------------------------------------------------------------------------

const (
	minPing OperationID = 1
)

// ---------------------------------------------------------------------------
// Composed operation IDs
// ---------------------------------------------------------------------------

// OpOther is returned for unrecognized HTTP method+path combinations.
const OpOther OperationID = -1

// Search operations.
const (
	OpSearch         = CatSearch | minSearch
	OpMSearch        = CatSearch | minMSearch
	OpCount          = CatSearch | minCount
	OpSearchTemplate = CatSearch | minSearchTemplate
	OpMSearchTmpl    = CatSearch | minMSearchTemplate
	OpValidate       = CatSearch | minValidate
	OpRankEval       = CatSearch | minRankEval
	OpExplain        = CatSearch | minExplain
	OpSearchShards   = CatSearch | minSearchShards
	OpFieldCaps      = CatSearch | minFieldCaps
)

// Scroll operations.
const (
	OpScrollGet    = CatScroll | minScrollGet
	OpScrollDelete = CatScrollWrite | minScrollDelete
)

// Point-in-time operations.
const (
	OpPITList   = CatPIT | minPITList
	OpPITCreate = CatPITWrite | minPITCreate
	OpPITDelete = CatPITWrite | minPITDelete
)

// Document read operations.
const (
	OpDocGet         = CatDocRead | minDocGet
	OpDocExists      = CatDocRead | minDocExists
	OpDocSourceGet   = CatDocRead | minDocSourceGet
	OpDocSourceExist = CatDocRead | minDocSourceExists
	OpMGet           = CatDocRead | minMGet
	OpTermVectors    = CatDocRead | minTermVectors
	OpMTermVectors   = CatDocRead | minMTermVectors
)

// Document write operations.
const (
	OpDocIndex  = CatDocWrite | minDocIndex
	OpDocCreate = CatDocWrite | minDocCreate
	OpDocUpdate = CatDocWrite | minDocUpdate
	OpDocDelete = CatDocWrite | minDocDelete
)

// Bulk and query-based write operations.
const (
	OpBulk              = CatBulk | minBulk
	OpBulkStream        = CatBulk | minBulkStream
	OpReindex           = CatBulk | minReindex
	OpDeleteByQuery     = CatBulk | minDeleteByQuery
	OpUpdateByQuery     = CatBulk | minUpdateByQuery
	OpReindexRethrottle = CatBulk | minReindexRethrottle
	OpDBQRethrottle     = CatBulk | minDBQRethrottle
	OpUBQRethrottle     = CatBulk | minUBQRethrottle
)

// Index management operations.
const (
	OpIndexGet      = CatIndex | minIndexGet
	OpIndexExists   = CatIndex | minIndexExists
	OpIndexCreate   = CatIndexWrite | minIndexCreate
	OpIndexDelete   = CatIndexWrite | minIndexDelete
	OpIndexOpen     = CatIndexWrite | minIndexOpen
	OpIndexClose    = CatIndexWrite | minIndexClose
	OpIndexClone    = CatIndexWrite | minIndexClone
	OpIndexShrink   = CatIndexWrite | minIndexShrink
	OpIndexSplit    = CatIndexWrite | minIndexSplit
	OpIndexRollover = CatIndexWrite | minIndexRollover
	OpIndexBlock    = CatIndexWrite | minIndexBlock
	OpIndexResolve  = CatIndex | minIndexResolve
	OpIndexAnalyze  = CatIndexWrite | minIndexAnalyze
)

// Mapping operations.
const (
	OpMappingGet = CatMapping | minMappingGet
	OpMappingPut = CatMappingWrite | minMappingPut
)

// Alias operations.
const (
	OpAliasGet    = CatAlias | minAliasGet
	OpAliasPut    = CatAliasWrite | minAliasPut
	OpAliasDelete = CatAliasWrite | minAliasDelete
	OpCatAliases  = CatAlias | minCatAliases
)

// Template operations.
const (
	OpIndexTemplateGet           = CatTemplate | minIndexTemplateGet
	OpIndexTemplateCreate        = CatTemplateWrite | minIndexTemplateCreate
	OpIndexTemplateDelete        = CatTemplateWrite | minIndexTemplateDelete
	OpIndexTemplateExists        = CatTemplate | minIndexTemplateExists
	OpIndexTemplateSimulate      = CatTemplate | minIndexTemplateSimulate
	OpIndexTemplateSimulateIndex = CatTemplate | minIndexTemplateSimulateIndex
	OpComponentTemplateGet       = CatTemplate | minComponentTemplateGet
	OpComponentTemplateCreate    = CatTemplateWrite | minComponentTemplateCreate
	OpComponentTemplateDelete    = CatTemplateWrite | minComponentTemplateDelete
	OpComponentTemplateExists    = CatTemplate | minComponentTemplateExists
	OpLegacyTemplateGet          = CatTemplate | minLegacyTemplateGet
	OpLegacyTemplateCreate       = CatTemplateWrite | minLegacyTemplateCreate
	OpLegacyTemplateDelete       = CatTemplateWrite | minLegacyTemplateDelete
	OpLegacyTemplateExists       = CatTemplate | minLegacyTemplateExists
)

// Maintenance operations.
const (
	OpRefresh     = CatMaintWrite | minRefresh
	OpFlush       = CatMaintWrite | minFlush
	OpForceMerge  = CatMaintWrite | minForceMerge
	OpCacheClear  = CatMaintWrite | minCacheClear
	OpSegments    = CatMaint | minSegments
	OpRecovery    = CatMaint | minRecovery
	OpShardStores = CatMaint | minShardStores
	OpStats       = CatMaint | minStats
)

// Ingest operations.
const (
	OpIngestGet      = CatIngest | minIngestGet
	OpIngestCreate   = CatIngestWrite | minIngestCreate
	OpIngestDelete   = CatIngestWrite | minIngestDelete
	OpIngestSimulate = CatIngestWrite | minIngestSimulate
	OpIngestGrok     = CatIngest | minIngestGrok
)

// Cluster operations.
const (
	OpClusterInfo           = CatCluster | minClusterInfo
	OpClusterHealth         = CatCluster | minClusterHealth
	OpClusterStats          = CatCluster | minClusterStats
	OpClusterState          = CatCluster | minClusterState
	OpClusterSettingsGet    = CatCluster | minClusterSettingsGet
	OpClusterSettingsPut    = CatClusterWrite | minClusterSettingsPut
	OpClusterReroute        = CatClusterWrite | minClusterReroute
	OpClusterPendingTasks   = CatCluster | minClusterPendingTasks
	OpClusterAllocExplain   = CatCluster | minClusterAllocExplain
	OpClusterRemoteInfo     = CatCluster | minClusterRemoteInfo
	OpClusterVotingConfigEx = CatClusterWrite | minClusterVotingConfigPut
)

// Admin — cat operations.
const (
	OpCatAllocation  = CatAdmin | minCatAllocation
	OpCatClusterMgr  = CatAdmin | minCatClusterMgr
	OpCatCount       = CatAdmin | minCatCount
	OpCatFielddata   = CatAdmin | minCatFielddata
	OpCatHealth      = CatAdmin | minCatHealth
	OpCatIndices     = CatAdmin | minCatIndices
	OpCatMaster      = CatAdmin | minCatMaster
	OpCatNodeAttrs   = CatAdmin | minCatNodeAttrs
	OpCatNodes       = CatAdmin | minCatNodes
	OpCatPendingTask = CatAdmin | minCatPendingTasks
	OpCatPlugins     = CatAdmin | minCatPlugins
	OpCatRecovery    = CatAdmin | minCatRecovery
	OpCatRepos       = CatAdmin | minCatRepositories
	OpCatSegments    = CatAdmin | minCatSegments
	OpCatShards      = CatAdmin | minCatShards
	OpCatSnapshots   = CatAdmin | minCatSnapshots
	OpCatTasks       = CatAdmin | minCatTasks
	OpCatTemplates   = CatAdmin | minCatTemplates
	OpCatThreadPool  = CatAdmin | minCatThreadPool
)

// Admin — node operations.
const (
	OpNodesInfo           = CatAdmin | minNodesInfo
	OpNodesStats          = CatAdmin | minNodesStats
	OpNodesUsage          = CatAdmin | minNodesUsage
	OpNodesHotThreads     = CatAdmin | minNodesHotThreads
	OpNodesReloadSecurity = CatAdminWrite | minNodesReloadSecurity
)

// Admin — task operations.
const (
	OpTasksList   = CatAdmin | minTasksList
	OpTasksGet    = CatAdmin | minTasksGet
	OpTasksCancel = CatAdminWrite | minTasksCancel
)

// Admin — snapshot operations.
const (
	OpSnapshotCreate     = CatAdminWrite | minSnapshotCreate
	OpSnapshotGet        = CatAdmin | minSnapshotGet
	OpSnapshotDelete     = CatAdminWrite | minSnapshotDelete
	OpSnapshotClone      = CatAdminWrite | minSnapshotClone
	OpSnapshotRestore    = CatAdminWrite | minSnapshotRestore
	OpSnapshotStatus     = CatAdmin | minSnapshotStatus
	OpSnapshotRepoCreate = CatAdminWrite | minSnapshotRepoCreate
	OpSnapshotRepoGet    = CatAdmin | minSnapshotRepoGet
	OpSnapshotRepoDelete = CatAdminWrite | minSnapshotRepoDelete
	OpSnapshotRepoVerify = CatAdminWrite | minSnapshotRepoVerify
	OpSnapshotRepoClean  = CatAdminWrite | minSnapshotRepoCleanup
)

// Admin — script operations.
const (
	OpScriptGet          = CatAdmin | minScriptGet
	OpScriptPut          = CatAdminWrite | minScriptPut
	OpScriptDelete       = CatAdminWrite | minScriptDelete
	OpScriptContext      = CatAdmin | minScriptContext
	OpScriptLanguage     = CatAdmin | minScriptLanguage
	OpScriptPainlessExec = CatAdminWrite | minScriptPainlessExec
)

// Admin — dangling index operations.
const (
	OpDanglingGet    = CatAdmin | minDanglingGet
	OpDanglingDelete = CatAdminWrite | minDanglingDelete
	OpDanglingImport = CatAdminWrite | minDanglingImport
)

// Data stream operations.
const (
	OpDataStreamGet    = CatIndex | minDataStreamGet
	OpDataStreamCreate = CatIndexWrite | minDataStreamCreate
	OpDataStreamDelete = CatIndexWrite | minDataStreamDelete
	OpDataStreamStats  = CatIndex | minDataStreamStats
)

// OpRenderSearchTemplate identifies the render search template operation.
const OpRenderSearchTemplate = CatSearch | minRenderSearchTemplate

// OpPing identifies a cluster ping operation.
const OpPing = CatPing | minPing

// ---------------------------------------------------------------------------
// String
// ---------------------------------------------------------------------------

//nolint:cyclop,gocyclo,exhaustive,goconst // intentional large switch with default fallback
func (op OperationID) String() string {
	switch op {
	// Search
	case OpSearch:
		return "search"
	case OpMSearch:
		return "msearch"
	case OpCount:
		return "count"
	case OpSearchTemplate:
		return "search_template"
	case OpMSearchTmpl:
		return "msearch_template"
	case OpValidate:
		return "validate"
	case OpRankEval:
		return "rank_eval"
	case OpExplain:
		return "explain"
	case OpSearchShards:
		return "search_shards"
	case OpFieldCaps:
		return "field_caps"

	// Scroll
	case OpScrollGet:
		return "scroll_get"
	case OpScrollDelete:
		return "scroll_delete"

	// PIT
	case OpPITList:
		return "pit_list"
	case OpPITCreate:
		return "pit_create"
	case OpPITDelete:
		return "pit_delete"

	// Document read
	case OpDocGet:
		return "doc_get"
	case OpDocExists:
		return "doc_exists"
	case OpDocSourceGet:
		return "doc_source_get"
	case OpDocSourceExist:
		return "doc_source_exists"
	case OpMGet:
		return "mget"
	case OpTermVectors:
		return "termvectors"
	case OpMTermVectors:
		return "mtermvectors"

	// Document write
	case OpDocIndex:
		return "doc_index"
	case OpDocCreate:
		return "doc_create"
	case OpDocUpdate:
		return "doc_update"
	case OpDocDelete:
		return "doc_delete"

	// Bulk
	case OpBulk:
		return "bulk"
	case OpBulkStream:
		return "bulk_stream"
	case OpReindex:
		return "reindex"
	case OpDeleteByQuery:
		return "delete_by_query"
	case OpUpdateByQuery:
		return "update_by_query"
	case OpReindexRethrottle:
		return "reindex_rethrottle"
	case OpDBQRethrottle:
		return "dbq_rethrottle"
	case OpUBQRethrottle:
		return "ubq_rethrottle"

	// Index management
	case OpIndexGet:
		return "index_get"
	case OpIndexExists:
		return "index_exists"
	case OpIndexCreate:
		return "index_create"
	case OpIndexDelete:
		return "index_delete"
	case OpIndexOpen:
		return "index_open"
	case OpIndexClose:
		return "index_close"
	case OpIndexClone:
		return "index_clone"
	case OpIndexShrink:
		return "index_shrink"
	case OpIndexSplit:
		return "index_split"
	case OpIndexRollover:
		return "index_rollover"
	case OpIndexBlock:
		return "index_block"
	case OpIndexResolve:
		return "index_resolve"
	case OpIndexAnalyze:
		return "index_analyze"

	// Mapping
	case OpMappingGet:
		return "mapping_get"
	case OpMappingPut:
		return "mapping_put"

	// Alias
	case OpAliasGet:
		return "alias_get"
	case OpAliasPut:
		return "alias_put"
	case OpAliasDelete:
		return "alias_delete"
	case OpCatAliases:
		return "cat_aliases"

	// Template
	case OpIndexTemplateGet:
		return "index_template_get"
	case OpIndexTemplateCreate:
		return "index_template_create"
	case OpIndexTemplateDelete:
		return "index_template_delete"
	case OpIndexTemplateExists:
		return "index_template_exists"
	case OpIndexTemplateSimulate:
		return "index_template_simulate"
	case OpIndexTemplateSimulateIndex:
		return "index_template_simulate_index"
	case OpComponentTemplateGet:
		return "component_template_get"
	case OpComponentTemplateCreate:
		return "component_template_create"
	case OpComponentTemplateDelete:
		return "component_template_delete"
	case OpComponentTemplateExists:
		return "component_template_exists"
	case OpLegacyTemplateGet:
		return "legacy_template_get"
	case OpLegacyTemplateCreate:
		return "legacy_template_create"
	case OpLegacyTemplateDelete:
		return "legacy_template_delete"
	case OpLegacyTemplateExists:
		return "legacy_template_exists"

	// Maintenance
	case OpRefresh:
		return "refresh"
	case OpFlush:
		return "flush"
	case OpForceMerge:
		return "forcemerge"
	case OpCacheClear:
		return "cache_clear"
	case OpSegments:
		return "segments"
	case OpRecovery:
		return "recovery"
	case OpShardStores:
		return "shard_stores"
	case OpStats:
		return "stats"

	// Ingest
	case OpIngestGet:
		return "ingest_get"
	case OpIngestCreate:
		return "ingest_create"
	case OpIngestDelete:
		return "ingest_delete"
	case OpIngestSimulate:
		return "ingest_simulate"
	case OpIngestGrok:
		return "ingest_grok"

	// Cluster
	case OpClusterInfo:
		return "cluster_info"
	case OpClusterHealth:
		return "cluster_health"
	case OpClusterStats:
		return "cluster_stats"
	case OpClusterState:
		return "cluster_state"
	case OpClusterSettingsGet:
		return "cluster_settings_get"
	case OpClusterSettingsPut:
		return "cluster_settings_put"
	case OpClusterReroute:
		return "cluster_reroute"
	case OpClusterPendingTasks:
		return "cluster_pending_tasks"
	case OpClusterAllocExplain:
		return "cluster_alloc_explain"
	case OpClusterRemoteInfo:
		return "cluster_remote_info"
	case OpClusterVotingConfigEx:
		return "cluster_voting_config"

	// Ping
	case OpPing:
		return "ping"

	case OpOther:
		return "other"
	}

	// Fallback: category name + numeric minor
	cat := op.Category()
	minor := op.Minor()

	var catName string
	switch cat {
	case CatSearch:
		catName = "search"
	case CatScroll:
		catName = "scroll"
	case CatScrollWrite:
		catName = "scroll_write"
	case CatPIT:
		catName = "pit"
	case CatPITWrite:
		catName = "pit_write"
	case CatDocRead:
		catName = "doc_read"
	case CatDocWrite:
		catName = "doc_write"
	case CatBulk:
		catName = "bulk"
	case CatIndex:
		catName = "index"
	case CatIndexWrite:
		catName = "index_write"
	case CatMapping:
		catName = "mapping"
	case CatMappingWrite:
		catName = "mapping_write"
	case CatAlias:
		catName = "alias"
	case CatAliasWrite:
		catName = "alias_write"
	case CatTemplate:
		catName = "template"
	case CatTemplateWrite:
		catName = "template_write"
	case CatMaint:
		catName = "maint"
	case CatMaintWrite:
		catName = "maint_write"
	case CatIngest:
		catName = "ingest"
	case CatIngestWrite:
		catName = "ingest_write"
	case CatCluster:
		catName = "cluster"
	case CatClusterWrite:
		catName = "cluster_write"
	case CatAdmin:
		catName = "admin"
	case CatAdminWrite:
		catName = "admin_write"
	case CatPing:
		catName = "ping"
	default:
		catName = "unknown"
	}

	return catName + "_" + strconv.FormatInt(int64(minor), 10)
}
