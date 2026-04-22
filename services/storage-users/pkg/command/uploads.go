package command

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/config"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/event"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/revaconfig"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/registry"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// Session contains the information of an upload session
type Session struct {
	ID         string         `json:"id"`
	Space      string         `json:"space"`
	Filename   string         `json:"filename"`
	Offset     int64          `json:"offset"`
	Size       int64          `json:"size"`
	Executant  userpb.UserId  `json:"executant"`
	SpaceOwner *userpb.UserId `json:"spaceowner,omitempty"`
	Expires    time.Time      `json:"expires"`
	Processing bool           `json:"processing"`
	ScanDate   time.Time      `json:"virus_scan_date"`
	ScanResult string         `json:"virus_scan_result"`
}

// Uploads is the entry point for the uploads command
func Uploads(cfg *config.Config) *cobra.Command {
	uploadsCmd := &cobra.Command{
		Use:   "uploads",
		Short: "manage unfinished uploads",
	}
	uploadsCmd.AddCommand([]*cobra.Command{
		ListUploadSessions(cfg),
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
			if restart || resume {
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
				raw   []Session
			)

			if !renderJson {
				fmt.Println(buildInfo(filter))

				table = tablewriter.NewTable(os.Stdout, tablewriter.WithHeaderAutoFormat(tw.Off))
				table.Header([]string{"Space", "Upload Id", "Name", "Offset", "Size", "Executant", "Owner", "Expires", "Processing", "Scan Date", "Scan Result"})
			}

			for _, u := range uploads {
				ref := u.Reference()
				sr, sd := u.ScanData()

				session := Session{
					Space:      ref.GetResourceId().GetSpaceId(),
					ID:         u.ID(),
					Filename:   u.Filename(),
					Offset:     u.Offset(),
					Size:       u.Size(),
					Executant:  u.Executant(),
					SpaceOwner: u.SpaceOwner(),
					Expires:    u.Expires(),
					Processing: u.IsProcessing(),
					ScanDate:   sd,
					ScanResult: sr,
				}

				if renderJson {
					raw = append(raw, session)
				} else {
					table.Append([]string{
						session.Space,
						session.ID,
						session.Filename,
						strconv.FormatInt(session.Offset, 10),
						strconv.FormatInt(session.Size, 10),
						session.Executant.OpaqueId,
						session.SpaceOwner.GetOpaqueId(),
						session.Expires.Format(time.RFC3339),
						strconv.FormatBool(session.Processing),
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
					if err := u.Purge(cmd.Context()); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to clean upload session '%s'\n", u.ID())
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
