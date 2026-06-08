package ignore

import (
	"path/filepath"
	"strings"

	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/blobstore"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/lookup"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/options"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/templates"
	"github.com/rs/zerolog"
)

// Ignorer handles checking if paths should be ignored in posix operations
type Ignorer struct {
	options            *options.Options
	log                *zerolog.Logger
	personalSpacesRoot string
	projectSpacesRoot  string
}

// NewIgnorer creates a new Ignorer given the posix options and logger
func NewIgnorer(options *options.Options, log *zerolog.Logger) *Ignorer {
	return &Ignorer{
		options:            options,
		log:                log,
		personalSpacesRoot: filepath.Clean(filepath.Join(options.Root, templates.Base(options.PersonalSpacePathTemplate))),
		projectSpacesRoot:  filepath.Clean(filepath.Join(options.Root, templates.Base(options.GeneralSpacePathTemplate))),
	}
}

// IsIgnored checking if paths should be ignored in posix operations
func (i *Ignorer) IsIgnored(path string) bool {
	return IsLockFile(path) || IsTrash(path) || i.IsUpload(path) || i.IsInternal(path) || i.IsRootPath(path) || i.IsSpaceRoot(path)
}

func (i *Ignorer) IsUpload(path string) bool {
	return strings.HasPrefix(path, i.options.UploadDirectory)
}

func (i *Ignorer) IsIndex(path string) bool {
	return strings.HasPrefix(path, filepath.Join(i.options.Root, "indexes"))
}

func (i *Ignorer) IsChanges(path string) bool {
	return strings.HasPrefix(path, filepath.Join(i.options.Root, "changes"))
}

func (i *Ignorer) IsTemporary(path string) bool {
	if filepath.IsAbs(path) {
		tmpDirPattern := filepath.Join(i.options.Root, "*", "*", blobstore.TMPDir)
		isTempDir, err := filepath.Match(tmpDirPattern, path)
		if err != nil {
			i.log.Error().Err(err).Str("pattern", tmpDirPattern).Str("path", path).Msg("error matching temporary path")
			return false
		}
		isTempParentDir, err := filepath.Match(tmpDirPattern, filepath.Dir(path))
		if err != nil {
			i.log.Error().Err(err).Str("pattern", tmpDirPattern).Str("path", filepath.Dir(path)).Msg("error matching temporary path")
			return false
		}
		return isTempDir || isTempParentDir
	}
	return path == blobstore.TMPDir || filepath.Dir(path) == blobstore.TMPDir
}

func (i *Ignorer) IsRootPath(path string) bool {
	return path == i.options.Root ||
		path == i.personalSpacesRoot ||
		path == i.projectSpacesRoot
}

func (i *Ignorer) IsSpaceRoot(path string) bool {
	parent := filepath.Dir(path)
	return parent == i.personalSpacesRoot || parent == i.projectSpacesRoot
}

func (i *Ignorer) IsInternal(path string) bool {
	return i.IsIndex(path) || strings.Contains(path, lookup.MetadataDir) || i.IsTemporary(path) || i.IsChanges(path)
}

func IsLockFile(path string) bool {
	return strings.HasSuffix(path, ".flock") || strings.HasSuffix(path, ".mlock")
}

func IsTrash(path string) bool {
	return strings.HasSuffix(path, ".trashinfo") || strings.HasSuffix(path, ".trashitem") || strings.Contains(path, ".Trash")
}
