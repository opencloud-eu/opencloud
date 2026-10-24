package consumer

import (
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/eventhistory/pkg/config"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"go-micro.dev/v4/store"
)

// Option defines a single option function.
type Option func(o *Options)

// Options defines the available options for this package.
type Options struct {
	Logger      log.Logger
	Config      *config.Config
	Persistence store.Store
	Consumer    events.Consumer
}

// newOptions initializes the available default options.
func newOptions(opts ...Option) Options {
	opt := Options{}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

// Logger provides a function to set the logger option.
func Logger(val log.Logger) Option {
	return func(o *Options) {
		o.Logger = val
	}
}

// Config provides a function to set the config option.
func Config(val *config.Config) Option {
	return func(o *Options) {
		o.Config = val
	}
}

// Persistence provides a function to configure the store
func Persistence(store store.Store) Option {
	return func(o *Options) {
		o.Persistence = store
	}
}

// Stream provides a function to set the event source to consume from.
func Stream(consumer events.Consumer) Option {
	return func(o *Options) {
		o.Consumer = consumer
	}
}
