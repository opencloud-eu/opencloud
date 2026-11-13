package capabilities

import (
	"context"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/owncloud/ocs"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// resolveProviders builds the per-provider capability map from the spaces visible
// to the current user, keyed by provider ID. Returns nil on no user context or an
// unreachable gateway, so the response omits the section rather than failing.
func (h *Handler) resolveProviders(ctx context.Context) map[string]*ocs.ProviderCapabilities {
	log := appctx.GetLogger(ctx)

	gc, err := pool.GetGatewayServiceClient(h.gatewayAddr)
	if err != nil {
		log.Error().Err(err).Msg("capabilities: error getting gateway client")
		return nil
	}

	res, err := gc.ListStorageSpaces(ctx, &provider.ListStorageSpacesRequest{})
	if err != nil {
		log.Error().Err(err).Msg("capabilities: error listing storage spaces")
		return nil
	}
	if res.GetStatus().GetCode() != rpc.Code_CODE_OK {
		return nil
	}

	providers := map[string]*ocs.ProviderCapabilities{}
	for _, sp := range res.GetStorageSpaces() {
		id := sp.GetRoot().GetStorageId()
		if id == "" {
			continue
		}
		if _, done := providers[id]; done {
			continue
		}
		var caps storage.Capabilities
		if err := utils.ReadJSONFromOpaque(sp.GetOpaque(), storage.CapabilitiesOpaqueKey, &caps); err != nil {
			continue
		}
		providers[id] = ocs.NewProviderCapabilities(caps)
	}
	return providers
}
