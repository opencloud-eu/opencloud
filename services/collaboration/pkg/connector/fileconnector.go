package connector

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"time"

	appproviderv1beta1 "github.com/cs3org/go-cs3apis/cs3/app/provider/v1beta1"
	auth "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	gatewayv1beta1 "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpcv1beta1 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	providerv1beta1 "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/config"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/connector/fileinfo"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/helpers"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/middleware"
	"github.com/opencloud-eu/opencloud/services/collaboration/pkg/wopisrc"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/rs/zerolog"
	microstore "go-micro.dev/v4/store"
)

const (
	// WOPI Locks generally have a lock duration of 30 minutes and will be refreshed before expiration if needed
	// https://docs.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/concepts#lock
	lockDuration time.Duration = 30 * time.Minute
)

// FileConnectorService is the interface to implement the "Files"
// endpoint. Basically lock operations on the file plus the CheckFileInfo.
// All operations need a context containing a WOPI context and, optionally,
// a zerolog logger.
// Target file is within the WOPI context
type FileConnectorService interface {
	// GetLock will return the lockID present in the target file.
	GetLock(ctx context.Context) (*ConnectorResponse, error)
	// Lock will lock the target file with the provided lockID. If the oldLockID
	// is provided (not empty), the method will perform an unlockAndRelock
	// operation (unlock the file with the oldLockID and immediately relock
	// the file with the new lockID).
	// The current lockID will be returned if a conflict happens
	Lock(ctx context.Context, lockID, oldLockID string) (*ConnectorResponse, error)
	// RefreshLock will extend the lock time 30 minutes. The current lockID
	// needs to be provided.
	// The current lockID will be returned if a conflict happens
	RefreshLock(ctx context.Context, lockID string) (*ConnectorResponse, error)
	// UnLock will unlock the target file. The current lockID needs to be
	// provided.
	// The current lockID will be returned if a conflict happens
	UnLock(ctx context.Context, lockID string) (*ConnectorResponse, error)
	// CheckFileInfo will return the file information of the target file
	CheckFileInfo(ctx context.Context) (*ConnectorResponse, error)
	// PutRelativeFileSuggested will create a new file based on the contents of the
	// current file. Target is the filename that will be used for this
	// new file.
	// This implements the "suggested" code flow for the PutRelativeFile endpoint.
	// Since we need to upload contents, it will be done through the provided
	// The target must be UTF8-encoded.
	// ContentConnectorService
	PutRelativeFileSuggested(ctx context.Context, ccs ContentConnectorService, stream io.Reader, streamLength int64, target string) (*ConnectorResponse, error)
	// PutRelativeFileRelative will create a new file based on the contents of the
	// current file. Target is the filename that will be used for this
	// new file.
	// This implements the "relative" code flow for the PutRelativeFile endpoint.
	// The required headers that could need to be sent through HTTP will also
	// be returned if needed.
	// The target must be UTF8-encoded.
	// Since we need to upload contents, it will be done through the provided
	// ContentConnectorService
	PutRelativeFileRelative(ctx context.Context, ccs ContentConnectorService, stream io.Reader, streamLength int64, target string) (*ConnectorResponse, error)
	// DeleteFile will delete the provided file in the context. Although
	// not documented, a lockID can be used to try to delete a locked file
	// assuming the lock matches.
	// The current lockID will be returned if the file is locked.
	DeleteFile(ctx context.Context, lockID string) (*ConnectorResponse, error)
	// RenameFile will rename the provided file in the context to the requested
	// filename. The filename must be UTF8-encoded.
	// In case of conflict, this method will return the actual lockId in
	// the file as second return value.
	RenameFile(ctx context.Context, lockID, target string) (*ConnectorResponse, error)
}

// FileConnector implements the "File" endpoint.
// Currently, it handles file locks and getting the file info.
// Note that operations might return any kind of error, not just ConnectorError
type FileConnector struct {
	gws   pool.Selectable[gatewayv1beta1.GatewayAPIClient]
	cfg   *config.Config
	store microstore.Store
}

// NewFileConnector creates a new file connector
func NewFileConnector(gws pool.Selectable[gatewayv1beta1.GatewayAPIClient], cfg *config.Config, st microstore.Store) *FileConnector {
	return &FileConnector{
		gws:   gws,
		cfg:   cfg,
		store: st,
	}
}

// GetLock returns a lock or an empty string if no lock exists
// https://docs.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/files/getlock
//
// The context MUST have a WOPI context, otherwise an error will be returned.
// You can pass a pre-configured zerologger instance through the context that
// will be used to log messages.
//
// The lock ID applied to the file reference in the context will be returned
// (if any). An error will be returned if something goes wrong. The error
// could be a ConnectorError
func (f *FileConnector) GetLock(ctx context.Context) (*ConnectorResponse, error) {
	wopiContext, err := middleware.WopiContextFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	logger := zerolog.Ctx(ctx)

	req := &providerv1beta1.GetLockRequest{
		Ref: wopiContext.FileReference,
	}

	gwc, err := f.gws.Next()
	if err != nil {
		return nil, err
	}
	resp, err := gwc.GetLock(ctx, req)
	if err != nil {
		logger.Error().Err(err).Msg("GetLock failed")
		return nil, err
	}

	if resp.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
		logger.Error().
			Str("StatusCode", resp.GetStatus().GetCode().String()).
			Str("StatusMsg", resp.GetStatus().GetMessage()).
			Msg("GetLock failed with unexpected status")
		// Return 404 and the required header, even if it's empty
		// https://docs.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/files/getlock#response-headers
		// This is needed to fulfill the requirements of online WOPI validator
		return NewResponseWithLock(404, ""), nil
	}

	lockID := ""
	if resp.GetLock() != nil {
		lockID = resp.GetLock().GetLockId()
	}

	// log the success at debug level
	logger.Debug().
		Str("LockID", lockID).
		Msg("GetLock success")

	return NewResponseWithLock(200, lockID), nil
}

// Lock returns a WOPI lock or performs an unlock and relock
// https://docs.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/files/lock
// https://docs.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/files/unlockandrelock
//
// The context MUST have a WOPI context, otherwise an error will be returned.
// You can pass a pre-configured zerologger instance through the context that
// will be used to log messages.
//
// Lock the file reference contained in the context with the provided lockID.
// The oldLockID is only used for the "unlock and relock" operation. The "lock"
// operation doesn't use the oldLockID and needs to be empty in this case.
//
// For the "lock" operation, if the operation is successful, an empty lock id
// will be returned without any error. In case of conflict, the current lock
// id will be returned along with a 409 ConnectorError. For any other error,
// the method will return an empty lock id.
//
// For the "unlock and relock" operation, the behavior will be the same.
//
// On success, the mtime of the file will be returned in the X-Wopi-Version header.
func (f *FileConnector) Lock(ctx context.Context, lockID, oldLockID string) (*ConnectorResponse, error) {
	wopiContext, err := middleware.WopiContextFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	logger := zerolog.Ctx(ctx).With().
		Str("RequestedLockID", lockID).
		Str("RequestedOldLockID", oldLockID).
		Logger()
	logger.Debug().Msg("Lock: start")

	if lockID == "" {
		logger.Error().Msg("Lock failed due to empty lockID")
		return NewResponse(400), nil
	}

	var setOrRefreshStatus *rpcv1beta1.Status
	if oldLockID == "" {
		// If the oldLockID is empty, this is a "LOCK" request
		logger.Debug().Msg("Lock: this is a SetLock request")
		req := &providerv1beta1.SetLockRequest{
			Ref: wopiContext.FileReference,
			Lock: &providerv1beta1.Lock{
				LockId:  lockID,
				AppName: f.cfg.App.Name,
				Type:    providerv1beta1.LockType_LOCK_TYPE_WRITE,
				Expiration: &typesv1beta1.Timestamp{
					Seconds: uint64(time.Now().Add(lockDuration).Unix()),
				},
			},
		}

		gwc, err := f.gws.Next()
		if err != nil {
			return nil, err
		}
		resp, err := gwc.SetLock(ctx, req)
		if err != nil {
			logger.Error().Err(err).Msg("SetLock failed")
			return nil, err
		}
		setOrRefreshStatus = resp.GetStatus()
	} else {
		// If the oldLockID isn't empty, this is a "UnlockAndRelock" request. We'll
		// do a "RefreshLock" in reva and provide the old lock
		logger.Debug().Msg("Lock: this is a RefreshLock request")
		req := &providerv1beta1.RefreshLockRequest{
			Ref: wopiContext.FileReference,
			Lock: &providerv1beta1.Lock{
				LockId:  lockID,
				AppName: f.cfg.App.Name,
				Type:    providerv1beta1.LockType_LOCK_TYPE_WRITE,
				Expiration: &typesv1beta1.Timestamp{
					Seconds: uint64(time.Now().Add(lockDuration).Unix()),
				},
			},
			ExistingLockId: oldLockID,
		}

		gwc, err := f.gws.Next()
		if err != nil {
			return nil, err
		}
		resp, err := gwc.RefreshLock(ctx, req)
		if err != nil {
			logger.Error().Err(err).Msg("UnlockAndRefresh failed")
			return nil, err
		}
		setOrRefreshStatus = resp.GetStatus()
	}

	gwc, err := f.gws.Next()
	if err != nil {
		return nil, err
	}
	statResp, err := gwc.Stat(ctx, &providerv1beta1.StatRequest{
		Ref: wopiContext.FileReference,
	})
	if err != nil {
		logger.Error().Err(err).Msg("Lock failed trying to get the file info")
		return nil, err
	}
	if statResp.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
		logger.Error().
			Str("StatusCode", statResp.GetStatus().GetCode().String()).
			Str("StatusMsg", statResp.GetStatus().GetMessage()).
			Msg("Lock failed trying to get the file info with unexpected status")
		return NewResponse(500), nil
	}

	// we're checking the status of either the "SetLock" or "RefreshLock" operations
	switch setOrRefreshStatus.GetCode() {
	case rpcv1beta1.Code_CODE_OK:
		logger.Debug().Msg("SetLock successful")
		return NewResponseWithVersion(statResp.GetInfo().GetMtime()), nil

	case rpcv1beta1.Code_CODE_FAILED_PRECONDITION, rpcv1beta1.Code_CODE_ABORTED:
		// Code_CODE_FAILED_PRECONDITION -> Lock operation mismatched lock
		// Code_CODE_ABORTED -> UnlockAndRelock operation mismatched lock
		// In both cases, we need to get the current lock to return it in a
		// 409 response if needed
		req := &providerv1beta1.GetLockRequest{
			Ref: wopiContext.FileReference,
		}

		gwc, err = f.gws.Next()
		if err != nil {
			return nil, err
		}
		resp, err := gwc.GetLock(ctx, req)
		if err != nil {
			logger.Error().Err(err).Msg("SetLock failed, fallback to GetLock failed too")
			return nil, err
		}

		if resp.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
			logger.Error().
				Str("StatusCode", resp.GetStatus().GetCode().String()).
				Str("StatusMsg", resp.GetStatus().GetMessage()).
				Msg("SetLock failed, fallback to GetLock failed with unexpected status")
		}

		if resp.GetLock() != nil {
			if resp.GetLock().GetLockId() != lockID {
				// lockId is different -> return 409 with the current lockId
				logger.Warn().
					Str("LockID", resp.GetLock().GetLockId()).
					Msg("SetLock conflict")
				return NewResponseLockConflict(resp.GetLock().GetLockId(), "Conflicting LockID"), nil
			}

			// TODO: according to the spec we need to treat this as a RefreshLock
			// There was a problem with the lock, but the file has the same lockId now.
			// This should never happen unless there are race conditions.
			// Since the lockId matches now, we'll assume success for now.
			// As said in the todo, we probably should send a "RefreshLock" request here.
			logger.Warn().
				Str("LockID", resp.GetLock().GetLockId()).
				Msg("SetLock lock refreshed instead")
			return NewResponseWithVersion(statResp.GetInfo().GetMtime()), nil
		}

		logger.Error().Msg("SetLock failed and could not refresh")
		return NewResponse(500), nil

	case rpcv1beta1.Code_CODE_NOT_FOUND:
		logger.Error().Msg("SetLock failed, file not found")
		return NewResponse(404), nil

	default:
		logger.Error().
			Str("StatusCode", setOrRefreshStatus.GetCode().String()).
			Str("StatusMsg", setOrRefreshStatus.GetMessage()).
			Msg("SetLock failed with unexpected status")
		return NewResponse(500), nil
	}
}

// RefreshLock refreshes a provided lock for 30 minutes
// https://docs.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/files/refreshlock
//
// The context MUST have a WOPI context, otherwise an error will be returned.
// You can pass a pre-configured zerologger instance through the context that
// will be used to log messages.
//
// If the operation is successful, an empty lock id will be returned without
// any error. In case of conflict, the current lock id will be returned
// along with a 409 ConnectorError. For any other error, the method will
// return an empty lock id.
// The conflict happens if the provided lockID doesn't match the one actually
// applied in the target file.
//
// On success, the mtime of the file will be returned in the X-Wopi-Version header.
func (f *FileConnector) RefreshLock(ctx context.Context, lockID string) (*ConnectorResponse, error) {
	wopiContext, err := middleware.WopiContextFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	logger := zerolog.Ctx(ctx).With().
		Str("RequestedLockID", lockID).
		Logger()
	logger.Debug().Msg("RefreshLock: start")

	if lockID == "" {
		logger.Error().Msg("RefreshLock failed due to empty lockID")
		return NewResponse(400), nil
	}

	req := &providerv1beta1.RefreshLockRequest{
		Ref: wopiContext.FileReference,
		Lock: &providerv1beta1.Lock{
			LockId:  lockID,
			AppName: f.cfg.App.Name,
			Type:    providerv1beta1.LockType_LOCK_TYPE_WRITE,
			Expiration: &typesv1beta1.Timestamp{
				Seconds: uint64(time.Now().Add(lockDuration).Unix()),
			},
		},
	}

	gwc, err := f.gws.Next()
	if err != nil {
		return nil, err
	}
	resp, err := gwc.RefreshLock(ctx, req)
	if err != nil {
		logger.Error().Err(err).Msg("RefreshLock failed")
		return nil, err
	}

	gwc, err = f.gws.Next()
	if err != nil {
		return nil, err
	}
	statResp, err := gwc.Stat(ctx, &providerv1beta1.StatRequest{
		Ref: wopiContext.FileReference,
	})
	if err != nil {
		logger.Error().Err(err).Msg("RefreshLock failed trying to get the file info")
		return nil, err
	}
	if statResp.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
		logger.Error().
			Str("StatusCode", statResp.GetStatus().GetCode().String()).
			Str("StatusMsg", statResp.GetStatus().GetMessage()).
			Msg("RefreshLock failed trying to get the file info with unexpected status")
		return NewResponse(500), nil
	}

	switch resp.GetStatus().GetCode() {
	case rpcv1beta1.Code_CODE_OK:
		logger.Debug().Msg("RefreshLock successful")
		// The current lock should not be returned in the headers on success
		// https://learn.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/files/refreshlock#response-headers
		return NewResponseWithVersion(statResp.GetInfo().GetMtime()), nil

	case rpcv1beta1.Code_CODE_NOT_FOUND:
		logger.Error().
			Str("StatusCode", resp.GetStatus().GetCode().String()).
			Str("StatusMsg", resp.GetStatus().GetMessage()).
			Msg("RefreshLock failed, file reference not found")
		return NewResponse(404), nil

	case rpcv1beta1.Code_CODE_ABORTED:
		logger.Error().
			Str("StatusCode", resp.GetStatus().GetCode().String()).
			Str("StatusMsg", resp.GetStatus().GetMessage()).
			Msg("RefreshLock failed, lock mismatch")

		// Either the file is unlocked or there is no lock
		// We need to return 409 with the current lock
		req := &providerv1beta1.GetLockRequest{
			Ref: wopiContext.FileReference,
		}

		gwc, err = f.gws.Next()
		if err != nil {
			return nil, err
		}
		resp, err := gwc.GetLock(ctx, req)
		if err != nil {
			logger.Error().Err(err).Msg("RefreshLock failed trying to get the current lock")
			return nil, err
		}

		if resp.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
			logger.Error().
				Str("StatusCode", resp.GetStatus().GetCode().String()).
				Str("StatusMsg", resp.GetStatus().GetMessage()).
				Msg("RefreshLock failed, tried to get the current lock failed with unexpected status")
			return NewResponse(500), nil
		}

		if resp.GetLock() == nil {
			logger.Error().
				Str("StatusCode", resp.GetStatus().GetCode().String()).
				Str("StatusMsg", resp.GetStatus().GetMessage()).
				Msg("RefreshLock failed, no lock on file")
			return NewResponseLockConflict("", "No lock on file"), nil
		} else {
			// lock is different than the one requested, otherwise we wouldn't reached this point
			logger.Error().
				Str("LockID", resp.GetLock().GetLockId()).
				Str("StatusCode", resp.GetStatus().GetCode().String()).
				Str("StatusMsg", resp.GetStatus().GetMessage()).
				Msg("RefreshLock failed, lock mismatch")
			return NewResponseLockConflict(resp.GetLock().GetLockId(), "Lock mismatch"), nil
		}
	default:
		logger.Error().
			Str("StatusCode", resp.GetStatus().GetCode().String()).
			Str("StatusMsg", resp.GetStatus().GetMessage()).
			Msg("RefreshLock failed with unexpected status")
		return NewResponse(500), nil
	}
}

// UnLock removes a given lock from a file
// https://docs.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/files/unlock
//
// The context MUST have a WOPI context, otherwise an error will be returned.
// You can pass a pre-configured zerologger instance through the context that
// will be used to log messages.
//
// If the operation is successful, an empty lock id will be returned without
// any error. In case of conflict, the current lock id will be returned
// along with a 409 ConnectorError. For any other error, the method will
// return an empty lock id.
// The conflict happens if the provided lockID doesn't match the one actually
// applied in the target file.
//
// On success, the mtime of the file will be returned in the X-Wopi-Version header.
func (f *FileConnector) UnLock(ctx context.Context, lockID string) (*ConnectorResponse, error) {
	wopiContext, err := middleware.WopiContextFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	logger := zerolog.Ctx(ctx).With().
		Str("RequestedLockID", lockID).
		Logger()
	logger.Debug().Msg("UnLock: start")

	if lockID == "" {
		logger.Error().Msg("Unlock failed due to empty lockID")
		return NewResponse(400), nil
	}

	req := &providerv1beta1.UnlockRequest{
		Ref: wopiContext.FileReference,
		Lock: &providerv1beta1.Lock{
			LockId:  lockID,
			AppName: f.cfg.App.Name,
		},
	}

	gwc, err := f.gws.Next()
	if err != nil {
		return nil, err
	}
	resp, err := gwc.Unlock(ctx, req)
	if err != nil {
		logger.Error().Err(err).Msg("Unlock failed")
		return nil, err
	}

	gwc, err = f.gws.Next()
	if err != nil {
		return nil, err
	}
	statResp, err := gwc.Stat(ctx, &providerv1beta1.StatRequest{
		Ref: wopiContext.FileReference,
	})
	if err != nil {
		logger.Error().Err(err).Msg("Unlock failed trying to get the file info")
		return nil, err
	}
	if statResp.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
		logger.Error().
			Str("StatusCode", statResp.GetStatus().GetCode().String()).
			Str("StatusMsg", statResp.GetStatus().GetMessage()).
			Msg("Unlock failed trying to get the file info with unexpected status")
		return NewResponse(500), nil
	}

	switch resp.GetStatus().GetCode() {
	case rpcv1beta1.Code_CODE_OK:
		logger.Debug().Msg("Unlock successful")
		return NewResponseWithVersion(statResp.GetInfo().GetMtime()), nil
	case rpcv1beta1.Code_CODE_ABORTED:
		// File isn't locked. Need to return 409 with empty lock
		logger.Error().Err(err).Msg("Unlock failed, file isn't locked")
		return NewResponseLockConflict("", "File isn't locked"), nil
	case rpcv1beta1.Code_CODE_LOCKED:
		// We need to return 409 with the current lock
		req := &providerv1beta1.GetLockRequest{
			Ref: wopiContext.FileReference,
		}

		gwc, err = f.gws.Next()
		if err != nil {
			return nil, err
		}
		resp, err := gwc.GetLock(ctx, req)
		if err != nil {
			logger.Error().Err(err).Msg("Unlock failed trying to get the current lock")
			return nil, err
		}

		if resp.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
			logger.Error().
				Str("StatusCode", resp.GetStatus().GetCode().String()).
				Str("StatusMsg", resp.GetStatus().GetMessage()).
				Msg("Unlock failed, tried to get the current lock failed with unexpected status")
			return NewResponse(500), nil
		}

		var outLockId string
		if resp.GetLock() == nil {
			logger.Error().
				Str("StatusCode", resp.GetStatus().GetCode().String()).
				Str("StatusMsg", resp.GetStatus().GetMessage()).
				Msg("Unlock failed, no lock on file")
			outLockId = ""
		} else {
			// lock is different than the one requested, otherwise we wouldn't reached this point
			logger.Error().
				Str("LockID", resp.GetLock().GetLockId()).
				Str("StatusCode", resp.GetStatus().GetCode().String()).
				Str("StatusMsg", resp.GetStatus().GetMessage()).
				Msg("Unlock failed, lock mismatch")
			outLockId = resp.GetLock().GetLockId()
		}
		return NewResponseLockConflict(outLockId, "Lock mismatch"), nil
	default:
		logger.Error().
			Str("StatusCode", resp.GetStatus().GetCode().String()).
			Str("StatusMsg", resp.GetStatus().GetMessage()).
			Msg("Unlock failed with unexpected status")
		return NewResponse(500), nil
	}
}

// PutRelativeFileSuggested upload a file using the suggested target name
// https://learn.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/files/putrelativefile
//
// The PutRelativeFile have 2 variants based on the "X-WOPI-SuggestedTarget"
// and "X-WOPI-RelativeTarget" headers. This method only implements the first,
// so this method must be used only if the "X-WOPI-SuggestedTarget" is present.
//
// The "target" filename must be UTF8-encoded. The conversion between UTF7 and
// UTF8 must happen outside this function.
//
// The context MUST have a WOPI context, otherwise an error will be returned.
// You can pass a pre-configured zerologger instance through the context that
// will be used to log messages.
//
// Since the method involves uploading a file to a location, it will use the
// provided ContentConnectorService to upload the stream. Note that the
// associated wopicontext is modified in order to point to the right location
// before the upload (it shouldn't matter because we'll work on a copy).
//
// As per documentation, this method will try to upload the provided stream
// using the suggested name. If the upload fails, we'll try using a different
// name. This new name will be generated by prefixing a random string to the
// suggested name.
// Since the upload won't use any lock, the upload will fail if the target file
// already exists and it isn't empty. This means that, this method can only
// generate new files.
func (f *FileConnector) PutRelativeFileSuggested(ctx context.Context, ccs ContentConnectorService, stream io.Reader, streamLength int64, target string) (*ConnectorResponse, error) {
	// assume the target is a full name
	wopiContext, err := middleware.WopiContextFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	logger := zerolog.Ctx(ctx).With().
		Str("PutTarget", target).
		Logger()

	gwc, err := f.gws.Next()
	if err != nil {
		return nil, err
	}
	// stat the current file in order to get the reference of the parent folder
	oldStatRes, err := gwc.Stat(ctx, &providerv1beta1.StatRequest{
		Ref: wopiContext.FileReference,
	})
	if err != nil {
		logger.Error().Err(err).Msg("PutRelativeFileSuggested: stat failed")
		return nil, err
	}

	if oldStatRes.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
		logger.Error().
			Str("StatusCode", oldStatRes.GetStatus().GetCode().String()).
			Str("StatusMsg", oldStatRes.GetStatus().GetMessage()).
			Msg("PutRelativeFileSuggested: stat failed with unexpected status")
		return NewResponse(500), nil
	}

	if strings.HasPrefix(target, ".") {
		// the target is an extension, so we need to use the original
		// name with the modified extension
		oldStatPath := oldStatRes.GetInfo().GetPath()
		ext := path.Ext(oldStatPath)
		target = strings.TrimSuffix(path.Base(oldStatPath), ext) + target
	}

	finalTarget := target
	newLogger := logger
	for isDone := false; !isDone; {
		targetPath := utils.MakeRelativePath(finalTarget)
		// need to change the file reference of the wopicontext to point to the new path
		wopiContext.FileReference = &providerv1beta1.Reference{
			ResourceId: oldStatRes.GetInfo().GetParentId(),
			Path:       targetPath,
		}

		// create a new context for the modified wopicontext
		newLogger := logger.With().Str("NewFileReference", wopiContext.FileReference.String()).Logger()
		newCtx := middleware.WopiContextToCtx(newLogger.WithContext(ctx), wopiContext)

		// try to put the file. It mustn't return a 400 or 409
		putResponse, err := ccs.PutFile(newCtx, stream, streamLength, "")
		if err != nil {
			newLogger.Error().Err(err).Msg("PutRelativeFileSuggested: put file failed") // fails here
			return nil, err
		}

		switch putResponse.Status {
		case 200:
			// if the put is successful, exit the loop and move on
			isDone = true
			logger = newLogger
		case 409:
			// if conflict generate a different name and retry.
			// this should happen only once
			actualFilename, _ := f.extractFilenameAndPrefix(target)
			finalTarget = f.generatePrefix() + " " + actualFilename
		default:
			// TODO: code 400 might happen, what to do?
			// in other cases, just return the error
			newLogger.Error().Msg("PutRelativeFileSuggested: put file failed with unhandled status")
			return NewResponse(500), nil
		}
	}
	var newInfo *providerv1beta1.ResourceInfo
	// adjust the wopi file reference to use only the resource id without path
	if newInfo, err = f.adjustWopiReference(ctx, &wopiContext, newLogger); err != nil {
		return nil, err
	}

	wopiSrcURL, err := f.generateWOPISrc(wopiContext, newLogger)
	if err != nil {
		logger.Error().Err(err).Msg("PutRelativeFileSuggested: error generating the WOPISrc parameter")
		return nil, err
	}

	logger.Debug().
		Str("FinalReference", wopiContext.FileReference.String()).
		Msg("PutRelativeFileSuggested: success")

	webURL, err := url.Parse(f.cfg.Commons.OpenCloudURL)
	if err != nil {
		return nil, err
	}
	return NewResponseSuccessBodyNameUrl(
		finalTarget,
		wopiSrcURL.String(),
		createHostUrl("write", webURL, strings.ToLower(f.cfg.App.Name), newInfo),
		createHostUrl("view", webURL, strings.ToLower(f.cfg.App.Name), newInfo),
	), nil
}

// PutRelativeFileRelative upload a file using the provided target name
// https://learn.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/files/putrelativefile
//
// The PutRelativeFile have 2 variants based on the "X-WOPI-SuggestedTarget"
// and "X-WOPI-RelativeTarget" headers. This method only implements the second,
// so this method must be used only if the "X-WOPI-RelativeTarget" is present.
//
// The "target" filename must be UTF8-encoded. The conversion between UTF7 and
// UTF8 must happen outside this function.
//
// The context MUST have a WOPI context, otherwise an error will be returned.
// You can pass a pre-configured zerologger instance through the context that
// will be used to log messages.
//
// Since the method involves uploading a file to a location, it will use the
// provided ContentConnectorService to upload the stream. Note that the
// associated wopicontext is modified in order to point to the right location
// before the upload (it shouldn't matter because we'll work on a copy).
//
// As per documentation, this method will try to upload the provided stream
// using the provided name. The filename won't be changed.
// Since the upload won't use any lock, the upload will fail if the target file
// already exists and it isn't empty. This means that, this method can only
// generate new files.
func (f *FileConnector) PutRelativeFileRelative(ctx context.Context, ccs ContentConnectorService, stream io.Reader, streamLength int64, target string) (*ConnectorResponse, error) {
	// assume the target is a full name
	wopiContext, err := middleware.WopiContextFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	logger := zerolog.Ctx(ctx).With().
		Str("PutTarget", target).
		Logger()

	gwc, err := f.gws.Next()
	if err != nil {
		return nil, err
	}
	// stat the current file in order to get the reference of the parent folder
	oldStatRes, err := gwc.Stat(ctx, &providerv1beta1.StatRequest{
		Ref: wopiContext.FileReference,
	})
	if err != nil {
		logger.Error().Err(err).Msg("PutRelativeFileRelative: stat failed")
		return nil, err
	}

	if oldStatRes.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
		logger.Error().
			Str("StatusCode", oldStatRes.GetStatus().GetCode().String()).
			Str("StatusMsg", oldStatRes.GetStatus().GetMessage()).
			Msg("PutRelativeFileRelative: stat failed with unexpected status")
		return NewResponse(500), nil
	}

	targetPath := utils.MakeRelativePath(target)
	// need to change the file reference of the wopicontext to point to the new path
	wopiContext.FileReference = &providerv1beta1.Reference{
		ResourceId: oldStatRes.GetInfo().GetParentId(),
		Path:       targetPath,
	}

	// create a new context for the modified wopicontext
	newLogger := logger.With().Str("NewFileReference", wopiContext.FileReference.String()).Logger()
	newCtx := middleware.WopiContextToCtx(newLogger.WithContext(ctx), wopiContext)

	// try to put the file
	putResponse, err := ccs.PutFile(newCtx, stream, streamLength, "")
	if err != nil {
		newLogger.Error().Err(err).Msg("PutRelativeFileRelative: put file failed") // or here
		return nil, err
	}

	lockID := ""
	if putResponse.Headers != nil {
		lockID = putResponse.Headers[HeaderWopiLock]
	}

	var newInfo *providerv1beta1.ResourceInfo
	webURL, err := url.Parse(f.cfg.Commons.OpenCloudURL)
	if err != nil {
		return nil, err
	}

	switch putResponse.Status {
	case 200: // success case, so don't do anything
	case 409:
		if newInfo, err = f.adjustWopiReference(ctx, &wopiContext, newLogger); err != nil {
			return nil, err
		}
		// if conflict generate a different name and retry.
		// this should happen only once
		wopiSrcURL, err2 := f.generateWOPISrc(wopiContext, newLogger)
		if err2 != nil {
			newLogger.Error().
				Err(err2).
				Str("LockID", lockID).
				Msg("PutRelativeFileRelative: error generating the WOPISrc parameter for conflict response")
			return nil, err2
		}

		actualFilename, _ := f.extractFilenameAndPrefix(target)
		finalTarget := f.generatePrefix() + " " + actualFilename

		newLogger.Error().
			Str("LockID", lockID).
			Msg("PutRelativeFileRelative: error conflict")

		// need to build the response ourselves
		return &ConnectorResponse{
			Status: 409,
			Headers: map[string]string{
				HeaderWopiValidRT:           finalTarget,
				HeaderWopiLock:              lockID,
				HeaderWopiLockFailureReason: "Lock Conflict",
			},
			Body: map[string]interface{}{
				"Name":        target,
				"Url":         wopiSrcURL.String(),
				"HostViewUrl": createHostUrl("view", webURL, strings.ToLower(f.cfg.App.Name), newInfo),
				"HostEditUrl": createHostUrl("write", webURL, strings.ToLower(f.cfg.App.Name), newInfo),
			},
		}, nil
	default:
		newLogger.Error().
			Str("LockID", lockID).
			Msg("PutRelativeFileRelative: put file failed with unhandled status")
		return nil, NewConnectorError(putResponse.Status, "put file failed with unhandled status")
	}

	if newInfo, err = f.adjustWopiReference(ctx, &wopiContext, newLogger); err != nil {
		return nil, err
	}

	wopiSrcURL, err := f.generateWOPISrc(wopiContext, newLogger)
	if err != nil {
		newLogger.Error().Err(err).Msg("PutRelativeFileRelative: error generating the WOPISrc parameter")
		return nil, err
	}

	newLogger.Debug().Msg("PutRelativeFileRelative: success")

	return NewResponseSuccessBodyNameUrl(
		target,
		wopiSrcURL.String(),
		createHostUrl("write", webURL, strings.ToLower(f.cfg.App.Name), newInfo),
		createHostUrl("view", webURL, strings.ToLower(f.cfg.App.Name), newInfo),
	), nil
}

// DeleteFile will delete the requested file
// https://learn.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/files/deletefile
//
// The lock isn't part of the documentation, but it might be possible to
// delete a file as long as you have the lock. In addition, we'll need to
// return the lock if there is a conflict.
//
// The context MUST have a WOPI context, otherwise an error will be returned.
// You can pass a pre-configured zerologger instance through the context that
// will be used to log messages.
//
// Note that this method isn't required and it's likely used just for the
// WOPI validator
func (f *FileConnector) DeleteFile(ctx context.Context, lockID string) (*ConnectorResponse, error) {
	wopiContext, err := middleware.WopiContextFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	logger := zerolog.Ctx(ctx).With().
		Str("RequestedLockID", lockID).
		Logger()

	var deleteRes *providerv1beta1.DeleteResponse
	deleteReq := &providerv1beta1.DeleteRequest{
		Ref:    wopiContext.FileReference,
		LockId: lockID,
	}

	// we'll retry the request after a while if we get a "TOO_EARLY" code
	for retries := 0; deleteRes == nil || deleteRes.GetStatus().GetCode() == rpcv1beta1.Code_CODE_TOO_EARLY; retries++ {
		gwc, err := f.gws.Next()
		if err != nil {
			return nil, err
		}
		deleteRes, err = gwc.Delete(ctx, deleteReq)
		if err != nil {
			logger.Error().Err(err).Msg("DeleteFile: stat failed")
			return nil, err
		}

		if deleteRes.GetStatus().GetCode() == rpcv1beta1.Code_CODE_TOO_EARLY {
			// starting from 20ms, double the waiting time for each retry
			// capping at 5 secs
			var waitingTime time.Duration
			waitingTime = (20 * time.Millisecond) << retries
			if waitingTime.Seconds() > 5 {
				waitingTime = 5 * time.Second
			}

			logger.Warn().
				Str("StatusCode", deleteRes.GetStatus().GetCode().String()).
				Str("StatusMsg", deleteRes.GetStatus().GetMessage()).
				Dur("WaitingTime", waitingTime).
				Int("Retries", retries).
				Msg("DeleteFile: file isn't ready yet. Retrying")

			time.Sleep(waitingTime)
		}
	}

	if deleteRes.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
		logger.Error().
			Str("StatusCode", deleteRes.GetStatus().GetCode().String()).
			Str("StatusMsg", deleteRes.GetStatus().GetMessage()).
			Msg("DeleteFile: delete failed with unexpected status")

		if deleteRes.GetStatus().GetCode() == rpcv1beta1.Code_CODE_NOT_FOUND {
			// don't bother to check for locks of a missing file
			logger.Error().Msg("DeleteFile: tried to delete a missing file")
			return NewResponse(404), nil
		}

		// check if the file is locked to return a proper lockID
		req := &providerv1beta1.GetLockRequest{
			Ref: wopiContext.FileReference,
		}

		gwc, err := f.gws.Next()
		if err != nil {
			return nil, err
		}
		resp, err2 := gwc.GetLock(ctx, req)
		if err2 != nil {
			logger.Error().Err(err2).Msg("DeleteFile: GetLock failed")
			return nil, err2
		}

		if resp.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
			logger.Error().
				Str("StatusCode", resp.GetStatus().GetCode().String()).
				Str("StatusMsg", resp.GetStatus().GetMessage()).
				Msg("DeleteFile: GetLock failed with unexpected status")
			return NewResponse(500), nil
		}

		if resp.GetLock() != nil {
			logger.Error().
				Str("LockID", resp.GetLock().GetLockId()).
				Msg("DeleteFile: file is locked")
			return NewResponseLockConflict(resp.GetLock().GetLockId(), "File is locked"), nil
		} else {
			// return the original error since the file isn't locked
			logger.Error().Msg("DeleteFile: delete failed on unlocked file")
			return NewResponse(500), nil
		}
	}
	logger.Debug().Msg("DeleteFile: success")
	return NewResponse(200), nil
}

// RenameFile will rename the requested file
// https://learn.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/files/renamefile
//
// The "target" filename must be UTF8-encoded. The conversion between UTF7 and
// UTF8 must happen outside this function.
//
// The context MUST have a WOPI context, otherwise an error will be returned.
// You can pass a pre-configured zerologger instance through the context that
// will be used to log messages.
//
// The method will return the final target name as first return value (target
// is just a suggestion, so it could have changed) and the actual lockId in
// case of conflict as second return value, otherwise the returned lockId will
// be empty.
func (f *FileConnector) RenameFile(ctx context.Context, lockID, target string) (*ConnectorResponse, error) {
	wopiContext, err := middleware.WopiContextFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	logger := zerolog.Ctx(ctx).With().
		Str("RequestedLockID", lockID).
		Str("RenameTarget", target).
		Logger()

	gwc, err := f.gws.Next()
	if err != nil {
		return nil, err
	}
	// stat the current file in order to get the reference of the parent folder
	oldStatRes, err := gwc.Stat(ctx, &providerv1beta1.StatRequest{
		Ref: wopiContext.FileReference,
	})
	if err != nil {
		logger.Error().Err(err).Msg("RenameFile: stat failed")
		return nil, err
	}

	if oldStatRes.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
		if oldStatRes.GetStatus().GetCode() == rpcv1beta1.Code_CODE_NOT_FOUND {
			logger.Error().Msg("RenameFile: file not found")
			return NewResponse(404), nil
		} else {
			logger.Error().
				Str("StatusCode", oldStatRes.GetStatus().GetCode().String()).
				Str("StatusMsg", oldStatRes.GetStatus().GetMessage()).
				Msg("RenameFile: stat failed with unexpected status")
			return NewResponse(500), nil
		}
	}

	// the target doesn't include the extension
	targetWithExt := target + path.Ext(oldStatRes.GetInfo().GetPath())
	finalTarget := targetWithExt
	for isDone := false; !isDone; {
		targetPath := utils.MakeRelativePath(finalTarget)
		// need to change the file reference of the wopicontext to point to the new path
		targetFileReference := &providerv1beta1.Reference{
			ResourceId: oldStatRes.GetInfo().GetParentId(),
			Path:       targetPath,
		}

		// add the new file reference to the log context
		newLogger := logger.With().Str("NewFileReference", targetFileReference.String()).Logger()

		gwc, err = f.gws.Next()
		if err != nil {
			return nil, err
		}
		// try to put the file. It mustn't return a 400 or 409
		moveRes, err := gwc.Move(ctx, &providerv1beta1.MoveRequest{
			Source:      wopiContext.FileReference,
			Destination: targetFileReference,
			LockId:      lockID,
		})
		if err != nil {
			newLogger.Error().Err(err).Msg("RenameFile: move failed")
			return nil, err
		}
		if moveRes.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
			if moveRes.GetStatus().GetCode() == rpcv1beta1.Code_CODE_LOCKED || moveRes.GetStatus().GetCode() == rpcv1beta1.Code_CODE_ABORTED {
				currentLockID := ""
				if oldStatRes.GetInfo().GetLock() != nil {
					currentLockID = oldStatRes.GetInfo().GetLock().GetLockId()
				}
				newLogger.Error().
					Str("LockID", currentLockID).
					Str("StatusCode", moveRes.GetStatus().GetCode().String()).
					Str("StatusMsg", moveRes.GetStatus().GetMessage()).
					Msg("RenameFile: conflict")
				return NewResponseLockConflict(currentLockID, "Lock Conflict"), nil
			}

			if moveRes.GetStatus().GetCode() == rpcv1beta1.Code_CODE_ALREADY_EXISTS {
				// try to generate a different name. This should happen only once
				actualFilename, _ := f.extractFilenameAndPrefix(targetWithExt)
				finalTarget = f.generatePrefix() + " " + actualFilename
			} else {
				// TODO: code 400 might happen, what to do?
				// in other cases, just return the error
				newLogger.Error().
					Str("StatusCode", moveRes.GetStatus().GetCode().String()).
					Str("StatusMsg", moveRes.GetStatus().GetMessage()).
					Msg("RenameFile: move failed with unexpected status")

				return NewResponse(500), nil
			}
		} else {
			// if the put is successful, exit the loop and move on
			isDone = true
			logger = newLogger
		}
	}

	logger.Debug().Msg("RenameFile: success")
	// return the final filename without extension
	return NewResponseSuccessBodyName(strings.TrimSuffix(path.Base(finalTarget), path.Ext(finalTarget))), nil
}

// CheckFileInfo returns information about the requested file and capabilities of the wopi server
// https://docs.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/files/checkfileinfo
//
// The context MUST have a WOPI context, otherwise an error will be returned.
// You can pass a pre-configured zerologger instance through the context that
// will be used to log messages.
//
// If the operation is successful, a "FileInfo" instance will be returned,
// otherwise the "FileInfo" will be empty and an error will be returned.
func (f *FileConnector) CheckFileInfo(ctx context.Context) (*ConnectorResponse, error) {
	wopiContext, err := middleware.WopiContextFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	logger := zerolog.Ctx(ctx)

	gwc, err := f.gws.Next()
	if err != nil {
		return nil, err
	}

	fileRef := wopiContext.FileReference
	// set path to empty string to get the full path from the stat
	fileRef.Path = ""
	statRes, err := gwc.Stat(ctx, &providerv1beta1.StatRequest{
		Ref: fileRef,
	})

	if err != nil {
		logger.Error().Err(err).Msg("CheckFileInfo: stat failed")
		return nil, err
	}

	if statRes.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
		logger.Error().
			Str("StatusCode", statRes.GetStatus().GetCode().String()).
			Str("StatusMsg", statRes.GetStatus().GetMessage()).
			Msg("CheckFileInfo: stat failed with unexpected status")

		if statRes.GetStatus().GetCode() == rpcv1beta1.Code_CODE_NOT_FOUND {
			return NewResponse(404), nil
		}
		return NewResponse(500), nil
	}

	// If a not known app name is used, consider "Microsoft" as default.
	// This will help with the CI because we're using a "FakeOffice" app
	// for the wopi validator, which requires a Microsoft fileinfo
	var info fileinfo.FileInfo
	switch strings.ToLower(f.cfg.App.Product) {
	case "collabora":
		info = &fileinfo.Collabora{}
	case "onlyoffice":
		info = &fileinfo.OnlyOffice{}
	default:
		info = &fileinfo.Microsoft{}
	}

	hexEncodedOwnerId := hex.EncodeToString([]byte(statRes.GetInfo().GetOwner().GetOpaqueId() + "@" + statRes.GetInfo().GetOwner().GetIdp()))

	// UserId must use only alphanumeric chars (https://learn.microsoft.com/en-us/microsoft-365/cloud-storage-partner-program/rest/files/checkfileinfo/checkfileinfo-response#requirements-for-user-identity-properties)
	// assign userId, userFriendlyName and isAnonymousUser
	// assume we don't have a wopiContext.User
	randomID, _ := uuid.NewUUID()
	userId := hex.EncodeToString([]byte("guest-" + randomID.String()))
	userFriendlyName := "Guest " + randomID.String()
	isAnonymousUser := true

	isPublicShare := false
	user := ctxpkg.ContextMustGetUser(ctx)
	if user.String() != "" {
		// if we have a wopiContext.User
		isPublicShare = utils.ExistsInOpaque(user.GetOpaque(), "public-share-role")
		if !isPublicShare {
			hexEncodedWopiUserId := hex.EncodeToString([]byte(user.GetId().GetOpaqueId() + "@" + user.GetId().GetIdp()))
			isAnonymousUser = false
			userFriendlyName = user.GetDisplayName()
			userId = hexEncodedWopiUserId
		}
	}

	breadcrumbFolderName := path.Dir(statRes.GetInfo().GetPath())
	if breadcrumbFolderName == "." || breadcrumbFolderName == "" || breadcrumbFolderName == "/" {
		breadcrumbFolderName = statRes.GetInfo().GetSpace().GetName()
	}

	ocURL, err := url.Parse(f.cfg.Commons.OpenCloudURL)
	if err != nil {
		return nil, err
	}
	collaborationURL, err := url.Parse(f.cfg.Wopi.WopiSrc)
	if err != nil {
		return nil, err
	}
	privateLinkURL := &url.URL{}
	*privateLinkURL = *ocURL
	privateLinkURL.Path = path.Join(ocURL.Path, "f", storagespace.FormatResourceID(statRes.GetInfo().GetId()))
	parentFolderURL := &url.URL{}
	*parentFolderURL = *ocURL
	if !isPublicShare {
		parentFolderURL.Path = path.Join(ocURL.Path, "f", storagespace.FormatResourceID(statRes.GetInfo().GetParentId()))
	} else {
		if scopes, ok := ctxpkg.ContextGetScopes(ctx); ok {
			publicShare := &link.PublicShare{}
			if err := f.getScopeByKeyPrefix(scopes, "publicshare:", publicShare); err == nil {
				parentFolderURL.Path = path.Join(ocURL.Path, "s", publicShare.GetToken())
			} else {
				logger.Error().Err(err).Msg("CheckFileInfo: error getting public share scope")
			}
		} else {
			logger.Error().Err(err).Msg("CheckFileInfo: error getting scopes from the context")
		}
	}
	// fileinfo map
	infoMap := map[string]interface{}{
		fileinfo.KeyOwnerID:           hexEncodedOwnerId,
		fileinfo.KeySize:              int64(statRes.GetInfo().GetSize()),
		fileinfo.KeyVersion:           getVersion(statRes.GetInfo().GetMtime()),
		fileinfo.KeyBaseFileName:      path.Base(statRes.GetInfo().GetPath()),
		fileinfo.KeyBreadcrumbDocName: path.Base(statRes.GetInfo().GetPath()),
		// to get the folder we actually need to do a GetPath() request
		fileinfo.KeyBreadcrumbFolderName: breadcrumbFolderName,
		fileinfo.KeyBreadcrumbFolderURL:  parentFolderURL.String(),

		fileinfo.KeyHostViewURL:    createHostUrl("view", ocURL, f.cfg.App.Name, statRes.GetInfo()),
		fileinfo.KeyHostEditURL:    createHostUrl("write", ocURL, f.cfg.App.Name, statRes.GetInfo()),
		fileinfo.KeyFileSharingURL: createShareUrl(privateLinkURL),
		fileinfo.KeyFileVersionURL: createVersionsUrl(privateLinkURL),

		fileinfo.KeyEnableOwnerTermination:     true, // only for collabora
		fileinfo.KeySupportsExtendedLockLength: true,
		fileinfo.KeySupportsGetLock:            true,
		fileinfo.KeySupportsLocks:              true,
		fileinfo.KeySupportsUpdate:             true,
		fileinfo.KeySupportsDeleteFile:         true,
		fileinfo.KeySupportsRename:             true,

		fileinfo.KeyIsAnonymousUser:  isAnonymousUser,
		fileinfo.KeyUserFriendlyName: userFriendlyName,
		fileinfo.KeyUserID:           userId,

		fileinfo.KeyPostMessageOrigin:            f.cfg.Commons.OpenCloudURL,
		fileinfo.KeyLicenseCheckForEditIsEnabled: f.cfg.App.LicenseCheckEnable,

		fileinfo.KeyUserCanNotWriteRelative: false,
	}

	switch wopiContext.ViewMode {
	case appproviderv1beta1.ViewMode_VIEW_MODE_READ_WRITE:
		infoMap[fileinfo.KeyUserCanWrite] = true
		infoMap[fileinfo.KeyUserCanRename] = true

	case appproviderv1beta1.ViewMode_VIEW_MODE_READ_ONLY:
		// nothing special to do here for now

	case appproviderv1beta1.ViewMode_VIEW_MODE_VIEW_ONLY:
		infoMap[fileinfo.KeyDisableExport] = true // only for collabora
		infoMap[fileinfo.KeyDisableCopy] = true   // only for collabora
		infoMap[fileinfo.KeyDisablePrint] = true
		if !isPublicShare {
			infoMap[fileinfo.KeyWatermarkText] = f.watermarkText(user) // only for collabora
		}
	}

	// if the file content is empty and a template reference is set, add the template source URL
	if wopiContext.TemplateReference != nil && statRes.GetInfo().GetSize() == 0 {
		if tu, err := f.createDownloadURL(wopiContext, collaborationURL); err == nil {
			infoMap[fileinfo.KeyTemplateSource] = tu
		}
	}

	info.SetProperties(infoMap)

	logger.Debug().Interface("FileInfo", info).Msg("CheckFileInfo: success")
	return NewResponseSuccessBody(info), nil
}

// createDownloadURL will create a download URL for the template file.
// It uses a new wopi context with the template reference set as the file reference
// and a reva access token to download the file.
func (f *FileConnector) createDownloadURL(wopiContext middleware.WopiContext, collaborationURL *url.URL) (string, error) {
	templateContext := wopiContext
	templateContext.FileReference = wopiContext.TemplateReference
	templateContext.TemplateReference = nil

	token, _, err := middleware.GenerateWopiToken(templateContext, f.cfg, f.store)
	if err != nil {
		return "", err
	}
	downloadURL := *collaborationURL
	downloadURL.Path = path.Join(
		collaborationURL.Path,
		"wopi/templates/",
		helpers.HashResourceId(templateContext.FileReference.GetResourceId()),
	)
	q := downloadURL.Query()
	q.Add("access_token", token)
	downloadURL.RawQuery = q.Encode()
	return downloadURL.String(), nil
}

func createHostUrl(mode string, u *url.URL, appName string, info *providerv1beta1.ResourceInfo) string {
	webUrl := createAppExternalURL(u, appName, info)
	addURLParams(webUrl, map[string]string{"view_mode": mode})
	return webUrl.String()
}

func createShareUrl(u *url.URL) string {
	shareURL := *u
	addURLParams(&shareURL, map[string]string{"details": "sharing"})
	return shareURL.String()
}

func createVersionsUrl(u *url.URL) string {
	versionsURL := *u
	addURLParams(&versionsURL, map[string]string{"details": "versions"})
	return versionsURL.String()
}

func addURLParams(u *url.URL, params map[string]string) {
	q := u.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	u.RawQuery = q.Encode()
}

func createAppExternalURL(u *url.URL, appName string, info *providerv1beta1.ResourceInfo) *url.URL {
	spaceAlias := utils.ReadPlainFromOpaque(info.GetSpace().GetOpaque(), "spaceAlias")
	appExternalURL := *u
	appExternalURL.Path = path.Join(u.Path, "external-"+strings.ToLower(appName), spaceAlias, info.GetPath())
	q := u.Query()
	q.Add("fileId", storagespace.FormatResourceID(info.GetId()))
	appExternalURL.RawQuery = q.Encode()
	return &appExternalURL
}

func (f *FileConnector) watermarkText(user *userv1beta1.User) string {
	if user != nil {
		return strings.TrimSpace(user.GetDisplayName() + " " + user.GetMail())
	}
	return "Watermark"
}

// extractFilenameAndPrefix will extract the filename and the prefix from the
// provided filename. The prefix in the filename must have been generated
// using the generatePrefix() method below and there must be a space between
// the prefix and the actual filename. For example "AZBVUm5F Document99.docx".
//
// In order to prevent false positives, all prefixes must have been generated
// after Jan 1th, 2020 (so any generated prefix should be correctly detected).
//
// This method will return the expected filename as first value, and the prefix
// as second value. If the provided filename doesn't have a valid prefix, the
// whole filename will be returned as first parameter, and the second will be
// the empty string.
func (f *FileConnector) extractFilenameAndPrefix(filename string) (string, string) {
	before, after, found := strings.Cut(filename, " ")
	if !found {
		return filename, ""
	}

	// try to decode the prefix
	byteArray, err := base64.RawURLEncoding.DecodeString(before)
	if err != nil {
		// filename not prefixed
		return filename, ""
	}

	if len(byteArray) > 8 {
		// weird prefix, likely part of a regular filename, probably a false positive
		// return the whole filename
		return filename, ""
	}

	if len(byteArray) < 8 {
		newArray := make([]byte, 8)
		for i := 0; i < len(byteArray); i++ {
			// first bytes should be 0
			newArray[8-len(byteArray)+i] = byteArray[i]
		}
		byteArray = newArray
	}

	millis := binary.BigEndian.Uint64(byteArray)
	t := time.UnixMilli(int64(millis)) // the uint64 should fit

	baseT := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	if t.Before(baseT) {
		// decoded integer isn't recent and is too low, likely a false positive
		// return the whole filename
		return filename, ""
	}
	return after, before
}

// generatePrefix will generate a short unique prefix based on the current
// time. This prefix can be used as part of a filename
func (f *FileConnector) generatePrefix() string {
	byteArray := binary.BigEndian.AppendUint64([]byte{}, uint64(time.Now().UnixMilli()))
	return base64.RawURLEncoding.EncodeToString(bytes.TrimLeft(byteArray, "\x00"))
}

// The adjustWopiReference should be called first so the file reference
// contains the resource id of the target file without the path
// (storage, opaque and space points directly to the file). The path component
// will be ignored
func (f *FileConnector) generateWOPISrc(wopiContext middleware.WopiContext, logger zerolog.Logger) (*url.URL, error) {
	// get the WOPI token for the new file
	accessToken, _, err := middleware.GenerateWopiToken(wopiContext, f.cfg, f.store)
	if err != nil {
		logger.Error().Err(err).Msg("generateWOPISrc: failed to generate access token for the new file")
		return nil, err
	}

	// get the reference
	fileRef := helpers.HashResourceId(wopiContext.FileReference.GetResourceId())

	// generate the URL for the WOPI app to access the new created file
	wopiSrcURL, err := wopisrc.GenerateWopiSrc(fileRef, f.cfg)
	if err != nil {
		logger.Error().Err(err).Msg("generateWOPISrc: failed to generate WOPISrc URL for the new file")
		return nil, err
	}
	q := wopiSrcURL.Query()
	q.Add("access_token", accessToken)
	wopiSrcURL.RawQuery = q.Encode()
	return wopiSrcURL, nil
}

func (f *FileConnector) adjustWopiReference(ctx context.Context, wopiContext *middleware.WopiContext, logger zerolog.Logger) (*providerv1beta1.ResourceInfo, error) {
	gwc, err := f.gws.Next()
	if err != nil {
		return nil, err
	}
	// using resourceid + path won't do for WOPI, we need just the resource if of the new file
	// the wopicontext has resourceid + path, which is good enough for the stat request
	newStatRes, err := gwc.Stat(ctx, &providerv1beta1.StatRequest{
		Ref: wopiContext.FileReference,
	})
	if err != nil {
		logger.Error().Err(err).Msg("stat failed")
		return nil, err
	}

	if newStatRes.GetStatus().GetCode() != rpcv1beta1.Code_CODE_OK {
		logger.Error().
			Str("StatusCode", newStatRes.GetStatus().GetCode().String()).
			Str("StatusMsg", newStatRes.GetStatus().GetMessage()).
			Msg("stat failed with unexpected status")
		return nil, NewConnectorError(500, newStatRes.GetStatus().GetCode().String()+" "+newStatRes.GetStatus().GetMessage())
	}
	// adjust the reference in the wopi context to use only the resourceid without the path
	wopiContext.FileReference = &providerv1beta1.Reference{
		ResourceId: newStatRes.GetInfo().GetId(),
	}

	return newStatRes.GetInfo(), nil
}

// getScopeByKeyPrefix returns the scope from the AccessToken Scope map by key prefix
func (f *FileConnector) getScopeByKeyPrefix(scopes map[string]*auth.Scope, keyPrefix string, m proto.Message) error {
	for k, v := range scopes {
		if strings.HasPrefix(k, keyPrefix) && v.Resource.Decoder == "json" {
			err := utils.UnmarshalJSONToProtoV1(v.Resource.Value, m)
			if err != nil {
				return fmt.Errorf("can't unmarshal public share from scope: %w", err)
			}
			return nil
		}
	}
	return fmt.Errorf("scope %s not found", keyPrefix)
}
