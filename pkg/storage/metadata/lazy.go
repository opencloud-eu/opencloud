package metadata

import (
	"context"
	"errors"
	"sync"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-playground/validator/v10"
	"github.com/opencloud-eu/reva/v3/pkg/storage/utils/metadata"

	"github.com/opencloud-eu/opencloud/pkg/storage"
)

// Lazy is a lazy storage implementation that initializes the underlying storage only when needed.
type Lazy struct {
	next func() (metadata.Storage, error)

	initName string          `validate:"required"`
	initCTX  context.Context `validate:"required"`
}

func NewLazyStorage(next metadata.Storage) (*Lazy, error) {
	s := &Lazy{}
	s.next = sync.OnceValues[metadata.Storage, error](func() (metadata.Storage, error) {
		if err := validator.New(validator.WithPrivateFieldValidation()).Struct(s); err != nil {
			return nil, errors.Join(storage.ErrStorageInitialization, storage.ErrStorageValidation, err)
		}

		if err := next.Init(s.initCTX, s.initName); err != nil {
			return nil, errors.Join(storage.ErrStorageInitialization, err)
		}

		return next, nil
	})

	return s, nil
}

// Backend wraps the backend of the next storage
func (s *Lazy) Backend() string {
	next, err := s.next()
	if err != nil {
		return ""
	}

	return next.Backend()
}

// Init prepares the required data for the underlying lazy storage initialization
func (s *Lazy) Init(ctx context.Context, name string) (err error) {
	s.initCTX = ctx
	s.initName = name

	return nil
}

// Upload wraps the upload method of the next storage
func (s *Lazy) Upload(ctx context.Context, req metadata.UploadRequest) (*metadata.UploadResponse, error) {
	next, err := s.next()
	if err != nil {
		return nil, err
	}

	return next.Upload(ctx, req)
}

// Download wraps the download method of the next storage
func (s *Lazy) Download(ctx context.Context, req metadata.DownloadRequest) (*metadata.DownloadResponse, error) {
	next, err := s.next()
	if err != nil {
		return nil, err
	}

	return next.Download(ctx, req)
}

// SimpleUpload wraps the simple upload method of the next storage
func (s *Lazy) SimpleUpload(ctx context.Context, uploadpath string, content []byte) error {
	next, err := s.next()
	if err != nil {
		return err
	}

	return next.SimpleUpload(ctx, uploadpath, content)
}

// SimpleDownload wraps the simple download method of the next storage
func (s *Lazy) SimpleDownload(ctx context.Context, path string) ([]byte, error) {
	next, err := s.next()
	if err != nil {
		return nil, err
	}

	return next.SimpleDownload(ctx, path)
}

// Delete wraps the delete method of the next storage
func (s *Lazy) Delete(ctx context.Context, path string) error {
	next, err := s.next()
	if err != nil {
		return err
	}

	return next.Delete(ctx, path)
}

// Stat wraps the stat method of the next storage
func (s *Lazy) Stat(ctx context.Context, path string) (*provider.ResourceInfo, error) {
	next, err := s.next()
	if err != nil {
		return nil, err
	}

	return next.Stat(ctx, path)
}

// ReadDir wraps the read directory method of the next storage
func (s *Lazy) ReadDir(ctx context.Context, path string) ([]string, error) {
	next, err := s.next()
	if err != nil {
		return nil, err
	}

	return next.ReadDir(ctx, path)
}

// ListDir wraps the list directory method of the next storage
func (s *Lazy) ListDir(ctx context.Context, path string) ([]*provider.ResourceInfo, error) {
	next, err := s.next()
	if err != nil {
		return nil, err
	}

	return next.ListDir(ctx, path)
}

// CreateSymlink wraps the create symlink method of the next storage
func (s *Lazy) CreateSymlink(ctx context.Context, oldname, newname string) error {
	next, err := s.next()
	if err != nil {
		return err
	}

	return next.CreateSymlink(ctx, oldname, newname)
}

// ResolveSymlink wraps the resolve symlink method of the next storage
func (s *Lazy) ResolveSymlink(ctx context.Context, name string) (string, error) {
	next, err := s.next()
	if err != nil {
		return "", err
	}

	return next.ResolveSymlink(ctx, name)
}

// MakeDirIfNotExist wraps the make directory if not exist method of the next storage
func (s *Lazy) MakeDirIfNotExist(ctx context.Context, name string) error {
	next, err := s.next()
	if err != nil {
		return err
	}

	return next.MakeDirIfNotExist(ctx, name)
}
