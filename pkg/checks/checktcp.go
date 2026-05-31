package checks

import (
	"context"
	"net"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/handlers"
)

// NewTCPCheck returns a check that connects to a given tcp endpoint.
func NewTCPCheck(address string) func(context.Context) error {
	address, err := handlers.FailSaveAddress(address)
	if err != nil {
		return func(context.Context) error {
			return err
		}
	}

	return func(ctx context.Context) error {
		d := net.Dialer{Timeout: 3 * time.Second}
		conn, err := d.DialContext(ctx, "tcp", address)
		if err != nil {
			return err
		}
		return conn.Close()
	}
}
