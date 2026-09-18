package service

import (
	"context"
	"errors"
	"fmt"
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
	grpcmetadata "google.golang.org/grpc/metadata"

	"github.com/opencloud-eu/opencloud/pkg/log"
	v0 "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

// _spaceStateTrashed is the value of the "trashed" opaque entry that the storage
// provider sets on disabled spaces.
const _spaceStateTrashed = "trashed"

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

	key := cacheKey(in.Query, in.PageSize, in.Ref, u)
	res, ok := s.FromCache(key)
	if !ok {
		var err error
		res, err = s.searcher.Search(ctx, &searchsvc.SearchRequest{
			Query:    in.Query,
			PageSize: in.PageSize,
			Ref:      in.Ref,
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
	return nil
}

// IndexSpace (re)indexes all resources of a given space.
func (s Service) IndexSpace(_ context.Context, in *searchsvc.IndexSpaceRequest, _ *searchsvc.IndexSpaceResponse) error {
	if in.GetSpaceId() != "" {
		return s.searcher.IndexSpace(&provider.StorageSpaceId{OpaqueId: in.GetSpaceId()}, in.GetForceReindex())
	}

	// index all spaces instead
	gwc, err := s.gws.Next()
	if err != nil {
		return err
	}

	ctx, err := utils.GetServiceUserContext(s.cfg.ServiceAccount.ServiceAccountID, gwc, s.cfg.ServiceAccount.ServiceAccountSecret)
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

	var skipped int
	for _, space := range resp.GetStorageSpaces() {
		if utils.ReadPlainFromOpaque(space.GetOpaque(), _spaceStateTrashed) == _spaceStateTrashed {
			// A disabled space cannot be traversed, the storage provider answers with
			// "not found" for its root. Skip it instead of failing the whole run, it
			// has to be indexed again once it has been enabled.
			skipped++
			s.log.Warn().
				Str("space_id", space.GetId().GetOpaqueId()).
				Msg("skipping disabled space, run the index command again after enabling it")
			continue
		}

		if err := s.searcher.IndexSpace(space.GetId(), in.GetForceReindex()); err != nil {
			return err
		}
	}

	if skipped > 0 {
		s.log.Warn().
			Int("skipped_spaces", skipped).
			Msg("skipped disabled spaces, they have to be indexed again after they have been enabled")
	}

	return nil
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

func cacheKey(query string, pagesize int32, ref *v0.Reference, user *user.User) string {
	return fmt.Sprintf("%s|%d|%s$%s!%s/%s|%s", query, pagesize, ref.GetResourceId().GetStorageId(), ref.GetResourceId().GetSpaceId(), ref.GetResourceId().GetOpaqueId(), ref.GetPath(), user.GetId().GetOpaqueId())
}
