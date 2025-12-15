// Copyright 2025 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package tree

import (
	"context"
	"encoding/json"
	"path/filepath"

	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/watcher"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	kafka "github.com/segmentio/kafka-go"
)

const (
	CEPH_MDS_NOTIFY_ACCESS        = 0x0000000000000001 // 1
	CEPH_MDS_NOTIFY_ATTRIB        = 0x0000000000000002 // 2
	CEPH_MDS_NOTIFY_CLOSE_WRITE   = 0x0000000000000004 // 4
	CEPH_MDS_NOTIFY_CLOSE_NOWRITE = 0x0000000000000008 // 8
	CEPH_MDS_NOTIFY_CREATE        = 0x0000000000000010 // 16
	CEPH_MDS_NOTIFY_DELETE        = 0x0000000000000020 // 32
	CEPH_MDS_NOTIFY_DELETE_SELF   = 0x0000000000000040 // 64
	CEPH_MDS_NOTIFY_MODIFY        = 0x0000000000000080 // 128
	CEPH_MDS_NOTIFY_MOVE_SELF     = 0x0000000000000100 // 256
	CEPH_MDS_NOTIFY_MOVED_FROM    = 0x0000000000000200 // 512
	CEPH_MDS_NOTIFY_MOVED_TO      = 0x0000000000000400 // 1024
	CEPH_MDS_NOTIFY_OPEN          = 0x0000000000000800 // 2048
	CEPH_MDS_NOTIFY_CLOSE         = 0x0000000000001000 // 4096
	CEPH_MDS_NOTIFY_MOVE          = 0x0000000000002000 // 8192
	CEPH_MDS_NOTIFY_ONESHOT       = 0x0000000000004000 // 16384
	CEPH_MDS_NOTIFY_IGNORED       = 0x0000000000008000 // 32768
	CEPH_MDS_NOTIFY_ONLYDIR       = 0x0000000000010000 // 65536
)

type CephFSWatcher struct {
	tree    *Tree
	root    string
	brokers []string
	log     *zerolog.Logger
}

func NewCephfsWatcher(tree *Tree, brokers []string, log *zerolog.Logger) (*CephFSWatcher, error) {
	return &CephFSWatcher{
		tree:    tree,
		root:    tree.options.WatchRoot,
		brokers: brokers,
		log:     log,
	}, nil
}

type cephfsEvent struct {
	// Mask/Path are the event mask and path of the according entity
	Mask int    `json:"mask"`
	Path string `json:"path"`

	// Src*/Dst* are emitted for the source and destination of move events
	SrcMask  int    `json:"src_mask"`
	SrcPath  string `json:"src_path"`
	DestMask int    `json:"dest_mask"`
	DestPath string `json:"dest_path"`
}

func (w *CephFSWatcher) Watch(topic string) {
	w.log.Info().Str("topic", topic).Msg("cephfs watcher watching topic")
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: w.brokers,
		GroupID: "opencloud-posixfs",
		Topic:   topic,
	})

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Error().Err(err).Msg("error reading message")
			continue
		}

		ev := &cephfsEvent{}
		err = json.Unmarshal(m.Value, ev)
		if err != nil {
			w.log.Error().Err(err).Msg("error unmarshalling message")
			continue
		}

		if w.tree.isIgnored(ev.Path) {
			continue
		}

		mask := ev.Mask
		path := filepath.Join(w.tree.options.WatchRoot, ev.Path)
		if ev.DestMask > 0 {
			mask = ev.DestMask
			path = filepath.Join(w.tree.options.WatchRoot, ev.DestPath)
		}
		isDir := mask&CEPH_MDS_NOTIFY_ONLYDIR > 0
		go func() {
			switch {
			case mask&CEPH_MDS_NOTIFY_DELETE > 0:
				err = w.tree.Scan(path, watcher.ActionDelete, isDir)
			case mask&CEPH_MDS_NOTIFY_MOVED_TO > 0:
				if ev.SrcMask > 0 {
					// This is a move, clean up the old path
					err = w.tree.Scan(filepath.Join(w.tree.options.WatchRoot, ev.SrcPath), watcher.ActionMoveFrom, isDir)
				}
				err = w.tree.Scan(path, watcher.ActionMove, isDir)
			case mask&CEPH_MDS_NOTIFY_CREATE > 0:
				err = w.tree.Scan(path, watcher.ActionCreate, isDir)
			case mask&CEPH_MDS_NOTIFY_CLOSE_WRITE > 0:
				err = w.tree.Scan(path, watcher.ActionUpdate, isDir)
			case mask&CEPH_MDS_NOTIFY_CLOSE > 0:
				// ignore, already handled by CLOSE_WRITE
			default:
				w.log.Trace().Interface("event", ev).Msg("unhandled event")
				return
			}
			if err != nil {
				w.log.Error().Err(err).Str("path", path).Msg("error scanning file")
			}
		}()
	}
}
