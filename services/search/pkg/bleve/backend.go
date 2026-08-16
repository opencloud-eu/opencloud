package bleve

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	storageProvider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/opencloud-eu/opencloud/pkg/kql"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"

	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchService "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	searchQuery "github.com/opencloud-eu/opencloud/services/search/pkg/query"
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
	index        bleve.Index
	queryCreator searchQuery.Creator[query.Query]
	vectorizer   search.TextVectorizer
	log          log.Logger
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

func NewBackend(index bleve.Index, queryCreator searchQuery.Creator[query.Query], log log.Logger, opts ...Option) *Backend {
	b := &Backend{
		index:        index,
		queryCreator: queryCreator,
		log:          log,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// Search executes a search request operation within the index.
// Returns a SearchIndexResponse object or an error.
func (b *Backend) Search(ctx context.Context, sir *searchService.SearchIndexRequest) (*searchService.SearchIndexResponse, error) {
	// the semantic clause ranks, the remaining query filters: it scopes the
	// vector search and stays the only source of totals. The creator splits
	// the clause off the parsed KQL tree.
	createdQuery, semanticText, err := b.queryCreator.CreateWithSemantic(sir.Query)
	if err != nil {
		if kql.IsValidationError(err) {
			return nil, errtypes.BadRequest(err.Error())
		}
		return nil, err
	}
	pureSemantic := createdQuery == nil && semanticText != ""

	// scope holds the restrictions every hit must satisfy; it also pre-filters
	// the vector search
	scope := bleve.NewConjunctionQuery(
		// Skip documents that have been marked as deleted
		&query.BoolFieldQuery{
			Bool:     false,
			FieldVal: "Deleted",
		},
	)

	if sir.Ref != nil {
		scope.Conjuncts = append(
			scope.Conjuncts,
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
		// paging respect the path too. Path is a case-preserving keyword
		// (paths act as references, /Foo and /foo are distinct), so the exact
		// folder or the folder prefix matches all of, and only, the scope.
		if requestedPath := utils.MakeRelativePath(sir.Ref.Path); requestedPath != "." {
			scope.Conjuncts = append(scope.Conjuncts, query.NewDisjunctionQuery([]query.Query{
				&query.TermQuery{FieldVal: "Path", Term: requestedPath},
				&query.PrefixQuery{FieldVal: "Path", Prefix: requestedPath + "/"},
			}))
		}
	}

	// A purely semantic query has no filter part: the hits come exclusively
	// from the KNN clause (base match_none), so the reported total is the
	// number of semantic hits, not a library count (a similarity search has
	// no result set, only a ranking). With a filter part the hits are the
	// filter matches, re-ranked by the fusion.
	var q query.Query
	if pureSemantic {
		q = query.NewMatchNoneQuery()
	} else {
		q = bleve.NewConjunctionQuery(append([]query.Query{}, append(scope.Conjuncts, createdQuery)...)...)
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
		// pre-filter the vector search: scope only for a purely semantic
		// query, the full filter conjunction otherwise
		knnFilter := query.Query(scope)
		if !pureSemantic {
			knnFilter = q
		}
		if err := addSemanticKNN(bleveReq, vector, knnFilter, semanticK(bleveReq.Size), !pureSemantic); err != nil {
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
				Id:         resourceIDtoSearchID(rID),
				Name:       getFieldValue[string](hit.Fields, "Name"),
				ParentId:   resourceIDtoSearchID(pID),
				Size:       uint64(getFieldValue[float64](hit.Fields, "Size")),
				Type:       uint64(getFieldValue[float64](hit.Fields, "Type")),
				MimeType:   getFieldValue[string](hit.Fields, "MimeType"),
				Deleted:    getFieldValue[bool](hit.Fields, "Deleted"),
				Tags:       getFieldSliceValue[string](hit.Fields, "Tags"),
				Favorites:  getFieldSliceValue[string](hit.Fields, "Favorites"),
				Highlights: getFragmentValue(hit.Fragments, "Content", 0),
				Audio:      hitToFacet[searchMessage.Audio](hit.Fields, "audio"),
				Image:      hitToFacet[searchMessage.Image](hit.Fields, "image"),
				Location:   hitToFacet[searchMessage.GeoCoordinates](hit.Fields, "location"),
				Photo:      hitToFacet[searchMessage.Photo](hit.Fields, "photo"),
			},
		}

		if mtime, err := time.Parse(time.RFC3339, getFieldValue[string](hit.Fields, "Mtime")); err == nil {
			match.Entity.LastModifiedTime = &timestamppb.Timestamp{Seconds: mtime.Unix(), Nanos: int32(mtime.Nanosecond())}
		}

		matches = append(matches, match)
	}

	return &searchService.SearchIndexResponse{
		Matches:      matches,
		TotalMatches: int32(totalMatches),
	}, nil
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

func (b *Backend) Move(rootID, parentID, location string) error {
	batch, err := b.NewBatch(defaultBatchSize)
	if err != nil {
		return err
	}

	if err := batch.Move(rootID, parentID, location); err != nil {
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
	return NewBatch(b.index, size)
}
