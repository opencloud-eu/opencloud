package svc

import (
	"fmt"
	"net/http"
	"strings"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/go-chi/render"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/thumbnail"

	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
)

type driveItemsByResourceID map[string]libregraph.DriveItem

// GetSharedByMe implements the Service interface (/me/drives/sharedByMe endpoint)
func (g Graph) GetSharedByMe(w http.ResponseWriter, r *http.Request) {
	g.logger.Debug().Msg("Calling GetRootDriveChildren")
	ctx := r.Context()

	driveItems, err := g.listUserShares(ctx, nil, make(driveItemsByResourceID))
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}

	if g.config.IncludeOCMSharees {
		driveItems, err = g.listOCMShares(ctx, nil, driveItems)
		if err != nil {
			errorcode.RenderError(w, r, err)
			return
		}
	}

	driveItems, err = g.listPublicShares(ctx, nil, driveItems)
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}

	expand := r.URL.Query().Get("$expand")
	expandThumbnails := strings.Contains(expand, "thumbnails")
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
					item.GetId())
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

	res := make([]libregraph.DriveItem, 0, len(driveItems))
	for _, v := range driveItems {
		res = append(res, v)
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &ListResponse{Value: res})
}

func cs3StatusToErrCode(code rpc.Code) (errcode errorcode.ErrorCode) {
	switch code {
	case rpc.Code_CODE_UNAUTHENTICATED:
		errcode = errorcode.Unauthenticated
	case rpc.Code_CODE_PERMISSION_DENIED:
		errcode = errorcode.AccessDenied
	case rpc.Code_CODE_NOT_FOUND:
		errcode = errorcode.ItemNotFound
	case rpc.Code_CODE_LOCKED:
		errcode = errorcode.ItemIsLocked
	case rpc.Code_CODE_INVALID_ARGUMENT:
		errcode = errorcode.InvalidRequest
	case rpc.Code_CODE_FAILED_PRECONDITION:
		errcode = errorcode.InvalidRequest
	default:
		errcode = errorcode.GeneralException
	}
	return errcode
}
