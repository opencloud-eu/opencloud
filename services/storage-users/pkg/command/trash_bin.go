package command

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	zlog "github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/config"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/config/parser"
	"github.com/opencloud-eu/opencloud/services/storage-users/pkg/event"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/mohae/deepcopy"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

const (
	SKIP = iota
	REPLACE
	KEEP_BOTH
)

// TrashBin wraps trash-bin related sub-commands.
func TrashBin(cfg *config.Config) *cobra.Command {
	trashBinCmd := &cobra.Command{
		Use:   "trash-bin",
		Short: "manage trash-bin's",
	}

	trashBinCmd.AddCommand([]*cobra.Command{
		PurgeExpiredResources(cfg),
		listTrashBinItems(cfg),
		restoreAllTrashBinItems(cfg),
		restoreTrashBinItem(cfg),
	}...)
	return trashBinCmd
}

// PurgeExpiredResources cli command removes old trash-bin items.
func PurgeExpiredResources(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "purge-expired",
		Short: "Purge expired trash-bin items",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			stream, err := event.NewStream(cfg)
			if err != nil {
				return err
			}

			if err := events.Publish(cmd.Context(), stream, event.PurgeTrashBin{ExecutionTime: time.Now()}); err != nil {
				return err
			}

			// go-micro nats implementation uses async publishing,
			// therefore we need to manually wait.
			//
			// FIXME: upstream pr
			//
			// https://github.com/go-micro/plugins/blob/3e77393890683be4bacfb613bc5751867d584692/v4/events/natsjs/nats.go#L115
			time.Sleep(5 * time.Second)

			return nil
		},
	}
}

func listTrashBinItems(cfg *config.Config) *cobra.Command {
	listTrashBinItemsCmd := &cobra.Command{
		Use:   "list",
		Short: "Print a list of all trash-bin items of a space.",
		// TODO: n might need to equal 2 not sure.
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			log := cliLogger(verbose)
			var spaceID string
			if len(args) > 0 {
				spaceID = args[0]
			}
			if spaceID == "" {
				_ = cmd.Help()
				return fmt.Errorf("spaceID is requiered")
			}
			log.Info().Msgf("Getting trash-bin items for spaceID: '%s' ...", spaceID)

			ref, err := storagespace.ParseReference(spaceID)
			if err != nil {
				return err
			}
			client, err := pool.GetGatewayServiceClient(cfg.RevaGatewayGRPCAddr)
			if err != nil {
				return fmt.Errorf("error selecting gateway client %w", err)
			}
			ctx, err := utils.GetServiceUserContext(cfg.ServiceAccount.ServiceAccountID, client, cfg.ServiceAccount.ServiceAccountSecret)
			if err != nil {
				return fmt.Errorf("could not get service user context %w", err)
			}
			res, err := listRecycle(ctx, client, ref)
			if err != nil {
				return err
			}

			table := itemsTable(len(res.GetRecycleItems()))
			for _, item := range res.GetRecycleItems() {
				table.Append([]string{item.GetKey(), item.GetRef().GetPath(), itemType(item.GetType()), utils.TSToTime(item.GetDeletionTime()).UTC().Format(time.RFC3339)})
			}
			table.Render()
			fmt.Println("Use an itemID to restore an item.")
			return nil
		},
	}
	listTrashBinItemsCmd.Flags().BoolP(
		"verbose",
		"v",
		false,
		"Get more verbose output",
	)
	return listTrashBinItemsCmd
}

func restoreAllTrashBinItems(cfg *config.Config) *cobra.Command {
	var overwriteOption int
	restoreAllTrashBinItemsCmd := &cobra.Command{
		Use:   "restore-all",
		Short: "Restore all trash-bin items for a space.",
		// TODO: not sure this could also be 2
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			log := cliLogger(verbose)
			var spaceID string
			if len(args) > 0 {
				spaceID = args[0]
			}
			if spaceID == "" {
				_ = cmd.Help()
				return fmt.Errorf("spaceID is requiered")
			}
			option, _ := cmd.Flags().GetString("option")
			switch option {
			case "skip":
				overwriteOption = SKIP
			case "replace":
				overwriteOption = REPLACE
			case "keep-both":
				overwriteOption = KEEP_BOTH
			default:
				_ = cmd.Help()
				return fmt.Errorf("option flag '%s' is invalid", option)
			}
			log.Info().Msgf("Restoring trash-bin items for spaceID: '%s' ...", spaceID)

			ref, err := storagespace.ParseReference(spaceID)
			if err != nil {
				return err
			}
			client, err := pool.GetGatewayServiceClient(cfg.RevaGatewayGRPCAddr)
			if err != nil {
				return fmt.Errorf("error selecting gateway client %w", err)
			}
			ctx, err := utils.GetServiceUserContext(cfg.ServiceAccount.ServiceAccountID, client, cfg.ServiceAccount.ServiceAccountSecret)
			if err != nil {
				return fmt.Errorf("could not get service user context %w", err)
			}
			res, err := listRecycle(ctx, client, ref)
			if err != nil {
				return err
			}
			applyYesFlag, _ := cmd.Flags().GetBool("yes")
			if !applyYesFlag {
				for {
					fmt.Printf("Found %d items that could be restored, continue (Y/n), show the items list (s): ", len(res.GetRecycleItems()))
					var i string
					_, err := fmt.Scanf("%s", &i)
					if err != nil {
						log.Err(err).Send()
						continue
					}
					if strings.ToLower(i) == "y" {
						break
					} else if strings.ToLower(i) == "n" {
						return nil
					} else if strings.ToLower(i) == "s" {
						table := itemsTable(len(res.GetRecycleItems()))
						for _, item := range res.GetRecycleItems() {
							table.Append([]string{item.GetKey(), item.GetRef().GetPath(), itemType(item.GetType()), utils.TSToTime(item.GetDeletionTime()).UTC().Format(time.RFC3339)})
						}
						table.Render()
					}
				}
			}

			log.Info().Msgf("Run restoring-all with option=%s", option)
			for _, item := range res.GetRecycleItems() {
				log.Info().Msgf("restoring itemID: '%s', path: '%s', type: '%s'", item.GetKey(), item.GetRef().GetPath(), itemType(item.GetType()))
				dstRes, err := restore(ctx, client, ref, item, overwriteOption, cfg.CliMaxAttemptsRenameFile, log)
				if err != nil {
					log.Err(err).Msg("trash-bin item restoring error")
					continue
				}
				fmt.Printf("itemID: '%s', path: '%s', restored as '%s'\n", item.GetKey(), item.GetRef().GetPath(), dstRes.GetPath())
			}
			return nil
		},
	}
	restoreAllTrashBinItemsCmd.Flags().BoolP(
		"verbose",
		"v",
		false,
		"Get more verbose output",
	)
	restoreAllTrashBinItemsCmd.Flags().StringP(
		"option",
		"o",
		"skip",
		"The restore option defines the behavior for a file to be restored, where the file name already already exists in the target space. Supported values are: 'skip', 'replace' and 'keep-both'. The default value is 'skip' overwriting an existing file.",
	)
	restoreAllTrashBinItemsCmd.Flags().BoolP(
		"yes",
		"y",
		false,
		"Automatic yes to prompts. Assume 'yes' as answer to all prompts and run non-interactively.",
	)

	return restoreAllTrashBinItemsCmd
}

func restoreTrashBinItem(cfg *config.Config) *cobra.Command {
	var overwriteOption int
	restoreTrashBinItemCmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore a trash-bin item by ID.",
		Args:  cobra.ExactArgs(2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return configlog.ReturnFatal(parser.ParseConfig(cfg))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			log := cliLogger(verbose)
			var spaceID, itemID string
			spaceID = args[0]
			itemID = args[1]
			if spaceID == "" {
				_ = cmd.Help()
				return fmt.Errorf("spaceID is requered")
			}
			if itemID == "" {
				_ = cmd.Help()
				return fmt.Errorf("itemID is requered")
			}
			option, _ := cmd.Flags().GetString("option")
			switch option {
			case "skip":
				overwriteOption = SKIP
			case "replace":
				overwriteOption = REPLACE
			case "keep-both":
				overwriteOption = KEEP_BOTH
			default:
				_ = cmd.Help()
				return fmt.Errorf("option flag '%s' is invalid", option)
			}
			log.Info().Msgf("Restoring trash-bin item for spaceID: '%s' itemID: '%s' ...", spaceID, itemID)

			ref, err := storagespace.ParseReference(spaceID)
			if err != nil {
				return err
			}
			client, err := pool.GetGatewayServiceClient(cfg.RevaGatewayGRPCAddr)
			if err != nil {
				return fmt.Errorf("error selecting gateway client %w", err)
			}
			ctx, err := utils.GetServiceUserContext(cfg.ServiceAccount.ServiceAccountID, client, cfg.ServiceAccount.ServiceAccountSecret)
			if err != nil {
				return fmt.Errorf("could not get service user context %w", err)
			}
			res, err := listRecycle(ctx, client, ref)
			if err != nil {
				return err
			}

			var found bool
			var itemRef *provider.RecycleItem
			rid, err := storagespace.ParseID(itemID)
			if err != nil {
				return err
			}
			opaqueID := rid.GetOpaqueId()
			for _, item := range res.GetRecycleItems() {
				if item.GetKey() == opaqueID {
					itemRef = item
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("itemID '%s' not found", itemID)
			}
			log.Info().Msgf("Run restoring with option=%s", option)
			log.Info().Msgf("restoring itemID: '%s', path: '%s', type: '%s", itemRef.GetKey(), itemRef.GetRef().GetPath(), itemType(itemRef.GetType()))
			dstRes, err := restore(ctx, client, ref, itemRef, overwriteOption, cfg.CliMaxAttemptsRenameFile, log)
			if err != nil {
				return err
			}
			fmt.Printf("itemID: '%s', path: '%s', restored as '%s'\n", itemRef.GetKey(), itemRef.GetRef().GetPath(), dstRes.GetPath())
			return nil
		},
	}
	restoreTrashBinItemCmd.Flags().BoolP(
		"verbose",
		"v",
		false,
		"Get more verbose output",
	)
	restoreTrashBinItemCmd.Flags().StringP(
		"option",
		"o",
		"skip",
		"The restore option defines the behavior for a file to be restored, where the file name already already exists in the target space. Supported values are: 'skip', 'replace' and 'keep-both'. The default value is 'skip' overwriting an existing file.",
	)
	return restoreTrashBinItemCmd
}

func listRecycle(ctx context.Context, client gateway.GatewayAPIClient, ref provider.Reference) (*provider.ListRecycleResponse, error) {
	_retrievingErrorMsg := "trash-bin items retrieving error"
	res, err := client.ListRecycle(ctx, &provider.ListRecycleRequest{Ref: &ref, Key: ""})
	if err != nil {
		return nil, fmt.Errorf("%s %w", _retrievingErrorMsg, err)
	}
	if res.Status.Code != rpc.Code_CODE_OK {
		return nil, fmt.Errorf("%s %s", _retrievingErrorMsg, res.Status.Code)
	}
	return res, nil
}

func restore(ctx context.Context, client gateway.GatewayAPIClient, ref provider.Reference, item *provider.RecycleItem, overwriteOption int, maxRenameAttempt int, log zlog.Logger) (*provider.Reference, error) {
	dst, _ := deepcopy.Copy(ref).(provider.Reference)
	dst.Path = utils.MakeRelativePath(item.GetRef().GetPath())
	// Restore request
	req := &provider.RestoreRecycleItemRequest{
		Ref:        &ref,
		Key:        path.Join(item.GetKey(), "/"),
		RestoreRef: &dst,
	}

	exists, dstStatRes, err := isDestinationExists(ctx, client, dst)
	if err != nil {
		return &dst, err
	}

	if exists {
		log.Info().Msgf("destination '%s' exists.", dstStatRes.GetInfo().GetPath())
		switch overwriteOption {
		case SKIP:
			return &dst, nil
		case REPLACE:
			// delete existing tree
			delReq := &provider.DeleteRequest{Ref: &dst}
			delRes, err := client.Delete(ctx, delReq)
			if err != nil {
				return &dst, fmt.Errorf("error sending grpc delete request %w", err)
			}
			if delRes.Status.Code != rpc.Code_CODE_OK && delRes.Status.Code != rpc.Code_CODE_NOT_FOUND {
				return &dst, fmt.Errorf("deleting error %w", err)
			}
		case KEEP_BOTH:
			// modify the file name
			req.RestoreRef, err = resolveDestination(ctx, client, dst, maxRenameAttempt)
			if err != nil {
				return &dst, err
			}
		}
	}

	res, err := client.RestoreRecycleItem(ctx, req)
	if err != nil {
		return req.RestoreRef, fmt.Errorf("restoring error  %w", err)
	}
	if res.Status.Code != rpc.Code_CODE_OK {
		return req.RestoreRef, fmt.Errorf("can not restore %s", res.Status.Code)
	}
	return req.RestoreRef, nil
}

func resolveDestination(ctx context.Context, client gateway.GatewayAPIClient, dstRef provider.Reference, maxRenameAttempt int) (*provider.Reference, error) {
	dst := dstRef
	if maxRenameAttempt < 100 {
		maxRenameAttempt = 100
	}
	for i := 1; i < maxRenameAttempt; i++ {
		dst.Path = modifyFilename(dstRef.Path, i)
		exists, _, err := isDestinationExists(ctx, client, dst)
		if err != nil {
			return nil, err
		}
		if exists {
			continue
		}
		return &dst, nil
	}
	return nil, fmt.Errorf("too many attempts to resolve the destination")
}

func isDestinationExists(ctx context.Context, client gateway.GatewayAPIClient, dst provider.Reference) (bool, *provider.StatResponse, error) {
	dstStatReq := &provider.StatRequest{Ref: &dst}
	dstStatRes, err := client.Stat(ctx, dstStatReq)
	if err != nil {
		return false, nil, fmt.Errorf("error sending grpc stat request %w", err)
	}
	if dstStatRes.GetStatus().GetCode() == rpc.Code_CODE_OK {
		return true, dstStatRes, nil
	}
	if dstStatRes.GetStatus().GetCode() == rpc.Code_CODE_NOT_FOUND {
		return false, dstStatRes, nil
	}
	return false, dstStatRes, fmt.Errorf("stat request failed %s", dstStatRes.GetStatus())
}

// modify the file name like UI do
func modifyFilename(filename string, mod int) string {
	var extension string
	var found bool
	expected := []string{".tar.gz", ".tar.bz", ".tar.bz2"}
	for _, s := range expected {
		var prefix string
		prefix, found = strings.CutSuffix(strings.ToLower(filename), s)
		if found {
			extension = strings.TrimPrefix(filename, prefix)
			break
		}
	}
	if !found {
		extension = filepath.Ext(filename)
	}
	name := filename[0 : len(filename)-len(extension)]
	return fmt.Sprintf("%s (%d)%s", name, mod, extension)
}

func itemType(it provider.ResourceType) string {
	var itemType = "file"
	if it == provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		itemType = "folder"
	}
	return itemType
}

func itemsTable(total int) *tablewriter.Table {
	table := tablewriter.NewTable(os.Stdout, tablewriter.WithHeaderAutoFormat(tw.Off))
	table.Header([]string{"itemID", "path", "type", "delete at"})
	table.Footer([]string{"", "", "", "total count: " + strconv.Itoa(total)})
	return table
}

func cliLogger(verbose bool) zlog.Logger {
	logLvl := zerolog.ErrorLevel
	if verbose {
		logLvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339, NoColor: true}
	return zlog.Logger{zerolog.New(output).With().Timestamp().Logger().Level(logLvl)}
}
