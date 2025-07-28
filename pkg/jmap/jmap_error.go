package jmap

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
	return e.err.Error()
}

func simpleError(err error, code int) Error {
	if err != nil {
		return SimpleError{code: code, err: err}
	} else {
		return nil
	}
}
