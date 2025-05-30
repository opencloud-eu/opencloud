package metadata

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/metadata"
)

// Deep is a deep storage implementation that allows to create nested storage structures.
type Deep struct {
	storage metadata.Storage
}

func NewDeepStorage(s metadata.Storage) (*Deep, error) {
	return &Deep{storage: s}, nil
}

// Backend wraps the backend of the base storage
func (s *Deep) Backend() string {
	return s.storage.Backend()
}

// Init prepares the base storage
func (s *Deep) Init(ctx context.Context, name string) (err error) {
	return s.storage.Init(ctx, name)
}

// Upload wraps the upload method of the base storage;
// the difference is that it creates parent directories if they do not exist, which allows for deep storage structures.
func (s *Deep) Upload(ctx context.Context, req metadata.UploadRequest) (*metadata.UploadResponse, error) {
	if pwd := filepath.Join(strings.Split(path.Dir(req.Path), string(os.PathSeparator))...); pwd != "" {
		if err := s.MakeDirIfNotExist(ctx, pwd); err != nil {
			return nil, fmt.Errorf("failed to create parent directory for %s: %w", req.Path, err)
		}
	}

	return s.storage.Upload(ctx, req)
}

// Download wraps the download method of the base storage
func (s *Deep) Download(ctx context.Context, req metadata.DownloadRequest) (*metadata.DownloadResponse, error) {
	return s.storage.Download(ctx, req)
}

// SimpleUpload wraps the simple upload method of the base storage
func (s *Deep) SimpleUpload(ctx context.Context, uploadpath string, content []byte) error {
	_, err := s.Upload(ctx, metadata.UploadRequest{
		Path:    uploadpath,
		Content: content,
	})

	return err
}

// SimpleDownload wraps the simple download method of the base storage
func (s *Deep) SimpleDownload(ctx context.Context, path string) ([]byte, error) {
	return s.storage.SimpleDownload(ctx, path)
}

// Delete wraps the delete method of the base storage
func (s *Deep) Delete(ctx context.Context, path string) error {
	return s.storage.Delete(ctx, path)
}

// Stat wraps the stat method of the base storage
func (s *Deep) Stat(ctx context.Context, path string) (*provider.ResourceInfo, error) {
	return s.storage.Stat(ctx, path)
}

// ReadDir wraps the read directory method of the base storage
func (s *Deep) ReadDir(ctx context.Context, path string) ([]string, error) {
	return s.storage.ReadDir(ctx, path)
}

// ListDir wraps the list directory method of the base storage
func (s *Deep) ListDir(ctx context.Context, path string) ([]*provider.ResourceInfo, error) {
	return s.storage.ListDir(ctx, path)
}

// CreateSymlink wraps the create symlink method of the base storage
func (s *Deep) CreateSymlink(ctx context.Context, oldname, newname string) error {
	return s.storage.CreateSymlink(ctx, oldname, newname)
}

// ResolveSymlink wraps the resolve symlink method of the base storage
func (s *Deep) ResolveSymlink(ctx context.Context, name string) (string, error) {
	return s.storage.ResolveSymlink(ctx, name)
}

// MakeDirIfNotExist wraps the make directory if not exist method of the base storage
func (s *Deep) MakeDirIfNotExist(ctx context.Context, name string) error {
	paths := strings.Split(name, string(os.PathSeparator))
	// Create all parent directories if they do not exist
	for i := 0; i <= len(paths)-1; i++ {
		c := path.Join(paths[:i+1]...)
		if err := s.storage.MakeDirIfNotExist(ctx, c); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", c, err)
		}
	}

	return nil
}
