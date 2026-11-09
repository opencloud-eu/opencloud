package cache

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/xattr"

	"github.com/opencloud-eu/opencloud/services/webdav/pkg/thumbnail"
)

const contentTypeAttr = "user.oc.content_type"

// FileCache stores thumbnails on the local filesystem using hierarchical directories.
type FileCache struct {
	dir string
}

func NewFileCache(dir string) *FileCache {
	return &FileCache{dir: dir}
}

func (c *FileCache) probe() error {
	if err := os.MkdirAll(c.dir, 0700); err != nil {
		return err
	}

	f, err := os.CreateTemp(c.dir, "tmpprobe")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()

	if err := xattr.FSet(f, contentTypeAttr, []byte("image/png")); err != nil {
		return fmt.Errorf("the thumbnail cache in %s needs a filesystem with extended attributes, use another directory or the s3 or memory backend: %w", c.dir, err)
	}

	return nil
}

// The cache key is already a hierarchical relative path, so the on-disk
// location is the cache dir joined with it.
func (c *FileCache) Get(key string) (*thumbnail.Image, error) {
	f, err := os.Open(filepath.Join(c.dir, key))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrCacheMiss
		}
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	contentType, err := xattr.FGet(f, contentTypeAttr)
	if err != nil && !errors.Is(err, xattr.ENOATTR) {
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil
	}

	return &thumbnail.Image{ContentType: string(contentType), Data: data}, nil
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
		if _, err := f.Write(img.Data); err != nil {
			f.Close()
			os.Remove(tempPath)
			return err
		}
	}

	if img != nil && img.ContentType != "" {
		if err := xattr.FSet(f, contentTypeAttr, []byte(img.ContentType)); err != nil {
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
