package checks

import (
	"context"
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/handlers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

// NewGRPCCheck checks the reachability of a grpc server.
func NewGRPCCheck(address string) func(context.Context) error {
	address, err := handlers.FailSaveAddress(address)
	if err != nil {
		return func(context.Context) error {
			return fmt.Errorf("invalid address: %v", err)
		}
	}

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return func(context.Context) error {
			return fmt.Errorf("could not connect to grpc server: %v", err)
		}
	}

	return func(ctx context.Context) error {
		s := conn.GetState()
		if s == connectivity.TransientFailure || s == connectivity.Shutdown {
			return fmt.Errorf("grpc connection in bad state: %v", s)
		}
		return nil
	}
}
