//go:build !windows

package magick

import (
	"fmt"
	"os"

	"github.com/kovidgoyal/go-shm"
)

var _ = fmt.Print

func get_temp_dir() string {
	if shm.SHM_DIR != "" {
		tempFile, err := os.CreateTemp(shm.SHM_DIR, "write_check_*")
		if err != nil {
			return os.TempDir()
		}
		tempFile.Close()
		os.Remove(tempFile.Name())
		return shm.SHM_DIR
	}
	return os.TempDir()

}
