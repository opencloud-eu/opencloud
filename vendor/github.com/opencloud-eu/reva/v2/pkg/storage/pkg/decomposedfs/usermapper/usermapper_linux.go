//go:build linux

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

package usermapper

import (
	"context"
	"fmt"
	"os/user"
	"runtime"
	"strconv"

	"golang.org/x/sys/unix"

	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
)

// UnixMapper is a user mapper that maps users to unix uids and gids
type UnixMapper struct {
	baseUid int
	baseGid int
}

// New returns a new user mapper
func NewUnixMapper() (*UnixMapper, error) {
	baseUid, _ := unix.SetfsuidRetUid(-1)
	baseGid, _ := unix.SetfsgidRetGid(-1)

	return &UnixMapper{
		baseUid: baseUid,
		baseGid: baseGid,
	}, nil
}

// RunInUserScope runs the given function in the scope of the base user
func (um *UnixMapper) RunInBaseScope(f func() error) error {
	unscope, err := um.ScopeBase()
	if err != nil {
		return err
	}
	defer func() { _ = unscope() }()

	return f()
}

// ScopeBase returns to the base uid and gid returning a function that can be used to restore the previous scope
func (um *UnixMapper) ScopeBase() (func() error, error) {
	return um.ScopeUserByIds(-1, um.baseGid)
}

// ScopeUser returns to the base uid and gid returning a function that can be used to restore the previous scope
func (um *UnixMapper) ScopeUser(ctx context.Context) (func() error, error) {
	u := revactx.ContextMustGetUser(ctx)

	uid, gid, err := um.mapUser(u.Username)
	if err != nil {
		return nil, err
	}
	return um.ScopeUserByIds(uid, gid)
}

// ScopeUserByIds scopes the current user to the given uid and gid returning a function that can be used to restore the previous scope
func (um *UnixMapper) ScopeUserByIds(uid, gid int) (func() error, error) {
	// The fsuid and the fsgid belong to the OS thread, not to the goroutine: lock the goroutine to
	// its thread, so that it keeps running with them, and no other goroutine runs on the thread
	// with them, until the returned function restores them and unlocks it again.
	runtime.LockOSThread()

	var err error
	var prevUid int
	var prevGid int
	uidSwitched := false
	// fail restores the fsuid if it was switched already, and unlocks the goroutine from its
	// thread again: on an error, the caller gets no function to restore the previous scope, so
	// without this, the goroutine would stay locked to its thread for good. That matters for
	// long-lived goroutines, e.g. background workers that switch for every item they process:
	// they would stay pinned to their thread after a single failure, with one more lock nested
	// for every further one.
	fail := func(err error) (func() error, error) {
		if uidSwitched {
			_ = unix.Setfsuid(prevUid)
		}
		runtime.UnlockOSThread()
		return nil, err
	}
	if uid >= 0 {
		prevUid, err = unix.SetfsuidRetUid(uid)
		if err != nil {
			return fail(err)
		}
		uidSwitched = true
		if testUid, _ := unix.SetfsuidRetUid(-1); testUid != uid {
			return fail(fmt.Errorf("failed to setfsuid to %d", uid))
		}
	}
	if gid >= 0 {
		prevGid, err = unix.SetfsgidRetGid(gid)
		if err != nil {
			return fail(err)
		}
		if testGid, _ := unix.SetfsgidRetGid(-1); testGid != gid {
			// the kernel refused it, setting it back only makes sure
			_ = unix.Setfsgid(prevGid)
			return fail(fmt.Errorf("failed to setfsgid to %d", gid))
		}
	}

	return func() error {
		if uid >= 0 {
			_ = unix.Setfsuid(prevUid)
		}
		if gid >= 0 {
			_ = unix.Setfsgid(prevGid)
		}
		// undo the runtime.LockOSThread above, the thread has its previous fsuid and fsgid again
		runtime.UnlockOSThread()
		return nil
	}, nil
}

func (u *UnixMapper) mapUser(username string) (int, int, error) {
	userDetails, err := user.Lookup(username)
	if err != nil {
		return 0, 0, err
	}

	uid, err := strconv.Atoi(userDetails.Uid)
	if err != nil {
		return 0, 0, err
	}
	gid, err := strconv.Atoi(userDetails.Gid)
	if err != nil {
		return 0, 0, err
	}

	return uid, gid, nil
}
