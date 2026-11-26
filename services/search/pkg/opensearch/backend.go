package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"time"

	storageProvider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	opensearchgo "github.com/opensearch-project/opensearch-go/v4"
	opensearchgoAPI "github.com/opensearch-project/opensearch-go/v4/opensearchapi"

	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
	"github.com/opencloud-eu/reva/v3/pkg/storagespace"
	"github.com/opencloud-eu/reva/v3/pkg/utils"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/pkg/log"
	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
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
	index      string
	client     *opensearchgoAPI.Client
	vectorizer search.TextVectorizer
}

// Option configures a Backend.
type Option func(*Backend)

// WithTextVectorizer enables semantic queries (`semantic:"..."`); without it
// they are rejected.
func WithTextVectorizer(v search.TextVectorizer) Option {
	return func(b *Backend) {
		b.vectorizer = v
	}
}

// NewBackend creates a backend on the versioned generation of the named index.
func NewBackend(ctx context.Context, name string, client *opensearchgoAPI.Client, logger log.Logger, opts ...Option) (*Backend, error) {
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

	b := &Backend{index: index, client: client}
	for _, opt := range opts {
		opt(b)
	}
	return b, nil
}

func (b *Backend) Search(ctx context.Context, sir *search.IndexQuery) (*search.IndexResult, error) {
	boolQuery, err := convert.KQLToOpenSearchBoolQuery(sir.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to convert KQL query to OpenSearch bool query: %w", err)
	}
	// the semantic clause ranks, the query filters: it scopes the vector
	// search and stays the only source of totals
	semanticText, pureSemantic := sir.Semantic, sir.Semantic != "" && sir.SemanticOnly
	if semanticText != "" && b.vectorizer == nil {
		return nil, errtypes.BadRequest("semantic search is not configured")
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
		// Do not send back the full content (only needed for highlighting, the
		// snippets come back instead) or the raw image vectors (ranking data).
		SourceExcludes: []string{"Content", "imageVector"},
	}

	switch {
	case sir.PageSize == -1:
		searchParams.Size = conversions.ToPointer(1000)
	case sir.PageSize == 0:
		searchParams.Size = conversions.ToPointer(200)
	default:
		searchParams.Size = conversions.ToPointer(int(sir.PageSize))
	}

	// the filter request carries the totals and the lexical ranking; for a
	// purely semantic query it only supplies the totals
	filterParams := searchParams
	if pureSemantic {
		filterParams.Size = conversions.ToPointer(0)
	}
	req, err := osu.BuildSearchReq(&opensearchgoAPI.SearchReq{
		Indices: []string{b.index},
		Params:  filterParams,
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
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build search request: %w", err)
	}

	resp, err := b.client.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	matches, err := convertHits(resp.Hits.Hits)
	if err != nil {
		return nil, err
	}
	totalMatches := resp.Hits.Total.Value

	if semanticText != "" {
		vector, err := b.vectorizer.VectorizeText(ctx, semanticText)
		if err != nil {
			return nil, fmt.Errorf("failed to vectorize the semantic query: %w", err)
		}
		k := semanticK(searchParams.Size)
		knnParams := searchParams
		knnParams.Size = conversions.ToPointer(k)
		// the full filter query also pre-filters the neighbor search
		knnQuery := osu.NewKnnQuery("imageVector").Vector(vector).K(k).Filter(boolQuery)
		knnReq, err := osu.BuildSearchReq(&opensearchgoAPI.SearchReq{
			Indices: []string{b.index},
			Params:  knnParams,
		}, knnQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to build knn request: %w", err)
		}
		knnResp, err := b.client.Search(ctx, knnReq)
		if err != nil {
			return nil, fmt.Errorf("failed to run knn search: %w", err)
		}
		knnMatches, err := convertHits(knnResp.Hits.Hits)
		if err != nil {
			return nil, err
		}

		if pureSemantic {
			// the knn result is the ranking, and the only honest total is the
			// number of semantic hits (a similarity search has no result set,
			// only a ranking)
			matches = knnMatches
			totalMatches = len(knnMatches)
		} else {
			matches = fuseRRF(matches, knnMatches, searchParams.Size)
		}
	}

	return &search.IndexResult{
		Matches:      matches,
		TotalMatches: int32(totalMatches),
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

// convertHits converts OpenSearch hits to matches.
func convertHits(hits []opensearchgoAPI.SearchHit) ([]*searchMessage.Match, error) {
	matches := make([]*searchMessage.Match, 0, len(hits))
	for _, hit := range hits {
		match, err := convert.OpenSearchHitToMatch(hit)
		if err != nil {
			return nil, fmt.Errorf("failed to convert hit to match: %w", err)
		}
		matches = append(matches, match)
	}
	return matches, nil
}

// semanticK picks the number of nearest neighbors for a semantic clause: at
// least a fusion-friendly window, at most a sane cap (huge page sizes stand
// for "everything", but a similarity ranking beyond 1000 hits carries no
// signal).
func semanticK(size *int) int {
	const minK, maxK = 200, 1000
	switch {
	case size == nil || *size >= maxK:
		return maxK
	case *size < minK:
		return minK
	default:
		return *size
	}
}

// fuseRRF merges the lexical and the semantic ranking via reciprocal rank
// fusion (rank constant 60, like bleve's default) and trims to limit.
func fuseRRF(lexical, semantic []*searchMessage.Match, limit *int) []*searchMessage.Match {
	const rankConst = 60
	type entry struct {
		match *searchMessage.Match
		score float64
	}
	byID := map[string]*entry{}
	order := make([]*entry, 0, len(lexical)+len(semantic))
	add := func(list []*searchMessage.Match) {
		for i, m := range list {
			id := m.GetEntity().GetId()
			key := id.GetStorageId() + "$" + id.GetSpaceId() + "!" + id.GetOpaqueId()
			e, ok := byID[key]
			if !ok {
				e = &entry{match: m}
				byID[key] = e
				order = append(order, e)
			}
			e.score += 1.0 / float64(rankConst+i+1)
		}
	}
	add(lexical)
	add(semantic)
	sort.SliceStable(order, func(i, j int) bool { return order[i].score > order[j].score })

	out := make([]*searchMessage.Match, 0, len(order))
	for _, e := range order {
		e.match.Score = float32(e.score)
		out = append(out, e.match)
	}
	if limit != nil && len(out) > *limit {
		out = out[:*limit]
	}
	return out
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
