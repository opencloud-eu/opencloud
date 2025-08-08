package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	storageProvider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/engine"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/convert"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/osu"
)

var (
	ErrUnhealthyCluster = fmt.Errorf("cluster is not healthy")
)

type Backend struct {
	index  string
	client *opensearchgoAPI.Client
}

func NewBackend(index string, client *opensearchgoAPI.Client) (*Backend, error) {
	pingResp, err := client.Ping(context.TODO(), &opensearchgoAPI.PingReq{})
	switch {
	case err != nil:
		return nil, fmt.Errorf("%w, failed to ping opensearch: %w", ErrUnhealthyCluster, err)
	case pingResp.IsError():
		return nil, fmt.Errorf("%w, failed to ping opensearch", ErrUnhealthyCluster)
	}

	// apply the index template
	if err := IndexManagerLatest.Apply(context.TODO(), index, client); err != nil {
		return nil, fmt.Errorf("failed to apply index template: %w", err)
	}

	// first check if the cluster is healthy

	resp, err := client.Cluster.Health(context.TODO(), &opensearchgoAPI.ClusterHealthReq{
		Indices: []string{index},
		Params: opensearchgoAPI.ClusterHealthParams{
			Local:   opensearchgoAPI.ToPointer(true),
			Timeout: 5 * time.Second,
		},
	})
	switch {
	case err != nil:
		return nil, fmt.Errorf("%w, failed to get cluster health: %w", ErrUnhealthyCluster, err)
	case resp.TimedOut:
		return nil, fmt.Errorf("%w, cluster health request timed out", ErrUnhealthyCluster)
	case resp.Status != "green" && resp.Status != "yellow":
		return nil, fmt.Errorf("%w, cluster health is not green or yellow: %s", ErrUnhealthyCluster, resp.Status)
	}

	return &Backend{index: index, client: client}, nil
}

func (be *Backend) Search(ctx context.Context, sir *searchService.SearchIndexRequest) (*searchService.SearchIndexResponse, error) {
	boolQuery, err := convert.KQLToOpenSearchBoolQuery(sir.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to convert KQL query to OpenSearch bool query: %w", err)
	}

	// filter out deleted resources
	boolQuery.Filter(
		osu.NewTermQuery[bool]("Deleted").Value(false),
	)

	if sir.Ref != nil {
		// if a reference is provided, filter by the root ID
		boolQuery.Filter(
			osu.NewTermQuery[string]("RootID").Value(
				storagespace.FormatResourceID(
					&storageProvider.ResourceId{
						StorageId: sir.Ref.GetResourceId().GetStorageId(),
						SpaceId:   sir.Ref.GetResourceId().GetSpaceId(),
						OpaqueId:  sir.Ref.GetResourceId().GetOpaqueId(),
					},
				),
			),
		)
	}

	searchParams := opensearchgoAPI.SearchParams{}

	switch {
	case sir.PageSize == -1:
		searchParams.Size = conversions.ToPointer(1000)
	case sir.PageSize == 0:
		searchParams.Size = conversions.ToPointer(200)
	default:
		searchParams.Size = conversions.ToPointer(int(sir.PageSize))
	}

	req, err := osu.BuildSearchReq(&opensearchgoAPI.SearchReq{
		Indices: []string{be.index},
		Params:  searchParams,
	},
		boolQuery,
		osu.SearchReqOptions{
			Highlight: &osu.HighlightOption{
				PreTags:  []string{"<mark>"},
				PostTags: []string{"</mark>"},
				Fields: map[string]osu.HighlightOption{
					"Content": {},
				},
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build search request: %w", err)
	}

	resp, err := be.client.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	matches := make([]*searchMessage.Match, 0, len(resp.Hits.Hits))
	totalMatches := resp.Hits.Total.Value
	for _, hit := range resp.Hits.Hits {
		match, err := convert.OpenSearchHitToMatch(hit)
		if err != nil {
			return nil, fmt.Errorf("failed to convert hit to match: %w", err)
		}

		if sir.Ref != nil {
			hitPath := strings.TrimSuffix(match.GetEntity().GetRef().GetPath(), "/")
			requestedPath := utils.MakeRelativePath(sir.Ref.Path)
			isRoot := hitPath == requestedPath

			if !isRoot && requestedPath != "." && !strings.HasPrefix(hitPath, requestedPath+"/") {
				totalMatches--
				continue
			}
		}

		matches = append(matches, match)
	}

	return &searchService.SearchIndexResponse{
		Matches:      matches,
		TotalMatches: int32(totalMatches),
	}, nil
}

func (be *Backend) Upsert(id string, r engine.Resource) error {
	body, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("failed to marshal resource: %w", err)
	}

	_, err = be.client.Index(context.TODO(), opensearchgoAPI.IndexReq{
		Index:      be.index,
		DocumentID: id,
		Body:       bytes.NewReader(body),
	})
	if err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}

	return nil
}

func (be *Backend) Move(id string, parentID string, target string) error {
	return be.updateSelfAndDescendants(id, func(rootResource engine.Resource) *osu.ScriptOption {
		return &osu.ScriptOption{
			Source: `
					if (ctx._source.ID == params.id ) { ctx._source.Name = params.newName; ctx._source.ParentID = params.parentID; }
					ctx._source.Path = ctx._source.Path.replace(params.oldPath, params.newPath)
				`,
			Lang: "painless",
			Params: map[string]any{
				"id":       id,
				"parentID": parentID,
				"oldPath":  rootResource.Path,
				"newPath":  utils.MakeRelativePath(target),
				"newName":  path.Base(utils.MakeRelativePath(target)),
			},
		}
	})
}

func (be *Backend) Delete(id string) error {
	return be.updateSelfAndDescendants(id, func(_ engine.Resource) *osu.ScriptOption {
		return &osu.ScriptOption{
			Source: "ctx._source.Deleted = params.deleted",
			Lang:   "painless",
			Params: map[string]any{
				"deleted": true,
			},
		}
	})
}

func (be *Backend) Restore(id string) error {
	return be.updateSelfAndDescendants(id, func(_ engine.Resource) *osu.ScriptOption {
		return &osu.ScriptOption{
			Source: "ctx._source.Deleted = params.deleted",
			Lang:   "painless",
			Params: map[string]any{
				"deleted": false,
			},
		}
	})
}

func (be *Backend) Purge(id string) error {
	resource, err := be.getResource(id)
	if err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}

	req, err := osu.BuildDocumentDeleteByQueryReq(
		opensearchgoAPI.DocumentDeleteByQueryReq{
			Indices: []string{be.index},
			Params: opensearchgoAPI.DocumentDeleteByQueryParams{
				WaitForCompletion: conversions.ToPointer(true),
			},
		},
		osu.NewTermQuery[string]("Path").Value(resource.Path),
	)
	if err != nil {
		return fmt.Errorf("failed to build delete by query request: %w", err)
	}

	resp, err := be.client.Document.DeleteByQuery(context.TODO(), req)
	switch {
	case err != nil:
		return fmt.Errorf("failed to delete by query: %w", err)
	case len(resp.Failures) != 0:
		return fmt.Errorf("failed to delete by query, failures: %v", resp.Failures)
	}

	return nil
}

func (be *Backend) DocCount() (uint64, error) {
	req, err := osu.BuildIndicesCountReq(
		&opensearchgoAPI.IndicesCountReq{
			Indices: []string{be.index},
		},
		osu.NewTermQuery[bool]("Deleted").Value(false),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to build count request: %w", err)
	}

	resp, err := be.client.Indices.Count(context.TODO(), req)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}

	return uint64(resp.Count), nil
}

func (be *Backend) updateSelfAndDescendants(id string, scriptProvider func(engine.Resource) *osu.ScriptOption) error {
	if scriptProvider == nil {
		return fmt.Errorf("script cannot be nil")
	}

	resource, err := be.getResource(id)
	if err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}

	req, err := osu.BuildUpdateByQueryReq(
		opensearchgoAPI.UpdateByQueryReq{
			Indices: []string{be.index},
			Params: opensearchgoAPI.UpdateByQueryParams{
				WaitForCompletion: conversions.ToPointer(true),
			},
		},
		osu.NewTermQuery[string]("Path").Value(resource.Path),
		osu.UpdateByQueryReqOptions{
			Script: scriptProvider(resource),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to build update by query request: %w", err)
	}

	resp, err := be.client.UpdateByQuery(context.TODO(), req)
	switch {
	case err != nil:
		return fmt.Errorf("failed to update by query: %w", err)
	case len(resp.Failures) != 0:
		return fmt.Errorf("failed to update by query, failures: %v", resp.Failures)
	}

	return nil
}

func (be *Backend) getResource(id string) (engine.Resource, error) {
	req, err := osu.BuildSearchReq(
		&opensearchgoAPI.SearchReq{
			Indices: []string{be.index},
		},
		osu.NewIDsQuery(id),
	)
	if err != nil {
		return engine.Resource{}, fmt.Errorf("failed to build search request: %w", err)
	}

	resp, err := be.client.Search(context.TODO(), req)
	switch {
	case err != nil:
		return engine.Resource{}, fmt.Errorf("failed to search for resource: %w", err)
	case resp.Hits.Total.Value == 0 || len(resp.Hits.Hits) == 0:
		return engine.Resource{}, fmt.Errorf("document with id %s not found", id)
	}

	resource, err := conversions.To[engine.Resource](resp.Hits.Hits[0].Source)
	if err != nil {
		return engine.Resource{}, fmt.Errorf("failed to convert hit source: %w", err)
	}

	return resource, nil
}

func (be *Backend) StartBatch(_ int) error {
	return nil // todo: implement batch processing
}

func (be *Backend) EndBatch() error {
	return nil // todo: implement batch processing
}
