package connector

import (
	"strconv"

	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
)

// ConnectorResponse represent a response from the FileConnectorService.
// The ConnectorResponse is oriented to HTTP, so it has the Status, Headers
// and Body that the actual HTTP response should have. This includes HTTP
// errors with status 4xx and 5xx, which will also represent some error
// conditions for the FileConnectorService.
// Note that the Body is expected to be JSON-encoded outside before sending.
type ConnectorResponse struct {
	Status  int
	Headers map[string]string
	Body    any
}

// NewResponse creates a new ConnectorResponse with just the specified status.
// Headers and Body will be nil
func NewResponse(status int) *ConnectorResponse {
	return &ConnectorResponse{Status: status}
}

// NewResponse creates a new ConnectorResponse with the specified status
// and the "X-WOPI-Lock" header having the value in the lockID parameter.
//
// This is usually used for conflict responses where the current lock id needs
// to be returned, although the `GetLock` method also uses this method for a
// successful response (with the lock id included)
func NewResponseWithLock(status int, lockID string) *ConnectorResponse {
	return &ConnectorResponse{
		Status: status,
		Headers: map[string]string{
			HeaderWopiLock: lockID,
		},
	}
}

// NewResponseLockConflict creates a new ConnectorResponse with the status 409
// and the "X-WOPI-Lock" header having the value in the lockID parameter.
//
// This is used for conflict responses where the current lock id needs
// to be returned, although the `GetLock` method also uses this method for a
// successful response (with the lock id included)
// The lockFailureReason parameter will be included in the "X-WOPI-LockFailureReason".
func NewResponseLockConflict(lockID string, lockFailureReason string) *ConnectorResponse {
	return &ConnectorResponse{
		Status: 409,
		Headers: map[string]string{
			HeaderWopiLock:              lockID,
			HeaderWopiLockFailureReason: lockFailureReason,
		},
	}
}

// NewResponseWithVersion creates a new ConnectorResponse with the specified status
// and the "X-WOPI-ItemVersion" header having the value in the mtime parameter.
func NewResponseWithVersion(mtime *types.Timestamp) *ConnectorResponse {
	return &ConnectorResponse{
		Status: 200,
		Headers: map[string]string{
			HeaderWopiVersion: getVersion(mtime),
		},
	}
}

// NewResponseWithVersionAndLock creates a new ConnectorResponse with the specified status
// and the "X-WOPI-ItemVersion" header and the "X-WOPI-Lock" header
// having the values in the mtime and lockID parameters.
func NewResponseWithVersionAndLock(status int, mtime *types.Timestamp, lockID string) *ConnectorResponse {
	r := &ConnectorResponse{
		Status: status,
		Headers: map[string]string{
			HeaderWopiVersion: getVersion(mtime),
			HeaderWopiLock:    lockID,
		},
	}
	return r
}

// NewResponseSuccessBody creates a new ConnectorResponse with a fixed 200
// (success) status and the specified body. The headers will be nil.
//
// This is used for the `CheckFileInfo` method in order to return the fileinfo
func NewResponseSuccessBody(body any) *ConnectorResponse {
	return &ConnectorResponse{
		Status: 200,
		Body:   body,
	}
}

// NewResponseSuccessBodyName creates a new ConnectorResponse with a fixed 200
// (success) status and a "map[string]interface{}" body. The body will contain
// a "Name" key with the supplied name as value.
//
// This is used for the `RenameFile` method in order to return the final name
// of the renamed file if the operation is successful
func NewResponseSuccessBodyName(name string) *ConnectorResponse {
	return &ConnectorResponse{
		Status: 200,
		Body: map[string]any{
			"Name": name,
		},
	}
}

// NewResponseSuccessBodyNameUrl creates a new ConnectorResponse with a fixed
// 200 (success) status and a "map[string]interface{}" body. The body will
// contain "Name" and "Url" keys with their respective suplied values
//
// This is used in the `PutRelativeFile` methods (both suggested and relative).
func NewResponseSuccessBodyNameUrl(name, url string, hostEditURL string, hostViewURL string) *ConnectorResponse {
	return &ConnectorResponse{
		Status: 200,
		Body: map[string]any{
			"Name":        name,
			"Url":         url,
			"HostEditUrl": hostEditURL,
			"HostViewUrl": hostViewURL,
		},
	}
}

// ConnectorError defines an error in the connector. It contains an error code
// and a message.
// For convenience, the error code can be used as HTTP error code, although
// the connector shouldn't know anything about HTTP.
type ConnectorError struct {
	HttpCodeOut int
	Msg         string
}

// Error gets the error message
func (e *ConnectorError) Error() string {
	return e.Msg
}

// NewConnectorError creates a new connector error using the provided parameters
func NewConnectorError(code int, msg string) *ConnectorError {
	return &ConnectorError{
		HttpCodeOut: code,
		Msg:         msg,
	}
}

// ConnectorService is the interface to implement the WOPI operations. They're
// divided into multiple endpoints.
// The IFileConnector will implement the "File" endpoint
// The IContentConnector will implement the "File content" endpoint
type ConnectorService interface {
	GetFileConnector() FileConnectorService
	GetContentConnector() ContentConnectorService
}

// Connector will implement the WOPI operations.
// For convenience, the connector splits the operations based on the
// WOPI endpoints, so you'll need to get the specific connector first.
//
// Available endpoints:
// * "Files" -> GetFileConnector()
// * "File contents" -> GetContentConnector()
//
// Other endpoints aren't available for now.
type Connector struct {
	fileConnector    FileConnectorService
	contentConnector ContentConnectorService
}

// NewConnector creates a new connector
func NewConnector(fc FileConnectorService, cc ContentConnectorService) *Connector {
	return &Connector{
		fileConnector:    fc,
		contentConnector: cc,
	}
}

// GetFileConnector gets the file connector service associated to this connector
func (c *Connector) GetFileConnector() FileConnectorService {
	return c.fileConnector
}

// GetContentConnector gets the content connector service associated to this connector
func (c *Connector) GetContentConnector() ContentConnectorService {
	return c.contentConnector
}

// getVersion returns a string representation of the timestamp
func getVersion(timestamp *types.Timestamp) string {
	return "v" + strconv.FormatUint(timestamp.GetSeconds(), 10) +
		strconv.FormatUint(uint64(timestamp.GetNanos()), 10)
}
