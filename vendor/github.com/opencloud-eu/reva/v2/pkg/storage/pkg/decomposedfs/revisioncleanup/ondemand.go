// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package revisioncleanup

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"path"
	"sync"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
)

// Request describes an on-demand cleanup of revisions, e.g. by a command line tool, as opposed
// to the cleanup that runs whenever a file gets a new revision.
//
// The strategy and the AsyncOptions default to the configuration of the storage driver, and
// can be overridden for this run. Without AsyncOptions of its own, though, a run waits for
// room in a full queue for as long as it takes (QueueFullBlock without a timeout), whatever
// the queue full policy of the storage driver: a run queues the files as fast as it finds
// them, so dropping them whenever the queue is full would skip most of them.
type Request struct {
	// SpaceIDs are the spaces to clean up. Exactly one of SpaceIDs, All and FSPath must be given.
	SpaceIDs []string
	// All cleans up all spaces.
	All bool
	// Path narrows the cleanup down to a file or a folder (recursively) within the space,
	// relative to the space root. It requires exactly one space in SpaceIDs.
	Path string
	// FSPath is a file or folder (recursively) given by its path on disk, as an admin sees it.
	// Only drivers that keep files under their own names on disk support it, i.e. posix, which
	// resolves it to the space and the path within it.
	FSPath string

	// Strategy overrides the configured revisions cleanup strategy for this run, see
	// revisions.NewRevisionCleanupStrategy.
	Strategy string
	// Async overrides the configured AsyncOptions for this run. Without it, the configured
	// ones are used, but with QueueFullBlock and no timeout.
	Async *AsyncOptions
	// DryRun only decides which revisions would be deleted, without deleting anything.
	DryRun bool
	// Shuffle cleans up the spaces, and the files and folders within each folder, in a random
	// order instead of the same order every time. A run that is stopped early through its
	// context (e.g. by a timeout) starts over from the beginning the next time; shuffled,
	// repeated runs start in different places, and eventually reach all files, also within a
	// space that is too large for a single run.
	Shuffle bool
}

// Validate checks the scope of the request.
func (r Request) Validate() error {
	switch {
	case r.FSPath != "" && (r.All || len(r.SpaceIDs) > 0 || r.Path != ""):
		return errors.New("a path on disk can't be combined with spaces or a path within a space")
	case r.FSPath != "":
		return nil
	case r.All && len(r.SpaceIDs) > 0:
		return errors.New("either all spaces or specific spaces can be cleaned up, not both")
	case !r.All && len(r.SpaceIDs) == 0:
		return errors.New("no space to clean up given")
	case r.Path != "" && len(r.SpaceIDs) != 1:
		return errors.New("a path requires exactly one space")
	}
	return nil
}

// OnDemand is implemented by the storage drivers that can clean up revisions on demand.
//
// CleanupRevisionsUnchecked cleans up the revisions of all files within the scope of the request
// and hands the Result of every file to report, one at a time, including the files that were
// skipped. It returns once all cleanups are done. It returns an error if the request is invalid or
// the cleanup could not be run; errors with single files are only reported.
//
// The end of ctx, e.g. a timeout, stops the cleanup gracefully: the files being cleaned up
// are finished, and the files already queued are reported as skipped. It then returns ctx's
// error.
type OnDemand interface {
	CleanupRevisionsUnchecked(ctx context.Context, req Request, report func(Result)) error
}

// Walker is the part of a tree needed to clean up the revisions of whole folders or spaces.
type Walker interface {
	RevisionStore
	// ListFolderKnownUnchecked lists the children of a folder that are known to the storage, for
	// maintenance. Unlike ListFolder:
	//   - it checks no permissions, so it needs no user in the context, and it also lists the
	//     files and folders that are denied to users;
	//   - it lists only the known children, the files and folders the storage has registered
	//     with a node id, and never assimilates (registers) unknown ones, which exist on disk
	//     but are not registered yet. Only posix has unknown files today, as files can be
	//     created on disk directly; in the other trees, every child is known.
	// So it changes nothing, and a cleanup, a dry run in particular, changes nothing but the
	// revisions it deletes. Unknown files have no revisions the storage knows of anyway, and a
	// later run covers them once they are registered.
	ListFolderKnownUnchecked(ctx context.Context, n *node.Node) ([]*node.Node, error)
}

// RunOptions are the settings of an on-demand Run.
type RunOptions struct {
	// Async are the AsyncOptions of the run's own Service.
	Async AsyncOptions
	// DryRun only decides which revisions would be deleted, without deleting anything.
	DryRun bool
	// Shuffle walks the files and folders within each folder in a random order, see
	// Request.Shuffle. The order of the roots is the caller's.
	Shuffle bool
}

// Run cleans up the revisions of the given roots, files or folders (recursively, e.g. space roots),
// with the given Cleaner and RunOptions, and hands the Result of every file to report, one at a
// time. It is the common implementation of OnDemand.CleanupRevisionsUnchecked for the storage
// drivers.
//
// It uses a Service of its own, independent of the storage driver's, so that the Cleaner and
// the AsyncOptions of the run apply: files are queued as they are found, while the workers
// already clean up, and Run waits for all cleanups before it returns. Files dropped by a full
// queue (queue full policy "drop") are reported as skipped. That Service leaves the metrics
// of the storage driver's queue and workers alone; the deleted revisions are counted.
//
// It returns an error if a cleanup could not be run at all, or ctx's error if it ends. Errors
// with single files or folders are only reported. When ctx ends, Run stops gracefully: it
// finishes the cleanups that are running, reports the files that are still queued as skipped,
// and doesn't look at the remaining ones.
//
// It checks no permissions: the Walker lists folders without them, and neither do the
// cleanups.
func Run(ctx context.Context, lu node.PathLookup, w Walker, roots []*node.Node, cleaner *Cleaner, o RunOptions, report func(Result), log *zerolog.Logger) error {
	if !cleaner.Enabled() {
		return errors.New("no revisions cleanup strategy configured or given")
	}
	if report == nil {
		report = func(Result) {}
	}
	// the workers report concurrently, report gets the results one at a time
	var reportMu sync.Mutex
	// the paths of the files queued by the walk, until their results are reported: the walk
	// knows them for free, while looking them up for every result would read all the parents
	paths := map[string]*queuedPath{}
	serialized := func(result Result) {
		reportMu.Lock()
		defer reportMu.Unlock()
		key := result.SpaceID + "/" + result.NodeID
		if p, ok := paths[key]; ok {
			if result.Path == "" {
				result.Path = p.path
			}
			if p.count--; p.count <= 0 {
				delete(paths, key)
			}
		}
		report(result)
	}
	remember := func(n *node.Node, path string) {
		reportMu.Lock()
		defer reportMu.Unlock()
		key := n.SpaceID + "/" + n.ID
		if p, ok := paths[key]; ok {
			p.count++ // the same file in more than one root
			return
		}
		paths[key] = &queuedPath{path: path, count: 1}
	}

	s, err := newService(cleaner, lu, w, o.Async, log, true, o.DryRun, serialized)
	if err != nil {
		return err
	}
	// stop the workers also when ctx ended; they are idle by then unless ctx ended
	defer func() { _ = s.Shutdown(context.WithoutCancel(ctx)) }()

	for _, root := range roots {
		if err := walkScoped(ctx, lu, w, root, s, o, serialized, remember); err != nil {
			return err
		}
	}
	return s.Wait(ctx)
}

// walkScoped walks root, see walk, switched to the group of its space if w is a SpaceScoper,
// so that the folders can be listed, see SpaceScoper. A space whose group can't be switched to
// is reported and left out.
func walkScoped(ctx context.Context, lu node.PathLookup, w Walker, root *node.Node, s *Service, o RunOptions, report func(Result), remember func(n *node.Node, path string)) error {
	ctx, unscope, err := scopeToSpace(ctx, w, root.SpaceID)
	if err != nil {
		report(Result{SpaceID: root.SpaceID, NodeID: root.ID, DryRun: o.DryRun, Err: fmt.Errorf("could not switch to the group of the space: %w", err)})
		return nil
	}
	defer func() { _ = unscope() }()
	return walk(ctx, w, root, rootPath(ctx, lu, root), s, o, report, remember)
}

// queuedPath is the path of a queued file, see Run
type queuedPath struct {
	path  string
	count int
}

// rootPath returns the path of root within its space, "/" for the space root, or "" if it
// can't be determined.
func rootPath(ctx context.Context, lu node.PathLookup, root *node.Node) string {
	if root.SpaceRoot == nil {
		return ""
	}
	p, err := lu.Path(ctx, root, node.NoCheck)
	if err != nil {
		return ""
	}
	return p
}

// walk queues the cleanup of n, if it is a file, or of all files below it, if it is a folder.
// rootPath is the path of n within its space, the paths of the files below it are derived
// from it and handed to remember before they are queued.
func walk(ctx context.Context, w Walker, n *node.Node, rootPath string, s *Service, o RunOptions, report func(Result), remember func(n *node.Node, path string)) error {
	type entry struct {
		n    *node.Node
		path string
	}
	stack := []entry{{n: n, path: rootPath}}
	for len(stack) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		switch current.n.Type(ctx) {
		case provider.ResourceType_RESOURCE_TYPE_FILE:
			remember(current.n, current.path)
			s.HandleRevisionCreated(ctx, current.n)
		case provider.ResourceType_RESOURCE_TYPE_CONTAINER:
			children, err := w.ListFolderKnownUnchecked(ctx, current.n)
			if err != nil && ctx.Err() != nil {
				return ctx.Err() // stopped while listing, not an error of the folder
			}
			if err != nil {
				report(Result{SpaceID: current.n.SpaceID, NodeID: current.n.ID, Path: current.path, DryRun: o.DryRun, Err: fmt.Errorf("could not list the folder at %q: %w", current.n.InternalPath(), err)})
				continue
			}
			// The second part of Request.Shuffle, after the order of the spaces (see
			// Decomposedfs.CleanupRevisionsUnchecked): the order of the children of this folder,
			// files and folders alike. As the walk takes them from a stack, depth first, this also
			// changes the order in which the subfolders are walked through.
			if o.Shuffle {
				rand.Shuffle(len(children), func(i, j int) { children[i], children[j] = children[j], children[i] })
			}
			for _, child := range children {
				childPath := ""
				if current.path != "" {
					childPath = path.Join(current.path, child.Name)
				}
				stack = append(stack, entry{n: child, path: childPath})
			}
		default:
			// references, symlinks and the like have no revisions
		}
	}
	return nil
}
