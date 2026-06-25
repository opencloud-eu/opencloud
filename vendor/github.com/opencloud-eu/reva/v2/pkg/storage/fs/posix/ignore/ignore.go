package ignore

import (
	"path/filepath"
	"strings"

	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/blobstore"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/lookup"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/options"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/tree/propagator"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/templates"
	"github.com/rs/zerolog"
)

const (
	// The needles include the separator at the beginning to avoid matching internal dir names in substrings
	// The end segment boundary is checked in isInternalDir
	trashNeedle    = needle(string(filepath.Separator) + lookup.TrashDir)
	metadataNeedle = needle(string(filepath.Separator) + lookup.MetadataDir)
	tmpNeedle      = needle(string(filepath.Separator) + blobstore.TMPDir)
)

type needle string

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
	return i.IsChanges(path) ||
		i.IsIndex(path) ||
		IsLockFile(path) ||
		i.IsTrash(path) ||
		i.IsMetadata(path) ||
		i.IsTemporary(path) ||
		i.IsUpload(path) ||
		i.IsRootPath(path) ||
		i.IsSpaceRoot(path)
}

func (i *Ignorer) IsChanges(path string) bool {
	return strings.HasPrefix(path, filepath.Join(i.options.Root, propagator.ChangesDir))
}

func (i *Ignorer) IsIndex(path string) bool {
	return strings.HasPrefix(path, filepath.Join(i.options.Root, lookup.IndexesDir))
}

func (i *Ignorer) IsUpload(path string) bool {
	return strings.HasPrefix(path, i.options.UploadDirectory)
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

func IsLockFile(path string) bool {
	return strings.HasSuffix(path, ".flock") || strings.HasSuffix(path, ".mlock")
}

func (i *Ignorer) IsMetadata(path string) bool {
	return i.isInternalDir(path, metadataNeedle)
}

func (i *Ignorer) IsTemporary(path string) bool {
	return i.isInternalDir(path, tmpNeedle)
}

func (i *Ignorer) IsTrash(path string) bool {
	return i.isInternalDir(path, trashNeedle)
}

// isInternalDir checks if the path contains the match dir and that the match lives
// in the space root, e.g. "/storage/users/user1/.metadata/file" -> match is ".metadata",
// parent dir is "/storage/users/user1" which is a space root, so this would return true
func (i *Ignorer) isInternalDir(path string, match needle) bool {
	idx := strings.Index(path, string(match))
	if idx <= 0 {
		return false
	}

	// must end at a segment boundary (end of path or separator)
	if length := idx + len(match); length != len(path) && path[length] != filepath.Separator {
		return false
	}

	// get the path of the parent dir, e.g. "/a/match" ->  index of "match" is 3
	// so parentPath is path[:2] -> "/a"
	parentPath := path[:idx-1]

	return len(parentPath) > 0 && i.IsSpaceRoot(parentPath)
}
