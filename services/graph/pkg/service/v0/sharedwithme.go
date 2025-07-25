package svc

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	ocm "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"
	"github.com/go-chi/render"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/thumbnail"
)

// ListSharedWithMe lists the files shared with the current user.
func (g Graph) ListSharedWithMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	expand := r.URL.Query().Get("$expand")
	expandThumbnails := strings.Contains(expand, "thumbnails")

	driveItems, err := g.listSharedWithMe(ctx, expandThumbnails)
	if err != nil {
		g.logger.Error().Err(err).Msg("listSharedWithMe failed")
		errorcode.RenderError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &ListResponse{Value: driveItems})
}

// listSharedWithMe is a helper function that lists the drive items shared with the current user.
func (g Graph) listSharedWithMe(ctx context.Context, expandThumbnails bool) ([]libregraph.DriveItem, error) {
	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		g.logger.Error().Err(err).Msg("could not select next gateway client")
		return nil, err
	}

	listReceivedSharesResponse, err := gatewayClient.ListReceivedShares(ctx, &collaboration.ListReceivedSharesRequest{})
	if err := errorcode.FromCS3Status(listReceivedSharesResponse.GetStatus(), err); err != nil {
		g.logger.Error().Err(err).Msg("listing shares failed")
		return nil, err
	}
	driveItems, err := cs3ReceivedSharesToDriveItems(ctx, g.logger, gatewayClient, g.identityCache, listReceivedSharesResponse.GetShares(), g.availableRoles)
	if err != nil {
		g.logger.Error().Err(err).Msg("could not convert received shares to drive items")
		return nil, err
	}

	if g.config.IncludeOCMSharees {
		listReceivedOCMSharesResponse, err := gatewayClient.ListReceivedOCMShares(ctx, &ocm.ListReceivedOCMSharesRequest{})
		if err := errorcode.FromCS3Status(listReceivedSharesResponse.GetStatus(), err); err != nil {
			g.logger.Error().Err(err).Msg("listing shares failed")
			return nil, err
		}
		ocmDriveItems, err := cs3ReceivedOCMSharesToDriveItems(ctx, g.logger, gatewayClient, g.identityCache, listReceivedOCMSharesResponse.GetShares(), g.availableRoles)
		if err != nil {
			g.logger.Error().Err(err).Msg("could not convert received ocm shares to drive items")
			return nil, err
		}
		driveItems = append(driveItems, ocmDriveItems...)
	}

	if expandThumbnails {
		for k, item := range driveItems {
			mt := item.GetFile().MimeType
			if mt == nil {
				continue
			}

			_, match := thumbnail.SupportedMimeTypes[*mt]
			if match {
				baseUrl := fmt.Sprintf("%s/dav/spaces/%s?scalingup=0&preview=1&processor=thumbnail",
					g.config.Commons.OpenCloudURL,
					item.RemoteItem.GetId())
				smallUrl := baseUrl + "&x=36&y=36"
				mediumUrl := baseUrl + "&x=48&y=48"
				largeUrl := baseUrl + "&x=96&y=96"

				item.SetThumbnails([]libregraph.ThumbnailSet{
					{
						Small:  &libregraph.Thumbnail{Url: &smallUrl},
						Medium: &libregraph.Thumbnail{Url: &mediumUrl},
						Large:  &libregraph.Thumbnail{Url: &largeUrl},
					},
				})

				driveItems[k] = item // assign modified item back to the map
			}
		}
	}

	return driveItems, err
}
