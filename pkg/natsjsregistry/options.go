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
type authKey struct{}

type auth struct {
	username string
	password string
}

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

// Authenticate sets the username and password for the connection to NATS.
// Without it, they are read from the MICRO_REGISTRY_AUTH_* env vars.
func Authenticate(username, password string) registry.Option {
	return func(o *registry.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, authKey{}, auth{username: username, password: password})
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
