package svc

import (
	"maps"
	"net/http"
	"slices"
	"strings"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/render"
	searchmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	revaCtx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/tags"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"go-micro.dev/v4/metadata"
)

// maxListedTags is the maximum number of tags that GetTags returns. It keeps
// the response small when many distinct tags are in use. Since the tags are
// sorted before the list is cut, the same tags are always the ones returned.
const maxListedTags = 100

// listedTags returns the distinct tags of the given search matches, sorted in
// byte order, and cut at maxListedTags.
//
// Byte order means that upper case comes before lower case ("Banana" before
// "apple"). It is used on purpose instead of a case-insensitive order, which
// would be nicer to read, because it is the order in which search engines sort
// keyword values, e.g. an OpenSearch terms aggregation ordered by key. With the
// same order here, the search can return just the first maxListedTags tags of
// every space, and merging those lists still gives exactly the first
// maxListedTags tags overall. With a case-insensitive order, that only works by
// aggregating on a lowercased copy of the field, which needs a mapping change
// and a reindex, and loses the original casing of the tags. Clients that want
// another order for display can sort the list themselves.
func listedTags(matches []*searchmsg.Match) []string {
	distinct := make(map[string]struct{})
	for _, match := range matches {
		for _, tag := range match.GetEntity().GetTags() {
			if tag != "" {
				distinct[tag] = struct{}{}
			}
		}
	}

	// never nil: an empty list is rendered as "value": [], not left out
	list := slices.AppendSeq(make([]string, 0, len(distinct)), maps.Keys(distinct))
	slices.Sort(list)
	return list[:min(len(list), maxListedTags)]
}

// GetTags returns the tags in use, see listedTags
func (g Graph) GetTags(w http.ResponseWriter, r *http.Request) {
	th := r.Header.Get(revaCtx.TokenHeader)
	ctx := revaCtx.ContextSetToken(r.Context(), th)
	ctx = metadata.Set(ctx, revaCtx.TokenHeader, th)
	sr, err := g.searchService.Search(ctx, &searchsvc.SearchRequest{
		Query:    "Tags:*",
		PageSize: -1,
	})
	if err != nil {
		g.logger.Error().Err(err).Msg("Could not search for tags")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tagCollection := libregraph.NewCollectionOfTags()
	tagCollection.Value = listedTags(sr.GetMatches())

	render.Status(r, http.StatusOK)
	render.JSON(w, r, tagCollection)
}

// AssignTags assigns a tag to a resource
func (g Graph) AssignTags(w http.ResponseWriter, r *http.Request) {
	var (
		assignment libregraph.TagAssignment
		ctx        = r.Context()
	)

	if err := StrictJSONUnmarshal(r.Body, &assignment); err != nil {
		g.logger.Debug().Err(err).Interface("body", r.Body).Msg("could not decode tag assignment request")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid body schema definition")
		return
	}

	rid, err := storagespace.ParseID(assignment.ResourceId)
	if err != nil {
		g.logger.Debug().Err(err).Msg("could not parse resourceId")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid resourceId")
		return
	}

	client, err := g.gatewaySelector.Next()
	if err != nil {
		g.logger.Error().Err(err).Msg("error selecting next gateway client")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	sres, err := client.Stat(ctx, &provider.StatRequest{
		Ref: &provider.Reference{ResourceId: &rid},
	})
	if err != nil {
		g.logger.Error().Err(err).Msg("error stating file")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if sres.GetStatus().GetCode() != rpc.Code_CODE_OK {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "can't stat resource")
		return
	}

	// use resource id from stat response to work on the actual resource and not a share jail item
	rid = *sres.GetInfo().GetId()

	pm := sres.GetInfo().GetPermissionSet()
	if pm == nil {
		g.logger.Error().Err(err).Msg("no permissionset on file")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// it says we need "write access" to set tags. One of those should do
	if !pm.InitiateFileUpload && !pm.CreateContainer {
		g.logger.Info().Msg("no permission to create a tag")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	var currentTags string
	if m := sres.GetInfo().GetArbitraryMetadata().GetMetadata(); m != nil {
		currentTags = m["tags"]
	}

	allTags := tags.New(currentTags)
	if !allTags.Add(assignment.Tags...) {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "no new tags in createtagsrequest or maximum reached")
		return
	}

	resp, err := client.SetArbitraryMetadata(ctx, &provider.SetArbitraryMetadataRequest{
		Ref: &provider.Reference{ResourceId: &rid},
		ArbitraryMetadata: &provider.ArbitraryMetadata{
			Metadata: map[string]string{
				"tags": allTags.AsList(),
			},
		},
	})
	if err != nil || resp.GetStatus().GetCode() != rpc.Code_CODE_OK {
		g.logger.Error().Err(err).Interface("status", resp.GetStatus()).Msg("error setting tags")
		if resp.GetStatus().GetCode() == rpc.Code_CODE_LOCKED {
			errorcode.InvalidRequest.Render(w, r, http.StatusLocked, "file is locked")
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if g.eventsPublisher != nil {
		ev := events.TagsAdded{
			Tags: strings.Join(assignment.Tags, ","),
			Ref: &provider.Reference{
				ResourceId: &rid,
				Path:       ".",
			},
			SpaceOwner: sres.Info.Owner,
			Executant:  revaCtx.ContextMustGetUser(r.Context()).Id,
		}
		if err := events.Publish(r.Context(), g.eventsPublisher, ev); err != nil {
			g.logger.Error().Err(err).Msg("Failed to publish TagsAdded event")
		}
	}
}

// UnassignTags removes a tag from a resource
func (g Graph) UnassignTags(w http.ResponseWriter, r *http.Request) {
	var (
		unassignment libregraph.TagUnassignment
		ctx          = r.Context()
	)

	if err := StrictJSONUnmarshal(r.Body, &unassignment); err != nil {
		g.logger.Debug().Err(err).Interface("body", r.Body).Msg("could not decode tag assignment request")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid body schema definition")
		return
	}

	rid, err := storagespace.ParseID(unassignment.ResourceId)
	if err != nil {
		g.logger.Debug().Err(err).Msg("could not parse resourceId")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid resourceId")
		return
	}

	client, err := g.gatewaySelector.Next()
	if err != nil {
		g.logger.Error().Err(err).Msg("error selecting next gateway client")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	sres, err := client.Stat(ctx, &provider.StatRequest{
		Ref: &provider.Reference{ResourceId: &rid},
	})
	if err != nil {
		g.logger.Error().Err(err).Msg("error stating file")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if sres.GetStatus().GetCode() != rpc.Code_CODE_OK {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "can't stat resource")
		return
	}

	// use resource id from stat response to work on the actual resource and not a share jail item
	rid = *sres.GetInfo().GetId()

	pm := sres.GetInfo().GetPermissionSet()
	if pm == nil {
		g.logger.Error().Err(err).Msg("no permissionset on file")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// it says we need "write access" to set tags. One of those should do
	if !pm.InitiateFileUpload && !pm.CreateContainer {
		g.logger.Info().Msg("no permission to create a tag")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	var currentTags string
	if m := sres.GetInfo().GetArbitraryMetadata().GetMetadata(); m != nil {
		currentTags = m["tags"]
	}

	allTags := tags.New(currentTags)
	if !allTags.Remove(unassignment.Tags...) {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "no new tags in createtagsrequest or maximum reached")
		return
	}

	resp, err := client.SetArbitraryMetadata(ctx, &provider.SetArbitraryMetadataRequest{
		Ref: &provider.Reference{ResourceId: &rid},
		ArbitraryMetadata: &provider.ArbitraryMetadata{
			Metadata: map[string]string{
				"tags": allTags.AsList(),
			},
		},
	})
	if err != nil || resp.GetStatus().GetCode() != rpc.Code_CODE_OK {
		g.logger.Error().Err(err).Msg("error setting tags")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if g.eventsPublisher != nil {
		ev := events.TagsRemoved{
			Tags: strings.Join(unassignment.Tags, ","),
			Ref: &provider.Reference{
				ResourceId: &rid,
				Path:       ".",
			},
			SpaceOwner: sres.Info.Owner,
			Executant:  revaCtx.ContextMustGetUser(r.Context()).Id,
		}
		if err := events.Publish(ctx, g.eventsPublisher, ev); err != nil {
			g.logger.Error().Err(err).Msg("Failed to publish TagsAdded event")
		}
	}
}
