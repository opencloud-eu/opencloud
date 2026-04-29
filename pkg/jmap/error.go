package jmap

import (
	"errors"
	"fmt"
	"strings"
)

const (
	JmapErrorAuthenticationFailed = iota
	JmapErrorInvalidHttpRequest
	JmapErrorServerResponse
	JmapErrorReadingResponseBody
	JmapErrorDecodingResponseBody
	JmapErrorEncodingRequestBody
	JmapErrorCreatingRequest
	JmapErrorSendingRequest
	JmapErrorInvalidSessionResponse
	JmapErrorInvalidJmapRequestPayload
	JmapErrorInvalidJmapResponsePayload
	JmapErrorSetError
	JmapErrorTooManyMethodCalls
	JmapErrorUnspecifiedType
	JmapErrorServerUnavailable
	JmapErrorServerFail
	JmapErrorUnknownMethod
	JmapErrorInvalidArguments
	JmapErrorInvalidResultReference
	JmapErrorForbidden
	JmapErrorAccountNotFound
	JmapErrorAccountNotSupportedByMethod
	JmapErrorAccountReadOnly
	JmapErrorFailedToEstablishWssConnection
	JmapErrorWssConnectionResponseMissingJmapSubprotocol
	JmapErrorWssFailedToSendWebSocketPushEnable
	JmapErrorWssFailedToSendWebSocketPushDisable
	JmapErrorWssFailedToClose
	JmapErrorWssFailedToRetrieveSession
	JmapErrorSocketPushUnsupported
	JmapErrorMissingCreatedObject
	JmapErrorInvalidObjectState
	JmapErrorPatchObjectSerialization
	JmapErrorInvalidProperties
)

var (
	errTooManyMethodCalls = errors.New("the amount of methodCalls in the request body would exceed the maximum that is configured in the session")
)

type Error interface {
	Code() int
	error
}

type JmapError struct {
	code        int
	err         error
	typ         string
	description string
}

var _ Error = &JmapError{}

func (e JmapError) Code() int {
	return e.code
}
func (e JmapError) Unwrap() error {
	return e.err
}
func (e JmapError) Error() string {
	if e.err != nil {
		return e.err.Error()
	} else {
		return ""
	}
}
func (e JmapError) Type() string {
	return e.typ
}
func (e JmapError) Description() string {
	return e.description
}

func jmapError(err error, code int) Error {
	if err != nil {
		return JmapError{code: code, err: err}
	} else {
		return nil
	}
}

func jmapResponseError(code int, err error, typ string, description string) JmapError {
	return JmapError{
		code:        code,
		err:         err,
		typ:         typ,
		description: description,
	}
}

func setErrorError(err SetError, objectType ObjectType) Error {
	var e error
	if len(err.Properties) > 0 {
		e = fmt.Errorf("failed to modify %s due to %s error in properties [%s]: %s", objectType, err.Type, strings.Join(err.Properties, ", "), err.Description)
	} else {
		e = fmt.Errorf("failed to modify %s due to %s error: %s", objectType, err.Type, err.Description)
	}
	code := JmapErrorSetError
	switch err.Type {
	case SetErrorTypeInvalidProperties:
		code = JmapErrorInvalidProperties
	}
	return JmapError{code: code, err: e}
}
