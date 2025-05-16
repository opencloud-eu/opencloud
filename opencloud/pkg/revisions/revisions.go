// Package revisions allows manipulating revisions in a storage provider.
package revisions

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/vmihailenco/msgpack/v5"
)

var (
	// regex to determine if a node versioned. Examples:
	// 9113a718-8285-4b32-9042-f930f1a58ac2.REV.2024-05-22T07:32:53.89969726Z
	// 9113a718-8285-4b32-9042-f930f1a58ac2.REV.2024-05-22T07:32:53.89969726Z.mpk
	// 9113a718-8285-4b32-9042-f930f1a58ac2.REV.2024-05-22T07:32:53.89969726Z.mlock
	_versionRegex = regexp.MustCompile(`\.REV\.[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]+Z*`)
)

// DelBlobstore is the interface for a blobstore that can delete blobs.
type DelBlobstore interface {
	Delete(node *node.Node) error
}

// Glob uses globbing to find all revision nodes in a storage provider.
func Glob(pattern string) <-chan string {
	ch := make(chan string)
	go func() {
		defer close(ch)
		nodes, err := filepath.Glob(filepath.Join(pattern))
		if err != nil {
			fmt.Println("error globbing", pattern, err)
			return
		}

		if len(nodes) == 0 {
			fmt.Println("no nodes found. Double check storage path")
			return
		}

		for _, n := range nodes {
			if _versionRegex.MatchString(n) {
				ch <- n
			}
		}
	}()

	return ch
}

// GlobWorkers uses multiple go routine to glob all revision nodes in a storage provider.
func GlobWorkers(pattern string, depth string, remainder string) <-chan string {
	wg := sync.WaitGroup{}
	ch := make(chan string)
	go func() {
		defer close(ch)
		nodes, err := filepath.Glob(pattern + depth)
		if err != nil {
			fmt.Println("error globbing", pattern, err)
			return
		}

		if len(nodes) == 0 {
			fmt.Println("no nodes found. Double check storage path")
			return
		}

		for _, node := range nodes {
			wg.Add(1)
			go func(node string) {
				defer wg.Done()
				nodes, err := filepath.Glob(node + remainder)
				if err != nil {
					fmt.Println("error globbing", node, err)
					return
				}
				for _, n := range nodes {
					if _versionRegex.MatchString(n) {
						ch <- n
					}
				}
			}(node)
		}

		wg.Wait()
	}()

	return ch
}

// Walk walks the storage provider to find all revision nodes.
func Walk(base string) <-chan string {
	ch := make(chan string)
	go func() {
		defer close(ch)
		err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Println("error walking", base, err)
				return err
			}

			if !_versionRegex.MatchString(info.Name()) {
				return nil
			}

			ch <- path
			return nil
		})
		if err != nil {
			fmt.Println("error walking", base, err)
			return
		}

	}()
	return ch
}

// List uses directory listing to find all revision nodes in a storage provider.
func List(base string, workers int) <-chan string {
	ch := make(chan string)
	go func() {
		defer close(ch)
		if err := listFolder(base, ch, make(chan struct{}, workers)); err != nil {
			fmt.Println("error listing", base, err)
			return
		}
	}()

	return ch
}

// PurgeRevisions removes all revisions from a storage provider.
func PurgeRevisions(nodes <-chan string, bs DelBlobstore, dryRun, verbose bool) (int, int, int) {
	countFiles := 0
	countBlobs := 0
	countRevisions := 0

	var err error
	for d := range nodes {
		if !_versionRegex.MatchString(d) {
			continue
		}

		var blobID string
		e := filepath.Ext(d)
		switch e {
		case ".mpk":
			blobID, err = getBlobID(d)
			if err != nil {
				fmt.Printf("error getting blobID from %s: %v\n", d, err)
				continue
			}

			countBlobs++
		case ".mlock":
			// no extra action on .mlock files
		default:
			countRevisions++
		}

		if !dryRun {
			if blobID != "" {
				//  TODO: needs spaceID for decomposeds3
				if err := bs.Delete(&node.Node{BlobID: blobID}); err != nil {
					fmt.Printf("error deleting blob %s: %v\n", blobID, err)
					continue
				}
			}

			if err := os.Remove(d); err != nil {
				fmt.Printf("error removing %s: %v\n", d, err)
				continue
			}

		}

		countFiles++

		if verbose {
			spaceID, nodeID := getIDsFromPath(d)
			if dryRun {
				fmt.Println("Would delete")
				fmt.Println("\tResourceID:", spaceID+"!"+nodeID)
				fmt.Println("\tSpaceID:", spaceID)
				fmt.Println("\tPath:", d)
				if blobID != "" {
					fmt.Println("\tBlob:", blobID)
				}
			} else {
				fmt.Println("Deleted")
				fmt.Println("\tResourceID:", spaceID+"!"+nodeID)
				fmt.Println("\tSpaceID:", spaceID)
				fmt.Println("\tPath:", d)
				if blobID != "" {
					fmt.Println("\tBlob:", blobID)
				}
			}
		}
	}

	return countFiles, countBlobs, countRevisions
}

func listFolder(path string, ch chan<- string, workers chan struct{}) error {
	workers <- struct{}{}
	wg := sync.WaitGroup{}

	children, err := os.ReadDir(path)
	if err != nil {
		<-workers
		return err
	}

	for _, child := range children {
		if child.IsDir() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := listFolder(filepath.Join(path, child.Name()), ch, workers); err != nil {
					fmt.Println("error listing", path, err)
				}
			}()
		}

		if _versionRegex.MatchString(child.Name()) {
			ch <- filepath.Join(path, child.Name())
		}

	}
	<-workers
	wg.Wait()
	return nil
}

func getBlobID(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	m := map[string][]byte{}
	if err := msgpack.Unmarshal(b, &m); err != nil {
		return "", err
	}

	if bid := m["user.oc.blobid"]; string(bid) != "" {
		return string(bid), nil
	}

	return "", nil
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

	n := strings.Split(rawIDs[1], ".REV.")
	if len(n) != 2 {
		return "", ""
	}

	spaceID := strings.Replace(s[1], "/", "", -1)
	nodeID := strings.Replace(n[0], "/", "", -1)
	return spaceID, filepath.Base(nodeID)
}
