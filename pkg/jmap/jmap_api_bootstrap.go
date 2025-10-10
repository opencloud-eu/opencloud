package jmap

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

type AccountBootstrapResult struct {
	Identities []Identity `json:"identities,omitempty"`
	Quotas     []Quota    `json:"quotas,omitempty"`
}

func (j *Client) GetBootstrap(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string]AccountBootstrapResult, SessionState, Language, Error) {
	uniqueAccountIds := structs.Uniq(accountIds)

	logger = j.logger("GetBootstrap", session, logger)

	calls := make([]Invocation, len(uniqueAccountIds)*2)
	for i, accountId := range uniqueAccountIds {
		calls[i*2+0] = invocation(CommandIdentityGet, IdentityGetCommand{AccountId: accountId}, mcid(accountId, "I"))
		calls[i*2+1] = invocation(CommandQuotaGet, QuotaGetCommand{AccountId: accountId}, mcid(accountId, "Q"))
	}

	cmd, err := j.request(session, logger, calls...)
	if err != nil {
		return nil, "", "", err
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string]AccountBootstrapResult, Error) {
		identityPerAccount := map[string][]Identity{}
		quotaPerAccount := map[string][]Quota{}
		for _, accountId := range uniqueAccountIds {
			var identityResponse IdentityGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandIdentityGet, mcid(accountId, "I"), &identityResponse)
			if err != nil {
				return nil, err
			} else {
				identityPerAccount[accountId] = identityResponse.List
			}

			var quotaResponse QuotaGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandQuotaGet, mcid(accountId, "Q"), &quotaResponse)
			if err != nil {
				return nil, err
			} else {
				quotaPerAccount[accountId] = quotaResponse.List
			}
		}

		result := map[string]AccountBootstrapResult{}
		for accountId, value := range identityPerAccount {
			r, ok := result[accountId]
			if !ok {
				r = AccountBootstrapResult{}
			}
			r.Identities = value
			result[accountId] = r
		}
		for accountId, value := range quotaPerAccount {
			r, ok := result[accountId]
			if !ok {
				r = AccountBootstrapResult{}
			}
			r.Quotas = value
			result[accountId] = r
		}
		return result, nil
	})
}
