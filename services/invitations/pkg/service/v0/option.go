package service

import (
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/invitations/pkg/config"
	"go-micro.dev/v4/store"
)

// Option defines a single option function.
type Option func(o *Options)

// UUIDGenerator is a function that returns a UUID string.
type UUIDGenerator func() string

// Options defines the available options for this package.
type Options struct {
	Logger log.Logger
	Config *config.Config

	Persistance *store.Store

	UUIDGenerator UUIDGenerator
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

func WithPersistance(val *store.Store) Option {
	return func(o *Options) {
		o.Persistance = val
	}
}

// WithUUIDGenerator provides a function to set the UUIDGenerator option.
func WithUUIDGenerator(val UUIDGenerator) Option {
	return func(o *Options) {
		o.UUIDGenerator = val
	}
}
