package theme

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/opencloud-eu/opencloud/pkg/x/io/fsx"
	"github.com/opencloud-eu/opencloud/pkg/x/path/filepathx"
)

// ServiceOptions defines the options to configure the Service.
type ServiceOptions struct {
	themeFS fsx.FS
}

// WithThemeFS sets the theme filesystem.
func (o ServiceOptions) WithThemeFS(fSys fsx.FS) ServiceOptions {
	o.themeFS = fSys
	return o
}

// validate validates the input parameters.
func (o ServiceOptions) validate() error {
	if o.themeFS == nil {
		return errors.New("themeFS is required")
	}

	return nil
}

// Service defines the http service.
type Service struct {
	themeFS fsx.FS
}

// NewService initializes a new Service.
func NewService(options ServiceOptions) (*Service, error) {
	if err := options.validate(); err != nil {
		return nil, err
	}

	return &Service{
		themeFS: options.themeFS,
	}, nil
}

// Build builds the theme, the theme is a merge of the default theme, the base theme, and the branding theme.
func (s Service) Build(id string) (KV, error) {
	themeFS := s.themeFS.IOFS()

	// there is no guarantee that the theme exists, its optional; therefore, we ignore the error
	baseTheme, _ := LoadKV(themeFS, filepathx.JailJoin(id, _themeFileName))

	// there is no guarantee that the theme exists, its optional; therefore, we ignore the error here too
	brandingTheme, _ := LoadKV(themeFS, filepathx.JailJoin(_brandingRoot, _themeFileName))

	// merge the themes, the order is important, the last one wins and overrides the previous ones
	// themeDefaults: contains all the default values, this is guaranteed to exist
	// baseTheme: contains the base theme from the theme fs, there is no guarantee that it exists
	// brandingTheme: contains the branding theme from the theme fs, there is no guarantee that it exists
	// mergedTheme = themeDefaults < baseTheme < brandingTheme
	mergedTheme, err := MergeKV(themeDefaults, baseTheme, brandingTheme)
	if err != nil {
		return nil, errors.Wrap(err, "failed to merge themes")
	}

	return mergedTheme, nil
}

func (s Service) Exists(id string) bool {
	info, err := s.themeFS.Stat(filepathx.JailJoin(id, _themeFileName))
	return err == nil && !info.IsDir() && info.Size() > 0
}

func (s Service) Remove(id string) error {
	if !s.Exists(id) {
		return errors.Errorf("theme %s does not exist", id)
	}

	// remove the theme directory
	return s.themeFS.RemoveAll(id)
}

func (s Service) Add(id string, r *zip.Reader) error {
	if s.Exists(id) {
		return errors.Errorf("theme %s already exists", id)
	}

	for _, f := range r.File {
		filePath := filepath.Join(id, f.Name)
		if f.FileInfo().IsDir() {
			err := s.themeFS.MkdirAll(filePath, os.ModePerm)
			if err != nil {
				return err
			}

			continue
		}

		if err := s.themeFS.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return err
		}

		source, err := f.Open()
		if err != nil {
			return err
		}

		dest, err := s.themeFS.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		if _, err := io.Copy(dest, source); err != nil {
			return err
		}

		_ = dest.Close()
		_ = source.Close()
	}

	return nil
}
