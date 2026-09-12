package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/jellydator/ttlcache/v2"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/token"
	"github.com/opencloud-eu/reva/v2/pkg/token/manager/jwt"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	merrors "go-micro.dev/v4/errors"
	"go-micro.dev/v4/metadata"
	"golang.org/x/sync/errgroup"
	grpcmetadata "google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/opencloud-eu/opencloud/pkg/log"
	v0 "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// NewHandler returns a service implementation for Service.
func NewHandler(opts ...Option) (searchsvc.SearchProviderHandler, error) {
	options := newOptions(opts...)
	cfg := options.Config
	if options.GatewaySelector == nil {
		return nil, errors.New("no GatewaySelector provided")
	}
	if options.Searcher == nil {
		return nil, errors.New("no Searcher provided")
	}

	cache := ttlcache.NewCache()
	if err := cache.SetTTL(time.Second); err != nil {
		return nil, err
	}

	tokenManager, err := jwt.New(map[string]any{
		"secret":  options.JWTSecret,
		"expires": int64(24 * 60 * 60),
	})
	if err != nil {
		return nil, err
	}

	return &Service{
		id:           cfg.GRPC.Namespace + "." + cfg.Service.Name,
		log:          &options.Logger,
		searcher:     options.Searcher,
		cache:        cache,
		tokenManager: tokenManager,
		gws:          options.GatewaySelector,
		cfg:          cfg,
	}, nil
}

// Service implements the searchServiceHandler interface
type Service struct {
	id           string
	log          *log.Logger
	searcher     search.Searcher
	cache        *ttlcache.Cache
	tokenManager token.Manager
	gws          *pool.Selector[gateway.GatewayAPIClient]
	cfg          *config.Config
}

// Search handles the search
func (s Service) Search(ctx context.Context, in *searchsvc.SearchRequest, out *searchsvc.SearchResponse) error {
	// Get token from the context (go-micro) and make it known to the reva client too (grpc)
	t, ok := metadata.Get(ctx, revactx.TokenHeader)
	if !ok {
		s.log.Error().Msg("Could not get token from context")
		return errors.New("could not get token from context")
	}
	ctx = grpcmetadata.AppendToOutgoingContext(ctx, revactx.TokenHeader, t)

	// unpack user
	u, _, err := s.tokenManager.DismantleToken(ctx, t)
	if err != nil {
		return err
	}
	ctx = revactx.ContextSetUser(ctx, u)

	key := cacheKey(in.Query, in.PageSize, in.Ref, u, in.Aggregations)
	res, ok := s.FromCache(key)
	if !ok {
		var err error
		res, err = s.searcher.Search(ctx, &searchsvc.SearchRequest{
			Query:        in.Query,
			PageSize:     in.PageSize,
			Ref:          in.Ref,
			Aggregations: in.Aggregations,
		})
		if err != nil {
			switch err.(type) {
			case errtypes.BadRequest:
				return merrors.BadRequest(s.id, "%s", err.Error())
			default:
				return merrors.InternalServerError(s.id, "%s", err.Error())
			}
		}

		s.Cache(key, res)
	}

	out.Matches = res.Matches
	out.TotalMatches = res.TotalMatches
	out.NextPageToken = res.NextPageToken
	out.Aggregations = res.Aggregations
	return nil
}

// IndexSpace (re)indexes all resources of a given space. Progress information is
// streamed back to the caller after every space that has been indexed.
func (s Service) IndexSpace(_ context.Context, in *searchsvc.IndexSpaceRequest, stream searchsvc.SearchProvider_IndexSpaceStream) error {
	// Use the stream's context so that indexing stops when the client cancels
	// the request or disconnects.
	ctx := stream.Context()

	if in.GetSpaceId() != "" {
		err := s.searcher.IndexSpace(&provider.StorageSpaceId{OpaqueId: in.GetSpaceId()}, in.GetForceReindex())
		resp := &searchsvc.IndexSpaceResponse{
			SpaceId:       in.GetSpaceId(),
			IndexedSpaces: 1,
			TotalSpaces:   1,
		}
		if err != nil {
			resp.Error = err.Error()
		}
		if sendErr := stream.Send(resp); sendErr != nil {
			return sendErr
		}
		return err
	}

	// index all spaces instead
	gwc, err := s.gws.Next()
	if err != nil {
		return err
	}

	ctx, err = utils.GetServiceUserContextWithContext(ctx, gwc, s.cfg.ServiceAccount.ServiceAccountID, s.cfg.ServiceAccount.ServiceAccountSecret)
	if err != nil {
		return err
	}

	resp, err := gwc.ListStorageSpaces(ctx, &provider.ListStorageSpacesRequest{})
	if err != nil {
		return err
	}

	if resp.GetStatus().GetCode() != rpc.Code_CODE_OK {
		return errors.New(resp.GetStatus().GetMessage())
	}

	spaces := resp.GetStorageSpaces()
	totalSpaces := int64(len(spaces))

	// Index all spaces concurrently, limited to a configurable number of spaces
	// being reindexed at the same time. The errgroup context is cancelled as
	// soon as the client goes away or a stream send fails, so the remaining
	// goroutines stop indexing early.
	concurrency := max(s.cfg.ReindexMaxConcurrency, 1)
	concurrency = min(concurrency, int(in.GetConcurrency()))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	// Serialize progress updates on the stream, as gRPC streams must not be
	// written to from multiple goroutines concurrently.
	var (
		mu           sync.Mutex
		indexedCount int64
	)

	for _, space := range spaces {
		// Stop early if the client cancelled the request.
		if err := ctx.Err(); err != nil {
			return err
		}
		g.Go(func() error {
			s.log.Info().Str("space_id", space.GetId().GetOpaqueId()).Msg("indexing space")
			t := time.Now()

			indexErr := s.searcher.IndexSpace(space.GetId(), in.GetForceReindex())
			if indexErr != nil {
				s.log.Error().Err(indexErr).Str("space_id", space.GetId().GetOpaqueId()).Msg("failed to index space")
			} else {
				s.log.Info().Str("space_id", space.GetId().GetOpaqueId()).Msg("finished indexing space")
			}

			mu.Lock()
			defer mu.Unlock()

			// Don't try to send progress on an already cancelled stream.
			if err := ctx.Err(); err != nil {
				return err
			}

			indexedCount++
			progress := &searchsvc.IndexSpaceResponse{
				SpaceId:       space.GetId().GetOpaqueId(),
				IndexedSpaces: indexedCount,
				TotalSpaces:   totalSpaces,
				SpaceDuration: durationpb.New(time.Since(t)),
			}
			if indexErr != nil {
				progress.Error = indexErr.Error()
			}
			return stream.Send(progress)
		})
	}

	return g.Wait()
}

// FromCache pulls a search result from cache
func (s Service) FromCache(key string) (*searchsvc.SearchResponse, bool) {
	v, err := s.cache.Get(key)
	if err != nil {
		return nil, false
	}

	sr, ok := v.(*searchsvc.SearchResponse)
	return sr, ok
}

// Cache caches the search result
func (s Service) Cache(key string, res *searchsvc.SearchResponse) {
	// lets ignore the error
	_ = s.cache.Set(key, res)
}

// cacheKey builds the cache identity for a search. Every result-affecting field
// must be in the key, including aggregations (serialised via deterministic proto
// marshalling). If the aggregation proto ever gains a map field, determinism
// requires all writers to set Deterministic=true.
func cacheKey(query string, pagesize int32, ref *v0.Reference, user *user.User, aggs []*searchsvc.AggregationOption) string {
	aggPart := ""
	if len(aggs) > 0 {
		b, _ := proto.MarshalOptions{Deterministic: true}.Marshal(&searchsvc.SearchRequest{Aggregations: aggs})
		aggPart = string(b)
	}
	return fmt.Sprintf("%s|%d|%s$%s!%s/%s|%s|%s",
		query, pagesize,
		ref.GetResourceId().GetStorageId(), ref.GetResourceId().GetSpaceId(), ref.GetResourceId().GetOpaqueId(),
		ref.GetPath(), user.GetId().GetOpaqueId(), aggPart)
}
