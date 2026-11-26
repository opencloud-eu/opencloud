// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package revisioncleanup

import (
	"fmt"
	"path/filepath"
	"sync"
)

// A process may set up the same storage more than once: e.g. the storage provider and the data
// provider each create an instance of the storage driver. Each instance used to get a Service
// of its own, so that every storage had twice the configured workers, and the de-duplication of
// the requests per node didn't work across the two, so that a node could be cleaned up by both
// at the same time.
//
// NewServiceFromOptions therefore shares one Service between the instances of a storage in a
// process, by the root of the storage. The first instance creates it, with its lookup and its
// revision store, and the others reuse it: all instances of a storage work on the same files
// with the same settings, so either instance's lookup and store do. The Service stops once the
// last instance has shut it down, see Service.Shutdown.
var shared = struct {
	sync.Mutex
	byRoot map[string]*Service
}{byRoot: map[string]*Service{}}

// share is the bookkeeping of a Service shared by the instances of a storage, see shared.
type share struct {
	root   string
	refs   int    // the instances that haven't shut the Service down yet
	policy string // the settings of the Policy, see Policy.String
	async  AsyncOptions
}

// sharedService returns the Service shared by the instances of the storage at root, creating it
// with create for the first one. The instances must agree on the strategies (policy, see
// Policy.String) and the AsyncOptions: a storage set up twice with different settings would
// otherwise be cleaned up with the settings of whichever instance happened to come first.
func sharedService(root, policy string, async AsyncOptions, create func() (*Service, error)) (*Service, error) {
	root = filepath.Clean(root)
	shared.Lock()
	defer shared.Unlock()
	if s, ok := shared.byRoot[root]; ok {
		if s.share.policy != policy || s.share.async != async {
			return nil, fmt.Errorf("the storage at %q is set up more than once in this process with different revision cleanup settings: strategies %q and %+v, then strategies %q and %+v",
				root, s.share.policy, s.share.async, policy, async)
		}
		s.share.refs++
		return s, nil
	}
	s, err := create()
	if err != nil {
		return nil, err
	}
	s.share = &share{root: root, refs: 1, policy: policy, async: async}
	shared.byRoot[root] = s
	return s, nil
}

// release gives up the share of one instance of the storage in s, and reports whether it was
// the last one, so that s is to stop. A Service that isn't shared is always the last one.
func (s *Service) release() bool {
	if s.share == nil {
		return true
	}
	shared.Lock()
	defer shared.Unlock()
	if s.share.refs == 0 {
		// released by every instance already: stopping it again is harmless
		return true
	}
	s.share.refs--
	if s.share.refs > 0 {
		return false
	}
	delete(shared.byRoot, s.share.root)
	return true
}
