package opensearch

import (
	"context"
	"fmt"
	"time"

	storageProvider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"

	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/pkg/kql"
	"github.com/opencloud-eu/opencloud/pkg/log"
	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/aggs"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/convert"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/osu"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

const defaultBatchSize = 50

var (
	ErrUnhealthyCluster = fmt.Errorf("cluster is not healthy")
)

type Backend struct {
	index  string
	client *opensearchgoAPI.Client
}

// NewBackend creates a backend on the versioned generation of the named index.
func NewBackend(ctx context.Context, name string, client *opensearchgoAPI.Client, logger log.Logger) (*Backend, error) {
	index := VersionedIndexName(name)

	pingResp, err := client.Ping(ctx, &opensearchgoAPI.PingReq{})
	switch {
	case err != nil:
		return nil, fmt.Errorf("%w, failed to ping opensearch: %w", ErrUnhealthyCluster, err)
	case pingResp.IsError():
		return nil, fmt.Errorf("%w, failed to ping opensearch", ErrUnhealthyCluster)
	}

	// apply the index template
	if err := IndexManagerLatest.Apply(ctx, index, client, logger); err != nil {
		return nil, fmt.Errorf("failed to apply index template: %w", err)
	}

	// first check if the cluster is healthy

	resp, err := client.Cluster.Health(ctx, &opensearchgoAPI.ClusterHealthReq{
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

func (b *Backend) Search(ctx context.Context, sir *searchService.SearchIndexRequest) (*searchService.SearchIndexResponse, error) {
	boolQuery, err := convert.KQLToOpenSearchBoolQuery(sir.Query)
	switch {
	case kql.IsValidationError(err):
		return nil, errtypes.BadRequest(err.Error())
	case err != nil:
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
		// Scope below the space root: restrict at query level so totals and
		// paging respect the path too. Path uses the case-preserving
		// path_hierarchy analyzer, so the folder path is an indexed token of
		// the folder itself and every descendant.
		if requestedPath := utils.MakeRelativePath(sir.Ref.Path); requestedPath != "." {
			boolQuery.Filter(
				osu.NewTermQuery[string]("Path").Value(requestedPath),
			)
		}
	}

	searchParams := opensearchgoAPI.SearchParams{
		SourceExcludes: []string{"Content"}, // Do not send back the full content in the search response, as it is only needed for highlighting and can be large. The highlighted snippets will be sent back in the response instead.
	}

	switch {
	case sir.PageSize == -1:
		searchParams.Size = conversions.ToPointer(1000)
	case sir.PageSize == 0:
		searchParams.Size = conversions.ToPointer(200)
	default:
		searchParams.Size = conversions.ToPointer(int(sir.PageSize))
	}

	req, err := osu.BuildSearchReq(&opensearchgoAPI.SearchReq{
		Indices: []string{b.index},
		Params:  searchParams,
	},
		boolQuery,
		osu.SearchBodyParams{
			Highlight: &osu.BodyParamHighlight{
				HighlightOptions: osu.HighlightOptions{
					NumberOfFragments: 2,
					PreTags:           []string{"<mark>"},
					PostTags:          []string{"</mark>"},
				},
				Fields: map[string]osu.HighlightOptions{
					"Content": {
						Type: osu.HighlightTypeFvh,
					},
				},
			},
			Aggs: aggs.Build(sir.GetAggregations()),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build search request: %w", err)
	}

	resp, err := b.client.Search(ctx, req)
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

		matches = append(matches, match)
	}

	aggResults, err := aggs.Parse(resp.Aggregations, sir.GetAggregations())
	if err != nil {
		return nil, fmt.Errorf("failed to parse aggregations: %w", err)
	}

	return &searchService.SearchIndexResponse{
		Matches:      matches,
		TotalMatches: int32(totalMatches),
		Aggregations: aggResults,
	}, nil
}

func (b *Backend) DocCount() (uint64, error) {
	req, err := osu.BuildIndicesCountReq(
		&opensearchgoAPI.IndicesCountReq{
			Indices: []string{b.index},
		},
		osu.NewMatchAllQuery(),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to build count request: %w", err)
	}

	resp, err := b.client.Indices.Count(context.TODO(), req)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}

	return uint64(resp.Count), nil
}

func (b *Backend) Upsert(id string, r search.Resource) error {
	batch, err := b.NewBatch(defaultBatchSize)
	if err != nil {
		return err
	}

	if err := batch.Upsert(id, r); err != nil {
		return err
	}

	return batch.Push()
}

func (b *Backend) Move(id string, parentID string, targetPath string) error {
	batch, err := b.NewBatch(defaultBatchSize)
	if err != nil {
		return err
	}

	if err := batch.Move(id, parentID, targetPath); err != nil {
		return err
	}

	return batch.Push()
}

func (b *Backend) Delete(id string) error {
	batch, err := b.NewBatch(defaultBatchSize)
	if err != nil {
		return err
	}

	if err := batch.Delete(id); err != nil {
		return err
	}

	return batch.Push()
}

func (b *Backend) Restore(id string) error {
	batch, err := b.NewBatch(defaultBatchSize)
	if err != nil {
		return err
	}

	if err := batch.Restore(id); err != nil {
		return err
	}

	return batch.Push()
}

func (b *Backend) Purge(id string, onlyDeleted bool) error {
	batch, err := b.NewBatch(defaultBatchSize)
	if err != nil {
		return err
	}

	if err := batch.Purge(id, onlyDeleted); err != nil {
		return err
	}

	return batch.Push()
}

func (b *Backend) PurgeSpace(rootID string) error {
	req, err := osu.BuildDocumentDeleteByQueryReq(
		opensearchgoAPI.DocumentDeleteByQueryReq{
			Indices: []string{b.index},
			Params: opensearchgoAPI.DocumentDeleteByQueryParams{
				WaitForCompletion: conversions.ToPointer(true),
				Refresh:           conversions.ToPointer(true),
			},
		},
		osu.NewBoolQuery().Must(osu.NewTermQuery[string]("RootID").Value(rootID)),
	)
	if err != nil {
		return fmt.Errorf("failed to build the space purge request %s: %w", rootID, err)
	}

	resp, err := b.client.Document.DeleteByQuery(context.TODO(), req)
	switch {
	case err != nil:
		return fmt.Errorf("failed to purge space %s: %w", rootID, err)
	case len(resp.Failures) != 0:
		return fmt.Errorf("failed to purge space %s: %v", rootID, resp.Failures)
	}

	return nil
}

func (b *Backend) NewBatch(size int) (search.BatchOperator, error) {
	return NewBatch(b.client, b.index, size)
}
