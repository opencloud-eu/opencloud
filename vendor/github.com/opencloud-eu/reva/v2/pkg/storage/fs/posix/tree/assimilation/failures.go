// Package assimilation contains helpers for assimilating files into the posix fs
package assimilation

import (
	"io/fs"
	"syscall"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/pkg/errors"
)

const (
	// a failed file's retry delay starts at minRetryDelay and doubles per failure, up to maxRetryDelay
	minRetryDelay = time.Minute
	maxRetryDelay = 24 * time.Hour

	// maxFailures caps the memory use at about 8 MB. Files that fail beyond it aren't remembered.
	maxFailures = 10000
)

// Failures remembers files that failed to assimilate, so scans can skip them while they are unchanged
type Failures struct {
	lru *expirable.LRU[string, failure]
}

// failure is the state of a file when it last failed to assimilate
type failure struct {
	err     error
	modTime time.Time
	size    int64
	mode    fs.FileMode
	uid     uint32
	gid     uint32
	delay   time.Duration
	retryAt time.Time
}

// NewFailures returns a new Failures
func NewFailures() *Failures {
	// entries expire so that deleted files don't pile up
	return &Failures{lru: expirable.NewLRU[string, failure](maxFailures, nil, 2*maxRetryDelay)}
}

// Recent returns the last error of the file at path if it is unchanged and not due for a retry yet
func (f *Failures) Recent(path string, fi fs.FileInfo) error {
	last, ok := f.lru.Get(path)
	if !ok || !last.unchanged(fi) || !time.Now().Before(last.retryAt) {
		return nil
	}
	return errors.Wrapf(last.err, "item is unchanged since it failed to assimilate, not retrying before %s", last.retryAt.Format(time.RFC3339))
}

// Record stores the result of assimilating the file at path. fi is its state before the attempt.
func (f *Failures) Record(path string, fi fs.FileInfo, err error) {
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		f.lru.Remove(path)
		return
	}
	// when full, don't evict: a scan over more failing files than maxFailures would evict each one
	// before it comes around again
	if !f.lru.Contains(path) && f.lru.Len() >= maxFailures {
		return
	}

	delay := minRetryDelay
	if last, ok := f.lru.Peek(path); ok && last.unchanged(fi) {
		delay = min(2*last.delay, maxRetryDelay)
	}
	uid, gid := owner(fi)
	f.lru.Add(path, failure{
		err:     err,
		modTime: fi.ModTime(),
		size:    fi.Size(),
		mode:    fi.Mode(),
		uid:     uid,
		gid:     gid,
		delay:   delay,
		retryAt: time.Now().Add(delay),
	})
}

// unchanged reports whether fi still matches the file that failed. It includes the owner, so a chown
// that fixes the file triggers a retry.
func (f failure) unchanged(fi fs.FileInfo) bool {
	uid, gid := owner(fi)
	return f.modTime.Equal(fi.ModTime()) && f.size == fi.Size() && f.mode == fi.Mode() && f.uid == uid && f.gid == gid
}

func owner(fi fs.FileInfo) (uint32, uint32) {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return st.Uid, st.Gid
	}
	return 0, 0
}
