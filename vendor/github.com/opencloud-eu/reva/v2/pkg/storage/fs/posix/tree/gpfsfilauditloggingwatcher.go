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

package tree

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/watcher"
	"github.com/rs/zerolog"
)

type GpfsFileAuditLoggingWatcher struct {
	tree *Tree
	log  *zerolog.Logger
}

type lwe struct {
	Event        string
	Path         string
	BytesWritten string
}

func NewGpfsFileAuditLoggingWatcher(tree *Tree, auditLogFile string, log *zerolog.Logger) (*GpfsFileAuditLoggingWatcher, error) {
	w := &GpfsFileAuditLoggingWatcher{
		tree: tree,
		log:  log,
	}

	_, err := os.Stat(auditLogFile)
	if err != nil {
		return nil, err
	}

	return w, nil
}

func (w *GpfsFileAuditLoggingWatcher) Watch(path string) {
start:
	file, err := os.Open(path)
	if err != nil {
		// try again later
		time.Sleep(5 * time.Second)
		goto start
	}
	defer file.Close()

	// Seek to the end of the file
	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		time.Sleep(5 * time.Second)
		goto start
	}

	reader := bufio.NewReader(file)
	ev := &lwe{}
	for {
		line, err := reader.ReadString('\n')
		switch err {
		case nil:
			err := json.Unmarshal([]byte(line), ev)
			if err != nil {
				w.log.Error().Err(err).Str("line", line).Msg("error unmarshalling line")
				continue
			}
			if w.tree.isIgnored(ev.Path) {
				continue
			}
			go func() {
				switch ev.Event {
				case "CREATE":
					err = w.tree.Scan(ev.Path, watcher.ActionCreate, false)
				case "CLOSE":
					var bytesWritten int
					bytesWritten, err = strconv.Atoi(ev.BytesWritten)
					if err == nil && bytesWritten > 0 {
						err = w.tree.Scan(ev.Path, watcher.ActionUpdate, false)
					}
				case "RENAME":
					err = w.tree.Scan(ev.Path, watcher.ActionMove, false)
					if warmupErr := w.tree.WarmupIDCache(ev.Path, false, false); warmupErr != nil {
						w.log.Error().Err(warmupErr).Str("path", ev.Path).Msg("error warming up id cache")
					}
				}
				if err != nil {
					w.log.Error().Err(err).Str("line", line).Msg("error unmarshalling line")
				}
			}()

		case io.EOF:
			time.Sleep(1 * time.Second)
		default:
			time.Sleep(5 * time.Second)
			goto start
		}
	}
}
