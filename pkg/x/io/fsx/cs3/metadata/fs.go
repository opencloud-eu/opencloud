package metadata

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/afero"

	revaMetadata "github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/metadata"
)

func NewMetadataFs(storage metadata.Storage) *Fs {
	return &Fs{storage: storage}
}

type Fs struct {
	storage metadata.Storage
}

func (fs *Fs) Create(_ string) (afero.File, error) {
	return nil, syscall.EPERM
}

func (fs *Fs) Mkdir(name string, _ os.FileMode) error {
	return fs.storage.MakeDirIfNotExist(context.Background(), name)
}

func (fs *Fs) MkdirAll(name string, _ os.FileMode) error {
	paths := strings.Split(name, string(os.PathSeparator))
	// Create all parent directories if they do not exist
	for i := 0; i <= len(paths)-1; i++ {
		c := path.Join(paths[:i+1]...)
		if err := fs.storage.MakeDirIfNotExist(context.Background(), c); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", c, err)
		}
	}

	return nil
}

func (fs *Fs) Open(name string) (afero.File, error) {
	return fs.OpenFile(name, os.O_RDONLY, 0)
}

func (fs *Fs) OpenFile(name string, _ int, _ os.FileMode) (afero.File, error) {
	res, err := fs.storage.Download(context.Background(), metadata.DownloadRequest{Path: name})
	if err != nil && !revaMetadata.IsNotExist(err) {
		return nil, err
	}

	var contend []byte
	if res != nil {
		contend = res.Content
	}

	return newFile(name, fs, 0, contend)
}

func (fs *Fs) Remove(name string) error {
	return fs.RemoveAll(name)
}

func (fs *Fs) RemoveAll(path string) error {
	return fs.storage.Delete(context.Background(), path)
}

func (fs *Fs) Rename(_, _ string) error {
	return syscall.EPERM
}

func (fs *Fs) Stat(name string) (fs.FileInfo, error) {
	return newFileInfo(name, fs, 0)
}

func (fs *Fs) Name() string {
	return "MetadataFS"
}

func (fs *Fs) Chmod(_ string, _ os.FileMode) error {
	return syscall.EPERM
}

func (fs *Fs) Chown(_ string, _, _ int) error {
	return syscall.EPERM
}

func (fs *Fs) Chtimes(_ string, _ time.Time, _ time.Time) error {
	return syscall.EPERM
}
