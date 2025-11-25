//go:build windows

package magick

import (
	"fmt"
	"os"
)

var _ = fmt.Print

func get_temp_dir() string {
	return os.TempDir()
}
