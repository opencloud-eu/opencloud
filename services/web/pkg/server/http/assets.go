// Package http provides the HTTP server of the web service.
package http

import (
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/x/io/fsx"
	"github.com/opencloud-eu/opencloud/services/web"
	"github.com/opencloud-eu/opencloud/services/web/pkg/config"
)

// assetFSs are the filesystems of the web assets: each one serves the files of
// its directory on disk, and the embedded files for the ones missing there.
type assetFSs struct {
	apps   *fsx.FallbackFS
	core   *fsx.FallbackFS
	themes *fsx.FallbackFS
}

// newAssetFSs returns the filesystems of the asset directories in cfg. The
// directories on disk are confined with os.Root: no symlink leads out of them.
// The apps directory is read-only, the themes directory is writable for the
// logo upload.
func newAssetFSs(cfg config.Asset, logger log.Logger) (assetFSs, error) {
	appsDirFS, err := fsx.NewRootFs(cfg.AppsPath, logger)
	if err != nil {
		return assetFSs{}, fmt.Errorf("could not open the apps asset directory: %w", err)
	}
	coreDirFS, err := fsx.NewRootFs(cfg.CorePath, logger)
	if err != nil {
		return assetFSs{}, fmt.Errorf("could not open the core asset directory: %w", err)
	}
	themesDirFS, err := fsx.NewRootFs(cfg.ThemesPath, logger)
	if err != nil {
		return assetFSs{}, fmt.Errorf("could not open the themes asset directory: %w", err)
	}

	return assetFSs{
		apps: fsx.NewFallbackFS(
			fsx.NewReadOnlyFs(appsDirFS),
			fsx.NewBasePathFs(fsx.FromIOFS(web.Assets), "assets/apps"),
		),
		core: fsx.NewFallbackFS(
			coreDirFS,
			fsx.NewBasePathFs(fsx.FromIOFS(web.Assets), "assets/core"),
		),
		themes: fsx.NewFallbackFS(
			themesDirFS,
			fsx.NewBasePathFs(fsx.FromIOFS(web.Assets), "assets/themes"),
		),
	}, nil
}
