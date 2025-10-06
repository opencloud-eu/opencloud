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

package node

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/filelocks"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/pkg/errors"
)

// SetLock sets a lock on the node
func (n *Node) SetLock(ctx context.Context, lock *provider.Lock) error {
	ctx, span := tracer.Start(ctx, "SetLock")
	defer span.End()
	lockFilePath := n.LockFilePaths()[0]

	// ensure parent path exists
	if err := os.MkdirAll(filepath.Dir(lockFilePath), 0700); err != nil {
		return errors.Wrap(err, "Decomposedfs: error creating parent folder for lock")
	}

	// get file lock, so that nobody can create the lock in the meantime
	fileLock, err := filelocks.AcquireWriteLock(n.InternalPath())
	if err != nil {
		return err
	}

	defer func() {
		rerr := filelocks.ReleaseLock(fileLock)

		// if err is non nil we do not overwrite that
		if err == nil {
			err = rerr
		}
	}()

	// check if already locked
	l, err := n.ReadLock(ctx, true) // we already have a write file lock, so ReadLock() would fail to acquire a read file lock -> skip it
	switch err.(type) {
	case errtypes.NotFound:
		// file not locked, continue
	case nil:
		if l != nil {
			return errtypes.PreconditionFailed("already locked")
		}
	default:
		return errors.Wrap(err, "Decomposedfs: could check if file already is locked")
	}

	// O_EXCL to make open fail when the file already exists
	f, err := os.OpenFile(lockFilePath, os.O_EXCL|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return errors.Wrap(err, "Decomposedfs: could not create lock file")
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(lock); err != nil {
		return errors.Wrap(err, "Decomposedfs: could not write lock file")
	}

	return err
}

// ReadLock reads the lock id for a node
func (n *Node) ReadLock(ctx context.Context, skipFileLock bool) (*provider.Lock, error) {
	ctx, span := tracer.Start(ctx, "ReadLock")
	defer span.End()

	// ensure parent path exists
	_, subspan := tracer.Start(ctx, "os.MkdirAll")
	err := os.MkdirAll(filepath.Dir(n.InternalPath()), 0700)
	subspan.End()
	if err != nil {
		return nil, errors.Wrap(err, "Decomposedfs: error creating parent folder for lock")
	}

	// the caller of ReadLock already may hold a file lock
	if !skipFileLock {
		_, subspan := tracer.Start(ctx, "filelocks.AcquireReadLock")
		fileLock, err := filelocks.AcquireReadLock(n.InternalPath())
		subspan.End()

		if err != nil {
			return nil, err
		}

		defer func() {
			_, subspan := tracer.Start(ctx, "filelocks.ReleaseLock")
			rerr := filelocks.ReleaseLock(fileLock)
			subspan.End()

			// if err is non nil we do not overwrite that
			if err == nil {
				err = rerr
			}
		}()
	}

	f, err := openAndMigrateLockFile(ctx, n, os.O_RDONLY, 0)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, errtypes.NotFound("no lock found")
		}
		return nil, errors.Wrap(err, "Decomposedfs: could not open lock file")
	}
	defer f.Close()

	lock := &provider.Lock{}
	if err := json.NewDecoder(f).Decode(lock); err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Msg("Decomposedfs: could not decode lock file, ignoring")
		return nil, errors.Wrap(err, "Decomposedfs: could not read lock file")
	}

	// lock already expired
	if lock.Expiration != nil && time.Now().After(time.Unix(int64(lock.Expiration.Seconds), int64(lock.Expiration.Nanos))) {

		_, subspan = tracer.Start(ctx, "os.Remove")
		err = os.Remove(f.Name())
		subspan.End()
		if err != nil {
			return nil, errors.Wrap(err, "Decomposedfs: could not remove expired lock file")
		}
		// we successfully deleted the expired lock
		return nil, errtypes.NotFound("no lock found")
	}

	return lock, nil
}

// RefreshLock refreshes the node's lock
func (n *Node) RefreshLock(ctx context.Context, lock *provider.Lock, existingLockID string) error {
	ctx, span := tracer.Start(ctx, "RefreshLock")
	defer span.End()

	// ensure parent path exists
	if err := os.MkdirAll(filepath.Dir(n.InternalPath()), 0700); err != nil {
		return errors.Wrap(err, "Decomposedfs: error creating parent folder for lock")
	}
	fileLock, err := filelocks.AcquireWriteLock(n.InternalPath())

	if err != nil {
		return err
	}

	defer func() {
		rerr := filelocks.ReleaseLock(fileLock)

		// if err is non nil we do not overwrite that
		if err == nil {
			err = rerr
		}
	}()

	f, err := openAndMigrateLockFile(ctx, n, os.O_RDWR, os.ModeExclusive)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return errtypes.PreconditionFailed("lock does not exist")
	case err != nil:
		return errors.Wrap(err, "Decomposedfs: could not open lock file")
	}
	defer f.Close()
	readLock := &provider.Lock{}
	if err := json.NewDecoder(f).Decode(readLock); err != nil {
		return errors.Wrap(err, "Decomposedfs: could not read lock")
	}

	// check refresh lockID match
	if existingLockID == "" && readLock.LockId != lock.LockId {
		return errtypes.Aborted("mismatching lock ID")
	}

	// check if UnlockAndRelock sends the correct lockID
	if existingLockID != "" && readLock.LockId != existingLockID {
		return errtypes.Aborted("mismatching existing lock ID")
	}

	if ok, err := isLockModificationAllowed(ctx, readLock, lock); !ok {
		return err
	}

	// Rewind to the beginning of the file before writing a refreshed lock
	_, err = f.Seek(0, 0)
	if err != nil {
		return errors.Wrap(err, "could not seek to the beginning of the lock file")
	}
	if err := json.NewEncoder(f).Encode(lock); err != nil {
		return errors.Wrap(err, "Decomposedfs: could not write lock file")
	}

	return err
}

// Unlock unlocks the node
func (n *Node) Unlock(ctx context.Context, lock *provider.Lock) error {
	ctx, span := tracer.Start(ctx, "Unlock")
	defer span.End()

	// ensure parent path exists
	if err := os.MkdirAll(filepath.Dir(n.InternalPath()), 0700); err != nil {
		return errors.Wrap(err, "Decomposedfs: error creating parent folder for lock")
	}
	fileLock, err := filelocks.AcquireWriteLock(n.InternalPath())

	if err != nil {
		return err
	}

	defer func() {
		rerr := filelocks.ReleaseLock(fileLock)

		// if err is non nil we do not overwrite that
		if err == nil {
			err = rerr
		}
	}()

	f, err := openAndMigrateLockFile(ctx, n, os.O_RDONLY, os.ModeExclusive)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return errtypes.Aborted("lock does not exist")
	case err != nil:
		return errors.Wrap(err, "Decomposedfs: could not open lock file")
	}
	defer f.Close()

	oldLock := &provider.Lock{}
	if err := json.NewDecoder(f).Decode(oldLock); err != nil {
		return errors.Wrap(err, "Decomposedfs: could not read lock")
	}

	// check lock
	if lock == nil || (oldLock.LockId != lock.LockId) {
		return errtypes.Locked(oldLock.LockId)
	}

	if ok, err := isLockModificationAllowed(ctx, oldLock, lock); !ok {
		return err
	}

	if err = os.Remove(f.Name()); err != nil {
		return errors.Wrap(err, "Decomposedfs: could not remove lock file")
	}
	return err
}

// CheckLock compares the context lock with the node lock
func (n *Node) CheckLock(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "CheckLock")
	defer span.End()
	contextLock, _ := ctxpkg.ContextGetLockID(ctx)
	diskLock, _ := n.ReadLock(ctx, false)
	if diskLock != nil {
		switch contextLock {
		case "":
			return errtypes.Locked(diskLock.LockId) // no lockid in request
		case diskLock.LockId:
			return nil // ok
		default:
			return errtypes.Aborted("mismatching lock")
		}
	}
	if contextLock != "" {
		return errtypes.Aborted("not locked") // no lock on disk. why is there a lockid in the context
	}
	return nil // ok
}

func readLocksIntoOpaque(ctx context.Context, n *Node, ri *provider.ResourceInfo) error {
	lock, err := n.ReadLock(ctx, false)
	if err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Msg("Decomposedfs: could not read lock")
		return err
	}

	// reencode to ensure valid json
	var b []byte
	if b, err = json.Marshal(lock); err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Msg("Decomposedfs: could not marshal locks")
	}
	if ri.Opaque == nil {
		ri.Opaque = &types.Opaque{
			Map: map[string]*types.OpaqueEntry{},
		}
	}
	ri.Opaque.Map["lock"] = &types.OpaqueEntry{
		Decoder: "json",
		Value:   b,
	}
	ri.Lock = lock
	return err
}

func (n *Node) hasLocks(ctx context.Context) bool {
	for _, p := range n.LockFilePaths() {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

func isLockModificationAllowed(ctx context.Context, oldLock *provider.Lock, newLock *provider.Lock) (bool, error) {
	if oldLock.Type == provider.LockType_LOCK_TYPE_SHARED {
		return true, nil
	}

	appNameEquals := oldLock.AppName == newLock.AppName
	if !appNameEquals {
		return false, errtypes.PermissionDenied("app names of the locks are mismatching")
	}

	var lockUserEquals, contextUserEquals bool
	if oldLock.User == nil && newLock.GetUser() == nil {
		// no user lock set
		lockUserEquals = true
		contextUserEquals = true
	} else {
		lockUserEquals = utils.UserIDEqual(oldLock.User, newLock.GetUser())
		if !lockUserEquals {
			return false, errtypes.PermissionDenied("users of the locks are mismatching")
		}

		u := ctxpkg.ContextMustGetUser(ctx)
		contextUserEquals = utils.UserIDEqual(oldLock.User, u.Id)
		if !contextUserEquals {
			return false, errtypes.PermissionDenied("lock holder and current user are mismatching")
		}
	}

	return appNameEquals && lockUserEquals && contextUserEquals, nil

}

// openAndMigrateLockFile opens the lock file for a node, migrating any legacy lock files if found
// This is needed to support the transition to the new lock file scheme in the posix driver.
// Once this transition period is over the migration code can be removed again.
func openAndMigrateLockFile(ctx context.Context, n *Node, flag int, perm os.FileMode) (*os.File, error) {
	lockfilePaths := n.LockFilePaths()

	// check for legacy lock files and migrate them
	for i := 1; i < len(lockfilePaths); i++ {
		fi, err := os.Stat(lockfilePaths[i])
		if err == nil {
			if fi.Size() == 0 {
				// this is not a lock file of ours. ignore
				continue
			}

			f, err := os.Open(lockfilePaths[i])
			if err != nil {
				continue
			}
			defer f.Close()
			readLock := &provider.Lock{}
			if err := json.NewDecoder(f).Decode(readLock); err != nil {
				// this is not a lock file of ours. ignore
				continue
			}

			// legacy lock file found, move it to the new location
			err = os.MkdirAll(filepath.Dir(lockfilePaths[0]), 0700)
			if err != nil {
				return nil, errors.Wrap(err, "Decomposedfs: could not create lock file directory")
			}
			if err := os.Rename(lockfilePaths[i], lockfilePaths[0]); err != nil {
				return nil, errors.Wrap(err, "Decomposedfs: could not migrate lock file")
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, errors.Wrap(err, "Decomposedfs: could not stat legacy lock file")
		}
	}

	return os.OpenFile(lockfilePaths[0], flag, perm)
}
