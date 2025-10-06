package jmap

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

func (j *Client) GetQuotas(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string]QuotaGetResponse, SessionState, Language, Error) {
	logger = j.logger("GetQuotas", session, logger)

	uniqueAccountIds := structs.Uniq(accountIds)

	invocations := make([]Invocation, len(uniqueAccountIds))
	for i, accountId := range uniqueAccountIds {
		invocations[i] = invocation(CommandQuotaGet, MailboxQueryCommand{AccountId: accountId}, mcid(accountId, "0"))
	}
	cmd, err := j.request(session, logger, invocations...)
	if err != nil {
		return nil, "", "", err
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string]QuotaGetResponse, Error) {
		result := map[string]QuotaGetResponse{}
		for _, accountId := range uniqueAccountIds {
			var response QuotaGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandQuotaGet, mcid(accountId, "0"), &response)
			if err != nil {
				return nil, err
			}
			result[accountId] = response
		}
		return result, nil
	})
}
