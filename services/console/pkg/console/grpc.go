package console

import (
	"go-micro.dev/v4/client"

	"github.com/opencloud-eu/opencloud/pkg/service/grpc"

	"github.com/opencloud-eu/opencloud/pkg/tracing"
	"github.com/opencloud-eu/opencloud/services/console/pkg/config"
)

func NewGRPCClient(cfg *config.Config) (client.Client, error) {
	traceProvider, err := tracing.GetServiceTraceProvider(cfg.Tracing, cfg.Service.Name)
	if err != nil {
		return nil, err
	}

	grpcClient, err := grpc.NewClient(
		append(grpc.GetClientOptions(cfg.GRPCClientTLS),
			grpc.WithTraceProvider(traceProvider),
		)...,
	)
	if err != nil {
		return nil, err
	}

	return grpcClient, nil
}
