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

	"github.com/opencloud-eu/opencloud/opencloud/pkg/register"
	"github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/pkg/x/path/filepathx"
	storageUsersConfig "github.com/opencloud-eu/opencloud/services/storage-users/pkg/config"
	storageUsersParser "github.com/opencloud-eu/opencloud/services/storage-users/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/event"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/revaconfig"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/ignore"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/options"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/registry"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"

	"github.com/pkg/xattr"
	"github.com/spf13/cobra"
	"github.com/vmihailenco/msgpack/v5"
)

var (
	restartRequired      = false
	recalculateChecksums = false
	ignorer              *ignore.Ignorer
)

type IDCacher interface {
	WarmupIDCache(root string, assimilate, onlyDirty bool) error
}

// EntryInfo holds information about a directory entry.
type EntryInfo struct {
	Path     string
	ModTime  time.Time
	ParentID string
}

// PosixfsCommand is the entrypoint for the posixfs command.
func PosixfsCommand(cfg *config.Config) *cobra.Command {
	posixCmd := &cobra.Command{
		Use:     "posixfs",
		Short:   `cli tools to inspect and manipulate a posixfs storage.`,
		GroupID: CommandGroupStorage,
	}

	posixCmd.AddCommand(consistencyCmd(cfg))
	posixCmd.AddCommand(scanCmd(cfg))

	return posixCmd
}

func init() {
	register.AddCommand(PosixfsCommand)
}

// scanCmd performs a posixfs id cache warmup scan
func scanCmd(ocCfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Perform a filesystem scan and update the ID and filemetadata cache",
		PreRunE: func(cmd *cobra.Command, args []string) error {

			if err := parser.ParseConfig(ocCfg, true); err != nil {
				return configlog.ReturnError(err)
			}

			// Parse storage users config
			ocCfg.StorageUsers.Commons = ocCfg.Commons

			return configlog.ReturnFatal(storageUsersParser.ParseConfig(ocCfg.StorageUsers))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := ocCfg.StorageUsers
			if cfg.Driver != "posix" {
				fmt.Fprintf(os.Stderr, "This command is only available when using the 'posix' driver. Current driver: '%s'\n", cfg.Driver)
				os.Exit(1)
			}

			storageRoot := cfg.Drivers.Posix.Root
			root := storageRoot
			defaultRoot := true
			if v, err := cmd.Flags().GetString("basepath"); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to parse command-line parameter '--basepath': %v\n", err)
				os.Exit(1)
			} else if v != "" {
				root = v
				if !filepath.IsAbs(v) {
					if v, err = filepath.Abs(v); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to make the basepath mentioned using '--basepath' absolute: %v\n", err)
						os.Exit(1)
					} else {
						root = v
					}
				} else {
					root = v
				}
				root = filepath.Clean(root)
				defaultRoot = false
			}

			// ensure that, if a basepath has been indicated, it is under the storage root
			if !defaultRoot {
				if contained, err := filepathx.IsSameOrContainedBy(storageRoot, root); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to determine whether the specified basepath %q is contained by the storage root %q: %v\n", root, storageRoot, err)
					os.Exit(1)
				} else if !contained {
					fmt.Fprintf(os.Stderr, "The specified basepath %q is neither the storage root %q, nor a subdirectory thereof, nor a file underneath it\n", root, storageRoot)
					os.Exit(1)
				}
			}

			// We want to initialize the driver but disable scanfs on boot, so we can trigger it manually afterwards
			drivers := revaconfig.StorageProviderDrivers(cfg)
			drivers["posix"] = revaconfig.Posix(cfg, false, false)

			var fsStream events.Stream
			var err error
			fsStream, err = event.NewStream(cfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create event stream for posix driver: %v\n", err)
				os.Exit(1)
			}
			log := logger("posixfs")

			if !defaultRoot {
				log = log.With().Str("basepath", root).Logger()
			}

			f, ok := registry.NewFuncs["posix"]
			if !ok {
				fmt.Fprintf(os.Stderr, "posix driver not found in registry\n")
				os.Exit(1)
			}

			fs, err := f(drivers["posix"].(map[string]any), fsStream, &log)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to initialize filesystem driver '%s': %v\n", cfg.Driver, err)
				return err
			}

			cacher, ok := fs.(IDCacher)
			if !ok {
				fmt.Fprintf(os.Stderr, "The posix driver does not expose WarmupIDCache.\n")
				os.Exit(1)
			}

			if defaultRoot {
				fmt.Println("Starting posixfs scan...")
			} else {
				fmt.Printf("Starting posixfs scan at '%s'...\n", root)
			}
			err = cacher.WarmupIDCache(root, true, false)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Scan failed: %v\n", err)
				return err
			}

			fmt.Println("Scan completed successfully.")
			return nil
		},
	}
	cmd.Flags().StringP("basepath", "p", "", "the root under which to scan files, which may be a directory or a file (when omitted, detaults to using the storage root)")
	return cmd
}

// consistencyCmd returns a command to check the consistency of the posixfs storage.
func consistencyCmd(ocCfg *config.Config) *cobra.Command {
	consCmd := &cobra.Command{
		Use:   "consistency [path ...]",
		Short: "Check the consistency of the posixfs storage",
		Long: `Check the consistency of the posixfs storage.

You can specify one or more paths to limit the scope of the check.
If no path is provided, the whole storage is checked.

The provided arguments determines the scope of the check:
  - a storage root:   the whole storage (all personal and project spaces) is checked
  - a space root:     only that space is checked
  - a file or folder: only that single entity is checked (and its children, if it is a folder)`,
		Args: cobra.ArbitraryArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {

			if err := parser.ParseConfig(ocCfg, true); err != nil {
				return configlog.ReturnError(err)
			}

			// Parse storage users config
			ocCfg.StorageUsers.Commons = ocCfg.Commons

			return configlog.ReturnFatal(storageUsersParser.ParseConfig(ocCfg.StorageUsers))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := ocCfg.StorageUsers
			return checkPosixfsConsistency(cfg, cmd, args)
		},
	}
	consCmd.Flags().Bool("fix-checksums", false, "Recalculate and fix the file checksums. This reads every file and can be slow on large storages.")

	return consCmd
}

// checkPosixfsConsistency checks the consistency of the posixfs storage. The
// given path determines the scope of the check: the whole storage, a single
// space or a single entity within a space.
func checkPosixfsConsistency(cfg *storageUsersConfig.Config, cmd *cobra.Command, paths []string) error {
	if len(paths) == 0 {
		paths = []string{cfg.Drivers.Posix.Root}
	}
	log := logger("posixfs")
	recalculateChecksums, _ = cmd.Flags().GetBool("fix-checksums")

	drivers := revaconfig.StorageProviderDrivers(cfg)
	drivers["posix"] = revaconfig.Posix(cfg, false, false)
	opts, err := options.New(drivers["posix"].(map[string]any))
	if err != nil {
		return err
	}
	ignorer = ignore.NewIgnorer(opts, &log)

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
			checkSpaces(filepath.Join(path, "users"))

			fmt.Println("Checking project spaces...")
			checkSpaces(filepath.Join(path, "projects"))
		case isSpaceRoot(path):
			fmt.Printf("Checking space '%s'...\n", path)
			checkSpace(path)
		case contained:
			fmt.Printf("Checking '%s'...\n", path)
			checkEntity(path)
		default:
			return fmt.Errorf("the provided path '%s' is neither a space root nor contained by the storage root '%s'", path, rootPath)
		}
	}

	if restartRequired {
		fmt.Println("\n\n  ⚠️  Please restart your openCloud instance to apply changes.")
	}
	return nil
}

// findStorageRoot walks up the directory tree starting at path until it finds a
// directory that contains an "indexes" subdirectory which marks the root of a
// posixfs storage. A user folder inside a space might also be named "indexes",
// so to disambiguate we require that the "indexes" directory is an internal
// directory: the storage's own indexes directory is skipped during assimilation
// and therefore never receives a node ID attribute, whereas a regular user
// folder would have one.
func findStorageRoot(path string) (string, error) {
	current := path
	for {
		indexesPath := filepath.Join(current, "indexes")
		if info, err := os.Stat(indexesPath); err == nil && info.IsDir() {
			if id, err := xattr.Get(indexesPath, prefixes.IDAttr); err != nil || len(id) == 0 {
				return current, nil
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("'%s' does not appear to be inside a posixfs storage (no 'indexes' directory found)", path)
		}
		current = parent
	}
}

// isSpaceRoot reports whether the given path is a space root, which is
// identified by the presence of the space ID attribute.
func isSpaceRoot(path string) bool {
	spaceID, err := xattr.Get(path, prefixes.SpaceIDAttr)
	return err == nil && len(spaceID) > 0
}

func checkSpaces(basePath string) {
	dirEntries, err := os.ReadDir(basePath)
	if err != nil {
		logFailure("Error reading spaces directory '%s': %v", basePath, err)
		return
	}

	for _, entry := range dirEntries {
		if entry.IsDir() {
			fullPath := filepath.Join(basePath, entry.Name())
			checkSpace(fullPath)
		}
	}
}

func checkSpace(spacePath string) {
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

	checkSpaceID(spacePath)
	checkNodes(spacePath)
}

func checkSpaceID(spacePath string) {
	entries, uniqueIDs, oldestEntry, err := gatherAttributes(spacePath)
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
		restartRequired = true

		obsoleteIDs := []string{}
		for id := range uniqueIDs {
			if id != targetID {
				obsoleteIDs = append(obsoleteIDs, id)
			}
		}
		fixSpaceID(spacePath, obsoleteIDs, targetID, entries)
	}
}

func walkNodes(dir string, parentID string) int {
	fixes := 0
	entries, err := os.ReadDir(dir)
	if err != nil {
		logFailure("Error reading directory '%s': %v", dir, err)
		return 0
	}

	for _, entry := range entries {
		fullPath := filepath.Join(dir, entry.Name())

		if ignorer.IsIgnored(fullPath) {
			continue
		}

		fixes += checkNodeAttributes(fullPath, entry.Name(), parentID, entry.IsDir())

		if entry.IsDir() {
			nodeID, err := xattr.Get(fullPath, prefixes.IDAttr)
			if err != nil || len(nodeID) == 0 {
				logFailure("Directory '%s' missing '%s', skipping its children", fullPath, prefixes.IDAttr)
				continue
			}
			fixes += walkNodes(fullPath, string(nodeID))
		}
	}
	return fixes
}

// checkNodeAttributes checks and fixes the parent ID and name attributes of a
// single node. For files it additionally checks the blobsize and, when
// requested, the checksums. It returns the number of fixes applied.
func checkNodeAttributes(path, name, parentID string, isDir bool) int {
	fixes := 0

	// Check if the parent ID attribute matches the expected parent ID, if not, fix it.
	actualParentID, err := xattr.Get(path, prefixes.ParentidAttr)
	if err != nil || string(actualParentID) != parentID {
		if err := xattr.Set(path, prefixes.ParentidAttr, []byte(parentID)); err != nil {
			logFailure("Failed to fix parent ID for '%s': %v", path, err)
		} else {
			fmt.Printf("  + Fixed parent ID for '%s'\n", path)
			fixes++
			restartRequired = true
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
			restartRequired = true
		}
	}

	if !isDir {
		fixes += checkBlobsize(path)
		if recalculateChecksums {
			fixes += fixChecksums(path)
		}
	}

	return fixes
}

// checkEntity checks a single file or folder within a space, including its own
// parent ID, name and (for files) blobsize/checksums. If the entity is a folder
// its children are checked recursively.
func checkEntity(path string) {
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

	fixes := checkNodeAttributes(path, info.Name(), string(parentID), info.IsDir())

	if info.IsDir() {
		nodeID, err := xattr.Get(path, prefixes.IDAttr)
		if err != nil || len(nodeID) == 0 {
			logFailure("Directory '%s' missing '%s' attribute", path, prefixes.IDAttr)
		} else {
			fixes += walkNodes(path, string(nodeID))
		}
	}

	if fixes > 0 {
		fmt.Printf("  ✓ Fixed %d incorrect node attributes for %s\n", fixes, filepath.Base(path))
	}
}

// checkBlobsize verifies that the stored blobsize attribute matches the actual
// file size and fixes it if it doesn't. It returns the number of fixes applied.
func checkBlobsize(path string) int {
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
	restartRequired = true
	return 1
}

// fixChecksums recalculates the sha1, md5 and adler32 checksums of the file and
// updates the stored attributes if they differ. It returns the number of fixes applied.
func fixChecksums(path string) int {
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
		restartRequired = true
		fixes++
	}
	return fixes
}

func checkNodes(spacePath string) {
	rootID, err := xattr.Get(spacePath, prefixes.IDAttr)
	if err != nil || len(rootID) == 0 {
		logFailure("Space root '%s' missing '%s' attribute", spacePath, prefixes.IDAttr)
		return
	}

	fixes := walkNodes(spacePath, string(rootID))

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

func gatherAttributes(path string) ([]EntryInfo, map[string]struct{}, EntryInfo, error) {
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
		if ignorer.IsIgnored(fullPath) {
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

func logFailure(message string, args ...any) {
	fmt.Fprintf(os.Stderr, message+"\n", args...)
}
