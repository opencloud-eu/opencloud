package bleve

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	storageProvider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
	"github.com/opencloud-eu/reva/v3/pkg/storagespace"
	"github.com/opencloud-eu/reva/v3/pkg/utils"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"

	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchQuery "github.com/opencloud-eu/opencloud/services/search/pkg/query"
	bleveQuery "github.com/opencloud-eu/opencloud/services/search/pkg/query/bleve"
)

const defaultBatchSize = 50

// semanticK picks the number of nearest neighbors for a semantic clause: at
// least a fusion-friendly window, at most a sane cap (huge page sizes stand
// for "everything", but a similarity ranking beyond 1000 hits carries no
// signal, see the KNNRequest semantics: k results per clause, no threshold).
func semanticK(size int) int64 {
	const minK, maxK = 200, 1000
	switch {
	case size >= maxK:
		return maxK
	case size < minK:
		return minK
	default:
		return int64(size)
	}
}

var _ search.Engine = (*Backend)(nil) // ensure Backend implements Engine

type Backend struct {
	index      *serialIndex
	vectorizer search.TextVectorizer
	log        log.Logger
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

func NewBackend(index bleve.Index, log log.Logger, opts ...Option) *Backend {
	b := &Backend{
		index: &serialIndex{bleveIndex: index},
		log:   log,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// bleveIndex lets serialIndex embed bleve.Index, whose Index method would clash
// with an embedded field of the same name
type bleveIndex = bleve.Index

// serialIndex runs one batch at a time. bleve analyzes a batch through a pool of
// workers shared by all batches and keeps the whole analysis in memory until the
// batch is written, so batches running side by side (e.g. spaces reindexed
// concurrently) add little throughput but stack up their memory.
type serialIndex struct {
	bleveIndex
	batchMu sync.Mutex
}

func (i *serialIndex) Batch(b *bleve.Batch) error {
	i.batchMu.Lock()
	defer i.batchMu.Unlock()
	return i.bleveIndex.Batch(b)
}

// Search executes a search request operation within the index.
func (b *Backend) Search(ctx context.Context, sir *search.IndexQuery) (*search.IndexResult, error) {
	createdQuery, err := bleveQuery.Compiler{}.Compile(searchQuery.Normalize(sir.Query, searchQuery.ResolveField))
	if err != nil {
		return nil, err
	}
	// the semantic clause ranks, the query filters: it scopes the vector
	// search and stays the only source of totals
	semanticText, pureSemantic := sir.Semantic, sir.Semantic != "" && sir.SemanticOnly

	q := bleve.NewConjunctionQuery(
		// Skip documents that have been marked as deleted
		&query.BoolFieldQuery{
			Bool:     false,
			FieldVal: "Deleted",
		},
		createdQuery,
	)

	if sir.Ref != nil {
		q.Conjuncts = append(
			q.Conjuncts,
			&query.TermQuery{
				FieldVal: "RootID",
				Term: storagespace.FormatResourceID(
					&storageProvider.ResourceId{
						StorageId: sir.Ref.GetResourceId().GetStorageId(),
						SpaceId:   sir.Ref.GetResourceId().GetSpaceId(),
						OpaqueId:  sir.Ref.GetResourceId().GetOpaqueId(),
					},
				),
			},
		)
		// Scope below the space root: restrict at query level so totals and
		// paging respect the path too. The folder term matches the folder and
		// its descendants (see PathAnalyzer).
		if requestedPath := utils.MakeRelativePath(sir.Ref.Path); requestedPath != "." {
			q.Conjuncts = append(q.Conjuncts, &query.TermQuery{FieldVal: "Path", Term: requestedPath})
		}
	}

	bleveReq := bleve.NewSearchRequest(q)
	bleveReq.Highlight = bleve.NewHighlight()

	switch {
	case sir.PageSize == -1:
		bleveReq.Size = math.MaxInt
	case sir.PageSize == 0:
		bleveReq.Size = 200
	default:
		bleveReq.Size = int(sir.PageSize)
	}

	if semanticText != "" {
		if b.vectorizer == nil {
			return nil, errtypes.BadRequest("semantic search is not configured")
		}
		vector, err := b.vectorizer.VectorizeText(ctx, semanticText)
		if err != nil {
			return nil, fmt.Errorf("failed to vectorize the semantic query: %w", err)
		}
		// a purely semantic search has no result set, only a ranking: the
		// hits come from the KNN clause alone and their number is the total
		if pureSemantic {
			bleveReq.Query = query.NewMatchNoneQuery()
		}
		if err := addSemanticKNN(bleveReq, vector, q, semanticK(bleveReq.Size), !pureSemantic); err != nil {
			return nil, err
		}
	}

	bleveReq.Fields = []string{"*"}
	res, err := b.index.Search(bleveReq)
	if err != nil {
		return nil, err
	}

	// hybrid semantic: the fusion reports the fused-hit count as Total, so the
	// filter-part total (the only meaningful one) needs its own cheap count
	if semanticText != "" && !pureSemantic {
		countReq := bleve.NewSearchRequest(q)
		countReq.Size = 0
		countRes, err := b.index.Search(countReq)
		if err != nil {
			return nil, err
		}
		res.Total = countRes.Total
	}

	matches := make([]*searchMessage.Match, 0, len(res.Hits))
	totalMatches := res.Total
	for _, hit := range res.Hits {
		rootID, err := storagespace.ParseID(getFieldValue[string](hit.Fields, "RootID"))
		if err != nil {
			return nil, err
		}

		rID, err := storagespace.ParseID(getFieldValue[string](hit.Fields, "ID"))
		if err != nil {
			return nil, err
		}

		pID, _ := storagespace.ParseID(getFieldValue[string](hit.Fields, "ParentID"))
		match := &searchMessage.Match{
			Score: float32(hit.Score),
			Entity: &searchMessage.Entity{
				Ref: &searchMessage.Reference{
					ResourceId: resourceIDtoSearchID(rootID),
					Path:       getFieldValue[string](hit.Fields, "Path"),
				},
				Id:          resourceIDtoSearchID(rID),
				Name:        getFieldValue[string](hit.Fields, "Name"),
				ParentId:    resourceIDtoSearchID(pID),
				Size:        uint64(getFieldValue[float64](hit.Fields, "Size")),
				Type:        uint64(getFieldValue[float64](hit.Fields, "Type")),
				MimeType:    getFieldValue[string](hit.Fields, "MimeType"),
				Deleted:     getFieldValue[bool](hit.Fields, "Deleted"),
				Tags:        getFieldSliceValue[string](hit.Fields, "Tags"),
				Favorites:   getFieldSliceValue[string](hit.Fields, "Favorites"),
				Highlights:  getFragmentValue(hit.Fragments, "Content", 0),
				Audio:       hitToFacet[searchMessage.Audio](hit.Fields, "audio"),
				Image:       hitToFacet[searchMessage.Image](hit.Fields, "image"),
				Location:    hitToFacet[searchMessage.GeoCoordinates](hit.Fields, "location"),
				Photo:       hitToFacet[searchMessage.Photo](hit.Fields, "photo"),
				Video:       hitToFacet[searchMessage.Video](hit.Fields, "video"),
				MotionPhoto: hitToFacet[searchMessage.MotionPhoto](hit.Fields, "motionPhoto"),
				LivePhoto:   hitToFacet[searchMessage.LivePhoto](hit.Fields, "livePhoto"),
			},
		}

		if mtime, err := time.Parse(time.RFC3339, getFieldValue[string](hit.Fields, "Mtime")); err == nil {
			match.Entity.LastModifiedTime = &timestamppb.Timestamp{Seconds: mtime.Unix(), Nanos: int32(mtime.Nanosecond())}
		}

		matches = append(matches, match)
	}

	return &search.IndexResult{
		Matches:      matches,
		TotalMatches: int32(totalMatches),
	}, nil
}

func (b *Backend) Get(_ context.Context, id string) (*search.Resource, error) {
	return searchResourceByID(id, b.index)
}

// Update holds the batch lock from the read to the write, so a batch pushed in
// between is not overwritten; a batch built before the update and pushed after
// it still writes the tags it read
func (b *Backend) Update(_ context.Context, id string, u search.MetadataUpdate) error {
	b.index.batchMu.Lock()
	defer b.index.batchMu.Unlock()

	r, err := searchResourceByID(id, b.index)
	if err != nil {
		return err
	}
	r.Tags = u.Tags
	r.Favorites = u.Favorites

	doc, err := mapping.PrepareForIndex(r, r.SearchFieldOverrides())
	if err != nil {
		return err
	}
	batch := b.index.NewBatch()
	err = batch.Index(id, doc)
	if err != nil {
		return err
	}
	return b.index.bleveIndex.Batch(batch)
}

func (b *Backend) DocCount() (uint64, error) {
	return b.index.DocCount()
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

func (b *Backend) Move(id, parentID, targetPath string) error {
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
	for {
		req := bleve.NewSearchRequest(&query.TermQuery{FieldVal: "RootID", Term: rootID})
		req.Size = defaultBatchSize

		res, err := b.index.Search(req)
		if err != nil {
			return err
		}

		if res.Hits.Len() == 0 {
			return nil
		}

		batch := b.index.NewBatch()
		for _, hit := range res.Hits {
			batch.Delete(hit.ID)
		}

		if err := b.index.Batch(batch); err != nil {
			return err
		}
	}
}

func (b *Backend) NewBatch(size int) (search.BatchOperator, error) {
	return NewBatch(b.index, size)
}
