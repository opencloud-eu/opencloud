// Copyright 2018-2024 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
package timemanager

import (
	"context"
	"os"
	"time"

	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
)

// Manager is responsible for managing time-related attributes of nodes in a decomposed file system.
type Manager struct {
}

// OverrideMtime overrides the modification time (mtime) attribute of a node with the given time.
func (m *Manager) OverrideMtime(ctx context.Context, _ *node.Node, attrs *node.Attributes, mtime time.Time) error {
	attrs.SetString(prefixes.MTimeAttr, mtime.UTC().Format(time.RFC3339Nano))
	return nil
}

// MTime retrieves the modification time (mtime) attribute of a node.
// If the attribute is not set, it falls back to the file's last modification time.
func (dtm *Manager) MTime(ctx context.Context, n *node.Node) (time.Time, error) {
	b, err := n.XattrString(ctx, prefixes.MTimeAttr)
	if err != nil {
		fi, err := os.Lstat(n.InternalPath())
		if err != nil {
			return time.Time{}, err
		}
		return fi.ModTime(), nil
	}
	return time.Parse(time.RFC3339Nano, b)
}

// SetMTime sets the modification time (mtime) attribute of a node to the given time.
// If the time is nil, the attribute is removed.
func (dtm *Manager) SetMTime(ctx context.Context, n *node.Node, mtime *time.Time) error {
	if mtime == nil {
		return n.RemoveXattr(ctx, prefixes.MTimeAttr)
	}
	return n.SetXattrString(ctx, prefixes.MTimeAttr, mtime.UTC().Format(time.RFC3339Nano))
}

// TMTime retrieves the tree modification time (tmtime) attribute of a node.
// If the attribute is not set, it falls back to the node's modification time (mtime).
func (dtm *Manager) TMTime(ctx context.Context, n *node.Node) (time.Time, error) {
	b, err := n.XattrString(ctx, prefixes.TreeMTimeAttr)
	if err == nil {
		return time.Parse(time.RFC3339Nano, b)
	}

	// no tmtime, use mtime
	return dtm.MTime(ctx, n)
}

// SetTMTime sets the tree modification time (tmtime) attribute of a node to the given time.
// If the time is nil, the attribute is removed.
func (dtm *Manager) SetTMTime(ctx context.Context, n *node.Node, tmtime *time.Time) error {
	if tmtime == nil {
		return n.RemoveXattr(ctx, prefixes.TreeMTimeAttr)
	}
	return n.SetXattrString(ctx, prefixes.TreeMTimeAttr, tmtime.UTC().Format(time.RFC3339Nano))
}

// CTime retrieves the creation time (ctime) attribute of a node.
// Since decomposedfs does not differentiate between ctime and mtime, it falls back to the node's modification time (mtime).
func (dtm *Manager) CTime(ctx context.Context, n *node.Node) (time.Time, error) {
	// decomposedfs does not differentiate between ctime and mtime
	return dtm.MTime(ctx, n)
}

// SetCTime sets the creation time (ctime) attribute of a node to the given time.
// Since decomposedfs does not differentiate between ctime and mtime, it sets the modification time (mtime) instead.
func (dtm *Manager) SetCTime(ctx context.Context, n *node.Node, mtime *time.Time) error {
	// decomposedfs does not differentiate between ctime and mtime
	return dtm.SetMTime(ctx, n, mtime)
}

// TCTime retrieves the tree creation time (tctime) attribute of a node.
// Since decomposedfs does not differentiate between ctime and mtime, it falls back to the tree modification time (tmtime).
func (dtm *Manager) TCTime(ctx context.Context, n *node.Node) (time.Time, error) {
	// decomposedfs does not differentiate between ctime and mtime
	return dtm.TMTime(ctx, n)
}

// SetTCTime sets the tree creation time (tctime) attribute of a node to the given time.
// Since decomposedfs does not differentiate between ctime and mtime, it sets the tree modification time (tmtime) instead.
func (dtm *Manager) SetTCTime(ctx context.Context, n *node.Node, tmtime *time.Time) error {
	// decomposedfs does not differentiate between ctime and mtime
	return dtm.SetTMTime(ctx, n, tmtime)
}

// DTime retrieves the deletion time (dtime) attribute of a node.
func (dtm *Manager) DTime(ctx context.Context, n *node.Node) (tmTime time.Time, err error) {
	b, err := n.XattrString(ctx, prefixes.DTimeAttr)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339Nano, b)
}

// SetDTime sets the deletion time (dtime) attribute of a node to the given time.
// If the time is nil, the attribute is removed.
func (dtm *Manager) SetDTime(ctx context.Context, n *node.Node, t *time.Time) (err error) {
	if t == nil {
		return n.RemoveXattr(ctx, prefixes.DTimeAttr)
	}
	return n.SetXattrString(ctx, prefixes.DTimeAttr, t.UTC().Format(time.RFC3339Nano))
}
