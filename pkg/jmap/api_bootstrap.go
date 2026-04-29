package jmap

import (
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

type AccountBootstrapResult struct {
	Identities []Identity `json:"identities,omitempty"`
	Quotas     []Quota    `json:"quotas,omitempty"`
}

var NS_MAIL_QUOTA = ns(JmapMail, JmapQuota)

func (j *Client) GetBootstrap(accountIds []string, ctx Context) (Result[map[string]AccountBootstrapResult], Error) { //NOSONAR
	uniqueAccountIds := structs.Uniq(accountIds)

	logger := j.logger("GetBootstrap", ctx)
	ctx = ctx.WithLogger(logger)

	calls := make([]Invocation, len(uniqueAccountIds)*2)
	for i, accountId := range uniqueAccountIds {
		calls[i*2+0] = invocation(IdentityGetCommand{AccountId: accountId}, mcid(accountId, "I"))
		calls[i*2+1] = invocation(QuotaGetCommand{AccountId: accountId}, mcid(accountId, "Q"))
	}

	cmd, err := j.request(ctx, NS_MAIL_QUOTA, calls...)
	if err != nil {
		return ZeroResult[map[string]AccountBootstrapResult](), err
	}
	return command(j, ctx, cmd, func(body *Response) (map[string]AccountBootstrapResult, State, Error) {
		identityPerAccount := map[string][]Identity{}
		quotaPerAccount := map[string][]Quota{}
		identityStatesPerAccount := map[string]State{}
		quotaStatesPerAccount := map[string]State{}
		for _, accountId := range uniqueAccountIds {
			var identityResponse IdentityGetResponse
			err = retrieveResponseMatchParameters(ctx, body, CommandIdentityGet, mcid(accountId, "I"), &identityResponse)
			if err != nil {
				return nil, "", err
			} else {
				identityPerAccount[accountId] = identityResponse.List
				identityStatesPerAccount[accountId] = identityResponse.State
			}

			var quotaResponse QuotaGetResponse
			err = retrieveResponseMatchParameters(ctx, body, CommandQuotaGet, mcid(accountId, "Q"), &quotaResponse)
			if err != nil {
				return nil, "", err
			} else {
				quotaPerAccount[accountId] = quotaResponse.List
				quotaStatesPerAccount[accountId] = quotaResponse.State
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

		return result, squashStateMaps(identityStatesPerAccount, quotaStatesPerAccount), nil
	})
}
