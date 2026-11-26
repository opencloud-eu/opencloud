package command

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/opencloud-eu/opencloud/opencloud/pkg/register"
	"github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/config/configlog"
	"github.com/opencloud-eu/opencloud/pkg/config/parser"
	mregistry "github.com/opencloud-eu/opencloud/pkg/registry"
	sharing "github.com/opencloud-eu/opencloud/services/sharing/pkg/config"
	sharingparser "github.com/opencloud-eu/opencloud/services/sharing/pkg/config/parser"
	"github.com/opencloud-eu/reva/v3/pkg/auth"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	sdk "github.com/opencloud-eu/reva/v3/pkg/sdk/common"
	"github.com/opencloud-eu/reva/v3/pkg/share"
	"github.com/opencloud-eu/reva/v3/pkg/share/manager/jsoncs3"
	migration "github.com/opencloud-eu/reva/v3/pkg/share/manager/jsoncs3/migrations"
	"github.com/opencloud-eu/reva/v3/pkg/share/manager/registry"
	"github.com/opencloud-eu/reva/v3/pkg/storage/utils/walker"
	"github.com/opencloud-eu/reva/v3/pkg/storagespace"
	"github.com/opencloud-eu/reva/v3/pkg/utils"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	storageregistry "github.com/cs3org/go-cs3apis/cs3/storage/registry/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

// need to be discussed, for now I will let it here
//
// Problem:
// in reva on CleanupStaleShares call there is migrations invokation, which at leat in tests takes some time,
// reva code was modified to wait until migrations are done, to prevent cases when migrations are stuck and the
// this executions is not returned this timeout is needed
const cleanupTimeout = 1 * time.Minute

// migrationTimeout is the default bound of a forced migration run. Importing the
// grants of many project spaces is not fast, so this is generous.
const migrationTimeout = 30 * time.Minute

const (
	jsoncs3Driver            = "jsoncs3"
	serviceAccountIDFlag     = "service-account-id"
	serviceAccountSecretFlag = "service-account-secret"
)

// SharesCommand is the entrypoint for the groups command.
func SharesCommand(cfg *config.Config) *cobra.Command {
	sharesCmd := &cobra.Command{
		Use:   "shares",
		Short: `cli tools to manage entries in the share manager.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Parse base config
			if err := parser.ParseConfig(cfg, true); err != nil {
				return configlog.ReturnError(err)
			}

			// Parse sharing config
			cfg.Sharing.Commons = cfg.Commons
			return configlog.ReturnError(sharingparser.ParseConfig(cfg.Sharing))
		},
	}
	sharesCmd.AddCommand(cleanupCmd(cfg))
	sharesCmd.AddCommand(cleanOrphanedGrantsCmd(cfg))
	sharesCmd.AddCommand(migrateCmd(cfg))

	return sharesCmd
}

func init() {
	register.AddCommand(SharesCommand)
}

func cleanupCmd(cfg *config.Config) *cobra.Command {
	cleanCmd := &cobra.Command{
		Use:   "cleanup",
		Short: `clean up stale entries in the share manager.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Parse base config
			if err := parser.ParseConfig(cfg, true); err != nil {
				return configlog.ReturnError(err)
			}

			// Parse sharing config
			cfg.Sharing.Commons = cfg.Commons
			return configlog.ReturnError(sharingparser.ParseConfig(cfg.Sharing))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cleanup(cmd, cfg)
		},
	}
	cleanCmd.Flags().String(serviceAccountIDFlag, "", "Name of the service account to use for the cleanup")
	_ = cleanCmd.MarkFlagRequired(serviceAccountIDFlag)
	_ = viper.BindEnv(serviceAccountIDFlag, "OC_SERVICE_ACCOUNT_ID")
	_ = viper.BindPFlag(serviceAccountIDFlag, cleanCmd.Flags().Lookup(serviceAccountIDFlag))

	cleanCmd.Flags().String(serviceAccountSecretFlag, "", "Secret for the service account")
	_ = cleanCmd.MarkFlagRequired(serviceAccountSecretFlag)
	_ = viper.BindEnv(serviceAccountSecretFlag, "OC_SERVICE_ACCOUNT_SECRET")
	_ = viper.BindPFlag(serviceAccountSecretFlag, cleanCmd.Flags().Lookup(serviceAccountSecretFlag))

	return cleanCmd
}

func cleanup(_ *cobra.Command, cfg *config.Config) error {
	driver := cfg.Sharing.UserSharingDriver
	// cleanup is only implemented for the jsoncs3 share manager
	if driver != jsoncs3Driver {
		return configlog.ReturnError(errors.New("cleanup is only implemented for the jsoncs3 share manager"))
	}

	l := logger("migrate")

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	mgr, err := jsoncs3Manager(cfg, driver, &l)
	if err != nil {
		return configlog.ReturnError(err)
	}

	// Initialize registry to make service lookup work
	_ = mregistry.GetRegistry()

	// get an authenticated context
	gatewaySelector, err := pool.GatewaySelector(cfg.Sharing.Reva.Address)
	if err != nil {
		return configlog.ReturnError(err)
	}

	client, err := gatewaySelector.Next()
	if err != nil {
		return configlog.ReturnError(err)
	}

	serviceAccountID := viper.GetString(serviceAccountIDFlag)
	serviceAccountSecret := viper.GetString(serviceAccountSecretFlag)
	serviceUserCtx, err := utils.GetServiceUserContext(serviceAccountID, client, serviceAccountSecret)
	if err != nil {
		return configlog.ReturnError(err)
	}
	serviceUserCtx = l.WithContext(serviceUserCtx)

	cleanupCtx, cancel := context.WithTimeout(serviceUserCtx, cleanupTimeout)
	defer cancel()
	if err := mgr.CleanupStaleShares(cleanupCtx); err != nil {
		return configlog.ReturnError(err)
	}

	return nil
}

func cleanOrphanedGrantsCmd(cfg *config.Config) *cobra.Command {
	cleanOrphanedCmd := &cobra.Command{
		Use:   "clean-orphaned-grants",
		Short: `Detect and clean orphaned share-manager grants.`,
		Long: `Reports storage grants of users and groups that have no share in the share
manager, and with --shares-without-grants also shares that have no storage grant
for their grantee. With --dry-run=false they are removed.

The dry run is not free of side effects: listing the shares of a resource makes
the jsoncs3 share manager remove expired shares and send the ShareExpired and
SpaceMembershipExpired events for them.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if force, _ := cmd.Flags().GetBool("force"); force && !cmd.Flags().Changed("resource-id") {
				return errors.New("--force needs --resource-id")
			}

			// bound when this command runs, not when it is built: the cleanup command binds the same keys
			_ = viper.BindPFlag(serviceAccountIDFlag, cmd.Flags().Lookup(serviceAccountIDFlag))
			_ = viper.BindPFlag(serviceAccountSecretFlag, cmd.Flags().Lookup(serviceAccountSecretFlag))

			if err := parser.ParseConfig(cfg, true); err != nil {
				return configlog.ReturnError(err)
			}

			cfg.Sharing.Commons = cfg.Commons
			return configlog.ReturnError(sharingparser.ParseConfig(cfg.Sharing))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cleanOrphanedGrants(cmd, cfg)
		},
	}
	cleanOrphanedCmd.Flags().String(serviceAccountIDFlag, "", "Name of the service account to use for the scan")
	_ = cleanOrphanedCmd.MarkFlagRequired(serviceAccountIDFlag)
	_ = viper.BindEnv(serviceAccountIDFlag, "OC_SERVICE_ACCOUNT_ID")

	cleanOrphanedCmd.Flags().String(serviceAccountSecretFlag, "", "Secret for the service account")
	_ = cleanOrphanedCmd.MarkFlagRequired(serviceAccountSecretFlag)
	_ = viper.BindEnv(serviceAccountSecretFlag, "OC_SERVICE_ACCOUNT_SECRET")

	cleanOrphanedCmd.Flags().String("space-id", "", "Limit the scan to one storage space, given as <storage-id>$<space-id>")
	cleanOrphanedCmd.Flags().Bool("dry-run", true, "Dry run mode enabled")
	cleanOrphanedCmd.Flags().String("resource-id", "", "Limit the scan to one resource, given as <storage-id>$<space-id>!<node-id>")
	cleanOrphanedCmd.Flags().Bool("force", false, "Remove the grants of --resource-id that have no share even when its shares cannot be listed")
	cleanOrphanedCmd.MarkFlagsMutuallyExclusive("space-id", "resource-id")
	cleanOrphanedCmd.Flags().Bool("shares-without-grants", false, "Also report shares without a storage grant for their grantee, and remove them with --dry-run=false (jsoncs3 only)")
	cleanOrphanedCmd.Flags().BoolP("verbose", "v", false, "Verbose logging enabled")

	return cleanOrphanedCmd
}

func cleanOrphanedGrants(cmd *cobra.Command, cfg *config.Config) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	verbose, _ := cmd.Flags().GetBool("verbose")
	spaceID, _ := cmd.Flags().GetString("space-id")
	resourceIDFlag, _ := cmd.Flags().GetString("resource-id")
	force, _ := cmd.Flags().GetBool("force")
	sharesWithoutGrants, _ := cmd.Flags().GetBool("shares-without-grants")
	if sharesWithoutGrants && cfg.Sharing.UserSharingDriver != jsoncs3Driver {
		return configlog.ReturnError(errors.New("--shares-without-grants is only implemented for the jsoncs3 share manager"))
	}
	var resourceID *provider.ResourceId
	if resourceIDFlag != "" {
		rid, err := parseResourceID(resourceIDFlag)
		if err != nil {
			return configlog.ReturnError(err)
		}
		resourceID = rid
		spaceID = storagespace.FormatStorageID(rid.GetStorageId(), rid.GetSpaceId())
	}
	serviceAccountID := viper.GetString(serviceAccountIDFlag)
	serviceAccountSecret := viper.GetString(serviceAccountSecretFlag)

	revaOpts := append(cfg.Sharing.Reva.GetRevaOptions(), pool.WithRegistry(mregistry.GetRegistry()))
	// Pre-flight: surface execution mode and flags before the main scan.
	fmt.Printf("\n== %s ==\n", "Pre-flight")
	if dryRun {
		fmt.Println("mode: dry run enabled")
	} else {
		fmt.Println("mode: dry run disabled: grants may be changed")
	}
	switch {
	case resourceID != nil:
		fmt.Printf("scope: limiting scan to resource %s\n", resourceIDFlag)
	case spaceID != "":
		fmt.Printf("scope: limiting scan to space %s\n", spaceID)
	}
	if force {
		fmt.Println("flags: --force active (will remove grants of the resource even on share list errors)")
	}

	gatewaySelector, err := pool.GatewaySelector(cfg.Sharing.Reva.Address, revaOpts...)
	if err != nil {
		return configlog.ReturnError(err)
	}

	client, err := gatewaySelector.Next()
	if err != nil {
		return configlog.ReturnError(err)
	}

	serviceUserCtx, err := utils.GetServiceUserContext(serviceAccountID, client, serviceAccountSecret)
	if err != nil {
		return configlog.ReturnError(err)
	}

	if !dryRun {
		if err := cleanupStaleShares(serviceUserCtx, cfg); err != nil {
			return configlog.ReturnError(err)
		}
	}

	storageRegistrySelector, err := pool.StorageRegistrySelector(cfg.Sharing.Reva.Address, revaOpts...)
	if err != nil {
		return configlog.ReturnError(err)
	}

	spaces, err := resolveTargetSpaces(serviceUserCtx, gatewaySelector, spaceID)
	if err != nil {
		return configlog.ReturnError(err)
	}
	if len(spaces) == 0 {
		fmt.Println("No storage spaces found")
		fmt.Println("Nothing to do")
		return nil
	}
	fmt.Printf("%d target space(s)\n", len(spaces))

	fmt.Printf("\n== %s ==\n", "Primary scan")
	stats := scanAndCleanOrphansForSpaces(serviceUserCtx, orphanScanArgs{
		GatewaySelector:         gatewaySelector,
		StorageRegistrySelector: storageRegistrySelector,
		RevaOpts:                revaOpts,
		Spaces:                  spaces,
		ResourceID:              resourceID,
		DryRun:                  dryRun,
		Verbose:                 verbose,
		Force:                   force,
	})
	fmt.Printf("Summary: %d space(s), %d node(s) visited, %d nodes with grants, %d grants total\n", len(spaces), stats.TotalNodes, stats.TotalNodesWithGrants, stats.TotalGrants)
	if stats.Unverified > 0 {
		fmt.Printf("Unverified: %d grant(s) skipped because the shares of their resource could not be listed\n", stats.Unverified)
	}
	if dryRun {
		fmt.Println("Dry run mode: no grants were modified")
	} else {
		fmt.Printf("Orphans: %d candidate(s), %d removed, %d errors\n", stats.OrphanCandidates, stats.OrphanDeleted, stats.OrphanDeleteErrors)
		if stats.OrphanDeleted > 0 {
			if err := cleanupStaleShares(serviceUserCtx, cfg); err != nil {
				return configlog.ReturnError(err)
			}
		}
	}

	if !sharesWithoutGrants {
		return nil
	}

	fmt.Printf("\n== %s ==\n", "Reverse orphan scan")
	l := logger("clean-orphaned-grants")
	mgr, err := jsoncs3Manager(cfg, jsoncs3Driver, &l)
	if err != nil {
		return configlog.ReturnError(err)
	}
	reverseStats := postprocessCleanReverseOrphans(serviceUserCtx, reverseOrphanArgs{
		GatewaySelector:         gatewaySelector,
		StorageRegistrySelector: storageRegistrySelector,
		RevaOpts:                revaOpts,
		Spaces:                  spaces,
		ResourceID:              resourceID,
		ListSpaceShares:         spaceShareLister(mgr),
		DryRun:                  dryRun,
		Verbose:                 verbose,
	})
	if !dryRun {
		fmt.Printf("Reverse orphans: %d candidate(s), %d removed, %d errors\n", reverseStats.Candidates, reverseStats.Deleted, reverseStats.Errors)
	}

	return nil
}

// OrphanStats aggregates counters for the main orphan scan pass.
type OrphanStats struct {
	TotalNodes           int
	TotalNodesWithGrants int
	TotalGrants          int
	OrphanCandidates     int
	OrphanDeleted        int
	OrphanDeleteErrors   int
	Unverified           int
}

// ReverseOrphanStats aggregates counters for the reverse orphan scan pass.
type ReverseOrphanStats struct {
	Candidates int
	Deleted    int
	Errors     int
}

// cleanupStaleShares removes the shares of deleted resources from the jsoncs3 share manager.
func cleanupStaleShares(ctx context.Context, cfg *config.Config) error {
	if cfg.Sharing.UserSharingDriver != jsoncs3Driver {
		return nil
	}
	l := logger("clean-orphaned-grants")
	mgr, err := jsoncs3Manager(cfg, jsoncs3Driver, &l)
	if err != nil {
		return fmt.Errorf("stale share cleanup: %w", err)
	}
	cleanupCtx, cancel := context.WithTimeout(l.WithContext(ctx), cleanupTimeout)
	defer cancel()
	if err := mgr.CleanupStaleShares(cleanupCtx); err != nil {
		return fmt.Errorf("stale share cleanup: %w", err)
	}
	return nil
}

type orphanScanArgs struct {
	GatewaySelector         pool.Selectable[gateway.GatewayAPIClient]
	StorageRegistrySelector pool.Selectable[storageregistry.RegistryAPIClient]
	RevaOpts                []pool.Option
	Spaces                  []*provider.StorageSpace
	ResourceID              *provider.ResourceId
	DryRun, Verbose, Force  bool
}

// scanAndCleanOrphansForSpaces performs the main pass to detect and remove orphan storage grants.
func scanAndCleanOrphansForSpaces(
	ctx context.Context,
	args orphanScanArgs,
) OrphanStats {
	stats := OrphanStats{}
	for idx, space := range args.Spaces {
		rootResourceID := space.GetRoot()
		identifier := describeSpace(space)
		if space.GetRoot() == nil {
			fmt.Printf("[%d/%d] Skipping space %s: missing root resource\n", idx+1, len(args.Spaces), identifier)
			continue
		}

		nodeCount := 0
		nodesWithGrants := 0
		grantCount := 0
		spaceProvidersCache := make(map[string][]*storageregistry.ProviderInfo)
		err := walkTarget(ctx, args.GatewaySelector, space.GetRoot(), args.ResourceID, func(wd string, info *provider.ResourceInfo, walkErr error) error {
			pathLabel := renderWalkerPath(wd, info)
			if walkErr != nil {
				fmt.Printf("  Warning: failed to access %s: %v\n", pathLabel, walkErr)
				return nil
			}

			if info == nil || info.GetId() == nil {
				fmt.Printf("  Warning: missing resource identifier for %s\n", pathLabel)
				return nil
			}

			nodeCount++

			grants, grantErr := listResourceGrants(ctx, args.StorageRegistrySelector, info.GetId(), args.RevaOpts, spaceProvidersCache)
			if grantErr != nil {
				fmt.Printf("  Warning: failed to list grants for %s: %v\n", pathLabel, grantErr)
				return nil
			}

			if len(grants) == 0 {
				if args.Verbose {
					fmt.Printf("  %s\n", pathLabel)
				}
				return nil
			}

			shares, shareErr := listSharesForResource(ctx, args.GatewaySelector, info.GetId())
			if shareErr != nil {
				fmt.Printf("  Warning: failed to list shares for %s: %v\n", pathLabel, shareErr)
			}

			nodesWithGrants++
			grantCount += len(grants)
			fmt.Printf("  %s -> %d grant(s)\n", pathLabel, len(grants))
			if args.Verbose {
				for _, g := range grants {
					fmt.Printf("    - %s\n", describeGrant(g))
				}
			}

			for _, g := range grants {
				if gr := g.GetGrantee(); gr != nil {
					if gr.GetType() != provider.GranteeType_GRANTEE_TYPE_USER && gr.GetType() != provider.GranteeType_GRANTEE_TYPE_GROUP {
						if args.Verbose {
							fmt.Printf("    SKIP(non-user/group): %s\n", describeGrant(g))
						}
						continue
					}
				}
				if resourceIDEqual(info.GetId(), rootResourceID) {
					if args.Verbose {
						fmt.Printf("    SKIP(root): %s\n", describeGrant(g))
					}
					continue
				}
				if shareErr != nil && (!args.Force || args.ResourceID == nil) {
					fmt.Printf("    UNVERIFIED: %s (shares could not be listed)\n", describeGrant(g))
					stats.Unverified++
					continue
				}
				if hasMatchingShareForGrant(shares, g) {
					continue
				}
				stats.OrphanCandidates++
				if args.DryRun {
					fmt.Printf("    ORPHAN: %s\n", describeGrant(g))
					continue
				}
				if err := removeGrantForResource(ctx, args.StorageRegistrySelector, info.GetId(), g, args.RevaOpts); err != nil {
					fmt.Printf("    REMOVE-FAILED: %s error=%v\n", describeGrant(g), err)
					stats.OrphanDeleteErrors++
					continue
				}
				fmt.Printf("    REMOVED: %s\n", describeGrant(g))
				stats.OrphanDeleted++
			}
			return nil
		})
		if err != nil {
			fmt.Printf("[%d/%d] Failed to walk space %s: %v\n", idx+1, len(args.Spaces), identifier, err)
			continue
		}

		stats.TotalNodes += nodeCount
		stats.TotalNodesWithGrants += nodesWithGrants
		stats.TotalGrants += grantCount
		fmt.Printf("[%d/%d] Space %s: %d nodes visited, %d nodes with grants, %d grants total\n", idx+1, len(args.Spaces), identifier, nodeCount, nodesWithGrants, grantCount)
	}
	return stats
}

type reverseOrphanArgs struct {
	GatewaySelector         pool.Selectable[gateway.GatewayAPIClient]
	StorageRegistrySelector pool.Selectable[storageregistry.RegistryAPIClient]
	RevaOpts                []pool.Option
	Spaces                  []*provider.StorageSpace
	ResourceID              *provider.ResourceId
	ListSpaceShares         func(ctx context.Context, storageID, spaceID string) ([]*collaboration.Share, error)
	DryRun, Verbose         bool
}

// spaceShareLister lists all shares of a space from the share manager cache. Unlike
// ListShares it does not expire shares on the way.
func spaceShareLister(mgr *jsoncs3.Manager) func(ctx context.Context, storageID, spaceID string) ([]*collaboration.Share, error) {
	return func(ctx context.Context, storageID, spaceID string) ([]*collaboration.Share, error) {
		readyCtx, cancel := context.WithTimeout(ctx, cleanupTimeout)
		defer cancel()
		select {
		case <-mgr.Ready():
		case <-readyCtx.Done():
			return nil, fmt.Errorf("share manager not ready: %w", readyCtx.Err())
		}
		list, err := mgr.Cache.ListSpace(ctx, storageID, spaceID)
		if err != nil {
			return nil, err
		}
		shares := make([]*collaboration.Share, 0, len(list.Shares))
		for _, s := range list.Shares {
			shares = append(shares, s)
		}
		return shares, nil
	}
}

// postprocessCleanReverseOrphans removes shares that have no storage grant for their grantee.
// It lists the shares of each space once and checks the grants of their resources only.
func postprocessCleanReverseOrphans(
	ctx context.Context,
	args reverseOrphanArgs,
) ReverseOrphanStats {
	stats := ReverseOrphanStats{}
	for idx, space := range args.Spaces {
		root := space.GetRoot()
		identifier := describeSpace(space)
		if root == nil {
			continue
		}
		shares, err := args.ListSpaceShares(ctx, root.GetStorageId(), root.GetSpaceId())
		if err != nil {
			fmt.Printf("[%d/%d] Warning: failed to list shares of space %s: %v\n", idx+1, len(args.Spaces), identifier, err)
			continue
		}

		byResource := make(map[string][]*collaboration.Share)
		for _, s := range shares {
			rid := s.GetResourceId()
			if rid == nil || rid.GetOpaqueId() == rid.GetSpaceId() || share.IsExpired(s) {
				continue
			}
			if args.ResourceID != nil && !resourceIDEqual(rid, args.ResourceID) {
				continue
			}
			key := storagespace.FormatResourceID(rid)
			byResource[key] = append(byResource[key], s)
		}

		spaceProvidersCache := make(map[string][]*storageregistry.ProviderInfo)
		for _, key := range slices.Sorted(maps.Keys(byResource)) {
			resourceShares := byResource[key]
			grants, grantsErr := listResourceGrants(ctx, args.StorageRegistrySelector, resourceShares[0].GetResourceId(), args.RevaOpts, spaceProvidersCache)
			if grantsErr != nil {
				fmt.Printf("  Warning: failed to list grants for %s: %v\n", key, grantsErr)
				continue
			}
			if args.Verbose {
				fmt.Printf("  %s -> %d share(s), %d grant(s)\n", key, len(resourceShares), len(grants))
			}
			for _, s := range resourceShares {
				granteeType := s.GetGrantee().GetType()
				if granteeType != provider.GranteeType_GRANTEE_TYPE_USER && granteeType != provider.GranteeType_GRANTEE_TYPE_GROUP {
					continue
				}
				if hasMatchingGrantForShare(grants, s) {
					continue
				}
				stats.Candidates++
				shareID := s.GetId().GetOpaqueId()
				granteeLabel := describeGrantee(s.GetGrantee())
				fmt.Printf("  WARN REV-ORPHAN: %s -> share:%s %s (no matching storage grant)\n", key, shareID, granteeLabel)
				if args.DryRun {
					continue
				}
				if err := removeShare(ctx, args.GatewaySelector, s); err != nil {
					fmt.Printf("    REV-REMOVE-FAILED share:%s %s error=%v\n", shareID, granteeLabel, err)
					stats.Errors++
				} else {
					fmt.Printf("    REV-REMOVED share:%s %s\n", shareID, granteeLabel)
					stats.Deleted++
				}
			}
		}
		fmt.Printf("[%d/%d] Space %s: %d shared resource(s) checked\n", idx+1, len(args.Spaces), identifier, len(byResource))
	}
	return stats
}

// walkTarget walks the tree below root, or visits only resourceID when it is set.
func walkTarget(ctx context.Context, gatewaySelector pool.Selectable[gateway.GatewayAPIClient], root, resourceID *provider.ResourceId, fn walker.WalkFunc) error {
	if resourceID == nil {
		return walker.NewWalker(gatewaySelector).Walk(auth.NewStaticSession(ctx), root, fn)
	}
	client, err := gatewaySelector.Next()
	if err != nil {
		return err
	}
	res, err := client.Stat(ctx, &provider.StatRequest{Ref: &provider.Reference{ResourceId: resourceID}})
	if err != nil {
		return err
	}
	if res.GetStatus().GetCode() != rpc.Code_CODE_OK {
		return fmt.Errorf("stat %s failed: %s", storagespace.FormatResourceID(resourceID), res.GetStatus().GetMessage())
	}
	return fn("", res.GetInfo(), nil)
}

// removeShare removes a collaboration share via the gateway, given its Share object.
func removeShare(ctx context.Context, gatewaySelector pool.Selectable[gateway.GatewayAPIClient], share *collaboration.Share) error {
	if share == nil || share.GetId() == nil {
		return errors.New("missing share id")
	}
	client, err := gatewaySelector.Next()
	if err != nil {
		return err
	}
	req := &collaboration.RemoveShareRequest{
		Ref: &collaboration.ShareReference{
			Spec: &collaboration.ShareReference_Id{
				Id: share.GetId(),
			},
		},
	}
	res, err := client.RemoveShare(ctx, req)
	if err != nil {
		return err
	}
	if res.GetStatus().GetCode() != rpc.Code_CODE_OK {
		return fmt.Errorf("remove share failed: %s", res.GetStatus().GetMessage())
	}
	return nil
}

// resourceIDEqual compares two provider.ResourceId values by fields.
func resourceIDEqual(a, b *provider.ResourceId) bool {
	if a == nil || b == nil {
		return false
	}
	return a.GetStorageId() == b.GetStorageId() &&
		a.GetSpaceId() == b.GetSpaceId() &&
		a.GetOpaqueId() == b.GetOpaqueId()
}

// listResourceGrants fetches grants for a resource, reusing storage provider
// resolution per space via the provided cache (keyed by space_id). It fails
// when no storage provider answers, so that callers never mistake that for a
// resource without grants.
func listResourceGrants(ctx context.Context, registrySelector pool.Selectable[storageregistry.RegistryAPIClient], resourceID *provider.ResourceId, revaOpts []pool.Option, providerCache map[string][]*storageregistry.ProviderInfo) ([]*provider.Grant, error) {
	if resourceID == nil {
		return nil, errors.New("missing resource id")
	}

	spaceID := resourceID.GetSpaceId()
	providerInfos, ok := providerCache[spaceID]
	if !ok {
		registryClient, err := registrySelector.Next()
		if err != nil {
			return nil, err
		}

		filters := map[string]string{
			"storage_id": resourceID.GetStorageId(),
			"space_id":   spaceID,
			"opaque_id":  resourceID.GetOpaqueId(),
		}
		listReq := &storageregistry.ListStorageProvidersRequest{Opaque: &types.Opaque{}}
		sdk.EncodeOpaqueMap(listReq.Opaque, filters)

		providersResp, err := registryClient.ListStorageProviders(ctx, listReq)
		if err != nil {
			return nil, err
		}
		if providersResp.GetStatus().GetCode() != rpc.Code_CODE_OK {
			return nil, fmt.Errorf("list storage providers failed: %s", providersResp.GetStatus().GetMessage())
		}
		providerInfos = providersResp.GetProviders()
		if len(providerInfos) > 0 {
			providerCache[spaceID] = providerInfos
		}
	}

	ref := &provider.Reference{ResourceId: resourceID}
	grants := make([]*provider.Grant, 0)
	queried := 0
	for _, info := range providerInfos {
		if info.GetAddress() == "" {
			continue
		}
		queried++
		provClient, err := pool.GetStorageProviderServiceClient(info.GetAddress(), revaOpts...)
		if err != nil {
			return nil, err
		}

		resp, err := provClient.ListGrants(ctx, &provider.ListGrantsRequest{Ref: ref})
		if err != nil {
			return nil, err
		}
		if resp.GetStatus().GetCode() != rpc.Code_CODE_OK {
			return nil, fmt.Errorf("list grants failed: %s", resp.GetStatus().GetMessage())
		}
		grants = append(grants, resp.GetGrants()...)
	}
	if queried == 0 {
		return nil, fmt.Errorf("no storage provider found for space %s", spaceID)
	}

	return grants, nil
}

func listSharesForResource(ctx context.Context, gatewaySelector pool.Selectable[gateway.GatewayAPIClient], resourceID *provider.ResourceId) ([]*collaboration.Share, error) {
	// ListShares filtered by resource id. This reads the share-manager indexes
	// to determine if a share exists that matches a storage grant.
	client, err := gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	req := &collaboration.ListSharesRequest{
		Filters: []*collaboration.Filter{
			{
				Type: collaboration.Filter_TYPE_RESOURCE_ID,
				Term: &collaboration.Filter_ResourceId{ResourceId: resourceID},
			},
		},
	}
	res, err := client.ListShares(ctx, req)
	if err != nil {
		return nil, err
	}
	if res.GetStatus().GetCode() != rpc.Code_CODE_OK {
		return nil, fmt.Errorf("list shares failed: %s", res.GetStatus().GetMessage())
	}
	return res.GetShares(), nil
}

func removeGrantForResource(ctx context.Context, registrySelector pool.Selectable[storageregistry.RegistryAPIClient], resourceID *provider.ResourceId, grant *provider.Grant, revaOpts []pool.Option) error {
	if resourceID == nil || grant == nil || grant.Grantee == nil {
		return errors.New("missing resource or grant")
	}
	registryClient, err := registrySelector.Next()
	if err != nil {
		return err
	}
	listReq := &storageregistry.ListStorageProvidersRequest{Opaque: &types.Opaque{}}
	sdk.EncodeOpaqueMap(listReq.Opaque, map[string]string{
		"storage_id": resourceID.GetStorageId(),
		"space_id":   resourceID.GetSpaceId(),
		"opaque_id":  resourceID.GetOpaqueId(),
	})
	providersResp, err := registryClient.ListStorageProviders(ctx, listReq)
	if err != nil {
		return err
	}
	if providersResp.GetStatus().GetCode() != rpc.Code_CODE_OK || len(providersResp.GetProviders()) == 0 {
		return fmt.Errorf("resolve provider failed: %s", providersResp.GetStatus().GetMessage())
	}
	ref := &provider.Reference{ResourceId: resourceID}
	for _, info := range providersResp.GetProviders() {
		if info.GetAddress() == "" {
			continue
		}
		provClient, err := pool.GetStorageProviderServiceClient(info.GetAddress(), revaOpts...)
		if err != nil {
			return err
		}
		_, err = provClient.RemoveGrant(ctx, &provider.RemoveGrantRequest{Ref: ref, Grant: grant})
		if err != nil {
			return err
		}
		// We consider success on first provider answering OK
		return nil
	}
	return errors.New("no provider with address")
}

func hasMatchingShareForGrant(shares []*collaboration.Share, g *provider.Grant) bool {
	// Compare grantee identity between storage grant and share-manager share.
	// ResourceId already matched by the ListShares filter.
	for _, s := range shares {
		// resource already filtered; compare grantee
		sg := s.GetGrantee()
		if sg == nil || g == nil || g.Grantee == nil {
			continue
		}
		if sg.GetType() != g.Grantee.GetType() {
			continue
		}
		switch sg.GetType() {
		case provider.GranteeType_GRANTEE_TYPE_USER:
			if sg.GetUserId() != nil && g.Grantee.GetUserId() != nil && utils.UserIDEqual(sg.GetUserId(), g.Grantee.GetUserId()) {
				return true
			}
		case provider.GranteeType_GRANTEE_TYPE_GROUP:
			if sg.GetGroupId() != nil && g.Grantee.GetGroupId() != nil && sg.GetGroupId().GetOpaqueId() == g.Grantee.GetGroupId().GetOpaqueId() {
				return true
			}
		}
	}
	return false
}

func hasMatchingGrantForShare(grants []*provider.Grant, s *collaboration.Share) bool {
	for _, g := range grants {
		if hasMatchingShareForGrant([]*collaboration.Share{s}, g) {
			return true
		}
	}
	return false
}

func resolveTargetSpaces(ctx context.Context, selector pool.Selectable[gateway.GatewayAPIClient], spaceID string) ([]*provider.StorageSpace, error) {
	client, err := selector.Next()
	if err != nil {
		return nil, err
	}

	req := &provider.ListStorageSpacesRequest{
		Opaque: utils.AppendPlainToOpaque(nil, "unrestricted", strconv.FormatBool(true)),
	}
	if spaceID != "" {
		id, err := parseSpaceID(spaceID)
		if err != nil {
			return nil, err
		}
		req.Filters = []*provider.ListStorageSpacesRequest_Filter{
			{
				Type: provider.ListStorageSpacesRequest_Filter_TYPE_ID,
				Term: &provider.ListStorageSpacesRequest_Filter_Id{Id: &provider.StorageSpaceId{OpaqueId: id}},
			},
		}
	}

	resp, err := client.ListStorageSpaces(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.GetStatus().GetCode() != rpc.Code_CODE_OK {
		return nil, fmt.Errorf("list storage spaces failed: %s", resp.GetStatus().GetMessage())
	}
	if spaceID != "" && len(resp.GetStorageSpaces()) == 0 {
		return nil, fmt.Errorf("space not found: %s", spaceID)
	}

	return resp.GetStorageSpaces(), nil
}

func parseSpaceID(spaceID string) (string, error) {
	rid, err := storagespace.ParseID(spaceID)
	if err != nil || rid.GetStorageId() == "" || rid.GetSpaceId() == "" || (rid.GetOpaqueId() != "" && rid.GetOpaqueId() != rid.GetSpaceId()) {
		return "", fmt.Errorf("invalid space id %q, expected <storage-id>$<space-id>", spaceID)
	}
	return storagespace.FormatStorageID(rid.GetStorageId(), rid.GetSpaceId()), nil
}

func parseResourceID(resourceID string) (*provider.ResourceId, error) {
	rid, err := storagespace.ParseID(resourceID)
	if err != nil || rid.GetStorageId() == "" || rid.GetSpaceId() == "" || rid.GetOpaqueId() == "" {
		return nil, fmt.Errorf("invalid resource id %q, expected <storage-id>$<space-id>!<node-id>", resourceID)
	}
	return &rid, nil
}

func describeSpace(space *provider.StorageSpace) string {
	if space == nil {
		return "<nil>"
	}
	spaceID := ""
	if space.GetId() != nil {
		spaceID = space.GetId().GetOpaqueId()
	}
	spaceName := space.GetName()
	if spaceName == "" {
		if spaceID == "" {
			return "<unknown>"
		}
		return spaceID
	}
	if spaceID == "" {
		return spaceName
	}
	return spaceName + " (" + spaceID + ")"
}

func describeGrant(grant *provider.Grant) string {
	if grant == nil {
		return "<nil>"
	}
	grantee := describeGrantee(grant.GetGrantee())
	perms := summarizePermissions(grant.GetPermissions())
	return grantee + " permissions=[" + perms + "]"
}

func describeGrantee(grantee *provider.Grantee) string {
	if grantee == nil {
		return "grantee:<nil>"
	}
	switch grantee.GetType() {
	case provider.GranteeType_GRANTEE_TYPE_USER:
		if grantee.GetUserId() == nil {
			return "user:<unknown>"
		}
		return "user:" + grantee.GetUserId().GetOpaqueId()
	case provider.GranteeType_GRANTEE_TYPE_GROUP:
		if grantee.GetGroupId() == nil {
			return "group:<unknown>"
		}
		return "group:" + grantee.GetGroupId().GetOpaqueId()
	default:
		return strings.ToLower(grantee.GetType().String())
	}
}

func summarizePermissions(p *provider.ResourcePermissions) string {
	if p == nil {
		return "none"
	}
	perms := make([]string, 0, 8)
	if p.GetAddGrant() {
		perms = append(perms, "add-grant")
	}
	if p.GetInitiateFileDownload() {
		perms = append(perms, "download")
	}
	if p.GetInitiateFileUpload() {
		perms = append(perms, "upload")
	}
	if p.GetListGrants() {
		perms = append(perms, "list-grants")
	}
	if p.GetListContainer() {
		perms = append(perms, "list")
	}
	if p.GetStat() {
		perms = append(perms, "stat")
	}
	if p.GetRemoveGrant() {
		perms = append(perms, "remove-grant")
	}
	if p.GetUpdateGrant() {
		perms = append(perms, "update-grant")
	}
	if len(perms) == 0 {
		return "none"
	}
	return strings.Join(perms, ",")
}

func renderWalkerPath(wd string, info *provider.ResourceInfo) string {
	base := strings.TrimPrefix(filepath.ToSlash(wd), "/")
	segment := ""
	if info != nil {
		segment = strings.TrimPrefix(filepath.ToSlash(info.GetPath()), "/")
		if segment == "." {
			segment = ""
		}
	}
	combined := path.Join(base, segment)
	combined = strings.TrimPrefix(combined, "/")
	if combined == "" {
		return "/"
	}
	return "/" + combined
}

func migrateCmd(cfg *config.Config) *cobra.Command {
	migrateCmd := &cobra.Command{
		Use: "migrate [migration]",
		Args: func(cmd *cobra.Command, args []string) error {
			if list, _ := cmd.Flags().GetBool("list"); list {
				return nil
			}
			if len(args) != 1 {
				return fmt.Errorf("name the migration to run, one of: %s", strings.Join(migration.Names(), ", "))
			}
			// reject an unknown name before touching the configuration
			if !slices.Contains(migration.Names(), args[0]) {
				return fmt.Errorf("unknown migration %q, one of: %s", args[0], strings.Join(migration.Names(), ", "))
			}
			return nil
		},
		Short: `re-run share manager migrations that have already been applied.`,
		Long: `Runs a share manager migration by name, whether or not it has been applied
before. On startup a migration runs only once; this runs the one you name.

The case this exists for is import_space_members: it reads the grants of every
project space from the storage provider and imports them into the share
manager. Space memberships that were written as storage grants by something
other than OpenCloud are invisible until that has run. The import is additive,
so running it again is safe.

Use --list to see the available migrations.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// listing the migrations needs no server configuration
			if list, _ := cmd.Flags().GetBool("list"); list {
				return nil
			}
			if err := parser.ParseConfig(cfg, true); err != nil {
				return configlog.ReturnError(err)
			}

			cfg.Sharing.Commons = cfg.Commons
			return configlog.ReturnError(sharingparser.ParseConfig(cfg.Sharing))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return migrate(cmd, cfg, args)
		},
	}

	migrateCmd.Flags().BoolP("list", "l", false, "list the available migrations and exit")
	migrateCmd.Flags().Duration("timeout", migrationTimeout, "abort the migration after this duration")
	_ = viper.BindPFlag("list", migrateCmd.Flags().Lookup("list"))

	return migrateCmd
}

func migrate(cmd *cobra.Command, cfg *config.Config, args []string) error {
	if viper.GetBool("list") {
		for _, name := range migration.Names() {
			fmt.Println(name)
		}
		return nil
	}

	driver := cfg.Sharing.UserSharingDriver
	if driver != "jsoncs3" {
		return configlog.ReturnError(errors.New("migrate is only implemented for the jsoncs3 share manager"))
	}

	l := logger("migrate")
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// Initialize registry to make service lookup work
	_ = mregistry.GetRegistry()

	mgr, err := jsoncs3Manager(cfg, driver, &l)
	if err != nil {
		return configlog.ReturnError(err)
	}

	timeout, err := cmd.Flags().GetDuration("timeout")
	if err != nil {
		return configlog.ReturnError(err)
	}
	ctx, cancel := context.WithTimeout(l.WithContext(context.Background()), timeout)
	defer cancel()

	return configlog.ReturnError(mgr.RunMigration(ctx, args[0]))
}

// jsoncs3Manager creates the share manager of the given driver and returns it as a jsoncs3 manager.
func jsoncs3Manager(cfg *config.Config, driver string, l *zerolog.Logger) (*jsoncs3.Manager, error) {
	f, ok := registry.NewFuncs[driver]
	if !ok {
		return nil, errors.New("Unknown share manager type '" + driver + "'")
	}
	d, ok := revaShareConfig(cfg.Sharing)[driver]
	if !ok {
		return nil, errors.New("no configuration for share manager type '" + driver + "'")
	}
	m, ok := d.(map[string]any)
	if !ok {
		return nil, errors.New("invalid configuration for share manager type '" + driver + "'")
	}
	mgr, err := f(m, l)
	if err != nil {
		return nil, err
	}
	jmgr, ok := mgr.(*jsoncs3.Manager)
	if !ok {
		return nil, errors.New("share manager type '" + driver + "' is not jsoncs3")
	}
	return jmgr, nil
}

func revaShareConfig(cfg *sharing.Config) map[string]any {
	return map[string]any{
		"json": map[string]any{
			"file":         cfg.UserSharingDrivers.JSON.File,
			"gateway_addr": cfg.Reva.Address,
		},
		"sql": map[string]any{ // cernbox sql
			"db_username":                   cfg.UserSharingDrivers.SQL.DBUsername,
			"db_password":                   cfg.UserSharingDrivers.SQL.DBPassword,
			"db_host":                       cfg.UserSharingDrivers.SQL.DBHost,
			"db_port":                       cfg.UserSharingDrivers.SQL.DBPort,
			"db_name":                       cfg.UserSharingDrivers.SQL.DBName,
			"password_hash_cost":            cfg.UserSharingDrivers.SQL.PasswordHashCost,
			"enable_expired_shares_cleanup": cfg.UserSharingDrivers.SQL.EnableExpiredSharesCleanup,
			"janitor_run_interval":          cfg.UserSharingDrivers.SQL.JanitorRunInterval,
		},
		"owncloudsql": map[string]any{
			"gateway_addr":     cfg.Reva.Address,
			"storage_mount_id": cfg.UserSharingDrivers.OwnCloudSQL.UserStorageMountID,
			"db_username":      cfg.UserSharingDrivers.OwnCloudSQL.DBUsername,
			"db_password":      cfg.UserSharingDrivers.OwnCloudSQL.DBPassword,
			"db_host":          cfg.UserSharingDrivers.OwnCloudSQL.DBHost,
			"db_port":          cfg.UserSharingDrivers.OwnCloudSQL.DBPort,
			"db_name":          cfg.UserSharingDrivers.OwnCloudSQL.DBName,
		},
		"cs3": map[string]any{
			"gateway_addr":        cfg.UserSharingDrivers.CS3.ProviderAddr,
			"provider_addr":       cfg.UserSharingDrivers.CS3.ProviderAddr,
			"service_user_id":     cfg.UserSharingDrivers.CS3.SystemUserID,
			"service_user_idp":    cfg.UserSharingDrivers.CS3.SystemUserIDP,
			"machine_auth_apikey": cfg.UserSharingDrivers.CS3.SystemUserAPIKey,
		},
		"jsoncs3": map[string]any{
			"gateway_addr":           cfg.Reva.Address,
			"provider_addr":          cfg.UserSharingDrivers.JSONCS3.ProviderAddr,
			"system_user_id":         cfg.UserSharingDrivers.JSONCS3.SystemUserID,
			"system_user_idp":        cfg.UserSharingDrivers.JSONCS3.SystemUserIDP,
			"machine_auth_apikey":    cfg.UserSharingDrivers.JSONCS3.SystemUserAPIKey,
			"service_account_id":     cfg.ServiceAccount.ServiceAccountID,
			"service_account_secret": cfg.ServiceAccount.ServiceAccountSecret,
		},
	}
}
