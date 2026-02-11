package ocservice

import (
	"context"

	occfg "github.com/opencloud-eu/opencloud/pkg/config"
)

type OCService struct {
	exec func(ctx context.Context) error
	name string
}

// NewSutureServiceBuilder creates a new suture service
func NewOCServiceBuilder(name string, f func(context.Context, *occfg.Config) error) func(*occfg.Config) OCService {
	return func(cfg *occfg.Config) OCService {
		return OCService{
			exec: func(ctx context.Context) error {
				return f(ctx, cfg)
			},
			name: name,
		}
	}
}

// Serve to fullfil Server interface
func (s OCService) Serve(ctx context.Context) error {
	return s.exec(ctx)
}

// String to fullfil fmt.Stringer interface, used to log the service name
func (s OCService) String() string {
	return s.name
}
