package search

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/golang-jwt/jwt/v5"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/metadata"
)

type staticGatewaySelector struct {
	client gateway.GatewayAPIClient
}

func indexTestSigningKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("generate test signing key: %v", err)
	}
	return key
}

func TestIndexAuthRefreshUsesTokenExpiry(t *testing.T) {
	signingKey := indexTestSigningKey(t)
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	calls := 0
	auth := &refreshingAuthContext{
		refreshInterval: 12 * time.Hour,
		now:             func() time.Time { return now },
		authenticate: func(ctx context.Context) (context.Context, error) {
			calls++
			token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			}).SignedString(signingKey)
			if err != nil {
				return nil, err
			}
			return metadata.AppendToOutgoingContext(ctx, revactx.TokenHeader, token), nil
		},
	}
	if _, err := auth.current(context.Background()); err != nil {
		t.Fatal(err)
	}
	now = now.Add(29 * time.Minute)
	if _, err := auth.current(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("premature refresh: %d calls", calls)
	}
	now = now.Add(time.Minute)
	if _, err := auth.current(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("short-lived token not refreshed: %d calls", calls)
	}
}

func TestIndexAuthPreservesCallerContext(t *testing.T) {
	type key struct{}
	parent, cancel := context.WithTimeout(context.WithValue(context.Background(), key{}, "value"), time.Minute)
	defer cancel()
	parent = metadata.AppendToOutgoingContext(parent, "trace-id", "trace", revactx.TokenHeader, "old-token")
	auth := &refreshingAuthContext{refreshInterval: time.Hour, now: time.Now,
		authenticate: func(ctx context.Context) (context.Context, error) {
			if ctx != parent {
				t.Fatal("authentication lost parent context")
			}
			return metadata.AppendToOutgoingContext(ctx, revactx.TokenHeader, "new-token"), nil
		},
	}
	ctx, err := auth.current(parent)
	if err != nil {
		t.Fatal(err)
	}
	wantDeadline, _ := parent.Deadline()
	gotDeadline, ok := ctx.Deadline()
	if !ok || !gotDeadline.Equal(wantDeadline) || ctx.Value(key{}) != "value" {
		t.Fatal("lost deadline or context value")
	}
	md, _ := metadata.FromOutgoingContext(ctx)
	if fmt.Sprint(md.Get("trace-id")) != "[trace]" || fmt.Sprint(md.Get(revactx.TokenHeader)) != "[new-token]" {
		t.Fatalf("unexpected metadata keys/token multiplicity")
	}
	original, _ := metadata.FromOutgoingContext(parent)
	if fmt.Sprint(original.Get(revactx.TokenHeader)) != "[old-token]" {
		t.Fatal("mutated parent metadata")
	}
	cancel()
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatal("lost cancellation")
	}
	if _, err := auth.current(parent); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled parent: %v", err)
	}
}

func TestIndexAuthRefreshFailureIsPropagated(t *testing.T) {
	now := time.Now()
	want := errors.New("authentication unavailable")
	calls := 0
	auth := &refreshingAuthContext{refreshInterval: time.Hour, now: func() time.Time { return now },
		authenticate: func(ctx context.Context) (context.Context, error) {
			calls++
			if calls > 1 {
				return nil, want
			}
			return metadata.AppendToOutgoingContext(ctx, revactx.TokenHeader, "first-token"), nil
		},
	}
	if _, err := auth.current(context.Background()); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Hour)
	for i := 0; i < 2; i++ {
		if ctx, err := auth.current(context.Background()); !errors.Is(err, want) || ctx != nil {
			t.Fatalf("refresh failure swallowed: %v", err)
		}
	}
}

func TestIndexAuthRejectsMissingOrExpiredToken(t *testing.T) {
	for _, expired := range []bool{false, true} {
		t.Run(fmt.Sprintf("expired=%v", expired), func(t *testing.T) {
			signingKey := indexTestSigningKey(t)
			auth := &refreshingAuthContext{refreshInterval: time.Hour, now: time.Now,
				authenticate: func(ctx context.Context) (context.Context, error) {
					if !expired {
						return ctx, nil
					}
					token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour))}).SignedString(signingKey)
					if err != nil {
						return nil, err
					}
					return metadata.AppendToOutgoingContext(ctx, revactx.TokenHeader, token), nil
				},
			}
			if _, err := auth.current(context.Background()); err == nil {
				t.Fatal("invalid authentication result accepted")
			}
		})
	}
}

func TestIndexAuthConcurrentCallsShareCredentials(t *testing.T) {
	calls := 0
	auth := &refreshingAuthContext{refreshInterval: time.Hour, now: time.Now,
		authenticate: func(ctx context.Context) (context.Context, error) {
			calls++
			return metadata.AppendToOutgoingContext(ctx, revactx.TokenHeader, "shared-token"), nil
		},
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := auth.current(context.Background()); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if calls != 1 {
		t.Fatalf("got %d authentications, want 1", calls)
	}
}

func (s staticGatewaySelector) Next(...pool.Option) (gateway.GatewayAPIClient, error) {
	return s.client, nil
}

func TestIndexSpaceWalkerRefreshesServiceTokenDuringLongWalk(t *testing.T) {
	currentTime := time.Date(2026, time.August, 19, 0, 0, 0, 0, time.UTC)
	authCalls := 0
	authenticate := func(ctx context.Context) (context.Context, error) {
		authCalls++
		return metadata.AppendToOutgoingContext(
			ctx,
			revactx.TokenHeader,
			fmt.Sprintf("token-%d", authCalls),
		), nil
	}

	rootID := &provider.ResourceId{StorageId: "storage", SpaceId: "space", OpaqueId: "root"}
	root := &provider.ResourceInfo{
		Id:   rootID,
		Path: ".",
		Type: provider.ResourceType_RESOURCE_TYPE_CONTAINER,
	}
	child := &provider.ResourceInfo{
		Id:   &provider.ResourceId{StorageId: "storage", SpaceId: "space", OpaqueId: "child"},
		Path: "document.pdf",
		Type: provider.ResourceType_RESOURCE_TYPE_FILE,
	}

	client := &cs3mocks.GatewayAPIClient{}
	observedTokens := make([]string, 0, 2)
	captureToken := func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)
		md, ok := metadata.FromOutgoingContext(ctx)
		tokens := md.Get(revactx.TokenHeader)
		if !ok || len(tokens) != 1 {
			t.Fatal("gateway call did not receive a service token")
		}
		observedTokens = append(observedTokens, tokens[0])
	}
	client.On("Stat", mock.Anything, mock.Anything, mock.Anything).
		Run(captureToken).
		Return(&provider.StatResponse{Status: &rpc.Status{Code: rpc.Code_CODE_OK}, Info: root}, nil).
		Once()
	client.On("ListContainer", mock.Anything, mock.Anything, mock.Anything).
		Run(captureToken).
		Return(&provider.ListContainerResponse{
			Status: &rpc.Status{Code: rpc.Code_CODE_OK},
			Infos:  []*provider.ResourceInfo{child},
		}, nil).
		Once()

	w, _ := newIndexSpaceWalker(
		staticGatewaySelector{client: client},
		time.Hour,
		func() time.Time { return currentTime },
		authenticate,
	)
	visited := make([]string, 0, 2)
	err := w.Walk(context.Background(), rootID, func(_ string, info *provider.ResourceInfo, err error) error {
		if err != nil {
			return err
		}
		visited = append(visited, info.Id.OpaqueId)
		if info.Id.OpaqueId == "root" {
			currentTime = currentTime.Add(2 * time.Hour)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
	if got, want := fmt.Sprint(visited), "[root child]"; got != want {
		t.Fatalf("visited resources: got %s, want %s", got, want)
	}
	if got, want := fmt.Sprint(observedTokens), "[token-1 token-2]"; got != want {
		t.Fatalf("gateway tokens: got %s, want %s", got, want)
	}
	if authCalls != 2 {
		t.Fatalf("authentication calls: got %d, want 2", authCalls)
	}
	client.AssertExpectations(t)
}
