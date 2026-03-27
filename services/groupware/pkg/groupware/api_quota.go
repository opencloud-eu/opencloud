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
			return req.error(accountId, err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))

		res, sessionState, state, lang, jerr := g.jmap.GetQuotas(single(accountId), req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		for _, v := range res {
			body := v.List
			return req.respond(accountId, body, sessionState, QuotaResponseObjectType, state)
		}
		return req.notFound(accountId, sessionState, QuotaResponseObjectType, state)
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
			return req.noopN(accountIds) // user has no accounts
		}
		logger := log.From(req.logger.With().Array(logAccountId, log.SafeStringArray(accountIds)))

		res, sessionState, state, lang, jerr := g.jmap.GetQuotas(accountIds, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.jmapErrorN(accountIds, jerr, sessionState, lang)
		}

		result := make(map[string]AccountQuota, len(res))
		for accountId, accountQuotas := range res {
			result[accountId] = AccountQuota{
				State:  accountQuotas.State,
				Quotas: accountQuotas.List,
			}
		}
		return req.respondN(accountIds, result, sessionState, QuotaResponseObjectType, state)
	})
}

// Get changes to Contacts since a given State
// @api:tags contact,changes
func (g *Groupware) GetQuotaChanges(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForQuota()
		if err != nil {
			return req.error(accountId, err)
		}

		l := req.logger.With().Str(logAccountId, accountId)

		var maxChanges uint = 0
		if v, ok, err := req.parseUIntParam(QueryParamMaxChanges, 0); err != nil {
			return req.error(accountId, err)
		} else if ok {
			maxChanges = v
			l = l.Uint(QueryParamMaxChanges, v)
		}

		sinceState := jmap.State(req.OptHeaderParamDoc(HeaderParamSince, "Specifies the state identifier from which on to list quota changes"))
		l = l.Str(HeaderParamSince, log.SafeString(string(sinceState)))

		logger := log.From(l)
		changes, sessionState, state, lang, jerr := g.jmap.GetQuotaChanges(accountId, req.session, req.ctx, logger, req.language(), sinceState, maxChanges)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		var body jmap.QuotaChanges = changes

		return req.respond(accountId, body, sessionState, QuotaResponseObjectType, state)
	})
}
