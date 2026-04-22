// Package backup contains OpenCloud backup functionality.
package backup

import (
	"fmt"
	"os"
	"regexp"
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

// CheckProviderConsistency checks the consistency of a space
func CheckProviderConsistency(storagepath string, lbs ListBlobstore, fail bool) error {
	fsys := os.DirFS(storagepath)

	p := NewProvider(fsys, storagepath, lbs)
	if err := p.ProduceData(); err != nil {
		return err
	}

	c := NewConsistency()
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

	for n := range c.Nodes {
		if len(c.Nodes[n]) == 0 {
			c.Nodes[n] = append(c.Nodes[n], InconsistencySymlinkMissing)
		}
	}
	for l := range c.LinkedNodes {
		c.LinkedNodes[l] = append(c.LinkedNodes[l], InconsistencyNodeMissing)
	}
	for b := range c.Blobs {
		c.Blobs[b] = append(c.Blobs[b], InconsistencyBlobOrphaned)
	}
	for b := range c.BlobReferences {
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
	if len(c.Nodes) == 0 && len(c.LinkedNodes) == 0 && len(c.Blobs) == 0 && len(c.BlobReferences) == 0 {
		fmt.Printf("💚 No inconsistency found. The backup in '%s' seems to be valid.\n", discpath)
	} else if fail {
		os.Exit(1)
	}
	return nil

}

func deleteInconsistency(incs map[string][]Inconsistency, path string) {
	if len(incs[path]) == 0 {
		delete(incs, path)
	}
}
