// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchapi

import (
	"encoding/json"
	"net/http"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/internal/build"
	ospath "github.com/opensearch-project/opensearch-go/v4/internal/path"
)

// ClusterStateReq represents possible options for the /_cluster/state request
type ClusterStateReq struct {
	Metrics []string
	Indices []string

	Header http.Header
	Params ClusterStateParams
}

// GetRequest returns the *http.Request that gets executed by the client
func (r ClusterStateReq) GetRequest(method string) (*http.Request, error) {
	path, err := ospath.ClusterStatePath{
		Index:  r.Indices,
		Metric: r.Metrics,
	}.Build()
	if err != nil {
		return nil, err
	}
	return build.Request(method, path, nil, r.Params.get(), r.Header)
}

// ClusterStateResp represents the returned struct of the ClusterStateReq response
type ClusterStateResp struct {
	ClusterName        string `json:"cluster_name"`
	ClusterUUID        string `json:"cluster_uuid"`
	Version            int    `json:"version"`
	StateUUID          string `json:"state_uuid"`
	MasterNode         string `json:"master_node"`
	ClusterManagerNode string `json:"cluster_manager_node"`
	Blocks             struct {
		Indices map[string]map[string]ClusterStateBlocksIndex `json:"indices"`
	} `json:"blocks"`
	Nodes        map[string]ClusterStateNodes `json:"nodes"`
	Metadata     ClusterStateMetaData         `json:"metadata"`
	response     *opensearch.Response
	RoutingTable struct {
		Indices map[string]struct {
			Shards map[string][]ClusterStateRoutingIndex `json:"shards"`
		} `json:"indices"`
	} `json:"routing_table"`
	RoutingNodes ClusterStateRoutingNodes `json:"routing_nodes"`
	Snapshots    struct {
		Snapshots []json.RawMessage `json:"snapshots"`
	} `json:"snapshots"`
	SnapshotDeletions struct {
		SnapshotDeletions []json.RawMessage `json:"snapshot_deletions"`
	} `json:"snapshot_deletions"`
	RepositoryCleanup struct {
		RepositoryCleanup []json.RawMessage `json:"repository_cleanup"`
	} `json:"repository_cleanup"`
	Restore struct {
		Snapshots []json.RawMessage `json:"snapshots"`
	} `json:"restore"`
}

// Inspect returns the Inspect type containing the raw *opensearch.Response
func (r ClusterStateResp) Inspect() Inspect {
	return Inspect{Response: r.response}
}

// ClusterStateBlocksIndex is a sub type of ClusterStateResp
type ClusterStateBlocksIndex struct {
	UUID        *string  `json:"uuid,omitempty"`
	Description string   `json:"description"`
	Retryable   bool     `json:"retryable"`
	Levels      []string `json:"levels"`
}

// ClusterStateNodes is a sub type of ClusterStateResp
type ClusterStateNodes struct {
	Name             string            `json:"name"`
	EphemeralID      string            `json:"ephemeral_id"`
	TransportAddress string            `json:"transport_address"`
	Attributes       map[string]string `json:"attributes"`
}

// ClusterStateMetaData is a sub type if ClusterStateResp containing metadata of the cluster
type ClusterStateMetaData struct {
	ClusterUUID          string `json:"cluster_uuid"`
	ClusterUUIDCommitted bool   `json:"cluster_uuid_committed"`
	ClusterCoordination  struct {
		Term                   int      `json:"term"`
		LastCommittedConfig    []string `json:"last_committed_config"`
		LastAcceptedConfig     []string `json:"last_accepted_config"`
		VotingConfigExclusions []struct {
			NodeID   string `json:"node_id"`
			NodeName string `json:"node_name"`
		} `json:"voting_config_exclusions"`
	} `json:"cluster_coordination"`
	Templates      map[string]json.RawMessage           `json:"templates"`
	Indices        map[string]ClusterStateMetaDataIndex `json:"indices"`
	IndexGraveyard struct {
		Tombstones []struct {
			Index struct {
				IndexName string `json:"index_name"`
				IndexUUID string `json:"index_uuid"`
			} `json:"index"`
			DeleteDateInMillis int `json:"delete_date_in_millis"`
		} `json:"tombstones"`
	} `json:"index-graveyard"`
	Repositories map[string]struct {
		Type              string            `json:"type"`
		Settings          map[string]string `json:"settings"`
		Generation        int               `json:"generation"`
		PendingGeneration int               `json:"pending_generation"`
	} `json:"repositories"`
	ComponentTemplate struct {
		ComponentTemplate map[string]json.RawMessage `json:"component_template"`
	} `json:"component_template"`
	IndexTemplate struct {
		IndexTemplate map[string]json.RawMessage `json:"index_template"`
	} `json:"index_template"`
	StoredScripts map[string]struct {
		Lang    string `json:"lang"`
		Source  string `json:"source"`
		Options *struct {
			ContentType string `json:"content_type,omitempty"`
		} `json:"options,omitempty"` // Present since OpenSearch 1.0.0
	} `json:"stored_scripts"`
	Ingest struct {
		Pipeline []struct {
			ID     string `json:"id"`
			Config struct {
				Description string          `json:"description"`
				Processors  json.RawMessage `json:"processors"`
			} `json:"config"`
		} `json:"pipeline"`
	} `json:"ingest"`
	DataStream struct {
		DataStream map[string]ClusterStateMetaDataStream `json:"data_stream"`
	} `json:"data_stream"`
}

// ClusterStateMetaDataIndex is a sub type of ClusterStateMetaData containing information about an index
type ClusterStateMetaDataIndex struct {
	Version           int                 `json:"version"`
	MappingVersion    int                 `json:"mapping_version"`
	SettingsVersion   int                 `json:"settings_version"`
	AliasesVersion    int                 `json:"aliases_version"`
	RoutingNumShards  int                 `json:"routing_num_shards"`
	State             string              `json:"state"`
	Settings          json.RawMessage     `json:"settings"`
	Mappings          json.RawMessage     `json:"mappings"`
	Aliases           []string            `json:"aliases"`
	PrimaryTerms      map[string]int      `json:"primary_terms"`
	InSyncAllocations map[string][]string `json:"in_sync_allocations"`
	RolloverInfo      map[string]struct {
		MetConditions map[string]string `json:"met_conditions"`
		Time          int               `json:"time"`
	} `json:"rollover_info"`
	System          bool            `json:"system"`
	IngestionStatus json.RawMessage `json:"ingestion_status"` // Available in OpenSearch 3.3.0+

	// Added in OpenSearch 3.6.0 (opensearch-project/OpenSearch@ed2e1006,
	// server/src/main/java/org/opensearch/cluster/metadata/IndexMetadata.java:1057-1059).
	PrimaryTermsMap     map[string]int               `json:"primary_terms_map,omitempty"`
	SplitShardsMetadata *ClusterStateSplitShardsMeta `json:"split_shards_metadata,omitempty"`
}

// ClusterStateSplitShardsMeta contains shard-split metadata for an index.
// Added in OpenSearch 3.6.0 (opensearch-project/OpenSearch@ed2e1006,
// server/src/main/java/org/opensearch/cluster/metadata/SplitShardsMetadata.java).
type ClusterStateSplitShardsMeta struct {
	NumOfRootShards         int                                 `json:"num_of_root_shards"`
	MaxShardID              int                                 `json:"max_shard_id"`
	InProgressSplitShardID  []int                               `json:"in_progress_split_shard_id,omitempty"`
	ActiveShardIDs          []int                               `json:"active_shard_ids"`
	RootShardsToAllChildren map[string][]ClusterStateShardRange `json:"root_shards_to_all_children"`
	ParentToChildShards     map[string][]ClusterStateShardRange `json:"parent_to_child_shards"`
}

// ClusterStateShardRange describes the hash-range owned by a single shard.
// Added in OpenSearch 3.6.0 (opensearch-project/OpenSearch@ed2e1006,
// server/src/main/java/org/opensearch/cluster/metadata/ShardRange.java).
type ClusterStateShardRange struct {
	ShardID int `json:"shard_id"`
	Start   int `json:"start"`
	End     int `json:"end"`
}

// ClusterStateMetaDataStream is a sub type of ClusterStateMetaData containing information about a data stream
type ClusterStateMetaDataStream struct {
	Name           string `json:"name"`
	TimestampField struct {
		Name string `json:"name"`
	} `json:"timestamp_field"`
	Indices []struct {
		IndexName string `json:"index_name"`
		IndexUUID string `json:"index_uuid"`
	} `json:"indices"`
	Generation int `json:"generation"`
}

// ClusterStateRoutingIndex is a sub type of ClusterStateResp and ClusterStateRoutingNodes containing information about shard routing
type ClusterStateRoutingIndex struct {
	State                    string  `json:"state"`
	Primary                  bool    `json:"primary"`
	SearchOnly               bool    `json:"searchOnly"`
	Node                     *string `json:"node"`
	RelocatingNode           *string `json:"relocating_node"`
	Shard                    int     `json:"shard"`
	Index                    string  `json:"index"`
	ExpectedShardSizeInBytes int     `json:"expected_shard_size_in_bytes"`
	AllocationID             *struct {
		ID           string  `json:"id"`
		RelocationID *string `json:"relocation_id,omitempty"` // Available since OpenSearch 1.0.0 (during shard relocation)
	} `json:"allocation_id,omitempty"`
	RecoverySource *struct {
		Type                    string `json:"type"`
		BootstrapNewHistoryUUID *bool  `json:"bootstrap_new_history_uuid,omitempty"` // Present since OpenSearch 1.0.0
	} `json:"recovery_source,omitempty"`
	UnassignedInfo *struct {
		Reason           string `json:"reason"`
		At               string `json:"at"`
		Delayed          bool   `json:"delayed"`
		AllocationStatus string `json:"allocation_status"`
		Details          string `json:"details"`
	} `json:"unassigned_info,omitempty"`
}

// ClusterStateRoutingNodes is a sub type of ClusterStateResp containing information about shard assigned to nodes
type ClusterStateRoutingNodes struct {
	Unassigned []ClusterStateRoutingIndex            `json:"unassigned"`
	Nodes      map[string][]ClusterStateRoutingIndex `json:"nodes"`
}
