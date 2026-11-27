package cache

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail"
)

// An entry is a header line naming the type of the image, followed by the image.
const entryMagic = "oc-thumbnail/1 "

// FileCache stores thumbnails on the local filesystem using hierarchical directories.
type FileCache struct {
	dir string
}

func NewFileCache(dir string) *FileCache {
	return &FileCache{dir: dir}
}

// The cache key is already a hierarchical relative path, so the on-disk
// location is the cache dir joined with it.
func (c *FileCache) Get(key string) (*thumbnail.Image, error) {
	raw, err := os.ReadFile(filepath.Join(c.dir, key))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrCacheMiss
		}
		return nil, err
	}

	if len(raw) == 0 {
		return nil, nil
	}

	header, data, found := bytes.Cut(raw, []byte("\n"))
	if !found || !bytes.HasPrefix(header, []byte(entryMagic)) {
		return &thumbnail.Image{Data: raw}, nil
	}

	return &thumbnail.Image{ContentType: string(header[len(entryMagic):]), Data: data}, nil
}

func (c *FileCache) Put(key string, img *thumbnail.Image) error {
	imgPath := filepath.Join(c.dir, key)
	dir := filepath.Dir(imgPath)

	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	f, err := os.CreateTemp(dir, "tmpthumb")
	if err != nil {
		return err
	}
	tempPath := f.Name()

	if img != nil {
		_, err := f.WriteString(entryMagic + img.ContentType + "\n")
		if err == nil {
			_, err = f.Write(img.Data)
		}
		if err != nil {
			f.Close()
			os.Remove(tempPath)
			return err
		}
	}

	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tempPath)
		return err
	}

	if err := f.Close(); err != nil {
		os.Remove(tempPath)
		return err
	}

	if err := os.Rename(tempPath, imgPath); err != nil {
		os.Remove(tempPath)
		return err
	}

	return nil
}
