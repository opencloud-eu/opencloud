package task

import (
	"time"

	apiGateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	apiRpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	apiProvider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v3/pkg/utils"
)

// PurgeTrashBin can be used to purge space trash-bin's,
// the provided executantID must have space access.
// removeBefore specifies how long an item must be in the trash-bin to be deleted,
// items that stay there for a shorter time are ignored and kept in place.
func PurgeTrashBin(serviceAccountID string, deleteBefore time.Time, spaceType SpaceType, gatewaySelector pool.Selectable[apiGateway.GatewayAPIClient], serviceAccountSecret string) error {
	gatewayClient, err := gatewaySelector.Next()
	if err != nil {
		return err
	}

	ctx, err := utils.GetServiceUserContext(serviceAccountID, gatewayClient, serviceAccountSecret)
	if err != nil {
		return err
	}

	gatewayClient, err = gatewaySelector.Next()
	if err != nil {
		return err
	}
	listStorageSpacesResponse, err := gatewayClient.ListStorageSpaces(ctx, &apiProvider.ListStorageSpacesRequest{
		Filters: []*apiProvider.ListStorageSpacesRequest_Filter{
			{
				Type: apiProvider.ListStorageSpacesRequest_Filter_TYPE_SPACE_TYPE,
				Term: &apiProvider.ListStorageSpacesRequest_Filter_SpaceType{
					SpaceType: string(spaceType),
				},
			},
		},
	})
	if err != nil {
		return err
	}

	for _, storageSpace := range listStorageSpacesResponse.StorageSpaces {
		if typ := storageSpace.GetSpaceType(); typ != "personal" && typ != "project" {
			// ignore spaces that are neither personal nor project
			continue
		}
		storageSpaceReference := &apiProvider.Reference{
			ResourceId: storageSpace.GetRoot(),
		}

		gatewayClient, err = gatewaySelector.Next()
		if err != nil {
			return err
		}
		listRecycleResponse, err := gatewayClient.ListRecycle(ctx, &apiProvider.ListRecycleRequest{Ref: storageSpaceReference})
		if err != nil {
			return err
		}

		for _, recycleItem := range listRecycleResponse.GetRecycleItems() {
			doDelete := utils.TSToUnixNano(recycleItem.DeletionTime) < utils.TSToUnixNano(utils.TimeToTS(deleteBefore))
			if !doDelete {
				continue
			}

			gatewayClient, err = gatewaySelector.Next()
			if err != nil {
				return err
			}
			purgeRecycleResponse, err := gatewayClient.PurgeRecycle(ctx, &apiProvider.PurgeRecycleRequest{
				Ref: storageSpaceReference,
				Key: recycleItem.Key,
			})

			if purgeRecycleResponse.GetStatus().GetCode() != apiRpc.Code_CODE_OK {
				return errtypes.NewErrtypeFromStatus(purgeRecycleResponse.Status)
			}

			if err != nil {
				return err
			}
		}
	}

	return nil
}
