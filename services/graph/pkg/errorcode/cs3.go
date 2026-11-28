package errorcode

import (
	"slices"

	cs3rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"

	"github.com/opencloud-eu/reva/v3/pkg/utils"
)

// FromCS3Status converts a CS3 status code and an error into a corresponding local Error representation.
//
// It takes a  *cs3rpc.Status, an error, and a variadic parameter of type cs3rpc.Code.
// If the error is not nil, it creates an Error object with the error message and a GeneralException code.
// If the error is nil, it evaluates the provided CS3 status code and returns an equivalent graph Error.
// If the CS3 status code does not have a direct equivalent within the app,
// or is ignored, a general purpose Error is returned.
//
// This function is particularly useful when dealing with CS3 responses,
// and a unified error handling within the application is necessary.
func FromCS3Status(status *cs3rpc.Status, inerr error, ignore ...cs3rpc.Code) error {
	err := Error{errorCode: GeneralException, msg: "unspecified error has occurred", origin: ErrorOriginCS3}

	if inerr != nil {
		err.msg = inerr.Error()
		return err
	}

	if status != nil {
		err.msg = status.GetMessage()
	}

	code := status.GetCode()
	switch {
	case slices.Contains(ignore, status.GetCode()):
		fallthrough
	case code == cs3rpc.Code_CODE_OK:
		return nil
	case code == cs3rpc.Code_CODE_NOT_FOUND:
		err.errorCode = ItemNotFound
	case code == cs3rpc.Code_CODE_PERMISSION_DENIED:
		err.errorCode = AccessDenied
	case code == cs3rpc.Code_CODE_UNAUTHENTICATED:
		err.errorCode = Unauthenticated
	case code == cs3rpc.Code_CODE_INVALID_ARGUMENT:
		err.errorCode = InvalidRequest
	case code == cs3rpc.Code_CODE_ALREADY_EXISTS:
		err.errorCode = NameAlreadyExists
	case code == cs3rpc.Code_CODE_FAILED_PRECONDITION:
		err.errorCode = InvalidRequest
	case code == cs3rpc.Code_CODE_OUT_OF_RANGE:
		err.errorCode = InvalidRange
	case code == cs3rpc.Code_CODE_UNIMPLEMENTED:
		err.errorCode = NotSupported
	case code == cs3rpc.Code_CODE_UNAVAILABLE:
		err.errorCode = ServiceNotAvailable
	case code == cs3rpc.Code_CODE_INSUFFICIENT_STORAGE:
		err.errorCode = QuotaLimitReached
	case code == cs3rpc.Code_CODE_LOCKED:
		err.errorCode = ItemIsLocked
	}

	return err
}

// FromStat transforms a *provider.StatResponse object and an error into an Error.
//
// It takes a stat of type *provider.StatResponse, an error, and a variadic parameter of type cs3rpc.Code.
// It invokes the FromCS3Status function with the StatResponse Status and the ignore codes.
func FromStat(stat *provider.StatResponse, err error, ignore ...cs3rpc.Code) error {
	// TODO: look into ResourceInfo to get the postprocessing state and map that to 425 status?
	return FromCS3Status(stat.GetStatus(), err, ignore...)
}

// FromUtilsStatusCodeError returns original error if `err` does not match to the statusCodeError type
func FromUtilsStatusCodeError(err error, ignore ...cs3rpc.Code) error {
	stat := utils.StatusCodeErrorToCS3Status(err)
	if stat == nil {
		return FromCS3Status(nil, err, ignore...)
	}
	return FromCS3Status(stat, nil, ignore...)
}
