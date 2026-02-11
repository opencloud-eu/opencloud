package runtime

import (
	"context"

	"github.com/opencloud-eu/opencloud/opencloud/pkg/runtime/service"
	"github.com/opencloud-eu/opencloud/pkg/config"
)

// Runtime represents an OpenCloud runtime environment.
type Runtime struct {
	c *config.Config
}

// New creates a new OpenCloud + micro runtime
func New(cfg *config.Config) Runtime {
	return Runtime{
		c: cfg,
	}
}

// Start rpc runtime
func (r *Runtime) Start(ctx context.Context) error {
	return service.StartV2(ctx, service.WithConfig(r.c))
	//return service.Start(ctx, service.WithConfig(r.c))
}
