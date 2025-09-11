package natsjsregistry

import (
	"context"
	"time"

	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/store"
)

type storeOptionsKey struct{}
type defaultTTLKey struct{}
type serviceNameKey struct{}

// StoreOptions sets the options for the underlying store
func StoreOptions(opts []store.Option) registry.Option {
	return func(o *registry.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, storeOptionsKey{}, opts)
	}
}

// DefaultTTL allows setting a default register TTL for services
func DefaultTTL(t time.Duration) registry.Option {
	return func(o *registry.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, defaultTTLKey{}, t)
	}
}

// ServiceName links the service name to the registry if possible.
// The name will be part of the connection name to the Nats registry
func ServiceName(name string) registry.Option {
	return func(o *registry.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, serviceNameKey{}, name)
	}
}
