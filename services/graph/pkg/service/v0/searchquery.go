package svc

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/render"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	revaCtx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	merrors "go-micro.dev/v4/errors"
	"go-micro.dev/v4/metadata"

	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	"github.com/opencloud-eu/opencloud/services/search/pkg/aggregation"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// SearchQuery handles POST /v1beta1/search/query (MS Graph searchQuery).
func (g Graph) SearchQuery(w http.ResponseWriter, r *http.Request) {
	var req libregraph.SearchQueryRequest
	if err := StrictJSONUnmarshal(r.Body, &req); err != nil {
		g.logger.Debug().Err(err).Msg("could not decode search query request")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid body schema definition")
		return
	}

	if len(req.Requests) == 0 {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "requests array must not be empty")
		return
	}

	if err := validateSearchExpand(r); err != nil {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}

	// every request is validated before the first one runs
	searches := make([]*searchsvc.SearchRequest, 0, len(req.Requests))
	for _, sr := range req.Requests {
		if property := unsupportedProperty(sr); property != "" {
			errorcode.NotSupported.Render(w, r, http.StatusNotImplemented, property+" is not supported yet")
			return
		}
		prepared, err := searchRequestOf(sr)
		if err != nil {
			errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
			return
		}
		searches = append(searches, prepared)
	}

	th := r.Header.Get(revaCtx.TokenHeader)
	ctx := revaCtx.ContextSetToken(r.Context(), th)
	ctx = metadata.Set(ctx, revaCtx.TokenHeader, th)

	expandThumbnails := driveItemRelationExpanded(r, _expandThumbnails)

	responses := make([]libregraph.SearchResponse, 0, len(req.Requests))
	for i, sr := range req.Requests {
		sresp, err := g.runSingleSearch(ctx, sr, searches[i], expandThumbnails)
		if err != nil {
			g.renderSearchError(w, r, err)
			return
		}
		responses = append(responses, sresp)
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, libregraph.SearchQuery200Response{Value: responses})
}

func validateSearchExpand(r *http.Request) error {
	for _, values := range r.URL.Query()["$expand"] {
		for _, relation := range strings.Split(values, ",") {
			if relation != _expandThumbnails {
				return fmt.Errorf("unsupported $expand %q; only %s is supported", relation, _expandThumbnails)
			}
		}
	}
	return nil
}

// unsupportedProperty names a property of the spec this endpoint does not
// evaluate yet.
func unsupportedProperty(sr libregraph.SearchRequest) string {
	if len(sr.SortProperties) > 0 {
		return "sortProperties"
	}
	if len(sr.AggregationFilters) > 0 {
		return "aggregationFilters"
	}
	for _, a := range sr.Aggregations {
		switch {
		case a.LibreGraphGeohashDefinition != nil:
			return "@libre.graph.geohashDefinition"
		case len(a.LibreGraphSubAggregations) > 0:
			return "@libre.graph.subAggregations"
		case a.LibreGraphMetricDefinition != nil:
			return "@libre.graph.metricDefinition"
		}
	}
	return ""
}

// searchRequestOf validates one request against the spec and the index rules
// (the search service checks them again) and translates it. Fields are
// validated under the request's names so an error reads like the request.
func searchRequestOf(sr libregraph.SearchRequest) (*searchsvc.SearchRequest, error) {
	if err := validateEntityTypes(sr.EntityTypes); err != nil {
		return nil, err
	}
	if err := validatePagination(sr.From, sr.Size); err != nil {
		return nil, err
	}
	if err := validateAggregations(sr.Aggregations); err != nil {
		return nil, err
	}
	aggregations := libregraphAggregationsToSearch(sr.Aggregations)
	requestFieldType := func(field string) string { return search.AggregatableFieldType(indexField(field)) }
	if err := aggregation.ValidateOptions(aggregations, requestFieldType); err != nil {
		return nil, err
	}
	resolveFields(aggregations)

	from, size := pagination(sr.From, sr.Size)
	return &searchsvc.SearchRequest{
		Query:        sr.Query.QueryString,
		From:         from,
		PageSize:     &size,
		Aggregations: aggregations,
	}, nil
}

func validateEntityTypes(entityTypes []string) error {
	if len(entityTypes) == 0 {
		return fmt.Errorf("entityTypes must contain at least one entry")
	}
	for _, t := range entityTypes {
		if t != "driveItem" {
			return fmt.Errorf("unsupported entity type %q; only driveItem is supported", t)
		}
	}
	return nil
}

func (g Graph) runSingleSearch(ctx context.Context, sr libregraph.SearchRequest, prepared *searchsvc.SearchRequest, expandThumbnails bool) (libregraph.SearchResponse, error) {
	from, size := prepared.GetFrom(), prepared.GetPageSize()

	rsp, err := g.searchService.Search(ctx, prepared)
	if err != nil {
		return libregraph.SearchResponse{}, err
	}

	// the caller's id decides @libre.graph.me.following (the WebDAV report's oc:favorite)
	uid := ""
	if u, ok := revaCtx.ContextGetUser(ctx); ok {
		uid = u.GetId().GetOpaqueId()
	}

	hits := make([]libregraph.SearchHit, 0, len(rsp.Matches))
	for i, match := range rsp.Matches {
		item := searchEntityToDriveItem(match.GetEntity(), uid)
		item.WebUrl = webURLForID(g.publicBaseURL, item.GetId())
		if expandThumbnails {
			setDriveItemThumbnailsByID(item, item.GetId(), g.config.Commons.OpenCloudURL)
		}
		hit := libregraph.SearchHit{HitId: item.Id, Rank: libregraph.PtrInt32(from + int32(i) + 1), Resource: item}
		if h := match.GetEntity().GetHighlights(); h != "" {
			hit.Summary = libregraph.PtrString(h)
		}
		hits = append(hits, hit)
	}

	total := int64(rsp.TotalMatches)
	// a next page has to exist and to be within reach
	next := int64(from) + int64(size)
	more := next < total && next < search.MaxResultWindow
	return libregraph.SearchResponse{
		SearchTerms: []string{sr.Query.QueryString},
		HitsContainers: []libregraph.SearchHitsContainer{{
			Hits:                 hits,
			Total:                &total,
			MoreResultsAvailable: &more,
			Aggregations:         searchAggregationsToLibregraph(rsp.Aggregations, sr.Aggregations),
		}},
	}, nil
}

// the spec's default and upper bound of SearchRequest.size
const (
	defaultPageSize = 25
	maxPageSize     = 500
)

func pagination(fromP, sizeP *int32) (from, size int32) {
	from, size = 0, defaultPageSize
	if fromP != nil {
		from = *fromP
	}
	if sizeP != nil {
		size = *sizeP
	}
	return from, size
}

// openapi-generator does not enforce the spec's [0,inf)/[0,500] bounds
func validatePagination(fromP, sizeP *int32) error {
	from, size := pagination(fromP, sizeP)
	if from < 0 {
		return fmt.Errorf("from must not be negative")
	}
	if size < 0 || size > maxPageSize {
		return fmt.Errorf("size must be between 0 and %d", maxPageSize)
	}
	return search.CheckResultWindow(from, size)
}

// openapi-generator enforces neither the enums nor the bounds of the spec.
// Whether an aggregation fits its field is for aggregation.ValidateOptions to
// say.
func validateAggregations(aggs []libregraph.AggregationOption) error {
	for _, a := range aggs {
		if a.Field == "" {
			return fmt.Errorf("aggregation field must not be empty")
		}
		if a.Size != nil && *a.Size < 1 {
			return fmt.Errorf("size of the aggregation on %q must be at least 1", a.Field)
		}
		if bd := a.BucketDefinition; bd != nil {
			if _, ok := bucketSortBy[bd.SortBy]; !ok {
				return fmt.Errorf("unsupported sortBy %q on field %q", bd.SortBy, a.Field)
			}
			if bd.MinimumCount != nil && *bd.MinimumCount < 0 {
				return fmt.Errorf("minimumCount of the aggregation on %q must not be negative", a.Field)
			}
		}
	}
	return nil
}

// renderSearchError answers with the status the search service failed with.
func (g Graph) renderSearchError(w http.ResponseWriter, r *http.Request, err error) {
	e := merrors.Parse(err.Error())
	switch e.Code {
	case http.StatusBadRequest:
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, e.Detail)
	case http.StatusRequestTimeout, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		g.logger.Error().Err(err).Msg("search service did not answer")
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusServiceUnavailable, e.Detail)
	default:
		g.logger.Error().Err(err).Msg("search service call failed")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, e.Detail)
	}
}
