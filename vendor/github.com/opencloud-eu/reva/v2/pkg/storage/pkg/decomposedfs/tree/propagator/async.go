// Copyright 2018-2021 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package propagator

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/renameio/v2"
	"github.com/google/uuid"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/options"
	"github.com/rs/zerolog"
	"github.com/vmihailenco/msgpack/v5"
)

const (
	ChangesDir = "changes"
)

var _propagationGracePeriod = 3 * time.Minute

// AsyncPropagator implements asynchronous treetime & treesize propagation
type AsyncPropagator struct {
	treeSizeAccounting bool
	treeTimeAccounting bool
	propagationDelay   time.Duration
	lookup             node.PathLookup
	log                *zerolog.Logger
}

// Change represents a change to the tree
type Change struct {
	SyncTime time.Time
	SizeDiff int64
}

// NewAsyncPropagator returns a new AsyncPropagator instance
func NewAsyncPropagator(treeSizeAccounting, treeTimeAccounting bool, o options.AsyncPropagatorOptions, lookup node.PathLookup, log *zerolog.Logger) AsyncPropagator {
	p := AsyncPropagator{
		treeSizeAccounting: treeSizeAccounting,
		treeTimeAccounting: treeTimeAccounting,
		propagationDelay:   o.PropagationDelay,
		lookup:             lookup,
		log:                log,
	}

	log.Info().Msg("async propagator starting up...")

	// spawn a goroutine that watches for stale .processing dirs and fixes them
	go func() {
		if !p.treeTimeAccounting && !p.treeSizeAccounting {
			// no propagation enabled
			log.Debug().Msg("propagation disabled or nothing to propagate")
			return
		}

		changesDirPath := filepath.Join(p.lookup.InternalRoot(), ChangesDir)
		doSleep := false // switch to not sleep on the first iteration
		for {
			if doSleep {
				time.Sleep(5 * time.Minute)
			}
			doSleep = true
			log.Debug().Msg("scanning for stale .processing dirs")

			entries, err := filepath.Glob(changesDirPath + "/**/*")
			if err != nil {
				log.Error().Err(err).Msg("failed to list changes")
				continue
			}

			for _, e := range entries {
				changesDirPath := e
				entry, err := os.Stat(changesDirPath)
				if err != nil {
					continue
				}
				// recover all dirs that seem to have been stuck
				if !entry.IsDir() || time.Now().Before(entry.ModTime().Add(_propagationGracePeriod)) {
					continue
				}

				go func() {
					if !strings.HasSuffix(changesDirPath, ".processing") {
						// first rename the existing node dir
						err = os.Rename(changesDirPath, changesDirPath+".processing")
						if err != nil {
							return
						}
						changesDirPath += ".processing"
					}

					log.Debug().Str("dir", changesDirPath).Msg("propagating stale .processing dir")
					parts := strings.SplitN(entry.Name(), ":", 2)
					if len(parts) != 2 {
						log.Error().Str("file", entry.Name()).Msg("encountered invalid .processing dir")
						return
					}

					now := time.Now()
					_ = os.Chtimes(changesDirPath, now, now)

					nodeID := strings.TrimSuffix(parts[1], ".processing")
					p.propagate(context.Background(), parts[0], nodeID, true, *log)
				}()
			}
		}
	}()

	return p
}

// Propagate triggers a propagation
func (p AsyncPropagator) Propagate(ctx context.Context, n *node.Node, sizeDiff int64) error {
	ctx, span := tracer.Start(ctx, "Propagate")
	defer span.End()
	log := appctx.GetLogger(ctx).With().
		Str("method", "async.Propagate").
		Str("spaceid", n.SpaceID).
		Str("nodeid", n.ID).
		Str("parentid", n.ParentID).
		Int64("sizeDiff", sizeDiff).
		Logger()

	if !p.treeTimeAccounting && (!p.treeSizeAccounting || sizeDiff == 0) {
		// no propagation enabled
		log.Debug().Msg("propagation disabled or nothing to propagate")
		return nil
	}

	// add a change to the parent node
	c := Change{
		// use a sync time and don't rely on the mtime of the current node, as the stat might not change when a rename happened too quickly
		SyncTime: time.Now().UTC(),
		SizeDiff: sizeDiff,
	}

	// Hand only the node's logical identity to the goroutine, never the live
	// *node.Node: sharing the instance would leak and race its metadata-lock state.
	go p.queuePropagation(ctx, n.SpaceID, n.ID, c, log)

	return nil
}

func (p AsyncPropagator) queuePropagation(ctx context.Context, spaceID, nodeID string, change Change, log zerolog.Logger) {
	// add a change to the parent node
	changePath := p.changesPath(spaceID, nodeID, uuid.New().String()+".mpk")

	data, err := msgpack.Marshal(change)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal Change")
		return
	}

	_, subspan := tracer.Start(ctx, "write changes file")
	ready := false
	triggerPropagation := false
	_ = os.MkdirAll(filepath.Dir(filepath.Dir(changePath)), 0700)
	err = os.Mkdir(filepath.Dir(changePath), 0700)
	triggerPropagation = err == nil || os.IsExist(err) // only the first goroutine, which succeeds to create the directory, is supposed to actually trigger the propagation
	for retries := 0; retries <= 500; retries++ {
		err := renameio.WriteFile(changePath, data, 0644)
		if err == nil {
			ready = true
			break
		}
		log.Debug().Err(err).Msg("failed to write Change to disk (retrying)")
		err = os.Mkdir(filepath.Dir(changePath), 0700)
		triggerPropagation = err == nil || os.IsExist(err) // only the first goroutine, which succeeds to create the directory, is supposed to actually trigger the propagation
	}

	if !ready {
		log.Error().Err(err).Msg("failed to write Change to disk")
		return
	}
	subspan.End()

	if !triggerPropagation {
		return
	}

	_, subspan = tracer.Start(ctx, "delay propagation")
	time.Sleep(p.propagationDelay) // wait a moment before propagating
	subspan.End()

	log.Debug().Msg("propagating")
	// add a change to the parent node
	changeDirPath := p.changesPath(spaceID, nodeID, "")

	// first rename the existing node dir
	err = os.Rename(changeDirPath, changeDirPath+".processing")
	if err != nil {
		// This can fail in 2 ways
		// 1. source does not exist anymore as it has already been propagated by another goroutine
		//    -> ignore, as the change is already being processed
		// 2. target already exists because a previous propagation is still running
		//    -> ignore, the previous propagation will pick the new changes up
		return
	}
	p.propagate(ctx, spaceID, nodeID, false, log)
}

func (p AsyncPropagator) propagate(ctx context.Context, spaceID, nodeID string, recalculateTreeSize bool, log zerolog.Logger) {
	changeDirPath := p.changesPath(spaceID, nodeID, "")
	processingPath := changeDirPath + ".processing"

	cleanup := func() {
		err := os.RemoveAll(processingPath)
		if err != nil {
			log.Error().Err(err).Msg("Could not remove .processing dir")
		}
	}

	_, subspan := tracer.Start(ctx, "list changes files")
	d, err := os.Open(processingPath)
	if err != nil {
		log.Error().Err(err).Msg("Could not open change .processing dir")
		cleanup()
		return
	}
	defer d.Close()
	names, err := d.Readdirnames(0)
	if err != nil {
		log.Error().Err(err).Msg("Could not read dirnames")
		cleanup()
		return
	}
	subspan.End()

	_, subspan = tracer.Start(ctx, "read changes files")
	pc := Change{}
	for _, name := range names {
		if !strings.HasSuffix(name, ".mpk") {
			continue
		}

		b, err := os.ReadFile(filepath.Join(processingPath, name))
		if err != nil {
			log.Error().Err(err).Msg("Could not read change")
			cleanup()
			return
		}
		c := Change{}
		err = msgpack.Unmarshal(b, &c)
		if err != nil {
			log.Error().Err(err).Msg("Could not unmarshal change")
			cleanup()
			return
		}
		if c.SyncTime.After(pc.SyncTime) {
			pc.SyncTime = c.SyncTime
		}
		pc.SizeDiff += c.SizeDiff
	}
	subspan.End()

	// TODO do we need to write an aggregated parentchange file?

	attrs := node.Attributes{}

	_, subspan = tracer.Start(ctx, "node.LockAndReadNode")
	n, unlock, err := node.LockAndReadNode(ctx, p.lookup, spaceID, nodeID, "", false, nil, false)
	subspan.End()
	if err != nil {
		if n != nil && !n.Exists {
			log.Debug().Str("attr", prefixes.PropagationAttr).Msg("node does not exist anymore, not propagating")
		} else {
			log.Error().Err(err).Msg("Propagation failed. Could not read node with lock.")
		}
		cleanup()
		return
	}
	defer func() { _ = unlock() }()

	if !n.Exists {
		log.Debug().Str("attr", prefixes.PropagationAttr).Msg("node does not exist anymore, not propagating")
		cleanup()
		return
	}

	if !n.HasPropagation(ctx) {
		log.Debug().Str("attr", prefixes.PropagationAttr).Msg("propagation attribute not set or unreadable, not propagating")
		cleanup()
		return
	}

	if p.treeTimeAccounting {
		// update the parent tree time if it is older than the nodes mtime
		updateSyncTime := false

		var tmTime time.Time
		tmTime, err = n.GetTMTime(ctx)
		switch {
		case err != nil:
			// missing attribute, or invalid format, overwrite
			log.Debug().Err(err).
				Msg("could not read tmtime attribute, overwriting")
			updateSyncTime = true
		case tmTime.Before(pc.SyncTime):
			log.Debug().
				Time("tmtime", tmTime).
				Time("stime", pc.SyncTime).
				Msg("parent tmtime is older than node mtime, updating")
			updateSyncTime = true
		default:
			log.Debug().
				Time("tmtime", tmTime).
				Time("stime", pc.SyncTime).
				Dur("delta", pc.SyncTime.Sub(tmTime)).
				Msg("node tmtime is younger than stime, not updating")
		}

		if updateSyncTime {
			// update the tree time of the parent node
			attrs.SetString(prefixes.TreeMTimeAttr, pc.SyncTime.UTC().Format(time.RFC3339Nano))
		}

		attrs.SetString(prefixes.TmpEtagAttr, "")
	}

	// size accounting
	if p.treeSizeAccounting && pc.SizeDiff != 0 {
		var newSize uint64

		// read treesize
		treeSize, err := n.GetTreeSize(ctx)
		switch {
		case recalculateTreeSize || metadata.IsAttrUnset(err):
			// fallback to calculating the treesize
			log.Warn().Msg("treesize attribute unset, falling back to calculating the treesize")
			newSize, err = calculateTreeSize(ctx, p.lookup, n)
			if err != nil {
				log.Error().Err(err).
					Msg("Error when calculating treesize of node.") // FIXME wat?
				cleanup()
				return
			}
		case err != nil:
			log.Error().Err(err).
				Msg("Failed to propagate treesize change. Error when reading the treesize attribute from node")
			cleanup()
			return
		case pc.SizeDiff > 0:
			newSize = treeSize + uint64(pc.SizeDiff)
		case uint64(-pc.SizeDiff) > treeSize:
			// The sizeDiff is larger than the current treesize. Which would result in
			// a negative new treesize. Something must have gone wrong with the accounting.
			// Reset the current treesize to 0.
			log.Error().Uint64("treeSize", treeSize).Int64("sizeDiff", pc.SizeDiff).
				Msg("Error when updating treesize of node. Updated treesize < 0. Resetting to 0")
			newSize = 0
		default:
			newSize = treeSize - uint64(-pc.SizeDiff)
		}

		// update the tree size of the node
		attrs.SetString(prefixes.TreesizeAttr, strconv.FormatUint(newSize, 10))
		log.Debug().Uint64("newSize", newSize).Msg("updated treesize of node")
	}

	if err = n.SetXattrsWithContext(ctx, attrs); err != nil {
		log.Error().Err(err).Msg("Failed to update extend attributes of node")
		cleanup()
		return
	}

	// Release node lock early, ignore already closed error
	_, subspan = tracer.Start(ctx, "f.Close")
	_ = unlock()
	subspan.End()

	log.Info().Msg("Propagation done. cleaning up")
	cleanup()

	if !n.IsSpaceRoot(ctx) {
		p.queuePropagation(ctx, n.SpaceID, n.ID, pc, log)
	}

	// Check for a changes dir that might have been added meanwhile and pick it up
	if _, err = os.Open(changeDirPath); err == nil {
		log.Info().Msg("Found a new changes dir. starting next propagation")
		time.Sleep(p.propagationDelay) // wait a moment before propagating
		err = os.Rename(changeDirPath, processingPath)
		if err != nil {
			// This can fail in 2 ways
			// 1. source does not exist anymore as it has already been propagated by another goroutine
			//    -> ignore, as the change is already being processed
			// 2. target already exists because a previous propagation is still running
			//    -> ignore, the previous propagation will pick the new changes up
			return
		}
		p.propagate(ctx, spaceID, nodeID, false, log)
	}
}

func (p AsyncPropagator) changesPath(spaceID, nodeID, filename string) string {
	return filepath.Join(p.lookup.InternalRoot(), ChangesDir, spaceID[0:2], spaceID+":"+nodeID, filename)
}
