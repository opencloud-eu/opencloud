//go:build linux

package magick

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

var _ = fmt.Print

func memfd(data []byte) (ans *os.File, err error) {
	fd, err := unix.MemfdCreate("memfile", unix.O_CLOEXEC)
	if err != nil {
		return nil, err
	}
	_, err = unix.Write(fd, data)
	if err != nil {
		return nil, err
	}
	_, err = unix.Seek(fd, 0, unix.SEEK_SET)
	if err != nil {
		return nil, err
	}
	ans = os.NewFile(uintptr(fd), "memfile")
	return
}
