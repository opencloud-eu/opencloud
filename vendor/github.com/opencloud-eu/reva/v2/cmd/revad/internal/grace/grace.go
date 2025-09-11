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

package grace

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// Watcher watches a process for a graceful restart
// preserving open network sockets to avoid packet loss.
type Watcher struct {
	log                     zerolog.Logger
	graceful                bool
	ppid                    int
	lns                     map[string]net.Listener
	ss                      map[string]Server
	pidFile                 string
	childPIDs               []int
	gracefulShutdownTimeout int
}

// Option represent an option.
type Option func(w *Watcher)

// WithLogger adds a logger to the Watcher.
func WithLogger(l zerolog.Logger) Option {
	return func(w *Watcher) {
		w.log = l
	}
}

// WithPIDFile specifies the pid file to use.
func WithPIDFile(fn string) Option {
	return func(w *Watcher) {
		w.pidFile = fn
	}
}

func WithGracefuleShutdownTimeout(seconds int) Option {
	return func(w *Watcher) {
		w.gracefulShutdownTimeout = seconds
	}
}

// NewWatcher creates a Watcher.
func NewWatcher(opts ...Option) *Watcher {
	w := &Watcher{
		log:      zerolog.Nop(),
		graceful: os.Getenv("GRACEFUL") == "true",
		ppid:     os.Getppid(),
		ss:       map[string]Server{},
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}

// Exit exits the current process cleaning up
// existing pid files.
func (w *Watcher) Exit(errc int) {
	w.Clean()
	os.Exit(errc)
}

// Clean removes the pid file.
func (w *Watcher) Clean() {
	err := w.clean()
	if err != nil {
		w.log.Warn().Err(err).Msg("error removing pid file")
	} else {
		w.log.Info().Msgf("pid file %q got removed", w.pidFile)
	}
}

func (w *Watcher) clean() error {
	// only remove PID file if the PID has been written by us
	filePID, err := w.readPID()
	if err != nil {
		return err
	}

	if filePID != os.Getpid() {
		// the pidfile may have been changed by a forked child
		// TODO(labkode): is there a way to list children pids for current process?
		return fmt.Errorf("pid:%d in pidfile is different from pid:%d, it can be a leftover from a hard shutdown or that a reload was triggered", filePID, os.Getpid())
	}

	return os.Remove(w.pidFile)
}

func (w *Watcher) readPID() (int, error) {
	piddata, err := os.ReadFile(w.pidFile)
	if err != nil {
		return 0, err
	}
	// Convert the file contents to an integer.
	pid, err := strconv.Atoi(string(piddata))
	if err != nil {
		return 0, err
	}
	return pid, nil
}

// GetProcessFromFile reads the pidfile and returns the running process or error if the process or file
// are not available.
func GetProcessFromFile(pfile string) (*os.Process, error) {
	data, err := os.ReadFile(pfile)
	if err != nil {
		return nil, err
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return nil, err
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return nil, err
	}

	return process, nil
}

// WritePID writes the pid to the configured pid file.
func (w *Watcher) WritePID() error {
	// Read in the pid file as a slice of bytes.
	if piddata, err := os.ReadFile(w.pidFile); err == nil {
		// Convert the file contents to an integer.
		if pid, err := strconv.Atoi(string(piddata)); err == nil {
			// Look for the pid in the process list.
			if process, err := os.FindProcess(pid); err == nil {
				// Send the process a signal zero kill.
				if err := process.Signal(syscall.Signal(0)); err == nil {
					if !w.graceful {
						return fmt.Errorf("pid already running: %d", pid)
					}

					if pid != w.ppid { // overwrite only if parent pid is pidfile
						// We only get an error if the pid isn't running, or it's not ours.
						return fmt.Errorf("pid %d is not this process parent", pid)
					}
				} else {
					w.log.Warn().Err(err).Msg("error sending zero kill signal to current process")
				}
			} else {
				w.log.Warn().Msgf("pid:%d not found", pid)
			}
		} else {
			w.log.Warn().Msg("error casting contents of pidfile to pid(int)")
		}
	} // else {
	// w.log.Info().Msg("error reading pidfile")
	//}

	// If we get here, then the pidfile didn't exist or we are are in graceful reload and thus we overwrite
	// or the pid in it doesn't belong to the user running this app.
	err := os.WriteFile(w.pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0664)
	if err != nil {
		return err
	}
	w.log.Info().Msgf("pidfile saved at: %s", w.pidFile)
	return nil
}

func newListener(network, addr string) (net.Listener, error) {
	return net.Listen(network, addr)
}

// GetListeners return grpc listener first and http listener second.
func (w *Watcher) GetListeners(servers map[string]Server) (map[string]net.Listener, error) {
	w.ss = servers
	lns := map[string]net.Listener{}
	if w.graceful {
		w.log.Info().Msg("graceful restart, inheriting parent ln fds for grpc and http")
		count := 3
		for k, s := range servers {
			network, addr := s.Network(), s.Address()
			fd := os.NewFile(uintptr(count), "") // 3 because ExtraFile passed to new process
			count++
			ln, err := net.FileListener(fd)
			if err != nil {
				w.log.Error().Err(err).Msg("error creating net.Listener from fd")
				// create new fd
				ln, err := newListener(network, addr)
				if err != nil {
					return nil, err
				}
				lns[k] = ln
			} else {
				lns[k] = ln
			}

		}

		// kill parent
		// TODO(labkode): maybe race condition here?
		// What do we do if we cannot kill the parent but we have valid fds?
		// Do we abort running the forked child? Probably yes, as if the parent cannot be
		// killed that means we run two version of the code indefinitely.
		w.log.Info().Msgf("killing parent pid gracefully with SIGQUIT: %d", w.ppid)
		p, err := os.FindProcess(w.ppid)
		if err != nil {
			w.log.Error().Err(err).Msgf("error finding parent process with ppid:%d", w.ppid)
			err = errors.Wrap(err, "error finding parent process")
			return nil, err
		}
		err = p.Kill()
		if err != nil {
			w.log.Error().Err(err).Msgf("error killing parent process with ppid:%d", w.ppid)
			err = errors.Wrap(err, "error killing parent process")
			return nil, err
		}
		w.lns = lns
		return lns, nil
	}

	// create two listeners for grpc and http
	for k, s := range servers {
		network, addr := s.Network(), s.Address()
		ln, err := newListener(network, addr)
		if err != nil {
			return nil, err
		}
		lns[k] = ln

	}
	w.lns = lns
	return lns, nil
}

// Server is the interface that servers like HTTP or gRPC
// servers need to implement.
type Server interface {
	Stop() error
	GracefulStop() error
	Network() string
	Address() string
}

// TrapSignals captures the OS signal.
func (w *Watcher) TrapSignals() {
	signalCh := make(chan os.Signal, 1024)
	signal.Notify(signalCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	for {
		s := <-signalCh
		w.log.Info().Msgf("%v signal received", s)

		switch s {
		case syscall.SIGHUP:
			w.log.Info().Msg("preparing for a hot-reload, forking child process...")

			// Fork a child process.
			listeners := w.lns
			p, err := forkChild(listeners)
			if err != nil {
				w.log.Error().Err(err).Msgf("unable to fork child process")
			} else {
				w.log.Info().Msgf("child forked with new pid %d", p.Pid)
				w.childPIDs = append(w.childPIDs, p.Pid)
			}
		case syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM:
			gracefulShutdown(w)
			return
		}
	}
}

func gracefulShutdown(w *Watcher) {
	defer w.Clean()
	w.log.Info().Int("Timeout", w.gracefulShutdownTimeout).Msg("preparing for a graceful shutdown with deadline")
	wg := sync.WaitGroup{}

	for _, s := range w.ss {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.log.Info().Str("network.transport", s.Network()).Str("network.local.address", s.Address()).Msg("fd gracefully closed")
			err := s.GracefulStop()
			if err != nil {
				w.log.Error().Err(err).Msg("error stopping server")
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-time.After(time.Duration(w.gracefulShutdownTimeout) * time.Second):
		w.log.Info().Msg("graceful shutdown timeout reached. running hard shutdown")
		for _, s := range w.ss {
			w.log.Info().Str("network.transport", s.Network()).Str("network.local.address", s.Address()).Msg("fd abruptly closed")
			err := s.Stop()
			if err != nil {
				w.log.Error().Err(err).Msg("error stopping server")
			}
		}
		return
	case <-done:
		w.log.Info().Msg("all servers gracefully stopped")
		return
	}
}

func getListenerFile(ln net.Listener) (*os.File, error) {
	switch t := ln.(type) {
	case *net.TCPListener:
		return t.File()
	case *net.UnixListener:
		return t.File()
	}
	return nil, fmt.Errorf("unsupported listener: %T", ln)
}

func forkChild(lns map[string]net.Listener) (*os.Process, error) {
	// Get the file descriptor for the listener and marshal the metadata to pass
	// to the child in the environment.
	fds := map[string]*os.File{}
	for k, ln := range lns {
		fd, err := getListenerFile(ln)
		if err != nil {
			return nil, err
		}
		fds[k] = fd
	}

	// Pass stdin, stdout, and stderr along with the listener file to the child
	files := []*os.File{
		os.Stdin,
		os.Stdout,
		os.Stderr,
	}

	// Get current environment and add in the listener to it.
	environment := append(os.Environ(), "GRACEFUL=true")
	var counter = 3
	for k, fd := range fds {
		k = strings.ToUpper(k)
		environment = append(environment, k+"FD="+fmt.Sprintf("%d", counter))
		files = append(files, fd)
		counter++
	}

	// Get current process name and directory.
	execName, err := os.Executable()
	if err != nil {
		return nil, err
	}
	execDir := filepath.Dir(execName)

	// Spawn child process.
	p, err := os.StartProcess(execName, os.Args, &os.ProcAttr{
		Dir:   execDir,
		Env:   environment,
		Files: files,
		Sys:   &syscall.SysProcAttr{},
	})

	// TODO(labkode): if the process dies (because config changed and is wrong
	// we need to return an error
	if err != nil {
		return nil, err
	}

	return p, nil
}
