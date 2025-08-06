package jmap

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

// https://jmap.io/spec-mail.html#identityget
func (j *Client) GetIdentity(accountId string, session *Session, ctx context.Context, logger *log.Logger) (IdentityGetResponse, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.logger(aid, "GetIdentity", session, logger)
	cmd, err := request(invocation(IdentityGet, IdentityGetCommand{AccountId: aid}, "0"))
	if err != nil {
		return IdentityGetResponse{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (IdentityGetResponse, Error) {
		var response IdentityGetResponse
		err = retrieveResponseMatchParameters(body, IdentityGet, "0", &response)
		return response, simpleError(err, JmapErrorInvalidJmapResponsePayload)
	})
}
