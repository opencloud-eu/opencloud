package svc

import (
	"net/http"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/go-chi/render"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
)

type driveItemsByResourceID map[string]libregraph.DriveItem

// GetSharedByMe implements the Service interface (/me/drives/sharedByMe endpoint)
func (g Graph) GetSharedByMe(w http.ResponseWriter, r *http.Request) {
	g.logger.Debug().Msg("Calling GetRootDriveChildren")
	ctx := r.Context()

	driveItems := make(driveItemsByResourceID)
	var err error
	driveItems, err = g.listUserShares(ctx, nil, driveItems)
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
