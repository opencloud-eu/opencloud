package fs

import (
	"github.com/opencloud-eu/opencloud/pkg/x/io/fsx"
	"github.com/opencloud-eu/opencloud/services/web"
	"github.com/opencloud-eu/opencloud/services/web/pkg/config"
)

// NewThemeFS
// fixMe: since the metadataFS (cs3) is not part of the current version anymore,
// the fs nesting can (could) be simplified, I also consider to:
//   - use a dedicated apace (path) for console-related things, e.g. OC_HOME/console/... instead of OC_HOME/console/...
//   - maybe a memFS is good enough? Maybe not the best idea, app (web-apps) size is unknown
func NewThemeFS(c *config.Config) (*fsx.FallbackFS, error) {
	writableFF := fsx.NewBasePathFs(fsx.NewOsFs(), c.Asset.ThemesPath)

	return fsx.NewFallbackFS(
		writableFF,
		fsx.NewFallbackFS(
			fsx.NewReadOnlyFs(fsx.NewBasePathFs(fsx.FromIOFS(web.Assets), "assets/themes")),
			fsx.NewReadOnlyFs(fsx.NewBasePathFs(fsx.NewOsFs(), c.Asset.ThemesPath)),
		),
	), nil
}
