package cache

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// FileCache stores thumbnails on the local filesystem using hierarchical directories.
type FileCache struct {
	dir string
}

func NewFileCache(dir string) *FileCache {
	return &FileCache{dir: dir}
}

// The cache key is already a hierarchical relative path (built to match main's
// BuildKey layout), so the on-disk location is simply the cache dir joined with it.
func (c *FileCache) Get(key string) ([]byte, error) {
	path := filepath.Join(c.dir, key)

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrCacheMiss
		}
		return nil, err
	}

	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

func (c *FileCache) Put(key string, data []byte) error {
	imgPath := filepath.Join(c.dir, key)
	dir := filepath.Dir(imgPath)

	if _, err := os.Stat(imgPath); err == nil {
		return nil
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	f, err := os.CreateTemp(dir, "tmpthumb")
	if err != nil {
		return err
	}
	tempPath := f.Name()

	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tempPath)
		return err
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
