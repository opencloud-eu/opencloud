package def

import (
	"errors"
	"fmt"
)

var (
	// base errors

	ErrMsgpack = errors.New("")

	// decoding errors

	ErrNoData                 = fmt.Errorf("%wno data", ErrMsgpack)
	ErrHasLeftOver            = fmt.Errorf("%wdata has left over", ErrMsgpack)
	ErrReceiverNotPointer     = fmt.Errorf("%wreceiver not pointer", ErrMsgpack)
	ErrNotMatchArrayElement   = fmt.Errorf("%wnot match array element", ErrMsgpack)
	ErrCanNotDecode           = fmt.Errorf("%winvalid code", ErrMsgpack)
	ErrCanNotSetSliceAsMapKey = fmt.Errorf("%wcan not set slice as map key", ErrMsgpack)
	ErrCanNotSetMapAsMapKey   = fmt.Errorf("%wcan not set map as map key", ErrMsgpack)
	ErrCanNotSetBinAsMapKey   = fmt.Errorf("%wcan not set bin as map key", ErrMsgpack)
	ErrUnhashableMapKey       = fmt.Errorf("%wunhashable map key", ErrMsgpack)
	ErrExceededMaxDepth       = fmt.Errorf("%wexceeded max nesting depth", ErrMsgpack)

	// encoding errors

	ErrTooShortBytes         = fmt.Errorf("%wtoo short bytes", ErrMsgpack)
	ErrLackDataLengthToSlice = fmt.Errorf("%wdata length lacks to create slice", ErrMsgpack)
	ErrLackDataLengthToMap   = fmt.Errorf("%wdata length lacks to create map", ErrMsgpack)
	ErrUnsupportedType       = fmt.Errorf("%wunsupported type", ErrMsgpack)
	ErrUnsupportedLength     = fmt.Errorf("%wunsupported length", ErrMsgpack)
	ErrNotMatchLastIndex     = fmt.Errorf("%wnot match last index", ErrMsgpack)
)
