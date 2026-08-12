package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	storageProvider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	opensearchgo "github.com/opensearch-project/opensearch-go/v4"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"

	revactx "github.com/opencloud-eu/reva/v3/pkg/ctx"
	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
	"github.com/opencloud-eu/reva/v3/pkg/storagespace"
	"github.com/opencloud-eu/reva/v3/pkg/utils"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/pkg/kql"
	"github.com/opencloud-eu/opencloud/pkg/log"
	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/aggregation"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/aggs"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/convert"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/osu"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

const defaultBatchSize = 50

// updateRetries lets OpenSearch merge a partial update again onto a document
// another write changed between its read and its write
const updateRetries = 3

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

func (b *Backend) Search(ctx context.Context, sir *search.IndexQuery) (*search.IndexResult, error) {
	boolQuery, err := convert.KQLToOpenSearchBoolQuery(sir.Query)
	switch {
	case kql.IsValidationError(err):
		return nil, errtypes.BadRequest(err.Error())
	case err != nil:
		return nil, fmt.Errorf("failed to convert KQL query to OpenSearch bool query: %w", err)
	}

	filters, err := aggregationFilterQueries(sir.AggregationFilters)
	if err != nil {
		return nil, errtypes.BadRequest(err.Error())
	}
	boolQuery.Filter(filters...)

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
		TrackScores:    conversions.ToPointer(true),
		// count every match, the default stops at 10000
		TrackTotalHits: true,
		Size:           conversions.ToPointer(search.EnginePageSize(sir.PageSize, 1000)),
	}
	// replicas may score a document differently, so a user reads every page
	// from the same shard copies; a preference must not start with an underscore
	if u, ok := revactx.ContextGetUser(ctx); ok && u.GetId().GetOpaqueId() != "" {
		searchParams.Preference = "user-" + u.GetId().GetOpaqueId()
	}

	// order_by first, missing values last in both directions, then like the
	// cross-space merge: score, ties by id
	sortClause := make([]map[string]any, 0, len(sir.OrderBy)+2)
	for _, sp := range sir.OrderBy {
		field, ok := search.SortIndexField(sp.GetName())
		if !ok {
			return nil, errtypes.BadRequest(fmt.Sprintf("field %q is not sortable", sp.GetName()))
		}
		order := "asc"
		if sp.GetIsDescending() {
			order = "desc"
		}
		sortClause = append(sortClause, map[string]any{field: map[string]any{"order": order, "missing": "_last"}})
	}
	sortClause = append(sortClause, map[string]any{"_score": map[string]any{"order": "desc"}}, map[string]any{"ID": map[string]any{"order": "asc"}})

	aggregationsBody, err := aggs.Build(sir.Aggregations)
	if err != nil {
		return nil, errtypes.BadRequest(err.Error())
	}

	req, err := osu.BuildSearchReq(&opensearchgoAPI.SearchReq{
		Indices: []string{b.index},
		Params:  searchParams,
	},
		boolQuery,
		osu.SearchBodyParams{
			Highlight: &osu.BodyParamHighlight{
				HighlightOptions: osu.HighlightOptions{
					Encoder:           "html",
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
			Aggs: aggregationsBody,
			Sort: sortClause,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build search request: %w", err)
	}

	resp, err := b.client.Search(ctx, req)
	switch {
	case tooManyBuckets(err):
		return nil, errtypes.BadRequest(aggregation.ErrTooManyBuckets.Error())
	case beyondResultWindow(err):
		return nil, errtypes.BadRequest("from and size reach beyond the result window of the index (index.max_result_window)")
	case err != nil:
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

	aggregations, err := aggs.Parse(sir.Aggregations, resp.Aggregations)
	if err == nil {
		err = aggregation.CheckBuckets(aggregations)
	}
	switch {
	case errors.Is(err, aggregation.ErrTooManyBuckets):
		return nil, errtypes.BadRequest(err.Error())
	case err != nil:
		return nil, fmt.Errorf("failed to parse aggregations: %w", err)
	}

	return &search.IndexResult{
		Matches:      matches,
		TotalMatches: int32(totalMatches),
		Aggregations: aggregations,
	}, nil
}

func (b *Backend) Get(ctx context.Context, id string) (*search.Resource, error) {
	resource, err := searchResourceByID(ctx, b.client, b.index, id)
	if err != nil {
		return nil, err
	}

	return &resource, nil
}

// Update sends a partial document, OpenSearch merges it into the stored one, so
// the extracted fields are never read and written back
func (b *Backend) Update(ctx context.Context, id string, u search.MetadataUpdate) error {
	r := search.Resource{}
	r.Tags = u.Tags
	r.Favorites = u.Favorites
	full, err := mapping.PrepareForIndex(r, r.SearchFieldOverrides())
	if err != nil {
		return fmt.Errorf("failed to prepare the update: %w", err)
	}

	// a field the update empties has no sibling in full, the explicit null clears the stale one
	siblings := mapping.SearchSiblings(reflect.TypeFor[search.Resource](), r.SearchFieldOverrides())
	partial := map[string]any{}
	for _, field := range []string{"Tags", "Favorites"} {
		partial[field] = full[field]
		if siblings[field].Lowercase {
			partial[field+mapping.LowercaseSuffix] = full[field+mapping.LowercaseSuffix]
		}
		if siblings[field].Words {
			partial[field+mapping.WordsSuffix] = full[field+mapping.WordsSuffix]
		}
	}

	body, err := json.Marshal(map[string]any{"doc": partial})
	if err != nil {
		return fmt.Errorf("failed to marshal the update: %w", err)
	}

	_, err = b.client.Update(ctx, opensearchgoAPI.UpdateReq{
		Index:      b.index,
		DocumentID: id,
		Body:       bytes.NewReader(body),
		Params: opensearchgoAPI.UpdateParams{
			Refresh:         "wait_for",
			RetryOnConflict: new(updateRetries),
		},
	})
	var structErr *opensearchgo.StructError
	switch {
	case errors.As(err, &structErr) && structErr.Err.Type == "document_missing_exception":
		return fmt.Errorf("document with id %s: %w", id, search.ErrResourceNotFound)
	case err != nil:
		return fmt.Errorf("failed to update the document: %w", err)
	}
	return nil
}

// tooManyBuckets tells whether OpenSearch refused the aggregations for their
// bucket count (search.max_buckets); the cause sits below the search phase
// error.
func tooManyBuckets(err error) bool {
	var structErr *opensearchgo.StructError
	if !errors.As(err, &structErr) {
		return false
	}
	for cause := structErr.Err.CausedBy; cause != nil; cause = cause.CausedBy {
		if cause.Type == "too_many_buckets_exception" {
			return true
		}
	}
	return false
}

// beyondResultWindow tells whether a search failed for a page that reaches
// beyond index.max_result_window, 10000 matches by default.
func beyondResultWindow(err error) bool {
	var structErr *opensearchgo.StructError
	if !errors.As(err, &structErr) {
		return false
	}
	tooLarge := func(reason string) bool { return strings.Contains(reason, "Result window is too large") }
	if tooLarge(structErr.Err.Reason) {
		return true
	}
	for _, cause := range structErr.Err.RootCause {
		if tooLarge(cause.Reason) {
			return true
		}
	}
	for cause := structErr.Err.CausedBy; cause != nil; cause = cause.CausedBy {
		if tooLarge(cause.Reason) {
			return true
		}
	}
	return false
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

// Optimize leaves the segments to the merge policy of the cluster, operators
// that want a forced merge run it against OpenSearch directly
func (b *Backend) Optimize(context.Context) error {
	return search.ErrOptimizeUnsupported
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
