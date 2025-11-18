//go:build !linux

package magick

import (
	"fmt"
	"os"
)

var _ = fmt.Print

func memfd(data []byte) (ans *os.File, err error) {
	return nil, fmt.Errorf("ENOSYS")
}
