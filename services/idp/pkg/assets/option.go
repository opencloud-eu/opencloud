package assets

import (
	"fmt"
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/x/io/fsx"
	"github.com/opencloud-eu/opencloud/services/idp"
	"github.com/opencloud-eu/opencloud/services/idp/pkg/config"
)

// New returns a new http filesystem to serve assets.
func New(opts ...Option) (http.FileSystem, error) {
	options := newOptions(opts...)

	var assetFS fsx.FS = fsx.NewBasePathFs(fsx.FromIOFS(idp.Assets), "assets")

	// only use a fsx.NewFallbackFS and a directory on disk if a path is set, use the embedded fs only otherwise
	if options.Config.Asset.Path != "" {
		dirFS, err := fsx.NewRootFs(options.Config.Asset.Path, options.Logger)
		if err != nil {
			return nil, fmt.Errorf("could not open the asset directory: %w", err)
		}
		assetFS = fsx.NewFallbackFS(dirFS, assetFS)
	}

	return http.FS(assetFS.IOFS()), nil
}

// Option defines a single option function.
type Option func(o *Options)

// Options define the available options for this package.
type Options struct {
	Logger log.Logger
	Config *config.Config
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
