package command

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/opencloud-eu/opencloud/opencloud/pkg/register"
	"github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/config/parser"
	oclog "github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/x/path/filepathx"
	storageUsersParser "github.com/opencloud-eu/opencloud/services/storage-users/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/event"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/revaconfig"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/ignore"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/options"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/registry"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/decomposedfs/spaceidindex"

	"github.com/pkg/xattr"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/theckman/yacspin"
	"github.com/vmihailenco/msgpack/v5"
)

// Define the names of the extended attributes we are working with.
const (
	parentIDAttrName = "user.oc.parentid"
	idAttrName       = "user.oc.id"
	nameAttrName     = "user.oc.name"
	spaceIDAttrName  = "user.oc.space.id"
	ownerIDAttrName  = "user.oc.owner.id"
)

var (
	spinner         *yacspin.Spinner
	restartRequired = false
	ignorer         *ignore.Ignorer
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
	posixCmd.AddCommand(indexCmd(cfg))
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
			log := oclog.NewLogger(
				oclog.Name("posixfs scan"),
				oclog.Level("error"),
				oclog.Pretty(true),
				oclog.Color(false)).Logger

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

func indexCmd(ocCfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Add spaces from the filesystem to the index",
		RunE: func(cmd *cobra.Command, args []string) error {
			return rebuildIndexes(cmd)
		},
	}
	cmd.Flags().StringP("root", "r", "", "Path to the root directory of the posixfs storage")
	cmd.Flags().StringSliceP("dirs", "d", []string{"users", "projects"}, "Subdirectories to scan for spaces")
	cmd.Flags().StringP("space", "s", "", "Only index a specific space by ID")
	_ = cmd.MarkFlagRequired("root")
	return cmd
}

func rebuildIndexes(cmd *cobra.Command) error {
	rootPath, _ := cmd.Flags().GetString("root")
	spaceFilter, _ := cmd.Flags().GetString("space")
	indexesDir := filepath.Join(rootPath, "indexes")

	if _, err := os.Stat(indexesDir); err != nil {
		return fmt.Errorf("'%s' is not a posixfs root: %w", rootPath, err)
	}

	userIndex := spaceidindex.New(indexesDir, "by-user-id")
	if err := userIndex.Init(); err != nil {
		return fmt.Errorf("failed to init by-user-id index: %w", err)
	}
	groupIndex := spaceidindex.New(indexesDir, "by-group-id")
	if err := groupIndex.Init(); err != nil {
		return fmt.Errorf("failed to init by-group-id index: %w", err)
	}
	typeIndex := spaceidindex.New(indexesDir, "by-type")
	if err := typeIndex.Init(); err != nil {
		return fmt.Errorf("failed to init by-type index: %w", err)
	}

	userEntries := map[string]map[string]string{}
	groupEntries := map[string]map[string]string{}
	typeEntries := map[string]map[string]string{}

	dirs, _ := cmd.Flags().GetStringSlice("dirs")
	for _, dir := range dirs {
		basePath := filepath.Join(rootPath, dir)
		entries, err := os.ReadDir(basePath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("failed to read '%s': %w", basePath, err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			spacePath := filepath.Join(basePath, entry.Name())

			spaceID, err := xattr.Get(spacePath, idAttrName)
			if err != nil || len(spaceID) == 0 {
				fmt.Fprintf(os.Stderr, "  - Skipping '%s': missing space ID\n", spacePath)
				continue
			}

			sid := string(spaceID)
			if spaceFilter != "" && sid != spaceFilter {
				continue
			}

			spaceType, err := xattr.Get(spacePath, "user.oc.space.type")
			if err != nil || len(spaceType) == 0 {
				fmt.Fprintf(os.Stderr, "  - Skipping '%s': missing space type\n", spacePath)
				continue
			}

			stype := string(spaceType)
			fmt.Printf("Collecting space '%s' (%s)\n", sid, stype)

			if typeEntries[stype] == nil {
				typeEntries[stype] = map[string]string{}
			}
			typeEntries[stype][sid] = sid

			ownerType, _ := xattr.Get(spacePath, "user.oc.owner.type")
			if string(ownerType) != "spaceowner" {
				ownerID, err := xattr.Get(spacePath, ownerIDAttrName)
				if err == nil && len(ownerID) > 0 {
					oid := string(ownerID)
					if userEntries[oid] == nil {
						userEntries[oid] = map[string]string{}
					}
					userEntries[oid][sid] = sid
				}
			}

			attrs, err := xattr.List(spacePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  - Failed to list xattrs for '%s': %v\n", spacePath, err)
				continue
			}
			for _, attr := range attrs {
				switch {
				case strings.HasPrefix(attr, "user.oc.grant.u:"):
					grantee := strings.TrimPrefix(attr, "user.oc.grant.u:")
					if userEntries[grantee] == nil {
						userEntries[grantee] = map[string]string{}
					}
					userEntries[grantee][sid] = sid
				case strings.HasPrefix(attr, "user.oc.grant.g:"):
					grantee := strings.TrimPrefix(attr, "user.oc.grant.g:")
					if groupEntries[grantee] == nil {
						groupEntries[grantee] = map[string]string{}
					}
					groupEntries[grantee][sid] = sid
				}
			}
		}
	}

	fmt.Println("Writing indexes...")

	for spaceType, entries := range typeEntries {
		if err := typeIndex.AddAll(spaceType, entries); err != nil {
			fmt.Fprintf(os.Stderr, "  - Failed to write type index '%s': %v\n", spaceType, err)
		}
	}
	for userID, entries := range userEntries {
		if err := userIndex.AddAll(userID, entries); err != nil {
			fmt.Fprintf(os.Stderr, "  - Failed to write user index '%s': %v\n", userID, err)
		}
	}
	for groupID, entries := range groupEntries {
		if err := groupIndex.AddAll(groupID, entries); err != nil {
			fmt.Fprintf(os.Stderr, "  - Failed to write group index '%s': %v\n", groupID, err)
		}
	}

	fmt.Println("Done.")
	return nil
}

// consistencyCmd returns a command to check the consistency of the posixfs storage.
func consistencyCmd(cfg *config.Config) *cobra.Command {
	consCmd := &cobra.Command{
		Use:   "consistency",
		Short: "check the consistency of the posixfs storage",
		RunE: func(cmd *cobra.Command, args []string) error {
			return checkPosixfsConsistency(cmd, cfg)
		},
	}
	consCmd.Flags().StringP("root", "r", "", "Path to the root directory of the posixfs storage")
	_ = consCmd.MarkFlagRequired("root")

	return consCmd
}

// checkPosixfsConsistency checks the consistency of the posixfs storage.
func checkPosixfsConsistency(cmd *cobra.Command, cfg *config.Config) error {
	rootPath, _ := cmd.Flags().GetString("root")
	indexesPath := filepath.Join(rootPath, "indexes")

	opt, _ := options.New(map[string]interface{}{
		"root": rootPath,
	})
	log := zerolog.Nop()
	ignorer = ignore.NewIgnorer(opt, &log)

	_, err := os.Stat(indexesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("consistency check failed: '%s' is not a posixfs root", rootPath)
		}
		return fmt.Errorf("error accessing '%s': %w", indexesPath, err)
	}

	spinnerCfg := yacspin.Config{
		Frequency:         100 * time.Millisecond,
		CharSet:           yacspin.CharSets[11],
		StopCharacter:     "✓",
		StopColors:        []string{"fgGreen"},
		StopFailCharacter: "✗",
		StopFailColors:    []string{"fgRed"},
	}

	spinner, err = yacspin.New(spinnerCfg)
	err = spinner.Start()
	if err != nil {
		return fmt.Errorf("error creating spinner: %w", err)
	}

	checkSpaces(filepath.Join(rootPath, "users"))
	spinner.Suffix(" Personal spaces check ")
	spinner.StopMessage("completed\n")
	spinner.Stop()

	checkSpaces(filepath.Join(rootPath, "projects"))
	spinner.Suffix(" Project spaces check ")
	spinner.StopMessage("completed")
	spinner.Stop()

	if restartRequired {
		fmt.Println("\n\n  ⚠️  Please restart your openCloud instance to apply changes.")
	}
	return nil
}

func checkSpaces(basePath string) {
	dirEntries, err := os.ReadDir(basePath)
	if err != nil {
		spinner.Message(fmt.Sprintf("Error reading spaces directory '%s'\n", basePath))
		spinner.StopFail()
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
	spinner.Message("")
	spinner.Suffix(fmt.Sprintf(" Checking space '%s'", spacePath))

	info, err := os.Stat(spacePath)
	if err != nil {
		logFailure("Error accessing path '%s': %v", spacePath, err)
		return
	}
	if !info.IsDir() {
		logFailure("Error: The provided path '%s' is not a directory\n", spacePath)
		return
	}

	spaceID, err := xattr.Get(spacePath, spaceIDAttrName)
	if err != nil || len(spaceID) == 0 {
		logFailure("Error: The directory '%s' does not seem to be a space root, it's missing the '%s' attribute\n", spacePath, spaceIDAttrName)
		return
	}

	checkSpaceID(spacePath)
	checkNodeIDs(spacePath)
}

func checkSpaceID(spacePath string) {
	spinner.Message(" - checking space ID uniqueness")

	entries, uniqueIDs, oldestEntry, err := gatherAttributes(spacePath)
	if err != nil {
		logFailure("Failed to gather attributes: %v", err)
		return
	}

	if len(entries) == 0 {
		return
	}

	if len(uniqueIDs) > 1 {
		spinner.Pause()
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
			spinner.Unpause()
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
		spinner.Unpause()
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

		// Check if the parent ID attribute matches the expected parent ID, if not, fix it.
		actualParentID, err := xattr.Get(fullPath, parentIDAttrName)
		if err != nil || string(actualParentID) != parentID {
			err = xattr.Set(fullPath, parentIDAttrName, []byte(parentID))
			if err != nil {
				logFailure("Failed to fix parent ID for '%s': %v", fullPath, err)
			} else {
				spinner.Pause()
				fmt.Printf("  + Fixed parent ID for '%s'", fullPath)
				spinner.Unpause()
				fixes++
				restartRequired = true
			}
		}

		// Check that the name attribute matches the actual name of the file/directory, if not, fix it.
		nameAttr, err := xattr.Get(fullPath, nameAttrName)
		if err != nil || string(nameAttr) != entry.Name() {
			err = xattr.Set(fullPath, nameAttrName, []byte(entry.Name()))
			if err != nil {
				logFailure("Failed to fix name attribute for '%s': %v", fullPath, err)
			} else {
				spinner.Pause()
				fmt.Printf("  + Fixed name attribute for '%s'", fullPath)
				spinner.Unpause()
				fixes++
				restartRequired = true
			}
		}

		if entry.IsDir() {
			nodeID, err := xattr.Get(fullPath, idAttrName)
			if err != nil || len(nodeID) == 0 {
				logFailure("Directory '%s' missing '%s', skipping its children", fullPath, idAttrName)
				continue
			}
			walkNodes(fullPath, string(nodeID))
		}
	}
	return fixes
}

func checkNodeIDs(spacePath string) {
	spinner.Message(" - checking nodes")

	rootID, err := xattr.Get(spacePath, idAttrName)
	if err != nil || len(rootID) == 0 {
		logFailure("Space root '%s' missing '%s' attribute", spacePath, idAttrName)
		return
	}

	fixes := walkNodes(spacePath, string(rootID))

	if fixes > 0 {
		spinner.Pause()
		fmt.Printf("\n  ✓ Fixed %d incorrect node attributes in %s\n", fixes, filepath.Base(spacePath))
		spinner.Unpause()
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
	fmt.Printf("  Updating directory '%s' with attribute '%s' -> %s\n", filepath.Base(spacePath), idAttrName, targetID)
	err = xattr.Set(spacePath, idAttrName, []byte(targetID))
	if err != nil {
		logFailure("Failed to set attribute on directory '%s': %v", spacePath, err)
		return
	}
	err = xattr.Set(spacePath, spaceIDAttrName, []byte(targetID))
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

		parentID, err := xattr.Get(fullPath, parentIDAttrName)
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

	ownerID, err := xattr.Get(basePath, ownerIDAttrName)
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
	spinner.StopFailMessage(fmt.Sprintf("\n"+message, args...))
	spinner.StopFail()
	spinner.Start()
}
