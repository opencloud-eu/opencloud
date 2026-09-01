package command

import (
	"errors"
	"fmt"
	"path/filepath"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"

	"github.com/opencloud-eu/opencloud/opencloud/pkg/register"
	"github.com/opencloud-eu/opencloud/opencloud/pkg/revisions"
	"github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/config/parser"
	decomposedbs "github.com/opencloud-eu/reva/v2/pkg/storage/fs/decomposed/blobstore"
	decomposeds3bs "github.com/opencloud-eu/reva/v2/pkg/storage/fs/decomposeds3/blobstore"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/posix/lookup"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"

	"github.com/spf13/cobra"
)

var (
	// _nodesGlobPattern is the glob pattern to find all nodes
	_nodesGlobPattern = "spaces/*/*/nodes/"
)

const posixDriver = "posix"

// RevisionsCommand is the entrypoint for the revisions command.
func RevisionsCommand(cfg *config.Config) *cobra.Command {
	revCmd := &cobra.Command{
		Use:   "revisions",
		Short: "OpenCloud revisions functionality",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnError(parser.ParseConfig(cfg, true))
		},
	}
	revCmd.AddCommand(PurgeRevisionsCommand(cfg))

	return revCmd
}

// PurgeRevisionsCommand allows removing all revisions from a storage provider.
func PurgeRevisionsCommand(cfg *config.Config) *cobra.Command {
	revCmd := &cobra.Command{
		Use:   "purge",
		Short: "purge revisions",
		RunE: func(cmd *cobra.Command, args []string) error {
			basePath, _ := cmd.Flags().GetString("basepath")

			var (
				bs  revisions.DelBlobstore
				err error
			)
			blobstoreFlag, _ := cmd.Flags().GetString("blobstore")
			switch blobstoreFlag {
			case "decomposeds3":
				bs, err = decomposeds3bs.New(
					cfg.StorageUsers.Drivers.DecomposedS3.Endpoint,
					cfg.StorageUsers.Drivers.DecomposedS3.Region,
					cfg.StorageUsers.Drivers.DecomposedS3.Bucket,
					cfg.StorageUsers.Drivers.DecomposedS3.AccessKey,
					cfg.StorageUsers.Drivers.DecomposedS3.SecretKey,
					decomposeds3bs.Options{},
				)
			case "decomposed":
				bs, err = decomposedbs.New(basePath)
			case "none":
				bs = nil
			default:
				err = errors.New("blobstore type not supported")
			}
			if err != nil {
				fmt.Println(err)
				return err
			}

			var rid *provider.ResourceId
			resourceIDFlag, _ := cmd.Flags().GetString("resource-id")
			resid, err := storagespace.ParseID(resourceIDFlag)
			if err == nil {
				rid = &resid
			}

			mechanism, _ := cmd.Flags().GetString("glob-mechanism")
			if rid.GetOpaqueId() != "" {
				mechanism = "glob"
			}

			var posix = cfg.StorageUsers.Driver == posixDriver

			var ch <-chan string
			switch mechanism {
			default:
				fallthrough
			case "glob":
				p := generatePath(basePath, rid, posix)
				if rid.GetOpaqueId() == "" {
					p = filepath.Join(p, "*/*/*/*/*")
				}
				ch = revisions.Glob(p)
			case "workers":
				p := generatePath(basePath, rid, posix)
				ch = revisions.GlobWorkers(p, "/*", "/*/*/*/*")
			case "list":
				p := basePath
				if !posix {
					p = filepath.Join(basePath, "spaces")
				}
				if rid != nil {
					p = generatePath(basePath, rid, posix)
				}
				ch = revisions.List(p, 10)
			}

			flagDryRun, err := cmd.Flags().GetBool("dry-run")
			if err != nil {
				return err
			}

			flagVerbose, err := cmd.Flags().GetBool("verbose")
			if err != nil {
				return err
			}

			files, blobs, revisionResults := revisions.PurgeRevisions(ch, bs, flagDryRun, flagVerbose)
			printResults(files, blobs, revisionResults, flagDryRun)
			return nil
		},
	}
	revCmd.Flags().StringP("basepath", "p", "", "the basepath of the decomposedfs (e.g. /var/tmp/opencloud/storage/metadata)")
	_ = revCmd.MarkFlagRequired("basepath")
	revCmd.Flags().StringP("blobstore", "b", "decomposed", "the blobstore type. Can be (none, decomposed, decomposeds3). Default decomposed")
	revCmd.Flags().Bool("dry-run", true, "do not delete anything, just print what would be deleted")
	revCmd.Flags().BoolP("verbose", "v", false, "print verbose output")
	revCmd.Flags().StringP("resource-id", "r", "", "purge all revisions of this file/space. If not set, all revisions will be purged")
	revCmd.Flags().String("glob-mechanism", "glob", "the glob mechanism to find all nodes. Can be 'glob', 'list' or 'workers'. 'glob' uses globbing with a single worker. 'workers' spawns multiple go routines, accelatering the command drastically but causing high cpu and ram usage. 'list' looks for references by listing directories with multiple workers. Default is 'glob'")

	return revCmd
}

func printResults(countFiles, countBlobs, countRevisions int, dryRun bool) {
	switch {
	case countFiles == 0 && countRevisions == 0 && countBlobs == 0:
		fmt.Println("❎ No revisions found. Storage provider is clean.")
	case !dryRun:
		fmt.Printf("✅ Deleted %d revisions (%d files / %d blobs)\n", countRevisions, countFiles, countBlobs)
	default:
		fmt.Printf("👉 Would delete %d revisions (%d files / %d blobs)\n", countRevisions, countFiles, countBlobs)
	}
}

func generatePath(basePath string, rid *provider.ResourceId, posix bool) string {
	// decomposedfs and posix store the revisions of a node at different
	// locations on disk, so the path has to be built per driver:
	//   - decomposedfs: <basePath>/spaces/<pathified spaceID>/nodes/<pathified nodeID>.REV.<ts>
	//   - posix:        <basePath>/<users|projects>/<spaceID>/.oc-nodes/<pathified nodeID>.REV.<ts>
	if posix {
		if rid == nil {
			return filepath.Join(basePath, "*", "*", ".oc-nodes")
		}

		nid := lookup.Pathify(rid.GetOpaqueId(), 4, 2)
		if nid != "" {
			return filepath.Join(basePath, "*", "*", ".oc-nodes", nid+"*")
		}

		if rid.GetSpaceId() == "" {
			return ""
		}
		return filepath.Join(basePath, "*", rid.GetSpaceId(), ".oc-nodes")
	} else {
		// decomposedfs
		if rid == nil {

			return filepath.Join(basePath, _nodesGlobPattern)
		}

		sid := lookup.Pathify(rid.GetSpaceId(), 1, 2)
		if sid == "" {
			return ""
		}

		nid := lookup.Pathify(rid.GetOpaqueId(), 4, 2)
		if nid == "" {
			return filepath.Join(basePath, "spaces", sid, "nodes")
		}

		return filepath.Join(basePath, "spaces", sid, "nodes", nid+"*")
	}
}

func init() {
	register.AddCommand(RevisionsCommand)
}
