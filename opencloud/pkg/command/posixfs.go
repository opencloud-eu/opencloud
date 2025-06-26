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
	"github.com/pkg/xattr"
	"github.com/theckman/yacspin"
	"github.com/urfave/cli/v2"
	"github.com/vmihailenco/msgpack/v5"
)

// Define the names of the extended attributes we are working with.
const (
	parentIDAttrName = "user.oc.parentid"
	idAttrName       = "user.oc.id"
	spaceIDAttrName  = "user.oc.space.id"
	ownerIDAttrName  = "user.oc.owner.id"
)

var (
	spinner         *yacspin.Spinner
	restartRequired = false
)

// EntryInfo holds information about a directory entry.
type EntryInfo struct {
	Path     string
	ModTime  time.Time
	ParentID string
}

// PosixfsCommand is the entrypoint for the posixfs command.
func PosixfsCommand(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:     "posixfs",
		Usage:    `cli tools to inspect and manipulate a posixfs storage.`,
		Category: "maintenance",
		Subcommands: []*cli.Command{
			consistencyCmd(cfg),
		},
	}
}

func init() {
	register.AddCommand(PosixfsCommand)
}

// consistencyCmd returns a command to check the consistency of the posixfs storage.
func consistencyCmd(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "consistency",
		Usage: "check the consistency of the posixfs storage",
		Action: func(c *cli.Context) error {
			return checkPosixfsConsistency(c, cfg)
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "root",
				Aliases:  []string{"r"},
				Required: true,
				Usage:    "Path to the root directory of the posixfs storage",
			},
		},
	}
}

// checkPosixfsConsistency checks the consistency of the posixfs storage.
func checkPosixfsConsistency(c *cli.Context, cfg *config.Config) error {
	rootPath := c.String("root")
	indexesPath := filepath.Join(rootPath, "indexes")

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
	spinner.StopMessage("completed")
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
}

func checkSpaceID(spacePath string) {
	spinner.Message("checking space ID uniqueness")

	entries, uniqueIDs, oldestEntry, err := gatherAttributes(spacePath)
	if err != nil {
		logFailure("Failed to gather attributes: %v", err)
		return
	}

	if len(entries) == 0 {
		logSuccess("(empty space)")
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
		fixSpaceID(spacePath, targetID, entries)
		spinner.Unpause()
	} else {
		logSuccess("")
	}
}

func fixSpaceID(spacePath, targetID string, entries []EntryInfo) {
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
	err = updateOwnerIndexFile(spacePath, targetID)
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
func updateOwnerIndexFile(basePath string, targetID string) error {
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
	for id := range indexMap {
		if id != targetID {
			if _, ok := indexMap[id]; ok {
				delete(indexMap, id)
				itemsRemoved++
				fmt.Printf("    - Removing obsolete ID '%s' from index.\n", id)
			}
		}
	}

	if itemsRemoved == 0 {
		fmt.Printf("  No obsolete IDs found in the index file. Nothing to change.\n")
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

	logSuccess("Successfully removed %d item(s) and saved index file.\n", itemsRemoved)
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
	spinner.StopFailMessage(fmt.Sprintf(message, args...))
	spinner.StopFail()
	spinner.Start()
}

func logSuccess(message string, args ...any) {
	spinner.StopMessage(fmt.Sprintf(message, args...))
	spinner.Stop()
	spinner.Start()
}
