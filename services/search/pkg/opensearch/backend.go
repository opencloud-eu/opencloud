package opensearch

import (
	"context"
	"fmt"
	"strings"
	"time"

	storageProvider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"

	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
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

func (b *Backend) Search(ctx context.Context, sir *searchService.SearchIndexRequest) (*searchService.SearchIndexResponse, error) {
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
		Indices: []string{b.index},
		Params:  searchParams,
	},
		boolQuery,
		osu.SearchBodyParams{
			Highlight: &osu.BodyParamHighlight{
				PreTags:  []string{"<mark>"},
				PostTags: []string{"</mark>"},
				Fields: map[string]osu.BodyParamHighlight{
					"Content": {},
				},
			},
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

func (b *Backend) DocCount() (uint64, error) {
	req, err := osu.BuildIndicesCountReq(
		&opensearchgoAPI.IndicesCountReq{
			Indices: []string{b.index},
		},
		osu.NewTermQuery[bool]("Deleted").Value(false),
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

func (b *Backend) Move(id string, parentID string, target string) error {
	batch, err := b.NewBatch(defaultBatchSize)
	if err != nil {
		return err
	}

	if err := batch.Move(id, parentID, target); err != nil {
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

func (b *Backend) NewBatch(size int) (search.BatchOperator, error) {
	return NewBatch(b.client, b.index, size)
}
