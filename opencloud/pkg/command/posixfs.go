package command

import (
	"errors"
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
	scanCmd := &cobra.Command{
		Use:   "scan [path ...]",
		Short: "Perform a filesystem scan and update the ID and filemetadata cache",
		Long: `Perform a filesystem scan and update the ID and filemetadata cache.

You can specify one or more paths to limit the scope of the scan.
If no path is provided, the whole storage is checked, starting at the storage root directory.

The provided arguments determines the scope of the check:
  - a storage root:   the whole storage (all personal and project spaces) is scanned
  - a space root:     only that space is scanned
  - a file or directory: only that single resource is scanned (and its children, if it is a directory)

Any specified file or directory must be underneath the storage root directory and if that is not the case,
the command is aborted with an error before performing any scanning.`,
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
			if cfg.Driver != "posix" {
				return fmt.Errorf("this command is only available when using the 'posix' driver. Current driver: '%s'", cfg.Driver)
			}

			haltOnError, err := cmd.Flags().GetBool("halt-on-error")
			if err != nil {
				return err
			}

			storageRoot := cfg.Drivers.Posix.Root
			paths := []string{storageRoot}
			if len(args) > 0 {
				paths = []string{}
				for _, v := range args {
					path := v
					if !filepath.IsAbs(path) {
						abs, err := filepath.Abs(path)
						if err != nil {
							return fmt.Errorf("failed to make the specified path %q absolute: %w", path, err)
						}
						path = abs
					}
					// not ensuring whether the path is under the storage root here, will be done when iterating over them
					path = filepath.Clean(path)
					paths = append(paths, path)
				}
			}

			var scan func(path string) error = nil
			{
				// We want to initialize the driver but disable scanfs on boot, so we can trigger it manually afterwards
				drivers := revaconfig.StorageProviderDrivers(cfg)
				drivers["posix"] = revaconfig.Posix(cfg, false, false)

				var fsStream events.Stream
				var err error
				fsStream, err = event.NewStream(cfg)
				if err != nil {
					return fmt.Errorf("failed to create event stream for posix driver: %w", err)
				}
				log := logger("posixfs")

				f, ok := registry.NewFuncs["posix"]
				if !ok {
					return errors.New("posix driver not found in registry")
				}

				fs, err := f(drivers["posix"].(map[string]any), fsStream, &log)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to initialize filesystem driver '%s': %v\n", cfg.Driver, err)
					return err
				}

				cacher, ok := fs.(IDCacher)
				if !ok {
					return errors.New("the posix driver does not expose WarmupIDCache")
				}

				scan = func(path string) error {
					err := cacher.WarmupIDCache(path, true, false)
					if err != nil {
						logFailure("Error scanning path '%s': %v", path, err)
					}
					return err
				}
			}

			errs := processPosixFsResources(paths, !haltOnError,
				func(path string) error {
					fmt.Println("Scanning personal spaces...")
					return scan(path)
				},
				func(path string) error {
					fmt.Println("Scanning project spaces...")
					return scan(path)
				},
				func(path string) error {
					fmt.Printf("Scanning space '%s'...\n", path)
					return scan(path)
				},
				func(path string) error {
					fmt.Printf("Scanning '%s'...\n", path)
					return scan(path)
				},
			)

			if len(errs) == 0 {
				fmt.Println("Scan completed successfully.")
				return nil
			} else {
				plural := "s"
				if len(errs) == 1 {
					plural = ""
				}
				verb := "completed"
				if haltOnError {
					verb = "aborted"
				}
				return fmt.Errorf("scan %s with %d error%s", verb, len(errs), plural)
			}
		},
	}
	scanCmd.Flags().BoolP("halt-on-error", "E", false, "Halt at once when an error occurs when processing one of the paths (default behaviour is to keep going and attempt to process all paths).")
	return scanCmd
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
			recalculateChecksums, err := cmd.Flags().GetBool("fix-checksums")
			if err != nil {
				return err
			}

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

// iterates over a list of paths and processes them all, using the appropriate function
// depending on the type of resource
//
// note that whenever an error occurs, it collects that error and continues processing
// subsequent paths, and then returns a slice of errors at the end (or an empty slice
// if no errors occured)
func processPosixFsResources(paths []string,
	keepGoing bool,
	personalSpaceDir func(string) error,
	projectSpaceDir func(string) error,
	spaceRoot func(string) error,
	entity func(string) error,
) []error {
	// no need to guard this with a mutex for now, since the implementation is not parallelized
	errors := []error{}

	for _, path := range paths {
		rootPath, err := findStorageRoot(path)
		if err != nil {
			errors = append(errors, err)
			logFailure("error: %s", err)
			if keepGoing {
				continue
			} else {
				return errors
			}
		}
		path = filepath.Clean(path)
		if _, err := os.Stat(path); err != nil {
			errors = append(errors, err)
			logFailure("error accessing '%s': %w", path, err)
			if keepGoing {
				continue
			} else {
				return errors
			}
		}
		contained, _ := filepathx.IsSameOrContainedBy(rootPath, path)

		switch {
		case path == rootPath:
			if err := personalSpaceDir(filepath.Join(path, "users")); err != nil {
				errors = append(errors, err)
				if !keepGoing {
					return errors
				}
			}
			if err := projectSpaceDir(filepath.Join(path, "projects")); err != nil {
				errors = append(errors, err)
				if !keepGoing {
					return errors
				}
			}
		case isSpaceRoot(path):
			if err := spaceRoot(path); err != nil {
				errors = append(errors, err)
				if !keepGoing {
					return errors
				}
			}
		case contained:
			if err := entity(path); err != nil {
				errors = append(errors, err)
				if !keepGoing {
					return errors
				}
			}
		default:
			err := fmt.Errorf("error: the provided path '%s' is neither a space root nor contained by the storage root '%s'", path, rootPath)
			errors = append(errors, err)
			logFailure(err.Error())
			if !keepGoing {
				return errors
			}
		}
	}
	return errors
}
