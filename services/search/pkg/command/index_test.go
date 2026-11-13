package command_test

import (
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/command"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
)

type fakeSearchProvider struct {
	searchsvc.UnimplementedSearchProviderServer
}

func (fakeSearchProvider) IndexSpace(_ *searchsvc.IndexSpaceRequest, stream grpc.ServerStreamingServer[searchsvc.IndexSpaceResponse]) error {
	return stream.Send(&searchsvc.IndexSpaceResponse{SpaceId: "space-1", IndexedSpaces: 1, TotalSpaces: 1})
}

var _ = Describe("Index", func() {
	var addr string

	// runIndex runs the index command against a search service without TLS
	// that listens on the configured gRPC address.
	runIndex := func(mode string, args ...string) error {
		cfg := &config.Config{
			GRPC:                  config.GRPCConfig{Addr: addr},
			GRPCClientTLS:         &shared.GRPCClientTLS{Mode: mode},
			ReindexMaxConcurrency: 3,
		}
		cmd := command.Index(cfg)
		Expect(cmd.ParseFlags(append([]string{"--all-spaces"}, args...))).To(Succeed())
		return cmd.RunE(cmd, nil)
	}

	BeforeEach(func() {
		lis, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).ToNot(HaveOccurred())
		srv := grpc.NewServer(grpc.Creds(insecure.NewCredentials()))
		searchsvc.RegisterSearchProviderServer(srv, fakeSearchProvider{})
		go func() { _ = srv.Serve(lis) }()
		DeferCleanup(srv.Stop)
		addr = lis.Addr().String()
	})

	DescribeTable("connects without TLS",
		func(mode string, args ...string) {
			Expect(runIndex(mode, args...)).To(Succeed())
		},
		Entry("when the mode is unset", ""),
		Entry("when the mode is off", "off"),
		Entry("when --insecure is passed", "", "--insecure"),
		Entry("when --insecure is passed with an unknown mode", "bogus", "--insecure"),
	)

	It("uses TLS when the mode is insecure", func() {
		Expect(runIndex("insecure")).To(MatchError(ContainSubstring("first record does not look like a TLS handshake")))
	})

	It("rejects an unknown mode", func() {
		Expect(runIndex("bogus")).To(MatchError(ContainSubstring("unknown TLS mode")))
	})

	It("prefers --endpoint over the configured gRPC address", func() {
		endpoint := addr
		addr = "127.0.0.1:1"
		Expect(runIndex("", "--endpoint", endpoint)).To(Succeed())
	})
})
