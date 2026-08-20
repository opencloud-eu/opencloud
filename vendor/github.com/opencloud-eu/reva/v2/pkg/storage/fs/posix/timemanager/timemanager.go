// Copyright 2018-2024 CERN
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

package timemanager

import (
	"context"
	"os"
	"syscall"
	"time"

	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
)

// Manager is responsible for managing time-related operations on files and directories.
type Manager struct {
}

// OverrideMtime overrides the modification time (mtime) of a node with the specified time.
func (m *Manager) OverrideMtime(ctx context.Context, n *node.Node, attrs *node.Attributes, mtime time.Time) error {
	attrs.SetTime(prefixes.MTimeAttr, mtime)
	return os.Chtimes(n.InternalPath(), mtime, mtime)
}

// MTime returns the modification time (mtime) of a node.
func (m *Manager) MTime(ctx context.Context, n *node.Node) (time.Time, error) {
	fi, err := os.Stat(n.InternalPath())
	if err != nil {
		return time.Time{}, err
	}
	return fi.ModTime(), nil
}

// SetMTime sets the modification time (mtime) of a node to the specified time.
func (m *Manager) SetMTime(ctx context.Context, n *node.Node, mtime *time.Time) error {
	return os.Chtimes(n.InternalPath(), *mtime, *mtime)
}

// TMTime returns the tree modification time (tmtime) of a node.
// If the tmtime is not set, it falls back to the modification time (mtime).
func (m *Manager) TMTime(ctx context.Context, n *node.Node) (time.Time, error) {
	b, err := n.XattrString(ctx, prefixes.TreeMTimeAttr)
	if err == nil {
		return time.Parse(time.RFC3339Nano, b)
	}

	// no tmtime, use mtime
	return m.MTime(ctx, n)
}

// SetTMTime sets the tree modification time (tmtime) of a node to the specified time.
// If tmtime is nil, the tmtime attribute is removed.
func (m *Manager) SetTMTime(ctx context.Context, n *node.Node, tmtime *time.Time) error {
	if tmtime == nil {
		return n.RemoveXattr(ctx, prefixes.TreeMTimeAttr)
	}
	return n.SetXattrString(ctx, prefixes.TreeMTimeAttr, tmtime.UTC().Format(time.RFC3339Nano))
}

// CTime returns the creation time (ctime) of a node.
func (m *Manager) CTime(ctx context.Context, n *node.Node) (time.Time, error) {
	fi, err := os.Stat(n.InternalPath())
	if err != nil {
		return time.Time{}, err
	}

	stat := fi.Sys().(*syscall.Stat_t)
	statCTime := StatCTime(stat)
	//nolint:unconvert
	return time.Unix(int64(statCTime.Sec), int64(statCTime.Nsec)), nil
}

// TCTime returns the tree creation time (tctime) of a node.
// Since decomposedfs does not differentiate between ctime and mtime, it falls back to TMTime.
func (m *Manager) TCTime(ctx context.Context, n *node.Node) (time.Time, error) {
	// decomposedfs does not differentiate between ctime and mtime
	return m.TMTime(ctx, n)
}

// SetTCTime sets the tree creation time (tctime) of a node to the specified time.
// Since decomposedfs does not differentiate between ctime and mtime, it falls back to SetTMTime.
func (m *Manager) SetTCTime(ctx context.Context, n *node.Node, tmtime *time.Time) error {
	// decomposedfs does not differentiate between ctime and mtime
	return m.SetTMTime(ctx, n, tmtime)
}

// DTime returns the deletion time (dtime) of a node.
func (m *Manager) DTime(ctx context.Context, n *node.Node) (tmTime time.Time, err error) {
	b, err := n.XattrString(ctx, prefixes.DTimeAttr)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339Nano, b)
}

// SetDTime sets the deletion time (dtime) of a node to the specified time.
// If t is nil, the dtime attribute is removed.
func (m *Manager) SetDTime(ctx context.Context, n *node.Node, t *time.Time) (err error) {
	if t == nil {
		return n.RemoveXattr(ctx, prefixes.DTimeAttr)
	}
	return n.SetXattrString(ctx, prefixes.DTimeAttr, t.UTC().Format(time.RFC3339Nano))
}
