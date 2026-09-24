package command

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/shamaton/msgpack/v2"
	"github.com/spf13/cobra"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/config"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/event"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/revaconfig"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/registry"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/lookup"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

const (
	MSGPACK_KEY_USER_OCIS_NODESTATUS = "user.ocis.nodestatus"

	// Log indentation levels
	LOG_INDENT_L1 = "  " // 2 spaces
	LOG_INDENT_L2 = LOG_INDENT_L1 + LOG_INDENT_L1
)

// Session contains the information of an upload session
type Session struct {
	ID            string         `json:"id"`
	Space         string         `json:"space"`
	Filename      string         `json:"filename"`
	Offset        int64          `json:"offset"`
	Size          int64          `json:"size"`
	Executant     userpb.UserId  `json:"executant"`
	SpaceOwner    *userpb.UserId `json:"spaceowner,omitempty"`
	Expires       time.Time      `json:"expires"`
	Processing    bool           `json:"processing"`
	ScanDate      time.Time      `json:"virus_scan_date"`
	ScanResult    string         `json:"virus_scan_result"`
	Status        string         `json:"status"`
	StatusMessage string         `json:"status_message"`
}

// Uploads is the entry point for the uploads command
func Uploads(cfg *config.Config) *cobra.Command {
	uploadsCmd := &cobra.Command{
		Use:   "uploads",
		Short: "manage unfinished uploads",
	}
	uploadsCmd.AddCommand([]*cobra.Command{
		ListUploadSessions(cfg),
		DeleteStaleProcessingNodes(cfg),
	}...)

	return uploadsCmd

}

// ListUploadSessions prints a list of upload sessiens
func ListUploadSessions(cfg *config.Config) *cobra.Command {
	listUploadSessionsCmd := &cobra.Command{
		Use:   "sessions",
		Short: "Print a list of upload sessions",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			f, ok := registry.NewFuncs[cfg.Driver]
			if !ok {
				fmt.Fprintf(os.Stderr, "Unknown filesystem driver '%s'\n", cfg.Driver)
				os.Exit(1)
			}
			drivers := revaconfig.StorageProviderDrivers(cfg)
			var fsStream events.Stream
			if cfg.Driver == "posix" {
				// We need to init the posix driver with 'scanfs' disabled
				drivers["posix"] = revaconfig.Posix(cfg, false, false)
				// Also posix refuses to start without an events stream
				fsStream, err = event.NewStream(cfg)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to create event stream for posix driver: %v\n", err)
					os.Exit(1)
				}
			}

			fs, err := f(drivers[cfg.Driver].(map[string]any), fsStream, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to initialize filesystem driver '%s'\n", cfg.Driver)
				return err
			}

			managingFS, ok := fs.(storage.UploadSessionLister)
			if !ok {
				fmt.Fprintf(os.Stderr, "'%s' storage does not support listing upload sessions\n", cfg.Driver)
				os.Exit(1)
			}

			restart, _ := cmd.Flags().GetBool("restart")
			resume, _ := cmd.Flags().GetBool("resume")
			clean, _ := cmd.Flags().GetBool("clean")
			renderJson, _ := cmd.Flags().GetBool("json")

			var stream events.Stream
			if restart || resume || clean {
				stream, err = event.NewStream(cfg)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to create event stream: %v\n", err)
					os.Exit(1)
				}
			}

			filter := buildFilter(cmd)
			uploads, err := managingFS.ListUploadSessions(cmd.Context(), filter)
			if err != nil {
				return err
			}

			var (
				table *tablewriter.Table
				raw   []*Session
			)

			if !renderJson {
				fmt.Println(buildInfo(filter))

				table = tablewriter.NewTable(os.Stdout, tablewriter.WithHeaderAutoFormat(tw.Off))
				table.Header([]string{"Space", "Upload Id", "Name", "Status", "Message", "Offset", "Size", "Executant", "Owner", "Expires", "Scan Date", "Scan Result"})
			}

			for _, u := range uploads {
				ref := u.Reference()
				sr, sd := u.ScanData()

				session := Session{
					Space:         ref.GetResourceId().GetSpaceId(),
					ID:            u.ID(),
					Filename:      u.Filename(),
					Offset:        u.Offset(),
					Size:          u.Size(),
					Executant:     u.Executant(),
					SpaceOwner:    u.SpaceOwner(),
					Expires:       u.Expires(),
					ScanDate:      sd,
					ScanResult:    sr,
					Status:        string(u.Status()),
					StatusMessage: u.StatusMessage(),
				}

				if renderJson {
					raw = append(raw, &session)
				} else {
					table.Append([]string{
						session.Space,
						session.ID,
						session.Filename,
						session.Status,
						session.StatusMessage,
						strconv.FormatInt(session.Offset, 10),
						strconv.FormatInt(session.Size, 10),
						session.Executant.OpaqueId,
						session.SpaceOwner.GetOpaqueId(),
						session.Expires.Format(time.RFC3339),
						session.ScanDate.Format(time.RFC3339),
						session.ScanResult,
					})
				}

				switch {
				case restart:
					if err := events.Publish(context.Background(), stream, events.RestartPostprocessing{
						UploadID:  u.ID(),
						Timestamp: utils.TSNow(),
					}); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to send restart event for upload session '%s'\n", u.ID())
						// if publishing fails there is no need to try publishing other events - they will fail too.
						os.Exit(1)
					}

				case resume:
					if err := events.Publish(context.Background(), stream, events.ResumePostprocessing{
						UploadID:  u.ID(),
						Timestamp: utils.TSNow(),
					}); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to send resume event for upload session '%s'\n", u.ID())
						// if publishing fails there is no need to try publishing other events - they will fail too.
						os.Exit(1)
					}

				case clean:
					if err := events.Publish(context.Background(), stream, events.CleanUpload{
						UploadID:  u.ID(),
						Timestamp: utils.TSNow(),
					}); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to send clean upload event for upload session '%s'\n", u.ID())
						// if publishing fails there is no need to try publishing other events - they will fail too.
						os.Exit(1)
					}
				}

			}

			if !renderJson {
				table.Render()
				return nil
			}

			j, err := json.Marshal(raw)
			if err != nil {
				fmt.Println(err)
				return err
			}
			fmt.Println(string(j))
			return nil
		},
	}
	listUploadSessionsCmd.Flags().String("id", "", "filter sessions by upload session id")
	listUploadSessionsCmd.Flags().Bool("processing", false, "filter sessions by processing status")
	listUploadSessionsCmd.Flags().Bool("expired", false, "filter sessions by expired status")
	listUploadSessionsCmd.Flags().Bool("has-virus", false, "filter sessions by virus scan result")
	listUploadSessionsCmd.Flags().Bool("json", false, "output as json")
	listUploadSessionsCmd.Flags().Bool("restart", false, "send restart event for all listed sessions. Only one of resume/restart/clean can be set.")
	listUploadSessionsCmd.Flags().Bool("resume", false, "send resume event for all listed sessions. Only one of resume/restart/clean can be set.")
	listUploadSessionsCmd.Flags().Bool("clean", false, "remove uploads for all listed sessions. Only one of resume/restart/clean can be set.")
	return listUploadSessionsCmd
}

func buildFilter(cmd *cobra.Command) storage.UploadSessionFilter {
	filter := storage.UploadSessionFilter{}
	if cmd.Flag("processing").Changed {
		processingValue, _ := cmd.Flags().GetBool("processing")
		filter.Processing = &processingValue
	}
	if cmd.Flag("expired").Changed {
		expiredValue, _ := cmd.Flags().GetBool("expired")
		filter.Expired = &expiredValue
	}
	if cmd.Flag("has-virus").Changed {
		infectedValue, _ := cmd.Flags().GetBool("has-virus")
		filter.HasVirus = &infectedValue
	}
	if cmd.Flag("id").Changed {
		idValue, _ := cmd.Flags().GetString("id")
		if idValue != "" {
			filter.ID = &idValue
		}
	}
	return filter
}

func buildInfo(filter storage.UploadSessionFilter) string {
	var b strings.Builder
	if filter.Processing != nil {
		if !*filter.Processing {
			b.WriteString("Not ")
		}
		if b.Len() == 0 {
			b.WriteString("Processing")
		} else {
			b.WriteString("processing")
		}
	}

	if filter.Expired != nil {
		if b.Len() != 0 {
			b.WriteString(", ")
		}
		if !*filter.Expired {
			if b.Len() == 0 {
				b.WriteString("Not ")
			} else {
				b.WriteString("not ")
			}
		}
		if b.Len() == 0 {
			b.WriteString("Expired")
		} else {
			b.WriteString("expired")
		}
	}

	if filter.HasVirus != nil {
		if b.Len() != 0 {
			b.WriteString(", ")
		}
		if !*filter.HasVirus {
			if b.Len() == 0 {
				b.WriteString("Not ")
			} else {
				b.WriteString("not ")
			}
		}
		if b.Len() == 0 {
			b.WriteString("Virusinfected")
		} else {
			b.WriteString("virusinfected")
		}
	}

	if b.Len() == 0 {
		b.WriteString("Session")
	} else {
		b.WriteString(" session")
	}

	if filter.ID != nil {
		b.WriteString(" with id '" + *filter.ID + "'")
	} else {
		// to make `session` plural
		b.WriteString("s")
	}

	b.WriteString(":")
	return b.String()
}

// DeleteStaleProcessingNodes is the entry point for the delete-stale-nodes command
func DeleteStaleProcessingNodes(cfg *config.Config) *cobra.Command {
	deleteStaleNodesCmd := &cobra.Command{
		Use:   "delete-stale-nodes",
		Short: "Delete all nodes in processing state that are not referenced by any upload session",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			spaceIDs := []string{}
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			verbose, _ := cmd.Flags().GetBool("verbose")
			start := time.Now()

			// Check if specific space ID provided
			if cmd.Flags().Changed("spaceid") {
				spaceID, _ := cmd.Flags().GetString("spaceid")
				spaceIDs = append(spaceIDs, spaceID)
			} else {
				fmt.Println("Scanning all spaces for stale processing nodes...")
				spaceIDs = globSpaceIDs(cfg)
			}

			if verbose {
				fmt.Printf("Spaces to cleanup: %d\n", len(spaceIDs))
				for _, spaceID := range spaceIDs {
					fmt.Printf("  - %s\n", spaceID)
				}
			}

			var stream events.Stream
			if !dryRun {
				s, err := event.NewStream(cfg)
				if err != nil {
					log.Fatalf("Failed to create event stream: %v", err)
				}
				stream = s
			}

			staleCount := 0
			for _, spaceID := range spaceIDs {
				staleCount += deleteStaleUploads(cfg, spaceID, dryRun, verbose, stream)
			}

			if verbose {
				fmt.Printf("Took %ds\n", int(time.Since(start).Seconds()))
			}
			fmt.Printf("Total stale nodes: %d\n", staleCount)

			return nil
		},
	}
	deleteStaleNodesCmd.Flags().String("spaceid", "", "Space ID to check for processing nodes (omit to check all spaces)")
	deleteStaleNodesCmd.Flags().Bool("dry-run", true, "Only show what would be deleted without actually deleting")
	deleteStaleNodesCmd.Flags().Bool("verbose", false, "Enable verbose logging")
	return deleteStaleNodesCmd
}

// globSpaceIDs returns a list of all space IDs in the storage root
func globSpaceIDs(cfg *config.Config) []string {
	fsys := os.DirFS(cfg.Drivers.Decomposed.Root)
	dirs, err := fs.Glob(fsys, "spaces/*/*/nodes")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error globbing spaces root directory %s: %v\n", cfg.Drivers.Decomposed.Root, err)
		return []string{}
	}

	spaceIDs := []string{}
	for _, dir := range dirs {
		// For dir i.e. spaces/9d/408cec-8f0a-4d33-8715-89df1217a10c/nodes
		// spaceID is 9d408cec-8f0a-4d33-8715-89df1217a10c
		spaceIDs = append(spaceIDs, strings.ReplaceAll(strings.TrimSuffix(strings.TrimPrefix(dir, "spaces/"), "/nodes"), "/", ""))
	}
	return spaceIDs
}

// delete stale processing nodes for a given spaceID
func deleteStaleUploads(cfg *config.Config, spaceID string, dryRun bool, verbose bool, stream events.Stream) int {
	if verbose {
		fmt.Printf("\nDeleting stale processing nodes for space: %s\n", spaceID)
	}

	// Find .mpk files in space directory
	spaceRoot := filepath.Join(cfg.Drivers.Decomposed.Root, "spaces", lookup.Pathify(spaceID, 1, 2))
	mpkFiles := []string{}
	err := filepath.Walk(spaceRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accessing path %s: %s\n", path, err)
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasSuffix(path, ".mpk") {
			mpkFiles = append(mpkFiles, path)
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking space directory %s: %s\n", spaceRoot, err)
		return 0
	}

	if verbose {
		fmt.Printf("%sFound total %d .mpk files\n", LOG_INDENT_L1, len(mpkFiles))
	}

	staleCount := 0
	for _, path := range mpkFiles {
		staleCount += deleteStaleNode(cfg, path, dryRun, verbose, stream)
	}

	if verbose {
		fmt.Printf("%sFound total %d stale nodes\n", LOG_INDENT_L1, staleCount)
	}

	return staleCount
}

// deleteStaleNode deletes a stale node: if it is not referenced by any upload session
// returns 1 if the node stale node was detected for deletion, 0 otherwise, for counting purposes
func deleteStaleNode(cfg *config.Config, path string, dryRun bool, verbose bool, stream events.Stream) int {
	nodeDir := filepath.Dir(path)

	// Read .mpk file to get processing info
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file %s: %s\n", path, err)
		return 0
	}
	var mpkData map[string]any
	if err := msgpack.Unmarshal(b, &mpkData); err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshaling file %s: %s\n", path, err)
		return 0
	}

	processingID := extractProcessingID(mpkData)
	if processingID == "" {
		return 0
	}

	// Construct path to upload info file:
	// i.e. ~/.ocis/storage/users/uploads/5329c14b-b786-4b27-8f7d-7429f03009d7.info
	// And pass only the .info file not exists: err is ErrNotExist
	pathUploadInfo := filepath.Join(cfg.Drivers.Decomposed.Root, "uploads", processingID) + ".info"
	_, infoStatErr := os.Stat(pathUploadInfo)
	if infoStatErr == nil {
		return 0
	}
	if !os.IsNotExist(infoStatErr) {
		// Tere was an error other than file not existing, log and return
		fmt.Fprintf(os.Stderr, "Error checking upload info %s: %s\n", pathUploadInfo, infoStatErr)
		return 0
	}

	if verbose {
		fmt.Printf("%sFound stale upload at %s (Processing ID: %s)\n", LOG_INDENT_L1, path, processingID)
		fmt.Printf("%sUpload info missing at: %s\n", LOG_INDENT_L2, pathUploadInfo)
	}

	if dryRun {
		return 1
	}

	rid := extractResourceID(strings.TrimSuffix(path, ".mpk"))
	if rid == nil {
		fmt.Fprintf(os.Stderr, "Failed to extract resource ID from path %s\n", path)
		return 0
	}

	if err := events.Publish(context.Background(), stream, events.RevertRevision{
		ResourceID: rid,
		Timestamp:  utils.TSNow(),
	}); err != nil {
		// if publishing fails there is no need to try publishing other events - they will fail too.
		log.Fatalf("Failed to send revert revision event for node '%s'\n", path)
	}

	if verbose {
		fmt.Printf("%sDeleted stale node: %s\n", LOG_INDENT_L2, nodeDir)
	}

	return 1
}

func extractProcessingID(mpkData map[string]any) string {
	processingID := ""
	for k, v := range mpkData {
		vStr := string(v.([]byte))
		if k == MSGPACK_KEY_USER_OCIS_NODESTATUS && strings.Contains(vStr, node.ProcessingStatus) {
			processingID = strings.Split(vStr, ":")[1]
			break
		}
	}
	return processingID
}

func extractResourceID(path string) *provider.ResourceId {
	// path looks like /.../storage/users/spaces/f2/06bccf-0f10-4070-9e63-40943f060667/nodes/5b/ba/1e/a7/-f185-4f31-8342-ed4b5743f096
	parts := strings.Split(path, "spaces")
	if len(parts) < 2 {
		return nil
	}

	spaceParts := strings.Split(parts[1], "nodes")
	if len(spaceParts) < 2 {
		return nil
	}

	return &provider.ResourceId{
		SpaceId:  strings.ReplaceAll(spaceParts[0], "/", ""),
		OpaqueId: strings.ReplaceAll(spaceParts[1], "/", ""),
	}
}
