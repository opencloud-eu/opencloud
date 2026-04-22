package console

import (
	"context"

	"go-micro.dev/v4/client"

	"github.com/opencloud-eu/opencloud/pkg/service/grpc"
	"github.com/opencloud-eu/opencloud/pkg/shared"

	"github.com/opencloud-eu/opencloud/pkg/tracing"
)

func NewGRPCClient(ctx context.Context, exporter, name string, tlsConfig *shared.GRPCClientTLS) (client.Client, error) {
	traceProvider, err := tracing.GetTraceProvider(ctx, exporter, name)
	if err != nil {
		return nil, err
	}

	grpcClient, err := grpc.NewClient(
		append(grpc.GetClientOptions(tlsConfig),
			grpc.WithTraceProvider(traceProvider),
		)...,
	)
	if err != nil {
		return nil, err
	}

	return grpcClient, nil
}
