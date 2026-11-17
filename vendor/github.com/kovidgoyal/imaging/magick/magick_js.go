//go:build js

package magick

import "os"

func get_temp_dir() string {
	return os.TempDir()
}