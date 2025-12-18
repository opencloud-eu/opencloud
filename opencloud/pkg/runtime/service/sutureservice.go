package service

import (
	"context"

	occfg "github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/thejerf/suture/v4"
)

// SutureService allows for the settings command to be embedded and supervised by a suture supervisor tree.
type SutureService struct {
	exec func(ctx context.Context) error
	name string
}

// NewSutureServiceBuilder creates a new suture service
func NewSutureServiceBuilder(name string, f func(context.Context, *occfg.Config) error) func(*occfg.Config) suture.Service {
	return func(cfg *occfg.Config) suture.Service {
		return SutureService{
			exec: func(ctx context.Context) error {
				return f(ctx, cfg)
			},
			name: name,
		}
	}
}

// Serve to fullfil Server interface
func (s SutureService) Serve(ctx context.Context) error {
	return s.exec(ctx)
}

// String to fullfil fmt.Stringer interface, used to log the service name
func (s SutureService) String() string {
	return s.name
}
