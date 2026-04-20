//go:build darwin && experimental_watchfs_darwin

package tree

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/options"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/watcher"
)

// FSnotifyWatcher fills the gap with fsnotify on Darwin, be careful with its limitations.
// The main reason for its existence is to provide a working watcher on Darwin for development and testing purposes.
type FSnotifyWatcher struct {
	tree    *Tree
	options *options.Options
	log     *zerolog.Logger

	mu      sync.Mutex
	watched map[string]struct{}
}

// NewWatcher creates a new FSnotifyWatcher which implements the Watcher interface for Darwin using fsnotify.
func NewWatcher(tree *Tree, o *options.Options, log *zerolog.Logger) (*FSnotifyWatcher, error) {
	log.Warn().Msg("fsnotify watcher on darwin has limitations and may not work as expected in all scenarios, not recommended for production use")

	return &FSnotifyWatcher{
		tree:    tree,
		options: o,
		log:     log,
		watched: make(map[string]struct{}),
	}, nil
}

// add takes care of adding watches for root and its subpaths.
func (w *FSnotifyWatcher) add(fsWatcher *fsnotify.Watcher, root string) error {
	// Check if the root is ignored before walking the tree
	if isPathIgnored(w.tree, root) {
		return nil
	}

	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// skip ignored paths or files
		if isPathIgnored(w.tree, p) || !d.IsDir() {
			return nil
		}

		w.mu.Lock()
		defer w.mu.Unlock()

		// path is known, skip
		if _, ok := w.watched[p]; ok {
			return nil
		}

		if err := fsWatcher.Add(p); err != nil {
			return err
		}

		w.watched[p] = struct{}{}

		return nil
	})
}

// remove takes care of removing watches for root and its subpaths.
func (w *FSnotifyWatcher) remove(fsWatcher *fsnotify.Watcher, root string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for p := range w.watched {
		if p == root || isSubpath(root, p) {
			if err := fsWatcher.Remove(p); err != nil {
				w.log.Debug().Err(err).Str("path", p).Msg("failed to remove watch")
			}

			delete(w.watched, p)
		}
	}
}

// handleEvent supervises the handling of fsnotify events.
func (w *FSnotifyWatcher) handleEvent(fsWatcher *fsnotify.Watcher, event fsnotify.Event) error {
	isCreate := event.Op&fsnotify.Create != 0
	isRemove := event.Op&fsnotify.Remove != 0
	isRename := event.Op&fsnotify.Rename != 0
	isWrite := event.Op&fsnotify.Write != 0

	isKnownEvent := isCreate || isRemove || isRename || isWrite
	isIgnored := isPathIgnored(w.tree, event.Name)

	// filter out unwanted events
	if isIgnored || !isKnownEvent {
		return nil
	}

	st, statErr := os.Stat(event.Name)
	exists := statErr == nil
	isDir := exists && st.IsDir()

	switch {
	case isRename:
		if exists {
			if isDir {
				_ = w.add(fsWatcher, event.Name)
			}

			return w.tree.Scan(event.Name, watcher.ActionMove, isDir)
		}

		w.remove(fsWatcher, event.Name)
		return w.tree.Scan(event.Name, watcher.ActionMoveFrom, false)
	case isRemove:
		w.remove(fsWatcher, event.Name)
		return w.tree.Scan(event.Name, watcher.ActionDelete, false)

	case isCreate:
		if exists {
			if isDir {
				_ = w.add(fsWatcher, event.Name)
			}

			return w.tree.Scan(event.Name, watcher.ActionCreate, isDir)
		}

		w.remove(fsWatcher, event.Name)
		return w.tree.Scan(event.Name, watcher.ActionMoveFrom, false)
	case isWrite:
		if exists {
			return w.tree.Scan(event.Name, watcher.ActionUpdate, isDir)
		}
	default:
		w.log.Warn().Interface("event", event).Msg("unhandled event")
	}

	return nil
}

// Watch starts watching the given path for changes.
func (w *FSnotifyWatcher) Watch(path string) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		w.log.Error().Err(err).Msg("failed to create watcher")
		return
	}
	defer func() { _ = fsWatcher.Close() }()

	if w.options.InotifyStatsFrequency > 0 {
		w.log.Debug().Str("watcher", "not implemented on darwin").Msg("fsnotify stats")
	}

	go func() {
		for {
			select {
			case event, ok := <-fsWatcher.Events:
				if !ok {
					return
				}

				if err := w.handleEvent(fsWatcher, event); err != nil {
					w.log.Error().Err(err).Str("path", event.Name).Msg("error scanning file")
				}
			case err, ok := <-fsWatcher.Errors:
				if !ok {
					return
				}

				w.log.Error().Err(err).Msg("fsnotify error")
			}
		}
	}()

	base := filepath.Join(path, "users")
	if err := w.add(fsWatcher, base); err != nil {
		w.log.Error().Err(err).Str("path", base).Msg("failed to add initial watches")
	}

	<-make(chan struct{})
}

// isSubpath checks if p is a subpath of root
func isSubpath(root, p string) bool {
	r, err := filepath.Abs(root)
	if err != nil {
		r = filepath.Clean(root)
	}

	pp, err := filepath.Abs(p)
	if err != nil {
		pp = filepath.Clean(p)
	}

	rel, err := filepath.Rel(r, pp)
	if err != nil {
		return false
	}

	return rel != "." && !strings.HasPrefix(rel, "..")
}

// isIgnored checks if the path is ignored by its tree.
func isPathIgnored(tree *Tree, path string) bool {

	isLockFile := isLockFile(path)
	isTrash := isTrash(path)
	isUpload := tree.isUpload(path)
	isInternal := tree.isInternal(path)

	// ask the tree if the path is internal or ignored
	return path == "" ||
		isLockFile ||
		isTrash ||
		isUpload ||
		isInternal
}
