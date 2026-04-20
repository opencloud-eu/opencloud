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
	getFromMap(Quota, w, r, g, func(accountIds, _ []string, ctx jmap.Context) (map[string]jmap.QuotaGetResponse, jmap.SessionState, jmap.State, jmap.Language, jmap.Error) {
		return g.jmap.GetQuotas(accountIds, ctx)
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
		ctx := req.ctx.WithLogger(logger)

		res, sessionState, state, lang, jerr := g.jmap.GetQuotas(accountIds, ctx)
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
		return req.respondN(accountIds, result, sessionState, QuotaResponseObjectType, state, lang)
	})
}

// Get changes to Quotas since a given State
//
// Currently unsupported in Stalwart.
// @api:tags contact,changes
// @api:ignore
func (g *Groupware) GetQuotaChanges(w http.ResponseWriter, r *http.Request) {
	changes(Quota, w, r, g, g.jmap.GetQuotaChanges)
}
