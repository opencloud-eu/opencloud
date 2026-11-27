// Package font serves the fonts of the collaboration service, e.g. for Collabora.
package font

import (
	"os"

	"github.com/spf13/afero"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/x/io/fsx"
)

// dirPerm is the mode of the fonts directory if it has to be created.
const dirPerm os.FileMode = 0o755

// NewFS returns the filesystem for the fonts in the directory dir, and creates
// the directory if it doesn't exist. The filesystem is confined to the
// directory: no symlink leads out of it.
func NewFS(dir string, logger log.Logger) (afero.Fs, error) {
	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return nil, err
	}
	return fsx.NewRootFs(dir, logger)
}
