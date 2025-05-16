package spaceidindex

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/mtimesyncedcache"
	"github.com/pkg/errors"
	"github.com/rogpeppe/go-internal/lockedfile"
	"github.com/vmihailenco/msgpack/v5"
)

// Index holds space id indexes
type Index struct {
	root  string
	name  string
	cache mtimesyncedcache.Cache[string, map[string]string]
}

type readWriteCloseSeekTruncater interface {
	io.ReadWriteCloser
	io.Seeker
	Truncate(int64) error
}

// New returns a new index instance
func New(root, name string) *Index {
	return &Index{
		root: root,
		name: name,
	}
}

// Init initializes the index and makes sure it can be used
func (i *Index) Init() error {
	// Make sure to work on an existing tree
	return os.MkdirAll(filepath.Join(i.root, i.name), 0700)
}

// Load returns the content of an index
func (i *Index) Load(index string) (map[string]string, error) {
	indexPath := filepath.Join(i.root, i.name, index+".mpk")
	fi, err := os.Stat(indexPath)
	if err != nil {
		return nil, err
	}
	return i.readSpaceIndex(indexPath, i.name+":"+index, fi.ModTime())
}

// Add adds an entry to an index
// Consider calling AddAll() when trying to add multiple entries as every Add call has to lock the index
func (i *Index) Add(index, key string, value string) error {
	return i.updateIndex(index, map[string]string{key: value}, []string{})
}

// AddAll adds multiple entries to the index
func (i *Index) AddAll(index string, m map[string]string) error {
	return i.updateIndex(index, m, []string{})
}

// Remove removes an entry from the index
func (i *Index) Remove(index, key string) error {
	return i.updateIndex(index, map[string]string{}, []string{key})
}

func (i *Index) updateIndex(index string, addLinks map[string]string, removeLinks []string) error {
	indexPath := filepath.Join(i.root, i.name, index+".mpk")

	var err error
	// acquire writelock
	var f readWriteCloseSeekTruncater
	f, err = lockedfile.OpenFile(indexPath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return errors.Wrap(err, "unable to lock index to write")
	}
	defer func() {
		rerr := f.Close()

		// if err is non nil we do not overwrite that
		if err == nil {
			err = rerr
		}
	}()

	// Read current state
	msgBytes, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	links := map[string]string{}
	if len(msgBytes) > 0 {
		err = msgpack.Unmarshal(msgBytes, &links)
		if err != nil {
			return err
		}
	}

	// set new metadata
	for key, val := range addLinks {
		links[key] = val
	}
	for _, key := range removeLinks {
		delete(links, key)
	}
	// Truncate file
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	err = f.Truncate(0)
	if err != nil {
		return err
	}

	// Write new metadata to file
	d, err := msgpack.Marshal(links)
	if err != nil {
		return errors.Wrap(err, "unable to marshal index")
	}
	_, err = f.Write(d)
	if err != nil {
		return errors.Wrap(err, "unable to write index")
	}
	return nil
}

func (i *Index) readSpaceIndex(indexPath, cacheKey string, mtime time.Time) (map[string]string, error) {
	return i.cache.LoadOrStore(cacheKey, mtime, func() (map[string]string, error) {
		// Acquire a read log on the index file
		f, err := lockedfile.Open(indexPath)
		if err != nil {
			return nil, errors.Wrap(err, "unable to lock index to read")
		}
		defer func() {
			rerr := f.Close()

			// if err is non nil we do not overwrite that
			if err == nil {
				err = rerr
			}
		}()

		// Read current state
		msgBytes, err := io.ReadAll(f)
		if err != nil {
			return nil, errors.Wrap(err, "unable to read index")
		}
		links := map[string]string{}
		if len(msgBytes) > 0 {
			err = msgpack.Unmarshal(msgBytes, &links)
			if err != nil {
				return nil, errors.Wrap(err, "unable to parse index")
			}
		}
		return links, nil
	})
}
