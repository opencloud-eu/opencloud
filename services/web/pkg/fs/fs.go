package fs

import (
	"context"
	"fmt"
	"time"

	revaMetadata "github.com/opencloud-eu/reva/v2/pkg/storage/utils/metadata"

	"github.com/opencloud-eu/opencloud/pkg/storage/metadata"
	"github.com/opencloud-eu/opencloud/pkg/x/io/fsx"
	metadataFs "github.com/opencloud-eu/opencloud/pkg/x/io/fsx/cs3/metadata"
	"github.com/opencloud-eu/opencloud/services/web"
	"github.com/opencloud-eu/opencloud/services/web/pkg/config"
)

func NewThemeFS(c *config.Config) (*fsx.FallbackFS, error) {
	storage, err := revaMetadata.NewCS3Storage(
		c.Metadata.GatewayAddress,
		c.Metadata.StorageAddress,
		c.Metadata.SystemUserID,
		c.Metadata.SystemUserIDP,
		c.Metadata.SystemUserAPIKey,
	)
	if err != nil {
		return nil, err
	}

	storage, err = metadata.NewLazyStorage(storage)
	if err != nil {
		return nil, err
	}

	time.Sleep(3 * time.Second) // fixme: wait for the storage to be initialized

	if err := storage.Init(context.Background(), "web-storage"); err != nil {
		return nil, err
	}

	storageFS := metadataFs.NewMetadataFs(storage)
	if err := storageFS.MkdirAll("assets/themes", 0755); err != nil {
		return nil, fmt.Errorf("failed to create themes directory: %w", err)
	}

	return fsx.NewFallbackFS(
		fsx.NewBasePathFs(fsx.FromAfero(storageFS), "assets/themes"),
		fsx.NewFallbackFS(
			fsx.NewReadOnlyFs(fsx.NewBasePathFs(fsx.FromIOFS(web.Assets), "assets/themes")),
			fsx.NewReadOnlyFs(fsx.NewBasePathFs(fsx.NewOsFs(), c.Asset.ThemesPath)),
		),
	), nil
}
