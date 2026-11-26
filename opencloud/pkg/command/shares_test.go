package command

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	grouppb "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	storageregistry "github.com/cs3org/go-cs3apis/cs3/storage/registry/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v3/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type staticSelector[T any] struct{ client T }

func (s staticSelector[T]) Next(...pool.Option) (T, error) { return s.client, nil }

var statusOK = &rpc.Status{Code: rpc.Code_CODE_OK}

type fakeStorageProvider struct {
	provider.UnimplementedProviderAPIServer
	mu      sync.Mutex
	grants  map[string][]*provider.Grant
	listErr bool
	removed []*provider.Grant
}

func (f *fakeStorageProvider) ListGrants(_ context.Context, req *provider.ListGrantsRequest) (*provider.ListGrantsResponse, error) {
	if f.listErr {
		return &provider.ListGrantsResponse{Status: &rpc.Status{Code: rpc.Code_CODE_INTERNAL, Message: "boom"}}, nil
	}
	return &provider.ListGrantsResponse{Status: statusOK, Grants: f.grants[req.GetRef().GetResourceId().GetOpaqueId()]}, nil
}

func (f *fakeStorageProvider) RemoveGrant(_ context.Context, req *provider.RemoveGrantRequest) (*provider.RemoveGrantResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removed = append(f.removed, req.GetGrant())
	return &provider.RemoveGrantResponse{Status: statusOK}, nil
}

func startStorageProvider(t *testing.T, srv *fakeStorageProvider) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	s := grpc.NewServer()
	provider.RegisterProviderAPIServer(s, srv)
	go func() { _ = s.Serve(lis) }()
	t.Cleanup(s.Stop)
	return lis.Addr().String()
}

func rid(opaqueID string) *provider.ResourceId {
	return &provider.ResourceId{StorageId: "storage", SpaceId: "space", OpaqueId: opaqueID}
}

func userGrant(id string) *provider.Grant {
	return &provider.Grant{Grantee: &provider.Grantee{
		Type: provider.GranteeType_GRANTEE_TYPE_USER,
		Id:   &provider.Grantee_UserId{UserId: &userpb.UserId{OpaqueId: id}},
	}}
}

func groupGrant(id string) *provider.Grant {
	return &provider.Grant{Grantee: &provider.Grantee{
		Type: provider.GranteeType_GRANTEE_TYPE_GROUP,
		Id:   &provider.Grantee_GroupId{GroupId: &grouppb.GroupId{OpaqueId: id}},
	}}
}

func shareFor(id string, g *provider.Grant) *collaboration.Share {
	return &collaboration.Share{Id: &collaboration.ShareId{OpaqueId: id}, Grantee: g.GetGrantee()}
}

type fixture struct {
	storage       *fakeStorageProvider
	shares        map[string][]*collaboration.Share
	shareErr      map[string]bool
	spaceListErr  bool
	noProviders   bool
	mu            sync.Mutex
	removedShares []string

	gatewaySelector  pool.Selectable[gateway.GatewayAPIClient]
	registrySelector pool.Selectable[storageregistry.RegistryAPIClient]
	revaOpts         []pool.Option
	spaces           []*provider.StorageSpace
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{
		storage: &fakeStorageProvider{grants: map[string][]*provider.Grant{
			"space": {userGrant("root-user")},
			"file":  {userGrant("alice"), userGrant("bob")},
		}},
		shares: map[string][]*collaboration.Share{
			"file": {shareFor("s1", userGrant("alice"))},
			"dir":  {shareFor("s2", groupGrant("devs"))},
		},
		shareErr: map[string]bool{},
		revaOpts: []pool.Option{pool.WithRegistry(nil)},
		spaces: []*provider.StorageSpace{{
			Id:   &provider.StorageSpaceId{OpaqueId: "storage$space"},
			Name: "test",
			Root: rid("space"),
		}},
	}
	addr := startStorageProvider(t, f.storage)

	infos := map[string]*provider.ResourceInfo{
		"space": {Id: rid("space"), Type: provider.ResourceType_RESOURCE_TYPE_CONTAINER},
		"dir":   {Id: rid("dir"), Path: "dir", Type: provider.ResourceType_RESOURCE_TYPE_CONTAINER},
		"file":  {Id: rid("file"), Path: "file", Type: provider.ResourceType_RESOURCE_TYPE_FILE},
	}
	children := map[string][]*provider.ResourceInfo{
		"space": {infos["dir"], infos["file"]},
	}

	gw := cs3mocks.NewGatewayAPIClient(t)
	gw.EXPECT().Stat(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, req *provider.StatRequest, _ ...grpc.CallOption) (*provider.StatResponse, error) {
			return &provider.StatResponse{Status: statusOK, Info: infos[req.GetRef().GetResourceId().GetOpaqueId()]}, nil
		}).Maybe()
	gw.EXPECT().ListContainer(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, req *provider.ListContainerRequest, _ ...grpc.CallOption) (*provider.ListContainerResponse, error) {
			return &provider.ListContainerResponse{Status: statusOK, Infos: children[req.GetRef().GetResourceId().GetOpaqueId()]}, nil
		}).Maybe()
	gw.EXPECT().ListShares(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, req *collaboration.ListSharesRequest, _ ...grpc.CallOption) (*collaboration.ListSharesResponse, error) {
			id := req.GetFilters()[0].GetResourceId().GetOpaqueId()
			if f.shareErr[id] {
				return &collaboration.ListSharesResponse{Status: &rpc.Status{Code: rpc.Code_CODE_INTERNAL, Message: "boom"}}, nil
			}
			return &collaboration.ListSharesResponse{Status: statusOK, Shares: f.shares[id]}, nil
		}).Maybe()
	gw.EXPECT().RemoveShare(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, req *collaboration.RemoveShareRequest, _ ...grpc.CallOption) (*collaboration.RemoveShareResponse, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.removedShares = append(f.removedShares, req.GetRef().GetId().GetOpaqueId())
			return &collaboration.RemoveShareResponse{Status: statusOK}, nil
		}).Maybe()

	reg := cs3mocks.NewStorageRegistryAPIClient(t)
	reg.EXPECT().ListStorageProviders(mock.Anything, mock.Anything).RunAndReturn(
		func(context.Context, *storageregistry.ListStorageProvidersRequest, ...grpc.CallOption) (*storageregistry.ListStorageProvidersResponse, error) {
			if f.noProviders {
				return &storageregistry.ListStorageProvidersResponse{Status: statusOK}, nil
			}
			return &storageregistry.ListStorageProvidersResponse{Status: statusOK, Providers: []*storageregistry.ProviderInfo{{Address: addr}}}, nil
		}).Maybe()

	f.gatewaySelector = staticSelector[gateway.GatewayAPIClient]{client: gw}
	f.registrySelector = staticSelector[storageregistry.RegistryAPIClient]{client: reg}
	return f
}

func (f *fixture) scan(dryRun bool) OrphanStats {
	return f.scanWith(orphanScanArgs{DryRun: dryRun})
}

func (f *fixture) scanWith(args orphanScanArgs) OrphanStats {
	args.GatewaySelector = f.gatewaySelector
	args.StorageRegistrySelector = f.registrySelector
	args.RevaOpts = f.revaOpts
	args.Spaces = f.spaces
	return scanAndCleanOrphansForSpaces(context.Background(), args)
}

func (f *fixture) reverse(dryRun bool) ReverseOrphanStats {
	return f.reverseWith(reverseOrphanArgs{DryRun: dryRun})
}

func (f *fixture) listSpaceShares(_ context.Context, storageID, spaceID string) ([]*collaboration.Share, error) {
	if f.spaceListErr {
		return nil, errors.New("boom")
	}
	var all []*collaboration.Share
	for opaqueID, shares := range f.shares {
		for _, s := range shares {
			c := proto.Clone(s).(*collaboration.Share)
			c.ResourceId = &provider.ResourceId{StorageId: storageID, SpaceId: spaceID, OpaqueId: opaqueID}
			all = append(all, c)
		}
	}
	return all, nil
}

func (f *fixture) reverseWith(args reverseOrphanArgs) ReverseOrphanStats {
	args.ListSpaceShares = f.listSpaceShares
	args.GatewaySelector = f.gatewaySelector
	args.StorageRegistrySelector = f.registrySelector
	args.RevaOpts = f.revaOpts
	args.Spaces = f.spaces
	return postprocessCleanReverseOrphans(context.Background(), args)
}

func removedUsers(grants []*provider.Grant) []string {
	ids := make([]string, 0, len(grants))
	for _, g := range grants {
		ids = append(ids, g.GetGrantee().GetUserId().GetOpaqueId())
	}
	return ids
}

func TestScanAndCleanOrphansForSpaces(t *testing.T) {
	t.Run("dry run reports the grant without share and removes nothing", func(t *testing.T) {
		f := newFixture(t)
		stats := f.scan(true)
		require.Equal(t, 3, stats.TotalNodes)
		require.Equal(t, 3, stats.TotalGrants)
		require.Equal(t, 1, stats.OrphanCandidates)
		require.Empty(t, f.storage.removed)
	})

	t.Run("removes only the grant without share and skips the space root", func(t *testing.T) {
		f := newFixture(t)
		stats := f.scan(false)
		require.Equal(t, 1, stats.OrphanDeleted)
		require.Equal(t, []string{"bob"}, removedUsers(f.storage.removed))
	})

	t.Run("a failing share list leaves the grants alone", func(t *testing.T) {
		for _, dryRun := range []bool{true, false} {
			f := newFixture(t)
			f.shareErr["file"] = true
			stats := f.scan(dryRun)
			require.Equal(t, 0, stats.OrphanCandidates)
			require.Equal(t, 2, stats.Unverified)
			require.Empty(t, f.storage.removed)
		}
	})

	t.Run("a resource id limits the scan to that resource", func(t *testing.T) {
		f := newFixture(t)
		stats := f.scanWith(orphanScanArgs{ResourceID: rid("file")})
		require.Equal(t, 1, stats.TotalNodes)
		require.Equal(t, []string{"bob"}, removedUsers(f.storage.removed))
	})

	t.Run("force removes the grants of the resource when its shares cannot be listed", func(t *testing.T) {
		f := newFixture(t)
		f.shareErr["file"] = true
		stats := f.scanWith(orphanScanArgs{ResourceID: rid("file"), Force: true})
		require.Equal(t, 2, stats.OrphanDeleted)
		require.ElementsMatch(t, []string{"alice", "bob"}, removedUsers(f.storage.removed))
	})

	t.Run("force without a resource id removes nothing", func(t *testing.T) {
		f := newFixture(t)
		f.shareErr["file"] = true
		stats := f.scanWith(orphanScanArgs{Force: true})
		require.Equal(t, 2, stats.Unverified)
		require.Empty(t, f.storage.removed)
	})
}

func TestPostprocessCleanReverseOrphans(t *testing.T) {
	t.Run("removes the share of a resource without grants", func(t *testing.T) {
		f := newFixture(t)
		stats := f.reverse(false)
		require.Equal(t, 1, stats.Deleted)
		require.Equal(t, []string{"s2"}, f.removedShares)
	})

	t.Run("a resource id limits the pass to that resource", func(t *testing.T) {
		f := newFixture(t)
		stats := f.reverseWith(reverseOrphanArgs{ResourceID: rid("file")})
		require.Equal(t, 0, stats.Candidates)
		require.Empty(t, f.removedShares)

		stats = f.reverseWith(reverseOrphanArgs{ResourceID: rid("dir")})
		require.Equal(t, 1, stats.Deleted)
		require.Equal(t, []string{"s2"}, f.removedShares)
	})

	t.Run("dry run removes nothing", func(t *testing.T) {
		f := newFixture(t)
		stats := f.reverse(true)
		require.Equal(t, 1, stats.Candidates)
		require.Empty(t, f.removedShares)
	})

	t.Run("no storage provider keeps all shares", func(t *testing.T) {
		f := newFixture(t)
		f.noProviders = true
		stats := f.reverse(false)
		require.Equal(t, 0, stats.Candidates)
		require.Empty(t, f.removedShares)
	})

	t.Run("removes the share whose grantee has no grant on a resource with other grants", func(t *testing.T) {
		f := newFixture(t)
		f.shares["file"] = append(f.shares["file"], shareFor("s3", userGrant("erin")), shareFor("s4", groupGrant("alice")))
		stats := f.reverse(false)
		require.Equal(t, 3, stats.Deleted)
		require.ElementsMatch(t, []string{"s2", "s3", "s4"}, f.removedShares)
	})

	t.Run("a failing share list of the space removes nothing", func(t *testing.T) {
		f := newFixture(t)
		f.spaceListErr = true
		stats := f.reverse(false)
		require.Equal(t, 0, stats.Candidates)
		require.Empty(t, f.removedShares)
	})

	t.Run("space root and expired shares are left alone", func(t *testing.T) {
		f := newFixture(t)
		f.storage.grants["space"] = nil
		f.shares["space"] = []*collaboration.Share{shareFor("member", userGrant("carol"))}
		expired := shareFor("expired", userGrant("dave"))
		expired.Expiration = &typespb.Timestamp{Seconds: 1}
		f.shares["dir"] = append(f.shares["dir"], expired)
		stats := f.reverse(false)
		require.Equal(t, 1, stats.Deleted)
		require.Equal(t, []string{"s2"}, f.removedShares)
	})

	t.Run("a failing grant list keeps all shares", func(t *testing.T) {
		f := newFixture(t)
		f.storage.listErr = true
		stats := f.reverse(false)
		require.Equal(t, 0, stats.Candidates)
		require.Empty(t, f.removedShares)
	})
}

func TestListResourceGrantsWithoutProvider(t *testing.T) {
	f := newFixture(t)
	f.noProviders = true
	cache := map[string][]*storageregistry.ProviderInfo{}

	grants, err := listResourceGrants(context.Background(), f.registrySelector, rid("file"), f.revaOpts, cache)
	require.Error(t, err)
	require.Nil(t, grants)
	require.NotContains(t, cache, "space")

	f.noProviders = false
	grants, err = listResourceGrants(context.Background(), f.registrySelector, rid("file"), f.revaOpts, cache)
	require.NoError(t, err)
	require.Len(t, grants, 2)
}

func TestHasMatchingShareForGrant(t *testing.T) {
	shares := []*collaboration.Share{shareFor("s1", userGrant("alice")), shareFor("s2", groupGrant("devs")), {}}
	tests := []struct {
		name   string
		shares []*collaboration.Share
		grant  *provider.Grant
		want   bool
	}{
		{"user with share", shares, userGrant("alice"), true},
		{"user without share", shares, userGrant("bob"), false},
		{"group with share", shares, groupGrant("devs"), true},
		{"group without share", shares, groupGrant("ops"), false},
		{"group id equal to a user id", shares, groupGrant("alice"), false},
		{"no shares", nil, userGrant("alice"), false},
		{"grant without grantee", shares, &provider.Grant{}, false},
		{"nil grant", shares, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, hasMatchingShareForGrant(tt.shares, tt.grant))
		})
	}
}

func TestHasMatchingGrantForShare(t *testing.T) {
	grants := []*provider.Grant{userGrant("alice"), groupGrant("devs")}
	require.True(t, hasMatchingGrantForShare(grants, shareFor("s1", userGrant("alice"))))
	require.True(t, hasMatchingGrantForShare(grants, shareFor("s2", groupGrant("devs"))))
	require.False(t, hasMatchingGrantForShare(grants, shareFor("s3", userGrant("bob"))))
	require.False(t, hasMatchingGrantForShare(grants, shareFor("s4", groupGrant("alice"))))
	require.False(t, hasMatchingGrantForShare(nil, shareFor("s5", userGrant("alice"))))
}

func TestRenderWalkerPath(t *testing.T) {
	tests := []struct {
		wd   string
		info *provider.ResourceInfo
		want string
	}{
		{"", nil, "/"},
		{"", &provider.ResourceInfo{Path: "."}, "/"},
		{"", &provider.ResourceInfo{Path: "dir"}, "/dir"},
		{"dir", &provider.ResourceInfo{Path: "file"}, "/dir/file"},
		{"/dir/", &provider.ResourceInfo{Path: "/sub/file"}, "/dir/sub/file"},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, renderWalkerPath(tt.wd, tt.info))
	}
}

func TestParseSpaceID(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"storage$space", "storage$space", false},
		{"storage$space!space", "storage$space", false},
		{"space", "", true},
		{"storage$space!node", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		got, err := parseSpaceID(tt.in)
		if tt.wantErr {
			require.Error(t, err, tt.in)
			continue
		}
		require.NoError(t, err, tt.in)
		require.Equal(t, tt.want, got)
	}
}

func TestParseResourceID(t *testing.T) {
	got, err := parseResourceID("storage$space!node")
	require.NoError(t, err)
	require.Equal(t, "storage", got.GetStorageId())
	require.Equal(t, "space", got.GetSpaceId())
	require.Equal(t, "node", got.GetOpaqueId())

	for _, in := range []string{"", "node", "storage$space", "space!node"} {
		_, err := parseResourceID(in)
		require.Error(t, err, in)
	}
}
