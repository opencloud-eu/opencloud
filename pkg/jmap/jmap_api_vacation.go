package jmap

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

// https://jmap.io/spec-mail.html#vacationresponseget
func (j *Client) GetVacationResponse(accountId string, session *Session, ctx context.Context, logger *log.Logger) (VacationResponseGetResponse, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.logger(aid, "GetVacationResponse", session, logger)
	cmd, err := request(invocation(VacationResponseGet, VacationResponseGetCommand{AccountId: aid}, "0"))
	if err != nil {
		return VacationResponseGetResponse{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (VacationResponseGetResponse, Error) {
		var response VacationResponseGetResponse
		err = retrieveResponseMatchParameters(body, VacationResponseGet, "0", &response)
		return response, simpleError(err, JmapErrorInvalidJmapResponsePayload)
	})
}
