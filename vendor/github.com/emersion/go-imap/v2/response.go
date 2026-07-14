package imap

import (
	"fmt"
	"strings"
)

// StatusResponseType is a generic status response type.
type StatusResponseType string

const (
	StatusResponseTypeOK      StatusResponseType = "OK"
	StatusResponseTypeNo      StatusResponseType = "NO"
	StatusResponseTypeBad     StatusResponseType = "BAD"
	StatusResponseTypePreAuth StatusResponseType = "PREAUTH"
	StatusResponseTypeBye     StatusResponseType = "BYE"
)

// ResponseCode is a response code.
type ResponseCode string

const (
	ResponseCodeAlert                ResponseCode = "ALERT"
	ResponseCodeAlreadyExists        ResponseCode = "ALREADYEXISTS"
	ResponseCodeAuthenticationFailed ResponseCode = "AUTHENTICATIONFAILED"
	ResponseCodeAuthorizationFailed  ResponseCode = "AUTHORIZATIONFAILED"
	ResponseCodeBadCharset           ResponseCode = "BADCHARSET"
	ResponseCodeCannot               ResponseCode = "CANNOT"
	ResponseCodeClientBug            ResponseCode = "CLIENTBUG"
	ResponseCodeContactAdmin         ResponseCode = "CONTACTADMIN"
	ResponseCodeCorruption           ResponseCode = "CORRUPTION"
	ResponseCodeExpired              ResponseCode = "EXPIRED"
	ResponseCodeHasChildren          ResponseCode = "HASCHILDREN"
	ResponseCodeInUse                ResponseCode = "INUSE"
	ResponseCodeLimit                ResponseCode = "LIMIT"
	ResponseCodeNonExistent          ResponseCode = "NONEXISTENT"
	ResponseCodeNoPerm               ResponseCode = "NOPERM"
	ResponseCodeOverQuota            ResponseCode = "OVERQUOTA"
	ResponseCodeParse                ResponseCode = "PARSE"
	ResponseCodePrivacyRequired      ResponseCode = "PRIVACYREQUIRED"
	ResponseCodeServerBug            ResponseCode = "SERVERBUG"
	ResponseCodeTryCreate            ResponseCode = "TRYCREATE"
	ResponseCodeUnavailable          ResponseCode = "UNAVAILABLE"
	ResponseCodeUnknownCTE           ResponseCode = "UNKNOWN-CTE"

	// METADATA
	ResponseCodeTooMany   ResponseCode = "TOOMANY"
	ResponseCodeNoPrivate ResponseCode = "NOPRIVATE"

	// APPENDLIMIT
	ResponseCodeTooBig ResponseCode = "TOOBIG"
)

// StatusResponse is a generic status response.
//
// See RFC 9051 section 7.1.
type StatusResponse struct {
	Type StatusResponseType
	Code ResponseCode
	Text string
}

// Error is an IMAP error caused by a status response.
type Error StatusResponse

var _ error = (*Error)(nil)

// Error implements the error interface.
func (err *Error) Error() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "imap: %v", err.Type)
	if err.Code != "" {
		fmt.Fprintf(&sb, " [%v]", err.Code)
	}
	text := err.Text
	if text == "" {
		text = "<unknown>"
	}
	fmt.Fprintf(&sb, " %v", text)
	return sb.String()
}
