package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	storageProvider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/pkg/kql"
	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/engine"
)

type Engine struct {
	index  string
	client *opensearchgoAPI.Client
}

func NewEngine(index string, client *opensearchgoAPI.Client) (*Engine, error) {
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
	_, healthy, err := clusterHealth(context.TODO(), client, []string{index})
	switch {
	case err != nil:
		return nil, fmt.Errorf("failed to get cluster health: %w", err)
	case !healthy:
		return nil, fmt.Errorf("cluster health is not healthy")
	}

	return &Engine{index: index, client: client}, nil
}

func (e *Engine) Search(ctx context.Context, sir *searchService.SearchIndexRequest) (*searchService.SearchIndexResponse, error) {
	ast, err := kql.Builder{}.Build(sir.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	transpiler, err := NewKQLToOsDSL()
	if err != nil {
		return nil, fmt.Errorf("failed to create KQL compiler: %w", err)
	}

	builder, err := transpiler.Compile(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to compile query: %w", err)
	}

	boolQuery := builderToBoolQuery(builder).Filter(
		NewTermQuery[bool]("Deleted").Value(false),
	)

	if sir.Ref != nil {
		boolQuery.Filter(
			NewTermQuery[string]("RootID").Value(
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

	req, err := BuildSearchReq(&opensearchgoAPI.SearchReq{
		Indices: []string{e.index},
		Params:  searchParams,
	},
		boolQuery,
		SearchReqOptions{
			Highlight: &HighlightOption{
				PreTags:  []string{"<mark>"},
				PostTags: []string{"</mark>"},
				Fields: map[string]HighlightOption{
					"Content": {},
				},
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build search request: %w", err)
	}

	resp, err := e.client.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	matches := make([]*searchMessage.Match, len(resp.Hits.Hits))
	totalMatches := resp.Hits.Total.Value
	for i, hit := range resp.Hits.Hits {
		match, err := searchHitToSearchMessageMatch(hit)
		if err != nil {
			return nil, fmt.Errorf("failed to convert hit %d: %w", i, err)
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

		matches[i] = match
	}

	return &searchService.SearchIndexResponse{
		Matches:      matches,
		TotalMatches: int32(totalMatches),
	}, nil
}

func (e *Engine) Upsert(id string, r engine.Resource) error {
	body, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("failed to marshal resource: %w", err)
	}

	_, err = e.client.Index(context.TODO(), opensearchgoAPI.IndexReq{
		Index:      e.index,
		DocumentID: id,
		Body:       bytes.NewReader(body),
	})
	if err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}

	return nil
}

func (e *Engine) Move(id string, parentID string, target string) error {
	return e.updateSelfAndDescendants(id, func(rootResource engine.Resource) *ScriptOption {
		return &ScriptOption{
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

func (e *Engine) Delete(id string) error {
	return e.updateSelfAndDescendants(id, func(_ engine.Resource) *ScriptOption {
		return &ScriptOption{
			Source: "ctx._source.Deleted = params.deleted",
			Lang:   "painless",
			Params: map[string]any{
				"deleted": true,
			},
		}
	})
}

func (e *Engine) Restore(id string) error {
	return e.updateSelfAndDescendants(id, func(_ engine.Resource) *ScriptOption {
		return &ScriptOption{
			Source: "ctx._source.Deleted = params.deleted",
			Lang:   "painless",
			Params: map[string]any{
				"deleted": false,
			},
		}
	})
}

func (e *Engine) Purge(id string) error {
	resource, err := e.getResource(id)
	if err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}

	req, err := BuildDocumentDeleteByQueryReq(
		opensearchgoAPI.DocumentDeleteByQueryReq{
			Indices: []string{e.index},
		},
		NewTermQuery[string]("Path").Value(resource.Path),
	)
	if err != nil {
		return fmt.Errorf("failed to build delete by query request: %w", err)
	}

	resp, err := e.client.Document.DeleteByQuery(context.TODO(), req)
	switch {
	case err != nil:
		return fmt.Errorf("failed to delete by query: %w", err)
	case len(resp.Failures) != 0:
		return fmt.Errorf("failed to delete by query, failures: %v", resp.Failures)
	}

	return nil
}

func (e *Engine) DocCount() (uint64, error) {
	req, err := BuildIndicesCountReq(
		&opensearchgoAPI.IndicesCountReq{
			Indices: []string{e.index},
		},
		NewTermQuery[bool]("Deleted").Value(false),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to build count request: %w", err)
	}

	resp, err := e.client.Indices.Count(context.TODO(), req)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}

	return uint64(resp.Count), nil
}

func (e *Engine) updateSelfAndDescendants(id string, scriptProvider func(engine.Resource) *ScriptOption) error {
	if scriptProvider == nil {
		return fmt.Errorf("script cannot be nil")
	}

	resource, err := e.getResource(id)
	if err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}

	req, err := BuildUpdateByQueryReq(
		opensearchgoAPI.UpdateByQueryReq{
			Indices: []string{e.index},
		},
		NewTermQuery[string]("Path").Value(resource.Path),
		UpdateByQueryReqOptions{
			Script: scriptProvider(resource),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to build update by query request: %w", err)
	}

	resp, err := e.client.UpdateByQuery(context.TODO(), req)
	switch {
	case err != nil:
		return fmt.Errorf("failed to update by query: %w", err)
	case len(resp.Failures) != 0:
		return fmt.Errorf("failed to update by query, failures: %v", resp.Failures)
	}

	return nil
}

func (e *Engine) getResource(id string) (engine.Resource, error) {
	req, err := BuildSearchReq(
		&opensearchgoAPI.SearchReq{
			Indices: []string{e.index},
		},
		NewIDsQuery([]string{id}),
	)
	if err != nil {
		return engine.Resource{}, fmt.Errorf("failed to build search request: %w", err)
	}

	resp, err := e.client.Search(context.TODO(), req)
	switch {
	case err != nil:
		return engine.Resource{}, fmt.Errorf("failed to search for resource: %w", err)
	case resp.Hits.Total.Value == 0 || len(resp.Hits.Hits) == 0:
		return engine.Resource{}, fmt.Errorf("document with id %s not found", id)
	}

	resource, err := convert[engine.Resource](resp.Hits.Hits[0].Source)
	if err != nil {
		return engine.Resource{}, fmt.Errorf("failed to convert hit source: %w", err)
	}

	return resource, nil
}

func (e *Engine) StartBatch(_ int) error {
	return nil // todo: implement batch processing
}

func (e *Engine) EndBatch() error {
	return nil // todo: implement batch processing
}
