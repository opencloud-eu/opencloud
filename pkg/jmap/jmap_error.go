package jmap

import (
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
	JmapErrorMethodLevel
	JmapErrorSetError
)

type Error interface {
	Code() int
	error
}

type SimpleError struct {
	code int
	err  error
}

var _ Error = &SimpleError{}

func (e SimpleError) Code() int {
	return e.code
}
func (e SimpleError) Unwrap() error {
	return e.err
}
func (e SimpleError) Error() string {
	if e.err != nil {
		return e.err.Error()
	} else {
		return ""
	}
}

func simpleError(err error, code int) Error {
	if err != nil {
		return SimpleError{code: code, err: err}
	} else {
		return nil
	}
}

func setErrorError(err SetError, objectType ObjectType) Error {
	var e error
	if len(err.Properties) > 0 {
		e = fmt.Errorf("failed to modify %s due to %s error in properties [%s]: %s", objectType, err.Type, strings.Join(err.Properties, ", "), err.Description)
	} else {
		e = fmt.Errorf("failed to modify %s due to %s error: %s", objectType, err.Type, err.Description)
	}
	return SimpleError{code: JmapErrorSetError, err: e}
}
