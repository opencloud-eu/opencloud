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

package filelocks

import (
	"errors"
	"io/fs"
	"os"
	"sync"
	"time"

	"github.com/gofrs/flock"
)

// LockFileSuffix to use for lock files
const LockFileSuffix = ".flock"

var (
	_localLocks sync.Map
	// waiting 20 lock cycles with a factor of 30 yields 6300ms, or a little over 6 sec
	_lockCycles              sync.Once
	_lockCyclesValue         = 20
	_lockCycleDuration       sync.Once
	_lockCycleDurationFactor = 30

	// ErrPathEmpty indicates that no path was specified
	ErrPathEmpty = errors.New("lock path is empty")
	// ErrAcquireLockFailed indicates that it was not possible to lock the resource.
	ErrAcquireLockFailed = errors.New("unable to acquire a lock on the file")
)

// SetMaxLockCycles configures the maximum amount of lock cycles. Subsequent calls to SetMaxLockCycles have no effect
func SetMaxLockCycles(v int) {
	_lockCycles.Do(func() {
		_lockCyclesValue = v
	})
}

// SetLockCycleDurationFactor configures the factor applied to the timeout allowed during a lock cycle. Subsequent calls to SetLockCycleDurationFactor have no effect
func SetLockCycleDurationFactor(v int) {
	_lockCycleDuration.Do(func() {
		_lockCycleDurationFactor = v
	})
}

// getMutexedFlock returns a new Flock struct for the given file.
// If there is already one in the local store, it returns nil.
// The caller has to wait until it can get a new one out of this
// mehtod.
func getMutexedFlock(file string) *flock.Flock {

	// Is there lock already?
	if _, ok := _localLocks.Load(file); ok {
		// There is already a lock for this file, another can not be acquired
		return nil
	}

	// Acquire the write log on the target node first.
	l := flock.New(file)
	_localLocks.Store(file, l)
	return l

}

// releaseMutexedFlock releases a Flock object that was acquired
// before by the getMutexedFlock function.
func releaseMutexedFlock(file string) {
	if len(file) > 0 {
		_localLocks.Delete(file)
	}
}

// acquireWriteLog acquires a lock on a file or directory.
// if the parameter write is true, it gets an exclusive write lock, otherwise a shared read lock.
// The function returns a Flock object, unlocking has to be done in the calling function.
func acquireLock(file string, write bool) (*flock.Flock, error) {
	var err error

	// Create a file to carry the log
	n := FlockFile(file)
	if len(n) == 0 {
		return nil, ErrPathEmpty
	}

	var flock *flock.Flock
	for i := 1; i <= _lockCyclesValue; i++ {
		if flock = getMutexedFlock(n); flock != nil {
			break
		}
		w := time.Duration(i*_lockCycleDurationFactor) * time.Millisecond

		time.Sleep(w)
	}
	if flock == nil {
		return nil, ErrAcquireLockFailed
	}

	var ok bool
	for i := 1; i <= _lockCyclesValue; i++ {
		if write {
			ok, err = flock.TryLock()
		} else {
			ok, err = flock.TryRLock()
		}

		if ok {
			break
		}

		time.Sleep(time.Duration(i*_lockCycleDurationFactor) * time.Millisecond)
	}

	if !ok {
		err = ErrAcquireLockFailed
	}

	if err != nil {
		return nil, err
	}
	return flock, nil
}

// FlockFile returns the flock filename for a given file name
// it returns an empty string if the input is empty
func FlockFile(file string) string {
	if file == "" {
		return ""
	}
	return file + LockFileSuffix
}

// AcquireReadLock tries to acquire a shared lock to read from the
// file and returns a lock object or an error accordingly.
// Call with the file to lock. This function creates .lock file next
// to it.
func AcquireReadLock(file string) (*flock.Flock, error) {
	return acquireLock(file, false)
}

// AcquireWriteLock tries to acquire a shared lock to write from the
// file and returns a lock object or an error accordingly.
// Call with the file to lock. This function creates an extra .lock
// file next to it.
func AcquireWriteLock(file string) (*flock.Flock, error) {
	return acquireLock(file, true)
}

// ReleaseLock releases a lock from a file that was previously created
// by AcquireReadLock or AcquireWriteLock.
func ReleaseLock(lock *flock.Flock) error {
	if lock == nil {
		return errors.New("cannot unlock nil lock")
	}

	// there is a probability that if the file can not be unlocked,
	// we also can not remove the file. We will only try to remove if it
	// was successfully unlocked.
	var err error
	n := lock.Path()
	// There is already a lock for this file

	err = lock.Unlock()
	if err == nil {
		if !lock.Locked() && !lock.RLocked() {
			err = os.Remove(n)
			// there is a concurrency issue when deleting the file
			// see https://github.com/owncloud/ocis/issues/3757
			// for now we just ignore "not found" errors when they pop up
			if err != nil && errors.Is(err, fs.ErrNotExist) {
				err = nil
			}
		}
	}
	releaseMutexedFlock(n)

	return err
}
