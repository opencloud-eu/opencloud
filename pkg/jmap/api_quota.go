package jmap

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

func (j *Client) GetQuotas(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string]QuotaGetResponse, SessionState, State, Language, Error) {
	return getTemplateN(j, "GetQuotas", CommandQuotaGet,
		func(accountId string, ids []string) QuotaGetCommand {
			return QuotaGetCommand{AccountId: accountId}
		},
		identity1,
		identity1,
		func(resp QuotaGetResponse) State { return resp.State },
		accountIds, session, ctx, logger, acceptLanguage, []string{},
	)
}
