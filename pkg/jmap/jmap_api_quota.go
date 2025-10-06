package jmap

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

func (j *Client) GetQuotas(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (QuotaGetResponse, SessionState, Language, Error) {
	logger = j.logger("GetQuotas", session, logger)
	cmd, err := j.request(session, logger, invocation(CommandQuotaGet, QuotaGetCommand{AccountId: accountId}, "0"))
	if err != nil {
		return QuotaGetResponse{}, "", "", err
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (QuotaGetResponse, Error) {
		var response QuotaGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandQuotaGet, "0", &response)
		if err != nil {
			return QuotaGetResponse{}, err
		}
		return response, nil
	})
}
