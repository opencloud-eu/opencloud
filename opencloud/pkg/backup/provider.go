package backup

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/vmihailenco/msgpack/v5"
)

// ListBlobstore required to check blob consistency
type ListBlobstore interface {
	List() ([]*node.Node, error)
	Path(node *node.Node) string
}

// DataProvider provides data for the consistency check
type DataProvider struct {
	Events chan interface{}

	fsys      fs.FS
	discpath  string
	lbs       ListBlobstore
	skipBlobs bool
}

// NodeData holds data about the nodes
type NodeData struct {
	NodePath        string
	BlobPath        string
	RequiresSymlink bool
	Inconsistencies []Inconsistency
}

// LinkData about the symlinks
type LinkData struct {
	LinkPath string
	NodePath string
}

// BlobData about the blobs in the blobstore
type BlobData struct {
	BlobPath string
}

// NewProvider creates a new DataProvider object
func NewProvider(fsys fs.FS, discpath string, lbs ListBlobstore) *DataProvider {
	return &DataProvider{
		Events: make(chan interface{}),

		fsys:      fsys,
		discpath:  discpath,
		lbs:       lbs,
		skipBlobs: lbs == nil,
	}
}

// ProduceData produces data for the consistency check
// Spawns 4 go-routines at the moment. If needed, this can be optimized.
func (dp *DataProvider) ProduceData() error {
	dirs, err := fs.Glob(dp.fsys, "spaces/*/*/nodes/*/*/*/*")
	if err != nil {
		return err
	}

	if len(dirs) == 0 {
		return errors.New("no backup found. Double check storage path")
	}

	wg := sync.WaitGroup{}

	// crawl spaces
	wg.Add(1)
	go func() {
		for _, d := range dirs {
			dp.evaluateNodeDir(d)
		}
		wg.Done()
	}()

	// crawl trash
	wg.Add(1)
	go func() {
		dp.evaluateTrashDir()
		wg.Done()
	}()

	// crawl blobstore
	if !dp.skipBlobs {
		wg.Add(1)
		go func() {
			bs, err := dp.lbs.List()
			if err != nil {
				fmt.Println("error listing blobs", err)
			}

			for _, bn := range bs {
				dp.Events <- BlobData{BlobPath: dp.lbs.Path(bn)}
			}
			wg.Done()
		}()
	}

	// wait for all crawlers to finish
	go func() {
		wg.Wait()
		dp.quit()
	}()

	return nil
}

func (dp *DataProvider) getBlobPath(path string) (string, Inconsistency) {
	if dp.skipBlobs {
		return "", ""
	}

	b, err := fs.ReadFile(dp.fsys, path+".mpk")
	if err != nil {
		return "", InconsistencyFilesMissing
	}

	m := map[string][]byte{}
	if err := msgpack.Unmarshal(b, &m); err != nil {
		return "", InconsistencyMalformedFile
	}

	// FIXME: how to check if metadata is complete?

	if bid := m["user.oc.blobid"]; string(bid) != "" {
		spaceID, _ := getIDsFromPath(filepath.Join(dp.discpath, path))
		return dp.lbs.Path(&node.Node{BaseNode: node.BaseNode{SpaceID: spaceID}, BlobID: string(bid)}), ""
	}

	return "", ""
}

func (dp *DataProvider) evaluateNodeDir(d string) {
	// d is something like spaces/a8/e5d981-41e4-4468-b532-258d5fb457d3/nodes/2d/08/8d/24
	// we could have multiple nodes under this, but we are only interested in one file per node - the one with "" extension
	entries, err := fs.ReadDir(dp.fsys, d)
	if err != nil {
		fmt.Println("error reading dir", err)
		return
	}

	if len(entries) == 0 {
		fmt.Println("empty dir", filepath.Join(dp.discpath, d))
		return
	}

	for _, e := range entries {
		switch {
		case e.IsDir():
			ls, err := fs.ReadDir(dp.fsys, filepath.Join(d, e.Name()))
			if err != nil {
				fmt.Println("error reading dir", err)
				continue
			}
			for _, l := range ls {
				linkpath := filepath.Join(dp.discpath, d, e.Name(), l.Name())

				r, _ := os.Readlink(linkpath)
				nodePath := filepath.Join(dp.discpath, d, e.Name(), r)
				dp.Events <- LinkData{LinkPath: linkpath, NodePath: nodePath}
			}
			fallthrough
		case filepath.Ext(e.Name()) == "" || _versionRegex.MatchString(e.Name()) || _trashRegex.MatchString(e.Name()):
			np := filepath.Join(dp.discpath, d, e.Name())
			var inc []Inconsistency
			if !dp.filesExist(filepath.Join(d, e.Name())) {
				inc = append(inc, InconsistencyFilesMissing)
			}
			bp, i := dp.getBlobPath(filepath.Join(d, e.Name()))
			if i != "" {
				inc = append(inc, i)
			}

			dp.Events <- NodeData{NodePath: np, BlobPath: bp, RequiresSymlink: requiresSymlink(np), Inconsistencies: inc}
		}
	}
}

func (dp *DataProvider) evaluateTrashDir() {
	linkpaths, err := fs.Glob(dp.fsys, "spaces/*/*/trash/*/*/*/*/*")
	if err != nil {
		fmt.Println("error reading trash", err)
	}
	for _, l := range linkpaths {
		linkpath := filepath.Join(dp.discpath, l)
		r, _ := os.Readlink(linkpath)
		p := filepath.Join(dp.discpath, l, "..", r)
		dp.Events <- LinkData{LinkPath: linkpath, NodePath: p}
	}
}

func (dp *DataProvider) filesExist(path string) bool {
	check := func(p string) bool {
		_, err := fs.Stat(dp.fsys, p)
		return err == nil
	}
	return check(path) && check(path+".mpk")
}

func (dp *DataProvider) quit() {
	close(dp.Events)
}

func requiresSymlink(path string) bool {
	spaceID, nodeID := getIDsFromPath(path)
	if nodeID != "" && spaceID != "" && (spaceID == nodeID || _versionRegex.MatchString(nodeID)) {
		return false
	}

	return true
}

func getIDsFromPath(path string) (string, string) {
	rawIDs := strings.Split(path, "/nodes/")
	if len(rawIDs) != 2 {
		return "", ""
	}

	s := strings.Split(rawIDs[0], "/spaces/")
	if len(s) != 2 {
		return "", ""
	}

	spaceID := strings.Replace(s[1], "/", "", -1)
	nodeID := strings.Replace(rawIDs[1], "/", "", -1)
	return spaceID, nodeID
}
