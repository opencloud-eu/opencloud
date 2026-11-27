package fsx

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/afero"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

var (
	// assert interfaces implemented
	_ afero.Fs = (*RootFs)(nil)
)

// errPathEscapesText is the text of the unexported error of os.Root for a path
// that leads out of the root (os/file.go errPathEscapes).
const errPathEscapesText = "path escapes from parent"

// createPerm is the mode of files made by Create, as with os.Create (before the umask).
const createPerm os.FileMode = 0o666

// maxWarnedPaths is the number of paths that lead out of the directory that
// are logged at Warn level, each once. Further paths are logged at Debug level:
// below a symlinked directory that leads out, clients can make up any number
// of such paths, and the set of warned paths must not grow without limit.
const maxWarnedPaths = 100

// RootFs is a filesystem that is confined to a directory with os.Root:
// symlinks inside the directory are followed as long as they stay inside it.
// A path that leads out of the directory, through ".." or a symlink, is
// reported as missing (fs.ErrNotExist), so that callers, e.g. a fallback to
// embedded files, treat it like any other missing file.
//
// os.Root refuses all absolute symlinks, also those that point inside the
// directory.
type RootFs struct {
	dir    string
	logger log.Logger

	mu     sync.Mutex
	warned map[string]struct{}
}

// NewRootFs returns a new BaseFS that is confined to the directory dir.
//
// The directory doesn't need to exist: until it does, all files are missing,
// and MkdirAll creates it. Any other problem with the directory, e.g. if it is
// a file or not readable, is returned as an error.
//
// The directory is opened again for each operation, so that a directory that
// is created or replaced later, or a symlink to it that is changed, is used.
func NewRootFs(dir string, logger log.Logger) (*BaseFS, error) {
	root, err := os.OpenRoot(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return nil, err
	default:
		_ = root.Close()
	}
	return FromAfero(&RootFs{dir: dir, logger: logger, warned: map[string]struct{}{}}), nil
}

// Name returns the name of the filesystem.
func (r *RootFs) Name() string {
	return "RootFs"
}

// Open opens the named file for reading.
func (r *RootFs) Open(name string) (afero.File, error) {
	return r.OpenFile(name, os.O_RDONLY, 0)
}

// OpenFile opens the named file with the given flag and permissions.
func (r *RootFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	var f *os.File
	err := r.do("open", name, func(root *os.Root, rel string) error {
		var err error
		f, err = root.OpenFile(rel, flag, perm)
		return err
	})
	if err != nil {
		// no typed nil in the interface
		return nil, err
	}
	return f, nil
}

// Create creates or truncates the named file.
func (r *RootFs) Create(name string) (afero.File, error) {
	return r.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, createPerm)
}

// Stat returns the FileInfo of the named file, following symlinks.
func (r *RootFs) Stat(name string) (os.FileInfo, error) {
	var info os.FileInfo
	err := r.do("stat", name, func(root *os.Root, rel string) error {
		var err error
		info, err = root.Stat(rel)
		return err
	})
	if err != nil {
		return nil, err
	}
	return info, nil
}

// Mkdir creates the named directory.
func (r *RootFs) Mkdir(name string, perm os.FileMode) error {
	return r.do("mkdir", name, func(root *os.Root, rel string) error {
		return root.Mkdir(rel, perm)
	})
}

// MkdirAll creates the named directory with all its parents, and the root
// directory itself if it doesn't exist yet.
func (r *RootFs) MkdirAll(name string, perm os.FileMode) error {
	_, err := os.Stat(r.dir)
	if errors.Is(err, fs.ErrNotExist) {
		err = os.MkdirAll(r.dir, perm)
		if err != nil {
			return err
		}
	}
	return r.do("mkdir", name, func(root *os.Root, rel string) error {
		return root.MkdirAll(rel, perm)
	})
}

// Remove removes the named file or empty directory. A symlink is removed
// itself, not its target.
func (r *RootFs) Remove(name string) error {
	return r.do("remove", name, func(root *os.Root, rel string) error {
		return root.Remove(rel)
	})
}

// RemoveAll removes the named file or directory with all its contents.
func (r *RootFs) RemoveAll(name string) error {
	return r.do("remove", name, func(root *os.Root, rel string) error {
		return root.RemoveAll(rel)
	})
}

// Rename renames oldname to newname, both inside the root.
func (r *RootFs) Rename(oldname, newname string) error {
	newRel, ok := relative(newname)
	if !ok {
		return &fs.PathError{Op: "rename", Path: newname, Err: fs.ErrNotExist}
	}
	return r.do("rename", oldname, func(root *os.Root, rel string) error {
		return root.Rename(rel, newRel)
	})
}

// Chmod changes the mode of the named file.
func (r *RootFs) Chmod(name string, mode os.FileMode) error {
	return r.do("chmod", name, func(root *os.Root, rel string) error {
		return root.Chmod(rel, mode)
	})
}

// Chown changes the owner and group of the named file.
func (r *RootFs) Chown(name string, uid, gid int) error {
	return r.do("chown", name, func(root *os.Root, rel string) error {
		return root.Chown(rel, uid, gid)
	})
}

// Chtimes changes the access and modification times of the named file.
func (r *RootFs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return r.do("chtimes", name, func(root *os.Root, rel string) error {
		return root.Chtimes(rel, atime, mtime)
	})
}

// do opens the root and runs fn with the path relative to it. A path that
// leads out of the root, or a root that doesn't exist, is reported as missing.
func (r *RootFs) do(op, name string, fn func(root *os.Root, rel string) error) error {
	notExist := &fs.PathError{Op: op, Path: name, Err: fs.ErrNotExist}

	rel, ok := relative(name)
	if !ok {
		// ".." out of the root, without a symlink: not worth a warning
		return notExist
	}

	root, err := os.OpenRoot(r.dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return notExist
	case err != nil:
		return err
	}
	defer root.Close()

	err = fn(root, rel)
	if isEscape(err) {
		r.logEscape(rel)
		return notExist
	}
	return err
}

// logEscape logs a path that leads out of the directory: at Warn level once
// per path, up to maxWarnedPaths paths, and at Debug level after that.
func (r *RootFs) logEscape(rel string) {
	r.mu.Lock()
	_, seen := r.warned[rel]
	warn := !seen && len(r.warned) < maxWarnedPaths
	if warn {
		r.warned[rel] = struct{}{}
	}
	r.mu.Unlock()

	if seen {
		return
	}
	level := zerolog.DebugLevel
	if warn {
		level = zerolog.WarnLevel
	}
	// only the requested path: where the symlink leads must not show up anywhere
	r.logger.WithLevel(level).Str("dir", r.dir).Str("path", rel).
		Msg("a symlink leads out of the directory or is absolute, the file is treated as missing")
}

// relative returns name relative to the root, and whether it stays inside it
// as text. Names may start with a slash, as in afero.BasePathFs.
func relative(name string) (string, bool) {
	rel := strings.TrimLeft(filepath.ToSlash(name), "/")
	if rel == "" {
		rel = "."
	}
	return rel, filepath.IsLocal(rel) || rel == "."
}

// isEscape reports whether err is the error of os.Root for a path that leads
// out of the root.
func isEscape(err error) bool {
	for ; err != nil; err = errors.Unwrap(err) {
		if err.Error() == errPathEscapesText {
			return true
		}
	}
	return false
}
