// Package backup contains OpenCloud backup functionality.
package backup

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/lookup"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v3/pkg/storage/pkg/decomposedfs/node"
	"github.com/vmihailenco/msgpack/v5"
)

// Inconsistency describes the type of inconsistency
type Inconsistency string

var (
	// InconsistencyBlobMissing is an inconsistency where a blob is missing in the blobstore
	InconsistencyBlobMissing Inconsistency = "blob missing"
	// InconsistencyBlobOrphaned is an inconsistency where a blob in the blobstore has no reference
	InconsistencyBlobOrphaned Inconsistency = "blob orphaned"
	// InconsistencyNodeMissing is an inconsistency where a symlink points to a non-existing node
	InconsistencyNodeMissing Inconsistency = "node missing"
	// InconsistencyMetadataMissing is an inconsistency where a node is missing metadata
	InconsistencyMetadataMissing Inconsistency = "metadata missing"
	// InconsistencySymlinkMissing is an inconsistency where a node is missing a symlink
	InconsistencySymlinkMissing Inconsistency = "symlink missing"
	// InconsistencyFilesMissing is an inconsistency where a node is missing metadata files like .mpk or .mlock
	InconsistencyFilesMissing Inconsistency = "files missing"
	// InconsistencyMalformedFile is an inconsistency where a node has a malformed metadata file
	InconsistencyMalformedFile Inconsistency = "malformed file"

	// regex to determine if a node is trashed or versioned.
	// 9113a718-8285-4b32-9042-f930f1a58ac2.REV.2024-05-22T07:32:53.89969726Z
	_versionRegex = regexp.MustCompile(`\.REV\.[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]+)?Z(\.\d+)?$`)
	//   9113a718-8285-4b32-9042-f930f1a58ac2.T.2024-05-23T08:25:20.006571811Z <- this HAS a symlink
	_trashRegex = regexp.MustCompile(`\.T\.[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]+)?Z$`)
)

// Consistency holds the node and blob data of a storage provider
type Consistency struct {
	// Storing the data like this might take a lot of memory
	// we might need to optimize this if we run into memory issues
	Nodes          map[string][]Inconsistency
	LinkedNodes    map[string][]Inconsistency
	BlobReferences map[string][]Inconsistency
	Blobs          map[string][]Inconsistency

	nodeToLink map[string]string
	blobToNode map[string]string

	// recheck drops findings that are resolved by now, nil disables the re-check
	recheck *rechecker
	// resolved counts the findings that disappeared when re-checked
	resolved int
	// inFlight counts the nodes that still process an upload, their missing blobs are not reported
	inFlight int
}

// NewConsistency creates a new Consistency object
func NewConsistency() *Consistency {
	return &Consistency{
		Nodes:          make(map[string][]Inconsistency),
		LinkedNodes:    make(map[string][]Inconsistency),
		BlobReferences: make(map[string][]Inconsistency),
		Blobs:          make(map[string][]Inconsistency),

		nodeToLink: make(map[string]string),
		blobToNode: make(map[string]string),
	}
}

func newRechecker(storagepath string, lbs ListBlobstore) *rechecker {
	sep := string(filepath.Separator)
	return &rechecker{
		localPrefix: strings.TrimSuffix(filepath.Clean(storagepath), sep) + sep,
		fsys:        os.DirFS(storagepath),
		lbs:         lbs,
	}
}

// CheckProviderConsistency checks the consistency of a space
func CheckProviderConsistency(storagepath string, lbs ListBlobstore, fail bool) error {
	fsys := os.DirFS(storagepath)

	p := NewProvider(fsys, storagepath, lbs)
	if err := p.ProduceData(); err != nil {
		return err
	}

	c := NewConsistency()
	c.recheck = newRechecker(storagepath, lbs)
	c.GatherData(p.Events)

	return c.PrintResults(storagepath, fail)
}

// GatherData gathers and evaluates data produced by the DataProvider
func (c *Consistency) GatherData(events <-chan any) {
	for ev := range events {
		switch d := ev.(type) {
		case NodeData:
			// does it have inconsistencies?
			if len(d.Inconsistencies) != 0 {
				c.Nodes[d.NodePath] = append(c.Nodes[d.NodePath], d.Inconsistencies...)
			}
			// is it linked?
			if _, ok := c.LinkedNodes[d.NodePath]; ok {
				deleteInconsistency(c.LinkedNodes, d.NodePath)
			} else if d.RequiresSymlink && c.Nodes[d.NodePath] == nil {
				c.Nodes[d.NodePath] = []Inconsistency{}
			}
			// does it have a blob?
			if d.BlobPath != "" {
				if _, ok := c.Blobs[d.BlobPath]; ok {
					deleteInconsistency(c.Blobs, d.BlobPath)
				} else {
					c.BlobReferences[d.BlobPath] = []Inconsistency{}
					c.blobToNode[d.BlobPath] = d.NodePath
				}
			}
		case LinkData:
			// does it have a node?
			if _, ok := c.Nodes[d.NodePath]; ok {
				deleteInconsistency(c.Nodes, d.NodePath)
			} else {
				c.LinkedNodes[d.NodePath] = []Inconsistency{}
				c.nodeToLink[d.NodePath] = d.LinkPath
			}
		case BlobData:
			// does it have a reference?
			if _, ok := c.BlobReferences[d.BlobPath]; ok {
				deleteInconsistency(c.BlobReferences, d.BlobPath)
			} else {
				c.Blobs[d.BlobPath] = []Inconsistency{}
			}

		}
	}

	for n, incs := range c.Nodes {
		if len(incs) == 0 {
			incs = append(incs, InconsistencySymlinkMissing)
		}
		found := len(incs)
		incs = slices.DeleteFunc(incs, func(inc Inconsistency) bool {
			return c.recheck.nodeInconsistencyResolved(n, inc)
		})
		c.resolved += found - len(incs)
		if len(incs) == 0 {
			delete(c.Nodes, n)
			continue
		}
		c.Nodes[n] = incs
	}
	for l := range c.LinkedNodes {
		if c.recheck.nodeExists(l) {
			delete(c.LinkedNodes, l)
			c.resolved++
			continue
		}
		c.LinkedNodes[l] = append(c.LinkedNodes[l], InconsistencyNodeMissing)
	}
	for b := range c.Blobs {
		c.Blobs[b] = append(c.Blobs[b], InconsistencyBlobOrphaned)
	}
	for b := range c.BlobReferences {
		if c.recheck.blobExists(b) {
			delete(c.BlobReferences, b)
			c.resolved++
			continue
		}
		if c.recheck.nodeProcessing(c.blobToNode[b]) {
			delete(c.BlobReferences, b)
			c.inFlight++
			continue
		}
		c.BlobReferences[b] = append(c.BlobReferences[b], InconsistencyBlobMissing)
	}
}

// PrintResults prints the results of the evaluation
func (c *Consistency) PrintResults(discpath string, fail bool) error {
	if len(c.Nodes) != 0 {
		fmt.Println("\n🚨 Inconsistent Nodes:")
	}
	for n := range c.Nodes {
		fmt.Printf("\t👉️ %v\tpath: %s\n", c.Nodes[n], n)
	}
	if len(c.LinkedNodes) != 0 {
		fmt.Println("\n🚨 Inconsistent Links:")
	}
	for l := range c.LinkedNodes {
		fmt.Printf("\t👉️ %v\tpath: %s\n\t\t\t\tmissing node:%s\n", c.LinkedNodes[l], c.nodeToLink[l], l)
	}
	if len(c.Blobs) != 0 {
		fmt.Println("\n🚨 Inconsistent Blobs:")
	}
	for b := range c.Blobs {
		fmt.Printf("\t👉️ %v\tblob: %s\n", c.Blobs[b], b)
	}
	if len(c.BlobReferences) != 0 {
		fmt.Println("\n🚨 Inconsistent BlobReferences:")
	}
	for b := range c.BlobReferences {
		fmt.Printf("\t👉️ %v\tblob: %s\n\t\t\t\treferencing node:%s\n", c.BlobReferences[b], b, c.blobToNode[b])
	}
	if c.resolved > 0 {
		fmt.Printf("\nℹ️  %d finding(s) resolved on re-check: the storage changed during the scan. For a reliable result run on a quiet system or on a filesystem snapshot.\n", c.resolved)
	}
	if c.inFlight > 0 {
		fmt.Printf("\nℹ️  %d node(s) still processing an upload, their blobs are skipped.\n", c.inFlight)
	}
	if len(c.Nodes) == 0 && len(c.LinkedNodes) == 0 && len(c.Blobs) == 0 && len(c.BlobReferences) == 0 {
		fmt.Printf("💚 No inconsistency found. The backup in '%s' seems to be valid.\n", discpath)
	} else if fail {
		os.Exit(1)
	}
	return nil

}

// rechecker checks if a finding is resolved by now. All methods are safe to call on a nil rechecker.
type rechecker struct {
	localPrefix string
	fsys        fs.FS
	lbs         ListBlobstore

	// the blobs are listed at most once and only if a blob outside of the storagepath needs a re-check
	hasListed bool
	blobs     map[string]struct{}
}

func (r *rechecker) blobExists(path string) bool {
	if r == nil {
		return false
	}

	if strings.HasPrefix(path, r.localPrefix) {
		_, err := os.Stat(path)
		return err == nil
	}

	if r.lbs == nil {
		return false
	}
	if !r.hasListed {
		r.hasListed = true
		nodes, err := r.lbs.List()
		if err == nil {
			r.blobs = make(map[string]struct{}, len(nodes))
			for _, n := range nodes {
				r.blobs[r.lbs.Path(n)] = struct{}{}
			}
		}
	}
	_, ok := r.blobs[path]
	return ok
}

func (r *rechecker) nodeExists(path string) bool {
	if r == nil {
		return false
	}

	_, err := os.Lstat(path)
	return err == nil
}

func (r *rechecker) nodeInconsistencyResolved(nodePath string, inc Inconsistency) bool {
	if r == nil {
		return false
	}

	mpkPath := nodePath + ".mpk"
	switch inc {
	case InconsistencySymlinkMissing:
		return r.symlinkExists(nodePath)
	case InconsistencyFilesMissing:
		_, nodeErr := os.Stat(nodePath)
		_, mpkErr := os.Stat(mpkPath)
		return nodeErr == nil && mpkErr == nil
	case InconsistencyMalformedFile:
		_, err := r.readMetadata(nodePath)
		return err == nil
	default:
		return false
	}
}

// symlinkExists checks if the symlink of a child node exists in its parent
func (r *rechecker) symlinkExists(nodePath string) bool {
	if _trashRegex.MatchString(nodePath) {
		return false
	}

	m, err := r.readMetadata(nodePath)
	if err != nil {
		return false
	}
	parentID := string(m[prefixes.ParentidAttr])
	name := string(m[prefixes.NameAttr])
	if parentID == "" || name == "" {
		return false
	}

	const (
		nodesDir     = "/nodes/"
		pathifyDepth = 4
		pathifyWidth = 2
	)
	idx := strings.LastIndex(nodePath, nodesDir)
	if idx < 0 {
		return false
	}

	_, err = os.Lstat(filepath.Join(nodePath[:idx+len(nodesDir)], lookup.Pathify(parentID, pathifyDepth, pathifyWidth), name))
	return err == nil
}

// nodeProcessing checks if the node still processes an upload, meaning its blob may not be written yet.
// A processing marker without an upload session is stale and does not count.
func (r *rechecker) nodeProcessing(nodePath string) bool {
	if r == nil {
		return false
	}

	m, err := r.readMetadata(nodePath)
	if err != nil {
		return false
	}
	id, ok := strings.CutPrefix(string(m[prefixes.StatusPrefix]), node.ProcessingStatus)
	if !ok || id == "" {
		return false
	}

	_, err = os.Stat(filepath.Join(r.localPrefix, "uploads", id+".info"))
	return err == nil
}

// readMetadata reads the metadata of a node below the storage root
func (r *rechecker) readMetadata(nodePath string) (map[string][]byte, error) {
	b, err := fs.ReadFile(r.fsys, strings.TrimPrefix(nodePath, r.localPrefix)+".mpk")
	if err != nil {
		return nil, err
	}

	m := map[string][]byte{}
	return m, msgpack.Unmarshal(b, &m)
}

func deleteInconsistency(incs map[string][]Inconsistency, path string) {
	if len(incs[path]) == 0 {
		delete(incs, path)
	}
}
