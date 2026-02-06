package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

// Get quota limits.
//
// Retrieves the list of Quota configurations for a given account.
//
// Note that there may be multiple Quota objects for different resource types.
func (g *Groupware) GetQuota(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForQuota()
		if err != nil {
			return errorResponse(single(accountId), err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))

		res, sessionState, state, lang, jerr := g.jmap.GetQuotas(single(accountId), req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.errorResponseFromJmap(single(accountId), jerr)
		}
		for _, v := range res {
			body := v.List
			return etagResponse(single(accountId), body, sessionState, QuotaResponseObjectType, state, lang)
		}
		return notFoundResponse(single(accountId), sessionState)
	})
}

type AccountQuota struct {
	Quotas []jmap.Quota `json:"quotas,omitempty"`
	State  jmap.State   `json:"state"`
}

// Get quota limits for all accounts.
//
// Retrieves the Quota configuration for all the accounts the user currently has access to,
// as a dictionary that has the account identifier as its key and an array of Quotas as its value.
func (g *Groupware) GetQuotaForAllAccounts(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountIds := req.AllAccountIds()
		if len(accountIds) < 1 {
			return noContentResponse(accountIds, "") // user has no accounts
		}
		logger := log.From(req.logger.With().Array(logAccountId, log.SafeStringArray(accountIds)))

		res, sessionState, state, lang, jerr := g.jmap.GetQuotas(accountIds, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.errorResponseFromJmap(accountIds, jerr)
		}

		result := make(map[string]AccountQuota, len(res))
		for accountId, accountQuotas := range res {
			result[accountId] = AccountQuota{
				State:  accountQuotas.State,
				Quotas: accountQuotas.List,
			}
		}
		return etagResponse(accountIds, result, sessionState, QuotaResponseObjectType, state, lang)
	})
}
