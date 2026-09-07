package bleve

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/blevesearch/bleve/v2"
	bleveSearch "github.com/blevesearch/bleve/v2/search"
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

var _ search.Engine = (*Backend)(nil) // ensure Backend implements Engine

type Backend struct {
	index        bleve.Index
	queryCreator searchQuery.Creator[query.Query]
	log          log.Logger
}

func NewBackend(index bleve.Index, queryCreator searchQuery.Creator[query.Query], log log.Logger) *Backend {
	return &Backend{
		index:        index,
		queryCreator: queryCreator,
		log:          log,
	}
}

// Search executes a search request operation within the index.
// Returns a SearchIndexResponse object or an error.
func (b *Backend) Search(_ context.Context, sir *searchService.SearchIndexRequest) (*searchService.SearchIndexResponse, error) {
	createdQuery, err := b.queryCreator.Create(sir.Query)
	if err != nil {
		if kql.IsValidationError(err) {
			return nil, errtypes.BadRequest(err.Error())
		}
		return nil, err
	}

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
		// paging respect the path too. Path is a case-preserving keyword
		// (paths act as references, /Foo and /foo are distinct), so the exact
		// folder or the folder prefix matches all of, and only, the scope.
		if requestedPath := utils.MakeRelativePath(sir.Ref.Path); requestedPath != "." {
			q.Conjuncts = append(q.Conjuncts, query.NewDisjunctionQuery([]query.Query{
				&query.TermQuery{FieldVal: "Path", Term: requestedPath},
				&query.PrefixQuery{FieldVal: "Path", Prefix: requestedPath + "/"},
			}))
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

	for _, agg := range sir.GetAggregations() {
		// Top-level metrics are computed by scanning the matched hits, they
		// have no facet representation.
		if agg.GetMetricKind() != searchService.MetricKind_METRIC_KIND_UNSPECIFIED {
			continue
		}
		fr, err := newBleveFacetRequest(agg)
		if err != nil {
			return nil, err
		}
		bleveReq.AddFacet(agg.GetField(), fr)
	}

	// Sub-aggregations and top-level metrics need the matched hit set, not just
	// count facets: widen the page so the emulator has enough docs. The caller's
	// larger PageSize wins.
	if needsSubAggScan(sir.GetAggregations()) && bleveReq.Size < subAggScanSize {
		bleveReq.Size = subAggScanSize
	}

	bleveReq.Fields = []string{"*"}
	res, err := b.index.Search(bleveReq)
	if err != nil {
		return nil, err
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

	return &searchService.SearchIndexResponse{
		Matches:      matches,
		TotalMatches: int32(totalMatches),
		Aggregations: extractBleveAggregations(res, sir.GetAggregations()),
	}, nil
}

// subAggScanSize caps how many hits we walk when emulating sub-aggregations;
// math.MaxInt returns everything.
const subAggScanSize = math.MaxInt

func needsSubAggScan(aggs []*searchService.AggregationOption) bool {
	for _, agg := range aggs {
		if len(agg.GetSubAggregations()) > 0 {
			return true
		}
		if agg.GetMetricKind() != searchService.MetricKind_METRIC_KIND_UNSPECIFIED {
			return true
		}
	}
	return false
}

// defaultFacetSize is used when no size is requested; the service layer trims
// after cross-space merge.
const defaultFacetSize = 1000

func newBleveFacetRequest(agg *searchService.AggregationOption) (*bleve.FacetRequest, error) {
	size := int(agg.GetSize())
	if size <= 0 {
		size = defaultFacetSize
	}
	fr := bleve.NewFacetRequest(agg.GetField(), size)
	ranges := aggregationRanges(agg)
	if rangesAreDates(ranges) {
		// bleve facets cannot mix numeric and date ranges, so one date-looking
		// bound switches the whole aggregation to date mode.
		for _, r := range ranges {
			start, err := parseRangeTime(r.GetFrom())
			if err != nil {
				return nil, fmt.Errorf("invalid date range bound %q on field %q", r.GetFrom(), agg.GetField())
			}
			end, err := parseRangeTime(r.GetTo())
			if err != nil {
				return nil, fmt.Errorf("invalid date range bound %q on field %q", r.GetTo(), agg.GetField())
			}
			fr.AddDateTimeRange(rangeBucketKey(r), start, end)
		}
		return fr, nil
	}
	for _, r := range ranges {
		minP := parseFloatPtr(r.GetFrom())
		maxP := parseFloatPtr(r.GetTo())
		fr.AddNumericRange(rangeBucketKey(r), minP, maxP)
	}
	return fr, nil
}

// rangeTimeLayouts are the accepted formats for date range bounds, tried in order.
var rangeTimeLayouts = []string{time.RFC3339, "2006-01-02"}

// rangesAreDates reports whether the ranges should be treated as datetime
// ranges: at least one bound parses as a date rather than a number.
func rangesAreDates(ranges []*searchService.BucketRange) bool {
	for _, r := range ranges {
		for _, s := range []string{r.GetFrom(), r.GetTo()} {
			if s == "" {
				continue
			}
			if _, err := strconv.ParseFloat(s, 64); err == nil {
				continue
			}
			if _, err := parseRangeTime(s); err == nil {
				return true
			}
		}
	}
	return false
}

// parseRangeTime parses a range bound; the zero time marks an open bound.
func parseRangeTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	for _, layout := range rangeTimeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time format %q", s)
}

func extractBleveAggregations(res *bleve.SearchResult, aggs []*searchService.AggregationOption) []*searchService.AggregationResult {
	if len(aggs) == 0 {
		return nil
	}
	out := make([]*searchService.AggregationResult, 0, len(aggs))
	for _, agg := range aggs {
		// Top-level metric: fold the matched hits through the sub-agg
		// accumulator, there is no facet to read from.
		if agg.GetMetricKind() != searchService.MetricKind_METRIC_KIND_UNSPECIFIED {
			acc := newSubAcc(agg)
			for _, hit := range res.Hits {
				accumulateHit(acc, agg, hit)
			}
			if r := emitAcc(acc, agg); r != nil {
				out = append(out, r)
			}
			continue
		}
		fr, ok := res.Facets[agg.GetField()]
		if !ok {
			continue
		}
		buckets := make([]*searchService.Bucket, 0)
		if len(aggregationRanges(agg)) > 0 {
			for _, nr := range fr.NumericRanges {
				buckets = append(buckets, &searchService.Bucket{
					Key:   nr.Name,
					Count: int64(nr.Count),
				})
			}
			for _, dr := range fr.DateRanges {
				buckets = append(buckets, &searchService.Bucket{
					Key:   dr.Name,
					Count: int64(dr.Count),
				})
			}
		} else {
			for _, t := range fr.Terms.Terms() {
				buckets = append(buckets, &searchService.Bucket{
					Key:   t.Term,
					Count: int64(t.Count),
				})
			}
		}
		if subAggs := agg.GetSubAggregations(); len(subAggs) > 0 {
			attachSubAggregations(res, agg.GetField(), subAggs, buckets)
		}
		out = append(out, &searchService.AggregationResult{
			Field:   agg.GetField(),
			Buckets: buckets,
		})
	}
	return out
}

// subAcc is the recursive accumulator emulating composite aggregations: one
// node per sub-aggregation under a parent bucket.
type subAcc struct {
	// terms: count + recursive accumulators per child value
	termCount map[string]int64
	termSubs  map[string][]*subAcc

	// metric
	metricVal float64 // SUM/MIN/MAX
	sum       float64 // AVG transport: numerator
	count     int64   // AVG transport: denominator

	seen bool // at least one hit contributed
}

// newSubAcc allocates an accumulator for the given sub-agg.
func newSubAcc(sa *searchService.AggregationOption) *subAcc {
	a := &subAcc{}
	if sa.GetMetricKind() == searchService.MetricKind_METRIC_KIND_UNSPECIFIED {
		a.termCount = map[string]int64{}
		if len(sa.GetSubAggregations()) > 0 {
			a.termSubs = map[string][]*subAcc{}
		}
	}
	return a
}

// accumulateHit folds one hit into a sub-agg accumulator, recursing into
// grand-sub-aggregations.
func accumulateHit(a *subAcc, sa *searchService.AggregationOption, hit *bleveSearch.DocumentMatch) {
	switch sa.GetMetricKind() {
	case searchService.MetricKind_METRIC_KIND_UNSPECIFIED:
		val, ok := hit.Fields[sa.GetField()].(string)
		if !ok || val == "" {
			return
		}
		a.termCount[val]++
		a.seen = true
		if subs := sa.GetSubAggregations(); len(subs) > 0 {
			childAccs, ok := a.termSubs[val]
			if !ok {
				childAccs = make([]*subAcc, len(subs))
				for i, ssa := range subs {
					childAccs[i] = newSubAcc(ssa)
				}
				a.termSubs[val] = childAccs
			}
			for i, ssa := range subs {
				accumulateHit(childAccs[i], ssa, hit)
			}
		}
	default:
		v, ok := numericFieldValue(hit.Fields[sa.GetField()])
		if !ok {
			return
		}
		switch sa.GetMetricKind() {
		case searchService.MetricKind_METRIC_KIND_SUM:
			a.metricVal += v
		case searchService.MetricKind_METRIC_KIND_MIN:
			if !a.seen || v < a.metricVal {
				a.metricVal = v
			}
		case searchService.MetricKind_METRIC_KIND_MAX:
			if !a.seen || v > a.metricVal {
				a.metricVal = v
			}
		case searchService.MetricKind_METRIC_KIND_AVG:
			a.sum += v
			a.count++
		}
		a.seen = true
	}
}

// emitAcc materialises a sub-agg accumulator into the proto result.
func emitAcc(a *subAcc, sa *searchService.AggregationOption) *searchService.AggregationResult {
	if sa.GetMetricKind() != searchService.MetricKind_METRIC_KIND_UNSPECIFIED {
		if !a.seen {
			return nil
		}
		r := &searchService.AggregationResult{
			Field:      sa.GetField(),
			MetricKind: sa.GetMetricKind(),
		}
		if sa.GetMetricKind() == searchService.MetricKind_METRIC_KIND_AVG {
			r.Sum = a.sum
			r.Count = a.count
		} else {
			r.Value = a.metricVal
		}
		return r
	}

	subs := sa.GetSubAggregations()
	childBuckets := make([]*searchService.Bucket, 0, len(a.termCount))
	for term, count := range a.termCount {
		b := &searchService.Bucket{Key: term, Count: count}
		if len(subs) > 0 {
			if childAccs, ok := a.termSubs[term]; ok {
				for i, ssa := range subs {
					if sub := emitAcc(childAccs[i], ssa); sub != nil {
						b.SubAggregations = append(b.SubAggregations, sub)
					}
				}
			}
		}
		childBuckets = append(childBuckets, b)
	}
	if sz := int(sa.GetSize()); sz > 0 && len(childBuckets) > sz {
		childBuckets = childBuckets[:sz]
	}
	return &searchService.AggregationResult{
		Field:   sa.GetField(),
		Buckets: childBuckets,
	}
}

// attachSubAggregations folds the matched hits into nested aggregation results
// per parent bucket, via a single hit walk dispatched through the accumulator tree.
func attachSubAggregations(res *bleve.SearchResult, parentField string, subAggs []*searchService.AggregationOption, buckets []*searchService.Bucket) {
	bucketByKey := make(map[string]*searchService.Bucket, len(buckets))
	for _, b := range buckets {
		bucketByKey[b.GetKey()] = b
	}

	perParent := make(map[string][]*subAcc, len(buckets))
	for _, b := range buckets {
		accs := make([]*subAcc, len(subAggs))
		for i, sa := range subAggs {
			accs[i] = newSubAcc(sa)
		}
		perParent[b.GetKey()] = accs
	}

	for _, hit := range res.Hits {
		parentVal, ok := hit.Fields[parentField].(string)
		if !ok || parentVal == "" {
			continue
		}
		accs, ok := perParent[parentVal]
		if !ok {
			continue
		}
		for i, sa := range subAggs {
			accumulateHit(accs[i], sa, hit)
		}
	}

	for key, accs := range perParent {
		b := bucketByKey[key]
		for i, sa := range subAggs {
			if r := emitAcc(accs[i], sa); r != nil {
				b.SubAggregations = append(b.SubAggregations, r)
			}
		}
	}
}

// numericFieldValue coerces a bleve stored value to float64 (also accepts
// string forms).
func numericFieldValue(raw interface{}) (float64, bool) {
	switch v := raw.(type) {
	case float64:
		return v, true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

func aggregationRanges(agg *searchService.AggregationOption) []*searchService.BucketRange {
	bd := agg.GetBucketDefinition()
	if bd == nil {
		return nil
	}
	return bd.GetRanges()
}

// rangeBucketKey formats a range as "from-to" for stable merge keys; open sides
// render as "-N" or "N-".
func rangeBucketKey(r *searchService.BucketRange) string {
	return r.GetFrom() + "-" + r.GetTo()
}

func parseFloatPtr(s string) *float64 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
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
