package opensearch

import (
	"context"
	"fmt"

	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"

	"github.com/opencloud-eu/opencloud/services/search/pkg/search"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/osu"
)

func searchResourceByID(ctx context.Context, client *opensearchgoAPI.Client, index string, id string) (search.Resource, error) {
	req, err := osu.BuildSearchReq(
		&opensearchgoAPI.SearchReq{
			Indices: []string{index},
		},
		osu.NewIDsQuery(id),
	)
	if err != nil {
		return search.Resource{}, fmt.Errorf("failed to build search request: %w", err)
	}

	resp, err := client.Search(ctx, req)
	switch {
	case err != nil:
		return search.Resource{}, fmt.Errorf("failed to search for resource: %w", err)
	case resp.Hits.Total.Value == 0 || len(resp.Hits.Hits) == 0:
		return search.Resource{}, fmt.Errorf("document with id %s not found", id)
	}

	resource, err := conversions.To[search.Resource](resp.Hits.Hits[0].Source)
	if err != nil {
		return search.Resource{}, fmt.Errorf("failed to convert hit source: %w", err)
	}

	return resource, nil
}

func updateSelfAndDescendants(ctx context.Context, client *opensearchgoAPI.Client, index string, id string, scriptProvider func(search.Resource) *osu.BodyParamScript) error {
	if scriptProvider == nil {
		return fmt.Errorf("script cannot be nil")
	}

	resource, err := searchResourceByID(context.Background(), client, index, id)
	if err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}

	req, err := osu.BuildUpdateByQueryReq(
		opensearchgoAPI.UpdateByQueryReq{
			Indices: []string{index},
			Params: opensearchgoAPI.UpdateByQueryParams{
				WaitForCompletion: conversions.ToPointer(true),
			},
		},
		osu.NewTermQuery[string]("Path").Value(resource.Path),
		osu.UpdateByQueryBodyParams{
			Script: scriptProvider(resource),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to build update by query request: %w", err)
	}

	resp, err := client.UpdateByQuery(ctx, req)
	switch {
	case err != nil:
		return fmt.Errorf("failed to update by query: %w", err)
	case len(resp.Failures) != 0:
		return fmt.Errorf("failed to update by query, failures: %v", resp.Failures)
	}

	return nil
}
