// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/x/path/filepathx"
	storageUsersConfig "github.com/opencloud-eu/opencloud/services/storage-users/pkg/config"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/ignore"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/pkg/xattr"
	"github.com/shamaton/msgpack/v2"
)

type consistencyChecker struct {
	cfg                  *storageUsersConfig.Config
	ignorer              *ignore.Ignorer
	recalculateChecksums bool

	restartRequired bool
}

// checkPosixfsConsistency checks the consistency of the posixfs storage. The
// given path determines the scope of the check: the whole storage, a single
// space or a single entity within a space.
func (c *consistencyChecker) Check(paths []string) error {
	for _, path := range paths {
		rootPath, err := findStorageRoot(path)
		if err != nil {
			return err
		}
		path = filepath.Clean(path)
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("error accessing '%s': %w", path, err)
		}
		contained, _ := filepathx.IsSameOrContainedBy(rootPath, path)

		switch {
		case path == rootPath:
			fmt.Println("Checking personal spaces...")
			c.checkSpaces(filepath.Join(path, "users"))

			fmt.Println("Checking project spaces...")
			c.checkSpaces(filepath.Join(path, "projects"))
		case isSpaceRoot(path):
			fmt.Printf("Checking space '%s'...\n", path)
			c.checkSpace(path)
		case contained:
			fmt.Printf("Checking '%s'...\n", path)
			c.checkEntity(path)
		default:
			return fmt.Errorf("the provided path '%s' is neither a space root nor contained by the storage root '%s'", path, rootPath)
		}
	}

	if c.restartRequired {
		fmt.Println("\n\n  ⚠️  Please restart your openCloud instance to apply changes.")
	}
	return nil
}

func (c *consistencyChecker) checkSpaces(basePath string) {
	dirEntries, err := os.ReadDir(basePath)
	if err != nil {
		logFailure("Error reading spaces directory '%s': %v", basePath, err)
		return
	}

	for _, entry := range dirEntries {
		if entry.IsDir() {
			fullPath := filepath.Join(basePath, entry.Name())
			c.checkSpace(fullPath)
		}
	}
}

func (c *consistencyChecker) checkSpace(spacePath string) {
	info, err := os.Stat(spacePath)
	if err != nil {
		logFailure("Error accessing path '%s': %v", spacePath, err)
		return
	}
	if !info.IsDir() {
		logFailure("Error: The provided path '%s' is not a directory\n", spacePath)
		return
	}

	spaceID, err := xattr.Get(spacePath, prefixes.SpaceIDAttr)
	if err != nil || len(spaceID) == 0 {
		logFailure("Error: The directory '%s' does not seem to be a space root, it's missing the '%s' attribute\n", spacePath, prefixes.SpaceIDAttr)
		return
	}

	c.checkSpaceID(spacePath)
	c.checkNodes(spacePath)
}

func (c *consistencyChecker) checkSpaceID(spacePath string) {
	entries, uniqueIDs, oldestEntry, err := c.gatherAttributes(spacePath)
	if err != nil {
		logFailure("Failed to gather attributes: %v", err)
		return
	}

	if len(entries) == 0 {
		return
	}

	if len(uniqueIDs) > 1 {
		fmt.Println("\n  ⚠ Multiple space IDs found:")
		for id := range uniqueIDs {
			fmt.Printf("    - %s\n", id)
		}

		fmt.Printf("\n  ⏳ Oldest entry is '%s' (modified on %s).\n",
			filepath.Base(oldestEntry.Path), oldestEntry.ModTime.Format(time.RFC1123))

		targetID := oldestEntry.ParentID
		fmt.Printf("  ✅ Proposed target Parent ID: %s\n", targetID)

		fmt.Printf("\n  Do you want to unify all parent IDs to '%s'? This will modify %d entries, the directory, and the user index. (y/N): ", targetID, len(entries))

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input != "y" {
			logFailure("Operation cancelled by user.")
			return
		}
		c.restartRequired = true

		obsoleteIDs := []string{}
		for id := range uniqueIDs {
			if id != targetID {
				obsoleteIDs = append(obsoleteIDs, id)
			}
		}
		fixSpaceID(spacePath, obsoleteIDs, targetID, entries)
	}
}

func (c *consistencyChecker) walkNodes(dir string, parentID string) int {
	fixes := 0
	entries, err := os.ReadDir(dir)
	if err != nil {
		logFailure("Error reading directory '%s': %v", dir, err)
		return 0
	}

	for _, entry := range entries {
		fullPath := filepath.Join(dir, entry.Name())

		if c.ignorer.IsIgnored(fullPath) {
			continue
		}

		fixes += c.checkNodeAttributes(fullPath, entry.Name(), parentID, entry.IsDir())

		if entry.IsDir() {
			nodeID, err := xattr.Get(fullPath, prefixes.IDAttr)
			if err != nil || len(nodeID) == 0 {
				logFailure("Directory '%s' missing '%s', skipping its children", fullPath, prefixes.IDAttr)
				continue
			}
			fixes += c.walkNodes(fullPath, string(nodeID))
		}
	}
	return fixes
}

// checkNodeAttributes checks and fixes the parent ID and name attributes of a
// single node. For files it additionally checks the blobsize and, when
// requested, the checksums. It returns the number of fixes applied.
func (c *consistencyChecker) checkNodeAttributes(path, name, parentID string, isDir bool) int {
	fixes := 0

	// Check if the parent ID attribute matches the expected parent ID, if not, fix it.
	actualParentID, err := xattr.Get(path, prefixes.ParentidAttr)
	if err != nil || string(actualParentID) != parentID {
		if err := xattr.Set(path, prefixes.ParentidAttr, []byte(parentID)); err != nil {
			logFailure("Failed to fix parent ID for '%s': %v", path, err)
		} else {
			fmt.Printf("  + Fixed parent ID for '%s'\n", path)
			fixes++
			c.restartRequired = true
		}
	}

	// Check that the name attribute matches the actual name of the file/directory, if not, fix it.
	nameAttr, err := xattr.Get(path, prefixes.NameAttr)
	if err != nil || string(nameAttr) != name {
		if err := xattr.Set(path, prefixes.NameAttr, []byte(name)); err != nil {
			logFailure("Failed to fix name attribute for '%s': %v", path, err)
		} else {
			fmt.Printf("  + Fixed name attribute for '%s'\n", path)
			fixes++
			c.restartRequired = true
		}
	}

	if !isDir {
		fixes += c.checkBlobsize(path)
		if c.recalculateChecksums {
			fixes += c.fixChecksums(path)
		}
	}

	return fixes
}

// checkEntity checks a single file or directory within a space, including its own
// parent ID, name and (for files) blobsize/checksums. If the entity is a directory
// its children are checked recursively.
func (c *consistencyChecker) checkEntity(path string) {
	info, err := os.Stat(path)
	if err != nil {
		logFailure("Error accessing path '%s': %v", path, err)
		return
	}

	// The expected parent ID is the ID attribute of the containing directory.
	parentDir := filepath.Dir(path)
	parentID, err := xattr.Get(parentDir, prefixes.IDAttr)
	if err != nil || len(parentID) == 0 {
		logFailure("Parent directory '%s' is missing the '%s' attribute", parentDir, prefixes.IDAttr)
		return
	}

	fixes := c.checkNodeAttributes(path, info.Name(), string(parentID), info.IsDir())

	if info.IsDir() {
		nodeID, err := xattr.Get(path, prefixes.IDAttr)
		if err != nil || len(nodeID) == 0 {
			logFailure("Directory '%s' missing '%s' attribute", path, prefixes.IDAttr)
		} else {
			fixes += c.walkNodes(path, string(nodeID))
		}
	}

	if fixes > 0 {
		fmt.Printf("  ✓ Fixed %d incorrect node attributes for %s\n", fixes, filepath.Base(path))
	}
}

// checkBlobsize verifies that the stored blobsize attribute matches the actual
// file size and fixes it if it doesn't. It returns the number of fixes applied.
func (c *consistencyChecker) checkBlobsize(path string) int {
	info, err := os.Stat(path)
	if err != nil {
		logFailure("Error accessing file '%s': %v", path, err)
		return 0
	}

	expectedSize := strconv.FormatInt(info.Size(), 10)
	blobsize, err := xattr.Get(path, prefixes.BlobsizeAttr)
	if err == nil && string(blobsize) == expectedSize {
		return 0
	}

	if err := xattr.Set(path, prefixes.BlobsizeAttr, []byte(expectedSize)); err != nil {
		logFailure("Failed to fix blobsize for '%s': %v", path, err)
		return 0
	}

	fmt.Printf("  + Fixed blobsize for '%s'\n", path)
	c.restartRequired = true
	return 1
}

// fixChecksums recalculates the sha1, md5 and adler32 checksums of the file and
// updates the stored attributes if they differ. It returns the number of fixes applied.
func (c *consistencyChecker) fixChecksums(path string) int {
	sha1h, md5h, adler32h, err := node.CalculateChecksums(context.Background(), path)
	if err != nil {
		logFailure("Failed to calculate checksums for '%s': %v", path, err)
		return 0
	}

	checksums := map[string][]byte{
		prefixes.ChecksumPrefix + "sha1":    sha1h.Sum(nil),
		prefixes.ChecksumPrefix + "md5":     md5h.Sum(nil),
		prefixes.ChecksumPrefix + "adler32": adler32h.Sum(nil),
	}

	fixes := 0
	for attrName, sum := range checksums {
		current, err := xattr.Get(path, attrName)
		if err == nil && bytes.Equal(current, sum) {
			continue
		}
		if err := xattr.Set(path, attrName, sum); err != nil {
			logFailure("Failed to fix checksum '%s' for '%s': %v", attrName, path, err)
			continue
		}
		fmt.Printf("  + Fixed checksum '%s' for '%s'\n", attrName, path)
		c.restartRequired = true
		fixes++
	}
	return fixes
}

func (c *consistencyChecker) checkNodes(spacePath string) {
	rootID, err := xattr.Get(spacePath, prefixes.IDAttr)
	if err != nil || len(rootID) == 0 {
		logFailure("Space root '%s' missing '%s' attribute", spacePath, prefixes.IDAttr)
		return
	}

	fixes := c.walkNodes(spacePath, string(rootID))

	if fixes > 0 {
		fmt.Printf("  ✓ Fixed %d incorrect node attributes in %s\n", fixes, filepath.Base(spacePath))
	}
}

func fixSpaceID(spacePath string, obsoleteIDs []string, targetID string, entries []EntryInfo) {
	// Set all parentid attributes to the proper space ID
	err := setAllParentIDAttributes(entries, targetID)
	if err != nil {
		logFailure("an error occurred during file attribute update: %v", err)
		return
	}

	// Update space ID itself
	fmt.Printf("  Updating directory '%s' with attribute '%s' -> %s\n", filepath.Base(spacePath), prefixes.IDAttr, targetID)
	err = xattr.Set(spacePath, prefixes.IDAttr, []byte(targetID))
	if err != nil {
		logFailure("Failed to set attribute on directory '%s': %v", spacePath, err)
		return
	}
	err = xattr.Set(spacePath, prefixes.SpaceIDAttr, []byte(targetID))
	if err != nil {
		logFailure("Failed to set attribute on directory '%s': %v", spacePath, err)
		return
	}

	// update the index
	err = updateOwnerIndexFile(spacePath, obsoleteIDs)
	if err != nil {
		logFailure("Could not update the owner index file: %v", err)
	}
}

func (c *consistencyChecker) gatherAttributes(path string) ([]EntryInfo, map[string]struct{}, EntryInfo, error) {
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, nil, EntryInfo{}, fmt.Errorf("failed to read directory: %w", err)
	}

	var allEntries []EntryInfo
	uniqueIDs := make(map[string]struct{})
	var oldestEntry EntryInfo
	oldestTime := time.Now().Add(100 * 365 * 24 * time.Hour) // Set to a future date to find the oldest entry

	for _, entry := range dirEntries {
		fullPath := filepath.Join(path, entry.Name())
		if c.ignorer.IsIgnored(fullPath) {
			continue
		}
		info, err := os.Stat(fullPath)
		if err != nil {
			fmt.Printf("  - Warning: could not stat %s: %v\n", entry.Name(), err)
			continue
		}

		parentID, err := xattr.Get(fullPath, prefixes.ParentidAttr)
		if err != nil {
			continue // Skip if attribute doesn't exist or can't be read
		}

		entryInfo := EntryInfo{
			Path:     fullPath,
			ModTime:  info.ModTime(),
			ParentID: string(parentID),
		}

		allEntries = append(allEntries, entryInfo)
		uniqueIDs[string(parentID)] = struct{}{}

		if entryInfo.ModTime.Before(oldestTime) {
			oldestTime = entryInfo.ModTime
			oldestEntry = entryInfo
		}
	}

	return allEntries, uniqueIDs, oldestEntry, nil
}

func setAllParentIDAttributes(entries []EntryInfo, targetID string) error {
	fmt.Printf("  Setting all parent IDs to '%s':\n", targetID)

	for _, entry := range entries {
		if entry.ParentID == targetID {
			fmt.Printf("    - Skipping '%s' (already has target ID).\n", filepath.Base(entry.Path))
			continue
		}

		fmt.Printf("    - Removing all attributes from '%s'. It will be re-assimilated\n", filepath.Base(entry.Path))
		filepath.WalkDir(entry.Path, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return fmt.Errorf("error walking path '%s': %w", path, err)
			}

			// Remove all attributes from the file.
			if err := removeAttributes(path); err != nil {
				fmt.Printf("failed to remove attributes from '%s': %v", path, err)
			}
			return nil
		})
	}
	return nil
}

// updateOwnerIndexFile handles the logic of reading, modifying, and writing the MessagePack index file.
func updateOwnerIndexFile(basePath string, obsoleteIDs []string) error {
	fmt.Printf("  Rewriting index file '%s'\n", basePath)

	ownerID, err := xattr.Get(basePath, prefixes.OwnerIDAttr)
	if err != nil {
		return fmt.Errorf("could not get owner ID from oldest entry '%s' to find index: %w", basePath, err)
	}

	indexPath := filepath.Join(basePath, "../../indexes/by-user-id", string(ownerID)+".mpk")
	indexPath = filepath.Clean(indexPath)

	// Read the MessagePack file
	fileData, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("index file does not exist, skipping update")
		}
		return fmt.Errorf("could not read index file: %w", err)
	}
	var indexMap map[string]string
	if err := msgpack.Unmarshal(fileData, &indexMap); err != nil {
		return fmt.Errorf("failed to parse MessagePack index file (is it corrupt?): %w", err)
	}

	// Remove obsolete IDs from the map
	itemsRemoved := 0
	for _, id := range obsoleteIDs {
		if _, exists := indexMap[id]; exists {
			fmt.Printf("    - Removing obsolete ID '%s' from index.\n", id)
			delete(indexMap, id)
			itemsRemoved++
		} else {
			fmt.Printf("    - Obsolete ID '%s' not found in index\n", id)
		}
	}

	if itemsRemoved == 0 {
		return nil
	}

	// Write the data back to the file
	updatedData, err := msgpack.Marshal(&indexMap)
	if err != nil {
		return fmt.Errorf("failed to marshal updated index map: %w", err)
	}
	if err := os.WriteFile(indexPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write updated index file: %w", err)
	}

	fmt.Printf("  ✓ Successfully removed %d item(s) and saved index file.\n", itemsRemoved)
	return nil
}

func removeAttributes(path string) error {
	attrNames, err := xattr.List(path)
	if err != nil {
		return fmt.Errorf("failed to list attributes for '%s': %w", path, err)
	}

	for _, attrName := range attrNames {
		if err := xattr.Remove(path, attrName); err != nil {
			return fmt.Errorf("failed to remove attribute '%s' from '%s': %w", attrName, path, err)
		}
	}
	return nil
}
