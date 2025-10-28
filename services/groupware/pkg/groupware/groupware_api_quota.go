package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

// When the request succeeds.
// swagger:response GetQuotaResponse200
type SwaggerGetQuotaResponse200 struct {
	// in: body
	Body []jmap.Quota
}

// swagger:route GET /groupware/accounts/{account}/quota quota get_quota
// Get quota limits.
//
// responses:
//
//	200: GetQuotaResponse200
//	400: ErrorResponse400
//	500: ErrorResponse500
func (g *Groupware) GetQuota(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForQuota()
		if err != nil {
			return errorResponse(err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))

		res, sessionState, state, lang, jerr := g.jmap.GetQuotas([]string{accountId}, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}
		for _, v := range res {
			return etagResponse(v.List, sessionState, state, lang)
		}
		return notFoundResponse(sessionState)
	})
}

type AccountQuota struct {
	Quotas []jmap.Quota `json:"quotas,omitempty"`
	State  jmap.State   `json:"state"`
}

// When the request succeeds.
// swagger:response GetQuotaForAllAccountsResponse200
type SwaggerGetQuotaForAllAccountsResponse200 struct {
	// in: body
	Body map[string]AccountQuota
}

// swagger:route GET /groupware/accounts/all/quota quota get_quota_for_all_accounts
// Get quota limits for all accounts.
//
// responses:
//
//	200: GetQuotaForAllAccountsResponse200
//	400: ErrorResponse400
//	500: ErrorResponse500
func (g *Groupware) GetQuotaForAllAccounts(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountIds := structs.Keys(req.session.Accounts)
		if len(accountIds) < 1 {
			return noContentResponse("")
		}
		logger := log.From(req.logger.With().Array(logAccountId, log.SafeStringArray(accountIds)))

		res, sessionState, state, lang, jerr := g.jmap.GetQuotas(accountIds, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		result := make(map[string]AccountQuota, len(res))
		for accountId, accountQuotas := range res {
			result[accountId] = AccountQuota{
				State:  accountQuotas.State,
				Quotas: accountQuotas.List,
			}
		}
		return etagResponse(result, sessionState, state, lang)
	})
}
