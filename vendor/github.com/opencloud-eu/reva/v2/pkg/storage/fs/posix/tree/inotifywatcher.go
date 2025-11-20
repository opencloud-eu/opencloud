//go:build linux

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
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/options"
	"github.com/pablodz/inotifywaitgo/inotifywaitgo"
	"github.com/rs/zerolog"
	slogzerolog "github.com/samber/slog-zerolog/v2"
)

type InotifyWatcher struct {
	tree    *Tree
	options *options.Options
	log     *zerolog.Logger
}

func NewInotifyWatcher(tree *Tree, o *options.Options, log *zerolog.Logger) (*InotifyWatcher, error) {
	return &InotifyWatcher{
		tree:    tree,
		options: o,
		log:     log,
	}, nil
}

func (iw *InotifyWatcher) Watch(path string) {
	if iw.options.InotifyStatsFrequency > 0 {
		go func() {
			for {
				iw.printStats()
				time.Sleep(iw.options.InotifyStatsFrequency)
			}
		}()
	}

	// create a slog logger to be passed to the settings of inotifywatcher to log into
	logger := slog.New(slogzerolog.Option{Level: slog.LevelDebug, Logger: iw.log}.NewZerologHandler())

	events := make(chan inotifywaitgo.FileEvent)
	errors := make(chan error)

	go inotifywaitgo.WatchPath(&inotifywaitgo.Settings{
		Dir:        path,
		FileEvents: events,
		ErrorChan:  errors,
		KillOthers: true,
		Options: &inotifywaitgo.Options{
			Recursive: true,
			Events: []inotifywaitgo.EVENT{
				inotifywaitgo.CREATE,
				inotifywaitgo.MOVED_TO,
				inotifywaitgo.MOVED_FROM,
				inotifywaitgo.CLOSE_WRITE,
				inotifywaitgo.DELETE,
			},
			Monitor: true,
		},
		Verbose: false,
		Log:     logger,
	})

	for {
		select {
		case event := <-events:
			if iw.tree.isIgnored(event.Filename) {
				continue
			}
			for _, e := range event.Events {
				go func() {
					var err error
					switch e {
					case inotifywaitgo.DELETE:
						err = iw.tree.Scan(event.Filename, ActionDelete, event.IsDir)
					case inotifywaitgo.MOVED_FROM:
						err = iw.tree.Scan(event.Filename, ActionMoveFrom, event.IsDir)
					case inotifywaitgo.MOVED_TO:
						err = iw.tree.Scan(event.Filename, ActionMove, event.IsDir)
					case inotifywaitgo.CREATE:
						err = iw.tree.Scan(event.Filename, ActionCreate, event.IsDir)
					case inotifywaitgo.CLOSE_WRITE:
						err = iw.tree.Scan(event.Filename, ActionUpdate, event.IsDir)
					case inotifywaitgo.CLOSE:
						// ignore, already handled by CLOSE_WRITE
					default:
						iw.log.Warn().Interface("event", event).Msg("unhandled event")
						return
					}
					if err != nil {
						iw.log.Error().Err(err).Str("path", event.Filename).Msg("error scanning file")
					}
				}()
			}

		case err := <-errors:
			switch err.Error() {
			case inotifywaitgo.NOT_INSTALLED:
				panic("Error: inotifywait is not installed")
			case inotifywaitgo.INVALID_EVENT:
				// ignore
			default:
				fmt.Printf("Error: %s\n", err)
			}
		}
	}
}

// InotifyUsage holds the number of inotify watches and instances.
type InotifyUsage struct {
	Watches      int
	Instances    int
	MaxWatches   int
	MaxInstances int
}

func countInotifyFDs(pid string) (int, int, error) {
	fds, err := os.ReadDir(filepath.Join("/proc", pid, "fd"))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil // Process may have exited, treat as 0.
		}
		return 0, 0, fmt.Errorf("failed to read /proc/%s/fd: %w", pid, err)
	}

	watches := 0
	instances := 0
	for _, fd := range fds {
		if !fd.IsDir() {
			if fd.Type()&os.ModeSymlink == 0 {
				continue
			}

			link, err := os.Readlink(filepath.Join("/proc", pid, "fd", fd.Name()))
			if err != nil || (link != "inotify" && link != "anon_inode:inotify") {
				continue
			}

			instances++
			fdinfoPath := filepath.Join("/proc", pid, "fdinfo", fd.Name())
			content, err := os.ReadFile(fdinfoPath)
			if err != nil {
				return 0, 0, fmt.Errorf("failed to read %s: %w", fdinfoPath, err)
			}

			lines := strings.SplitSeq(string(content), "\n")
			for line := range lines {
				if strings.HasPrefix(line, "inotify") {
					watches++
				}
			}
		}
	}
	return watches, instances, nil
}

func GetInotifyUsageFromProc() (InotifyUsage, error) {
	usage := InotifyUsage{}
	var err error

	usage.MaxWatches, err = readProcFile("sys/fs/inotify/max_user_watches")
	if err != nil {
		return usage, fmt.Errorf("failed to read max_user_watches: %w", err)
	}
	usage.MaxInstances, err = readProcFile("sys/fs/inotify/max_user_instances")
	if err != nil {
		return usage, fmt.Errorf("failed to read max_user_instances: %w", err)
	}

	dirs, err := os.ReadDir("/proc")
	if err != nil {
		return usage, fmt.Errorf("failed to read /proc: %w", err)
	}

	totalWatches := 0
	totalInstances := 0
	for _, dir := range dirs {
		if dir.IsDir() {
			pid := dir.Name()
			if _, err := strconv.Atoi(pid); err == nil {
				watches, instances, err := countInotifyFDs(pid)
				if err != nil {
					continue
				}
				totalWatches += watches
				totalInstances += instances
			}
		}
	}

	usage.Watches = totalWatches
	usage.Instances = totalInstances
	return usage, nil
}

func readProcFile(filename string) (int, error) {
	filePath := filepath.Join("/proc", filename)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0, err
	}
	i, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil {
		return 0, fmt.Errorf("failed to parse max_user_watches: %w", err)
	}
	return i, nil
}

func (iw *InotifyWatcher) printStats() {
	t := time.Now()
	usage, err := GetInotifyUsageFromProc()
	if err != nil {
		iw.log.Error().Err(err).Msg("failed to get inotify usage")
		return
	}
	d := time.Since(t)

	iw.log.Info().
		Str("watches", fmt.Sprintf("%d/%d (%.2f%%)", usage.Watches, usage.MaxWatches, float64(usage.Watches)/float64(usage.MaxWatches)*100)).
		Str("instances", fmt.Sprintf("%d/%d (%.2f%%)", usage.Instances, usage.MaxInstances, float64(usage.Instances)/float64(usage.MaxInstances)*100)).
		Str("duration", d.String()).
		Msg("Inotify usage stats")
}
