package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
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

var ErrNoContainerType = fmt.Errorf("not a container type")

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
	if err := IndexTemplateResourceV1.Apply(context.TODO(), client); err != nil {
		return nil, fmt.Errorf("failed to apply index template: %w", err)
	}

	indicesExistsResp, err := client.Indices.Exists(context.TODO(), opensearchgoAPI.IndicesExistsReq{
		Indices: []string{index},
	})
	switch {
	case indicesExistsResp != nil && indicesExistsResp.StatusCode == 404:
		break
	case err != nil:
		return nil, fmt.Errorf("failed to check if index exists: %w", err)
	case indicesExistsResp == nil:
		return nil, fmt.Errorf("unexpected nil response when checking if index exists")
	}

	// if the index does not exist, we need to create it
	if indicesExistsResp.StatusCode == 404 {
		resp, err := client.Indices.Create(context.TODO(), opensearchgoAPI.IndicesCreateReq{
			Index: index,
			// the body is not necessary; we will use an index template to define the index settings and mappings
		})
		switch {
		case err != nil:
			return nil, fmt.Errorf("failed to create index: %w", err)
		case !resp.Acknowledged:
			return nil, fmt.Errorf("failed to create index: %s", index)
		}
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

	body, err := NewRootQuery(boolQuery).MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	searchParams := opensearchgoAPI.SearchParams{}

	switch {
	case sir.PageSize == -1:
		searchParams.Size = conversions.ToPointer(math.MaxInt)
	case sir.PageSize == 0:
		searchParams.Size = conversions.ToPointer(200)
	default:
		searchParams.Size = conversions.ToPointer(int(sir.PageSize))
	}

	// fixMe: see getDescendants
	if *searchParams.Size > 250 {
		searchParams.Size = conversions.ToPointer(250)
	}

	resp, err := e.client.Search(ctx, &opensearchgoAPI.SearchReq{
		Indices: []string{e.index},
		Body:    bytes.NewReader(body),
		Params:  searchParams,
	})
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
	resource, err := e.getResource(id)
	if err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}

	oldPath := resource.Path
	resource.Path = utils.MakeRelativePath(target)
	resource.Name = path.Base(resource.Path)
	resource.ParentID = parentID

	if err := e.Upsert(id, resource); err != nil {
		return fmt.Errorf("failed to upsert resource: %w", err)
	}

	descendants, err := e.getDescendants(resource.Type, resource.RootID, oldPath)
	if err != nil && !errors.Is(err, ErrNoContainerType) {
		return fmt.Errorf("failed to find descendants: %w", err)
	}

	for _, descendant := range descendants {
		descendant.Path = strings.Replace(descendant.Path, oldPath, resource.Path, 1)
		if err := e.Upsert(descendant.ID, descendant); err != nil {
			return fmt.Errorf("failed to upsert resource: %w", err)
		}
	}

	return nil
}

func (e *Engine) Delete(id string) error {
	return e.deleteResource(id, true)
}

func (e *Engine) Restore(id string) error {
	return e.deleteResource(id, false)
}

func (e *Engine) Purge(id string) error {
	_, err := e.client.Document.Delete(context.TODO(), opensearchgoAPI.DocumentDeleteReq{
		Index:      e.index,
		DocumentID: id,
	})
	if err != nil {
		return fmt.Errorf("failed to purge document: %w", err)
	}

	return nil
}

func (e *Engine) DocCount() (uint64, error) {
	body, err := NewRootQuery(
		NewTermQuery[bool]("Deleted").Value(false),
	).MarshalJSON()
	if err != nil {
		return 0, fmt.Errorf("failed to marshal query: %w", err)
	}

	resp, err := e.client.Indices.Count(context.TODO(), &opensearchgoAPI.IndicesCountReq{
		Indices: []string{e.index},
		Body:    bytes.NewReader(body),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}

	return uint64(resp.Count), nil
}

func (e *Engine) deleteResource(id string, deleted bool) error {
	resource, err := e.getResource(id)
	if err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}

	descendants, err := e.getDescendants(resource.Type, resource.RootID, resource.Path)
	if err != nil && !errors.Is(err, ErrNoContainerType) {
		return fmt.Errorf("failed to find descendants: %w", err)
	}

	body, err := json.Marshal(map[string]any{
		"doc": map[string]bool{
			"Deleted": deleted,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal body: %w", err)
	}

	for _, resource := range append([]engine.Resource{resource}, descendants...) {
		if resource.Deleted == deleted {
			continue // already marked as the desired state
		}

		if _, err = e.client.Update(context.TODO(), opensearchgoAPI.UpdateReq{
			Index:      e.index,
			DocumentID: resource.ID,
			Body:       bytes.NewReader(body),
		}); err != nil {
			return fmt.Errorf("failed to mark document as deleted: %w", err)
		}
	}

	return nil
}

func (e *Engine) getResource(id string) (engine.Resource, error) {
	body, err := NewRootQuery(
		NewIDsQuery([]string{id}),
	).MarshalJSON()
	if err != nil {
		return engine.Resource{}, fmt.Errorf("failed to marshal query: %w", err)
	}

	resp, err := e.client.Search(context.TODO(), &opensearchgoAPI.SearchReq{
		Indices: []string{e.index},
		Body:    bytes.NewReader(body),
	})
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

func (e *Engine) getDescendants(resourceType uint64, rootID, rootPath string) ([]engine.Resource, error) {
	switch {
	case resourceType != uint64(storageProvider.ResourceType_RESOURCE_TYPE_CONTAINER):
		return nil, fmt.Errorf("%w: %d", ErrNoContainerType, resourceType)
	case rootID == "":
		return nil, fmt.Errorf("rootID cannot be empty")
	case rootPath == "":
		return nil, fmt.Errorf("rootPath cannot be empty")
	}

	if !strings.HasSuffix(rootPath, "*") {
		rootPath = strings.Join(append(strings.Split(rootPath, "/"), "*"), "/")
	}

	body, err := NewRootQuery(
		NewBoolQuery().Must(
			NewTermQuery[string]("RootID").Value(rootID),
			NewWildcardQuery("Path").Value(rootPath),
		),
	).MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	// unfortunately, we need to use recursion to fetch all descendants because of the paging
	// toDo: check the docs to find a better and most important more efficient way to fetch (or update) all descendants
	var doSearch func(params opensearchgoAPI.SearchParams) ([]engine.Resource, error)
	doSearch = func(params opensearchgoAPI.SearchParams) ([]engine.Resource, error) {
		resp, err := e.client.Search(context.TODO(), &opensearchgoAPI.SearchReq{
			Indices: []string{e.index},
			Body:    bytes.NewReader(body),
			Params:  params,
		})
		switch {
		case err != nil:
			return nil, fmt.Errorf("failed to search for document: %w", err)
		case resp.Hits.Total.Value == 0 || len(resp.Hits.Hits) == 0:
			return nil, nil // no descendants found, fin
		}

		descendants := make([]engine.Resource, len(resp.Hits.Hits))
		for i, hit := range resp.Hits.Hits {
			descendant, err := convert[engine.Resource](hit.Source)
			if err != nil {
				return nil, fmt.Errorf("failed to convert hit source %d: %w", i, err)
			}

			descendants[i] = descendant
		}

		if len(descendants) < resp.Hits.Total.Value {
			switch params.From {
			case nil:
				params.From = opensearchgoAPI.ToPointer(len(resp.Hits.Hits))
			default:
				params.From = opensearchgoAPI.ToPointer(*params.From + len(resp.Hits.Hits))

			}
			moreDescendants, err := doSearch(params)
			if err != nil {
				return nil, fmt.Errorf("failed to search for more descendants: %w", err)
			}

			descendants = append(descendants, moreDescendants...)
		}

		return descendants, nil
	}

	return doSearch(opensearchgoAPI.SearchParams{})
}
