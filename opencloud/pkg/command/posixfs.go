package command

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/opencloud-eu/opencloud/opencloud/pkg/register"
	"github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/pkg/x/path/filepathx"
	storageUsersParser "github.com/opencloud-eu/opencloud/services/storage-users/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/event"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/revaconfig"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/ignore"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/options"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/registry"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/metadata/prefixes"

	"github.com/pkg/xattr"
	"github.com/spf13/cobra"
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
  - a file or directory: only that single entity is checked (and its children, if it is a directory)`,
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
			if len(args) == 0 {
				args = []string{cfg.Drivers.Posix.Root}
			}
			log := logger("posixfs")
			recalculateChecksums, _ := cmd.Flags().GetBool("fix-checksums")

			drivers := revaconfig.StorageProviderDrivers(cfg)
			drivers["posix"] = revaconfig.Posix(cfg, false, false)
			opts, err := options.New(drivers["posix"].(map[string]any))
			if err != nil {
				return err
			}
			ignorer := ignore.NewIgnorer(opts, &log)

			checker := &consistencyChecker{
				cfg:                  cfg,
				ignorer:              ignorer,
				recalculateChecksums: recalculateChecksums,
			}

			return checker.Check(args)
		},
	}
	consCmd.Flags().Bool("fix-checksums", false, "Recalculate and fix the file checksums. This reads every file and can be slow on large storages.")

	return consCmd
}

func logFailure(message string, args ...any) {
	fmt.Fprintf(os.Stderr, message+"\n", args...)
}

// findStorageRoot walks up the directory tree starting at path until it finds a
// directory that contains an "indexes" subdirectory which marks the root of a
// posixfs storage. A user directory inside a space might also be named "indexes",
// so to disambiguate we require that the "indexes" directory is an internal
// directory: the storage's own indexes directory is skipped during assimilation
// and therefore never receives a node ID attribute, whereas a regular user
// directory would have one.
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
