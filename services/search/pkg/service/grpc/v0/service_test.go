package service

import (
	"context"
	"testing"
	"time"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/jellydator/ttlcache/v2"
	"github.com/opencloud-eu/reva/v2/pkg/auth/scope"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/token/manager/jwt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go-micro.dev/v4/metadata"

	"github.com/opencloud-eu/opencloud/pkg/log"
	searchmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	searchmocks "github.com/opencloud-eu/opencloud/services/search/pkg/search/mocks"
)

func newTestService(t *testing.T, searcher *searchmocks.Searcher) (Service, context.Context) {
	t.Helper()

	tm, err := jwt.New(map[string]interface{}{"secret": "test-secret"})
	require.NoError(t, err)

	u := &user.User{Id: &user.UserId{OpaqueId: "test-user", Idp: "idp"}, Username: "test"}
	scopes, err := scope.AddOwnerScope(nil)
	require.NoError(t, err)
	tok, err := tm.MintToken(context.Background(), u, scopes)
	require.NoError(t, err)
	ctx := metadata.Set(context.Background(), revactx.TokenHeader, tok)

	cache := ttlcache.NewCache()
	require.NoError(t, cache.SetTTL(30*time.Second))

	logger := log.NopLogger()
	return Service{
		log:          &logger,
		searcher:     searcher,
		cache:        cache,
		tokenManager: tm,
	}, ctx
}

func TestServiceSearchForwardsOrderBy(t *testing.T) {
	searcher := searchmocks.NewSearcher(t)
	svc, ctx := newTestService(t, searcher)

	var captured *searchsvc.SearchRequest
	searcher.EXPECT().
		Search(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req *searchsvc.SearchRequest) {
			captured = req
		}).
		Return(&searchsvc.SearchResponse{}, nil).
		Once()

	err := svc.Search(ctx, &searchsvc.SearchRequest{
		Query: "mediatype:image",
		OrderBy: []*searchsvc.SortProperty{
			{Name: "photo.takenDateTime", IsDescending: true},
		},
	}, &searchsvc.SearchResponse{})
	require.NoError(t, err)
	require.NotNil(t, captured)
	require.Len(t, captured.OrderBy, 1)
	require.Equal(t, "photo.takenDateTime", captured.OrderBy[0].Name)
	require.True(t, captured.OrderBy[0].IsDescending)
}

func TestServiceSearchCacheDistinguishesOrderBy(t *testing.T) {
	searcher := searchmocks.NewSearcher(t)
	svc, ctx := newTestService(t, searcher)

	responseFor := func(name string) *searchsvc.SearchResponse {
		return &searchsvc.SearchResponse{
			TotalMatches: 1,
			Matches:      []*searchmsg.Match{{Entity: &searchmsg.Entity{Name: name}}},
		}
	}
	searcher.EXPECT().
		Search(mock.Anything, mock.MatchedBy(func(req *searchsvc.SearchRequest) bool {
			return len(req.OrderBy) > 0 && req.OrderBy[0].IsDescending
		})).
		Return(responseFor("newest.jpg"), nil).
		Once()
	searcher.EXPECT().
		Search(mock.Anything, mock.MatchedBy(func(req *searchsvc.SearchRequest) bool {
			return len(req.OrderBy) > 0 && !req.OrderBy[0].IsDescending
		})).
		Return(responseFor("oldest.jpg"), nil).
		Once()

	descOut := &searchsvc.SearchResponse{}
	require.NoError(t, svc.Search(ctx, &searchsvc.SearchRequest{
		Query:   "mediatype:image",
		OrderBy: []*searchsvc.SortProperty{{Name: "photo.takenDateTime", IsDescending: true}},
	}, descOut))

	ascOut := &searchsvc.SearchResponse{}
	require.NoError(t, svc.Search(ctx, &searchsvc.SearchRequest{
		Query:   "mediatype:image",
		OrderBy: []*searchsvc.SortProperty{{Name: "photo.takenDateTime"}},
	}, ascOut))

	require.Equal(t, "newest.jpg", descOut.Matches[0].Entity.Name)
	require.Equal(t, "oldest.jpg", ascOut.Matches[0].Entity.Name)
}
