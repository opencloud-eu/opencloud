package search

import (
	"context"
	"errors"
	"sync"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/golang-jwt/jwt/v5"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/walker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type authContextFunc func(context.Context) (context.Context, error)

type refreshingAuthContext struct {
	refreshInterval time.Duration
	now             func() time.Time
	authenticate    authContextFunc

	mu        sync.Mutex
	token     string
	refreshAt time.Time
}

// current attaches cached credentials to the caller's context, retaining its
// cancellation, deadline, values and unrelated outgoing metadata.
func (r *refreshingAuthContext) current(parent context.Context) (context.Context, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := parent.Err(); err != nil {
		return nil, err
	}
	now := r.now()
	if r.token == "" || !now.Before(r.refreshAt) {
		ctx, err := r.authenticate(parent)
		if err != nil {
			return nil, err
		}
		md, _ := metadata.FromOutgoingContext(ctx)
		tokens := md.Get(revactx.TokenHeader)
		if len(tokens) == 0 || tokens[len(tokens)-1] == "" {
			return nil, errors.New("indexing authentication returned no service token")
		}
		token := tokens[len(tokens)-1]
		refreshAt := now.Add(r.refreshInterval)
		// The token comes from our authenticated gateway. Reading exp here only
		// schedules renewal; authorization and signature validation stay with
		// the gateway. Opaque tokens retain the bounded fallback interval.
		claims := &jwt.RegisteredClaims{}
		if _, _, err := jwt.NewParser().ParseUnverified(token, claims); err == nil && claims.ExpiresAt != nil {
			remaining := claims.ExpiresAt.Time.Sub(r.now())
			if remaining <= 0 {
				return nil, errors.New("indexing authentication returned an expired service token")
			}
			if earlier := r.now().Add(remaining / 2); earlier.Before(refreshAt) {
				refreshAt = earlier
			}
		}
		r.token, r.refreshAt = token, refreshAt
	}
	md, _ := metadata.FromOutgoingContext(parent)
	md = md.Copy()
	md.Set(revactx.TokenHeader, r.token)
	return metadata.NewOutgoingContext(parent, md), nil
}

type indexSpaceGatewaySelector struct {
	pool.Selectable[gateway.GatewayAPIClient]
	authContext *refreshingAuthContext
}

func (s *indexSpaceGatewaySelector) Next(opts ...pool.Option) (gateway.GatewayAPIClient, error) {
	client, err := s.Selectable.Next(opts...)
	if err != nil {
		return nil, err
	}
	return &indexSpaceGatewayClient{
		GatewayAPIClient: client,
		authContext:      s.authContext,
	}, nil
}

type indexSpaceGatewayClient struct {
	gateway.GatewayAPIClient
	authContext *refreshingAuthContext
}

func (c *indexSpaceGatewayClient) Stat(parent context.Context, req *provider.StatRequest, opts ...grpc.CallOption) (*provider.StatResponse, error) {
	ctx, err := c.authContext.current(parent)
	if err != nil {
		return nil, err
	}
	return c.GatewayAPIClient.Stat(ctx, req, opts...)
}

func (c *indexSpaceGatewayClient) ListContainer(parent context.Context, req *provider.ListContainerRequest, opts ...grpc.CallOption) (*provider.ListContainerResponse, error) {
	ctx, err := c.authContext.current(parent)
	if err != nil {
		return nil, err
	}
	return c.GatewayAPIClient.ListContainer(ctx, req, opts...)
}

func newIndexSpaceWalker(
	selector pool.Selectable[gateway.GatewayAPIClient],
	refreshInterval time.Duration,
	now func() time.Time,
	authenticate authContextFunc,
) (walker.Walker, *refreshingAuthContext) {
	authContext := &refreshingAuthContext{
		refreshInterval: refreshInterval,
		now:             now,
		authenticate:    authenticate,
	}
	return walker.NewWalker(&indexSpaceGatewaySelector{
		Selectable:  selector,
		authContext: authContext,
	}), authContext
}
