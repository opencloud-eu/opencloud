package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
	"github.com/opencloud-eu/reva/v3/pkg/share/manager/jsoncs3"
	migration "github.com/opencloud-eu/reva/v3/pkg/share/manager/jsoncs3/migrations"
	"github.com/opencloud-eu/reva/v3/pkg/share/manager/registry"
	"github.com/opencloud-eu/reva/v3/pkg/storage/utils/walker"
	"github.com/opencloud-eu/reva/v3/pkg/utils"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	storageregistry "github.com/cs3org/go-cs3apis/cs3/storage/registry/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/rs/zerolog"
	"github.com/shamaton/msgpack/v2"
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

// SharesCommand is the entrypoint for the groups command.
func SharesCommand(cfg *config.Config) *cobra.Command {
	sharesCmd := &cobra.Command{
		Use:   "shares",
		Short: `cli tools to manage entries in the share manager.`,
	}
	sharesCmd.AddCommand(cleanupCmd(cfg))
	sharesCmd.AddCommand(cleanOrphanedGrantsCmd(cfg))
	sharesCmd.AddCommand(migrateCmd(cfg))
	sharesCmd.AddCommand(moveStuckUploadBlobsCmd(cfg))

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
	cleanCmd.Flags().String("service-account-id", "", "Name of the service account to use for the cleanup")
	_ = cleanCmd.MarkFlagRequired("service-account-id")
	_ = viper.BindEnv("service-account-id", "OC_SERVICE_ACCOUNT_ID")
	_ = viper.BindPFlag("service-account-id", cleanCmd.Flags().Lookup("service-account-id"))

	cleanCmd.Flags().String("service-account-secret", "", "Secret for the service account")
	_ = cleanCmd.MarkFlagRequired("service-account-secret")
	_ = viper.BindEnv("service-account-secret", "OC_SERVICE_ACCOUNT_SECRET")
	_ = viper.BindPFlag("service-account-secret", cleanCmd.Flags().Lookup("service-account-secret"))

	return cleanCmd
}

func cleanup(_ *cobra.Command, cfg *config.Config) error {
	driver := cfg.Sharing.UserSharingDriver
	// cleanup is only implemented for the jsoncs3 share manager
	if driver != "jsoncs3" {
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

	serviceAccountIDFlag := viper.GetString("service-account-id")
	serviceAccountSecretFlag := viper.GetString("service-account-secret")
	serviceUserCtx, err := utils.GetServiceUserContext(serviceAccountIDFlag, client, serviceAccountSecretFlag)
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
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// bound when this command runs, not when it is built: the cleanup command binds the same keys
			_ = viper.BindPFlag("service-account-id", cmd.Flags().Lookup("service-account-id"))
			_ = viper.BindPFlag("service-account-secret", cmd.Flags().Lookup("service-account-secret"))

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
	cleanOrphanedCmd.Flags().String("service-account-id", "", "Name of the service account to use for the scan")
	_ = cleanOrphanedCmd.MarkFlagRequired("service-account-id")
	_ = viper.BindEnv("service-account-id", "OC_SERVICE_ACCOUNT_ID")

	cleanOrphanedCmd.Flags().String("service-account-secret", "", "Secret for the service account")
	_ = cleanOrphanedCmd.MarkFlagRequired("service-account-secret")
	_ = viper.BindEnv("service-account-secret", "OC_SERVICE_ACCOUNT_SECRET")

	cleanOrphanedCmd.Flags().String("space-id", "", "Limit the scan to a specific storage space (opaque ID)")
	cleanOrphanedCmd.Flags().Bool("dry-run", true, "Dry run mode enabled")
	cleanOrphanedCmd.Flags().Bool("force", false, "Force removal of suspected orphans even when listing shares fails")
	cleanOrphanedCmd.Flags().BoolP("verbose", "v", false, "Verbose logging enabled")

	return cleanOrphanedCmd
}

func cleanOrphanedGrants(cmd *cobra.Command, cfg *config.Config) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	verbose, _ := cmd.Flags().GetBool("verbose")
	spaceID, _ := cmd.Flags().GetString("space-id")
	force, _ := cmd.Flags().GetBool("force")

	revaOpts := append(cfg.Sharing.Reva.GetRevaOptions(), pool.WithRegistry(mregistry.GetRegistry()))

	gatewaySelector, err := pool.GatewaySelector(cfg.Sharing.Reva.Address, revaOpts...)
	if err != nil {
		return configlog.ReturnError(err)
	}

	client, err := gatewaySelector.Next()
	if err != nil {
		return configlog.ReturnError(err)
	}

	serviceUserCtx, err := utils.GetServiceUserContext(viper.GetString("service-account-id"), client, viper.GetString("service-account-secret"))
	if err != nil {
		return configlog.ReturnError(err)
	}

	// Mandatory stale cleanup: reconcile jsoncs3 caches before scanning
	if cfg.Sharing.UserSharingDriver == "jsoncs3" {
		rcfg := revaShareConfig(cfg.Sharing)
		if f, ok := registry.NewFuncs["jsoncs3"]; ok {
			l := logger("clean-orphaned-grants")
			if mgr, mErr := f(rcfg["jsoncs3"].(map[string]interface{}), &l); mErr == nil {
				zerolog.SetGlobalLevel(zerolog.InfoLevel)
				cleanCtx := l.WithContext(serviceUserCtx)
				mgr.(*jsoncs3.Manager).CleanupStaleShares(cleanCtx)
			}
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
		if dryRun {
			fmt.Println("Dry run mode enabled: no changes executed")
		}
		return nil
	}

	walker := walker.NewWalker(gatewaySelector)
	totalNodes := 0
	totalNodesWithGrants := 0
	totalGrants := 0
	orphanCandidates := 0
	orphanDeleted := 0
	orphanDeleteErrors := 0

	if dryRun {
		fmt.Println("Dry run mode enabled")
	} else {
		fmt.Println("Dry run disabled: later stages will alter the grants")
	}

	for idx, space := range spaces {
		// Copy loop variable to avoid Go for-range capture: otherwise the callback would
		// reference the final value of 'space' for all iterations; pin this iteration's space/root.
		currentSpace := space
		rootResourceID := currentSpace.GetRoot()
		identifier := describeSpace(space)
		if space.GetRoot() == nil {
			fmt.Printf("[%d/%d] Skipping space %s: missing root resource\n", idx+1, len(spaces), identifier)
			continue
		}

		nodeCount := 0
		nodesWithGrants := 0
		grantCount := 0
		// Cache storage providers per space during this walk to avoid repeated
		// ListStorageProviders calls for every node in the same space.
		// Gateway already has a provider cache, this reduces roundtrips further.
		spaceProvidersCache := make(map[string][]*storageregistry.ProviderInfo)
		err = walker.Walk(auth.NewStaticSession(serviceUserCtx), space.GetRoot(), func(wd string, info *provider.ResourceInfo, walkErr error) error {
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

			// Pass A (per-node): Query storage grants. If none, do not touch share-manager.
			grants, grantErr := listResourceGrants(serviceUserCtx, storageRegistrySelector, info.GetId(), revaOpts, spaceProvidersCache)
			if grantErr != nil {
				fmt.Printf("  Warning: failed to list grants for %s: %v\n", pathLabel, grantErr)
				return nil
			}

			if len(grants) == 0 {
				if verbose {
					fmt.Printf("  %s\n", pathLabel)
				}
				return nil
			}

			// Pass B (same node, only when grants>0): Query share-manager for this resource and detect orphans.
			shares, shareErr := listSharesForResource(serviceUserCtx, gatewaySelector, info.GetId())
			if shareErr != nil && !dryRun && !force {
				fmt.Printf("  Warning: failed to list shares for %s: %v\n", pathLabel, shareErr)
				fmt.Printf("  Info: %s is too damaged to safely verify; use '--force' to remove suspected orphans regardless of share-manager errors\n", pathLabel)
			}

			nodesWithGrants++
			grantCount += len(grants)
			fmt.Printf("  %s -> %d grant(s)\n", pathLabel, len(grants))
			if verbose {
				for _, g := range grants {
					fmt.Printf("    - %s\n", describeGrant(g))
				}
			}

			// compare grantees, report suspected orphans; optionally delete when not a dry-run and shares were listed
			for _, g := range grants {
				// Skip non-user/group grantees (aligns with spec: do not touch public links)
				if gr := g.GetGrantee(); gr != nil {
					if gr.GetType() != provider.GranteeType_GRANTEE_TYPE_USER && gr.GetType() != provider.GranteeType_GRANTEE_TYPE_GROUP {
						if verbose {
							fmt.Printf("    SKIP(non-user/group): %s\n", describeGrant(g))
						}
						continue
					}
				}
				// Ignore ANY grants on the space root: built-in ownership/manager grants are not backed by share-manager
				if resourceIDEqual(info.GetId(), rootResourceID) {
					if verbose {
						fmt.Printf("    SKIP(root): %s\n", describeGrant(g))
					}
					continue
				}
				if !hasMatchingShareForGrant(shares, g) {
					orphanCandidates++
					if !dryRun && (shareErr == nil || force) {
						if err := removeGrantForResource(serviceUserCtx, storageRegistrySelector, info.GetId(), g, revaOpts); err != nil {
							fmt.Printf("    REMOVE-FAILED: %s error=%v\n", describeGrant(g), err)
							orphanDeleteErrors++
						} else {
							fmt.Printf("    REMOVED: %s\n", describeGrant(g))
							orphanDeleted++
						}
					} else {
						if shareErr != nil && !dryRun && !force {
							fmt.Printf("    ORPHAN: %s (not removed due to share list error; rerun with --force to remove)\n", describeGrant(g))
						} else {
							fmt.Printf("    ORPHAN: %s\n", describeGrant(g))
						}
					}
				}
			}
			return nil
		})
		if err != nil {
			fmt.Printf("[%d/%d] Failed to walk space %s: %v\n", idx+1, len(spaces), identifier, err)
			continue
		}

		totalNodes += nodeCount
		totalNodesWithGrants += nodesWithGrants
		totalGrants += grantCount
		fmt.Printf("[%d/%d] Space %s: %d nodes visited, %d nodes with grants, %d grants total\n", idx+1, len(spaces), identifier, nodeCount, nodesWithGrants, grantCount)
	}

	fmt.Printf("Summary: %d space(s), %d node(s) visited, %d nodes with grants, %d grants total\n", len(spaces), totalNodes, totalNodesWithGrants, totalGrants)
	if dryRun {
		fmt.Println("Dry run mode: no grants were modified")
	} else {
		fmt.Printf("Orphans: %d candidate(s), %d removed, %d errors\n", orphanCandidates, orphanDeleted, orphanDeleteErrors)
		// Post-cleanup reconciliation: run stale cleanup again to ensure caches reflect removals
		if cfg.Sharing.UserSharingDriver == "jsoncs3" {
			rcfg := revaShareConfig(cfg.Sharing)
			if f, ok := registry.NewFuncs["jsoncs3"]; ok {
				l := logger("clean-orphaned-grants")
				if mgr, mErr := f(rcfg["jsoncs3"].(map[string]interface{}), &l); mErr == nil {
					zerolog.SetGlobalLevel(zerolog.InfoLevel)
					cleanCtx := l.WithContext(serviceUserCtx)
					mgr.(*jsoncs3.Manager).CleanupStaleShares(cleanCtx)
				}
			}
		}
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
// resolution per space via the provided cache (keyed by space_id).
// Test locally with: simple-opencloud-blobs/test_clean_orphaned_grants.sh
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
		if len(providersResp.GetProviders()) == 0 {
			// Cache empty to avoid repeated lookups for this space in this run
			providerCache[spaceID] = nil
			return nil, nil
		}
		providerInfos = providersResp.GetProviders()
		providerCache[spaceID] = providerInfos
	}

	ref := &provider.Reference{ResourceId: resourceID}
	grants := make([]*provider.Grant, 0)
	for _, info := range providerInfos {
		if info.GetAddress() == "" {
			continue
		}
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

func resolveTargetSpaces(ctx context.Context, selector pool.Selectable[gateway.GatewayAPIClient], spaceID string) ([]*provider.StorageSpace, error) {
	client, err := selector.Next()
	if err != nil {
		return nil, err
	}

	req := &provider.ListStorageSpacesRequest{}
	if spaceID != "" {
		req.Filters = []*provider.ListStorageSpacesRequest_Filter{
			{
				Type: provider.ListStorageSpacesRequest_Filter_TYPE_ID,
				Term: &provider.ListStorageSpacesRequest_Filter_Id{Id: &provider.StorageSpaceId{OpaqueId: spaceID}},
			},
		}
	} else {
		req.Opaque = &types.Opaque{Map: map[string]*types.OpaqueEntry{
			"unrestricted": {
				Decoder: "plain",
				Value:   []byte(strconv.FormatBool(true)),
			},
		}}
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

// OpenCloud directory structure for share-manager metadata and user spaces:
//
// opencloudHome/storage/
// ...
// └── metadata/
//     ├── spaces/js/oncs3-share-manager-metadata/    (rootMetadata - Phase 1,3,4)
//     │   ├── blobs/
//     │   │   ├── 9c/a3/b2/f5/-42a1-4b8e-9123-456789abcdef   (Phase 4: MISSING received.json blob - reconstructed here)
//     │   │   │   {"Spaces": {"215fee7a-...:480049db-...": {"States": {"...:...:84652da9-...": {State: 2, MountPoint: {path: "file.txt"}}}}}}
//     │   │   └── d7/02/d7/e1/-37b0-4d41-b8dc-4b90c1d1f907   (Phase 1: read <spaceID>.json blob for Shares data)
//     │   │       {"Shares": {"215fee7a-...:480049db-...:84652da9-...": {resource_id: {...}, grantee: {...}, creator: {...}}}}
//     │   └── nodes/
//     │       ├── 3a/5f/c2/d8/-1234-5678-abcd-ef0123456789.mpk  (Phase 4: received.json MPK → points to MISSING blob)
//     │       │   {"user.oc.name": "received.json", "user.oc.blobid": "9ca3b2f5-42a1-4b8e-9123-456789abcdef", "user.oc.parentid": "a9a54ce7-..."}
//     │       ├── 99/98/b8/bf/-6871-49cc-aca9-dab4984dc1e4.mpk  (Phase 1: <spaceID>.json MPK → points to Shares blob)
//     │       │   {"user.oc.name": "480049db-...-...-....json", "user.oc.blobid": "d702d7e1-37b0-4d41-b8dc-4b90c1d1f907"}
//     │       └── a9/a5/4c/e7/-de30-4d27-94f8-10e4612c66c2.mpk  (Phase 3: parent node for ancestry lookup)
//     │           {"user.oc.name": "einstein", "user.oc.id": "a9a54ce7-...", "user.oc.parentid": "...users-node-id..."}
//     └── uploads/                                   (rootMetadataUploads)
//         ├── d702d7e1-37b0-4d41-b8dc-4b90c1d1f907   (Phase 1: read <spaceID>.json blob for Shares data; blobUploadsPath = filepath.Join(rootMetadataUploads, blobID))
//         │       {"Shares": {"215fee7a-...:480049db-...:84652da9-...": {resource_id: {...}, grantee: {...}, creator: {...}}}}
//         └── 1c93b82b-d22d-41e0-8038-5a706e9b409e.info
//                 {"MetaData": {"dir": "/users/4c510ada-c86b-4815-8820-42cdf82c3d51", "filename": "received.json", ...}, "Storage": {"NodeName": "received.json", "SpaceRoot": "jsoncs3-share-manager-metadata", ...}}

func moveStuckUploadBlobsCmd(cfg *config.Config) *cobra.Command {
	moveCmd := &cobra.Command{
		Use:   "move-stuck-upload-blobs",
		Short: `move stuck upload blobs to the jsoncs3 share-manager metadata`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Parse base config to align with other shares subcommands; no config fields are required here
			if err := parser.ParseConfig(cfg, true); err != nil {
				return configlog.ReturnError(err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			filename, _ := cmd.Flags().GetString("filename")
			verbose, _ := cmd.Flags().GetBool("verbose")

			dryRun := true
			if cmd.Flags().Changed("dry-run") {
				dryRun, _ = cmd.Flags().GetBool("dry-run")
			}
			if dryRun {
				fmt.Print("Dry run mode enabled\n\n")
			}

			home, err := os.UserHomeDir()
			if err != nil {
				return configlog.ReturnError(err)
			}

			opencloudHome := filepath.Join(home, ".opencloud")
			if cmd.Flags().Changed("opencloud-home") {
				opencloudHome, _ = cmd.Flags().GetString("opencloud-home")
			}

			rootMetadata := filepath.Join(opencloudHome, "storage", "metadata")
			rootMetadataBlobs := filepath.Join(rootMetadata, "spaces", "js", "oncs3-share-manager-metadata")

			fmt.Printf("Scanning for missing blobs in: %s \n\n", rootMetadataBlobs)
			missingBlobs, err := scanMissingBlobs(rootMetadataBlobs, filename)
			if err != nil {
				return err
			}
			if verbose {
				printJSON(missingBlobs, "missingBlobs")
			}

			if len(missingBlobs) == 0 {
				fmt.Println("No missing blobs found")
				return nil
			}

			rootMetadataUploads := filepath.Join(rootMetadata, "uploads")
			fmt.Printf("Found %d missing blobs. Restoring from %s\n", len(missingBlobs), rootMetadataUploads)
			remainingBlobIDs := restoreFromUploads(rootMetadataUploads, missingBlobs, dryRun)

			if verbose {
				printJSON(remainingBlobIDs, "remainingBlobIDs")
			}

			return nil
		},
	}
	moveCmd.Flags().Bool("dry-run", false, "Dry run mode enabled")
	moveCmd.Flags().String("opencloud-home", "~/.opencloud", "OpenCloud home directory")
	moveCmd.Flags().String("filename", "received.json", "file to move from uploads/ to share manager metadata blobs/")
	moveCmd.Flags().Bool("verbose", false, "Verbose logging enabled")

	return moveCmd
}

func printJSON(v any, label string) {
	jbs, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Printf("Error marshalling JSON: %v\n", err)
		return
	}
	fmt.Println(label, string(jbs))
}

// Scan for missing received.json blobs
func scanMissingBlobs(rootMetadata, filename string) (map[string]string, error) {
	missingBlobs := make(map[string]string) // blobID -> blobPathAbs
	nodesRoot := filepath.Join(rootMetadata, "nodes")

	_ = filepath.WalkDir(nodesRoot, func(path string, dir os.DirEntry, err error) error {
		if err != nil || dir.IsDir() || filepath.Ext(path) != ".mpk" {
			return nil
		}
		mpkBin, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		mpk := unmarshalMPK(mpkBin)
		if mpk["user.oc.name"] != filename {
			return nil
		}
		blobID := mpk["user.oc.blobid"]
		blobPathRel, ok := computeBlobPathRelative(blobID)
		if !ok {
			return nil
		}
		blobPathAbs := filepath.Join(rootMetadata, blobPathRel)
		if _, statErr := os.Stat(blobPathAbs); statErr == nil {
			return nil
		}
		missingBlobs[blobID] = blobPathAbs
		return nil
	})

	return missingBlobs, nil
}

// Attempt fast path restoration from uploads/ folder
func restoreFromUploads(rootMetadataUploads string, missing map[string]string, dryRun bool) map[string]bool {
	remainingBlobIDs := make(map[string]bool)

	for blobID, blobPathAbs := range missing {
		remainingBlobIDs[blobID] = true

		blobUploadsPath := filepath.Join(rootMetadataUploads, blobID)
		if dryRun {
			fmt.Printf("    DRY RUN: move %s to %s\n", blobUploadsPath, blobPathAbs)
			continue
		}

		// Check if the blob exists in the uploads folder and move it to the share manager metadata blobs/ folder
		if _, err := os.Stat(blobUploadsPath); err != nil {
			fmt.Printf("    Blob %s: not found in %s\n", blobID, blobUploadsPath)
			continue
		}
		fmt.Printf("    Move %s to %s\n", blobUploadsPath, blobPathAbs)
		if err := os.MkdirAll(filepath.Dir(blobPathAbs), 0755); err != nil {
			fmt.Printf("    Warning: Failed to create dir: %v\n", err)
			continue
		}
		if err := os.Rename(blobUploadsPath, blobPathAbs); err != nil {
			fmt.Printf("    Warning: Failed to move blob: %v\n", err)
			continue
		}

		// Remove the info file after the blob is moved
		infoPath := blobUploadsPath + ".info"
		if _, err := os.Stat(infoPath); err != nil {
			fmt.Printf("    Info file %s: not found\n", infoPath)
			continue
		}
		if err := os.Remove(infoPath); err != nil {
			fmt.Printf("    Warning: Failed to remove info file: %v\n", err)
			continue
		}

		remainingBlobIDs[blobID] = false
	}

	return remainingBlobIDs
}

func computeBlobPathRelative(bid string) (string, bool) {
	hyphen := strings.Index(bid, "-")
	if hyphen < 0 || hyphen < 8 {
		return "", false
	}
	prefix8 := bid[:hyphen]
	if len(prefix8) < 8 {
		return "", false
	}
	d1, d2, d3, d4 := prefix8[0:2], prefix8[2:4], prefix8[4:6], prefix8[6:8]
	suffix := bid[hyphen:]
	return filepath.Join("blobs", d1, d2, d3, d4, suffix), true
}

func unmarshalMPK(bin []byte) map[string]string {
	keyValue := map[string][]byte{}
	_ = msgpack.Unmarshal(bin, &keyValue)
	out := make(map[string]string, len(keyValue))
	for k, v := range keyValue {
		out[k] = string(v)
	}
	return out
}
