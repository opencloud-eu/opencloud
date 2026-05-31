package checks

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
)

// NewNatsCheck checks the reachability of a nats server.
func NewNatsCheck(natsCluster string, options ...nats.Option) func(context.Context) error {
	conn, err := nats.Connect(natsCluster, options...)
	if err != nil {
		return func(context.Context) error {
			return fmt.Errorf("could not connect to nats server: %v", err)
		}
	}

	return func(_ context.Context) error {
		if conn.Status() != nats.CONNECTED {
			return fmt.Errorf("nats server not connected: %v", conn.Status())
		}
		return nil
	}
}
