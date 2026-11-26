package command_test

import (
	"net"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	merrors "go-micro.dev/v4/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/command"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
)

var _ = Describe("Optimize", func() {
	// runOptimize runs the optimize command against a search service without
	// TLS that answers with provider
	runOptimize := func(provider fakeSearchProvider) error {
		lis, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).ToNot(HaveOccurred())
		srv := grpc.NewServer(grpc.Creds(insecure.NewCredentials()))
		searchsvc.RegisterSearchProviderServer(srv, provider)
		go func() { _ = srv.Serve(lis) }()
		DeferCleanup(srv.Stop)

		cfg := &config.Config{
			GRPC:          config.GRPCConfig{Addr: lis.Addr().String()},
			GRPCClientTLS: &shared.GRPCClientTLS{},
		}
		cmd := command.Optimize(cfg)
		Expect(cmd.ParseFlags(nil)).To(Succeed())
		return cmd.RunE(cmd, nil)
	}

	It("asks the service to optimize its index", func() {
		Expect(runOptimize(fakeSearchProvider{})).To(Succeed())
	})

	It("returns the detail of the go-micro error the service answers with", func() {
		merr := merrors.New("eu.opencloud.api.search", "the search engine merges its index on its own", http.StatusNotImplemented)
		err := runOptimize(fakeSearchProvider{optimizeErr: status.Error(codes.Unimplemented, merr.Error())})
		Expect(err).To(MatchError("the search engine merges its index on its own"))
	})
})
