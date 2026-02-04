package groupware

import (
	"net/http"
	"slices"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

// When the request succeeds.
// swagger:response GetAccountResponse200
type SwaggerGetAccountResponse struct {
	// in: body
	Body struct {
		*jmap.Account
	}
}

// swagger:route GET /groupware/accounts/{account} account account
// Get attributes of a given account.
//
// responses:
//
//	200: GetAccountResponse200
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) GetAccount(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, account, err := req.GetAccountForMail()
		if err != nil {
			return errorResponse(single(accountId), err)
		}
		var body jmap.Account = account
		return etagResponse(single(accountId), body, req.session.State, AccountResponseObjectType, jmap.State(req.session.State), "")
	})
}

// When the request succeeds.
// swagger:response GetAccountsResponse200
type SwaggerGetAccountsResponse struct {
	// in: body
	Body map[string]jmap.Account
}

// swagger:route GET /groupware/accounts account accounts
// Get the list of all of the user's accounts.
//
// responses:
//
//	200: GetAccountsResponse200
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) GetAccounts(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		list := make([]AccountWithId, len(req.session.Accounts))
		i := 0
		for accountId, account := range req.session.Accounts {
			list[i] = AccountWithId{
				AccountId: accountId,
				Account:   account,
			}
			i++
		}
		// sort on accountId to have a stable order that remains the same with every query
		slices.SortFunc(list, func(a, b AccountWithId) int { return strings.Compare(a.AccountId, b.AccountId) })
		var RBODY []AccountWithId = list
		return etagResponse(structs.Map(list, func(a AccountWithId) string { return a.AccountId }), RBODY, req.session.State, AccountResponseObjectType, jmap.State(req.session.State), "")
	})
}

func (g *Groupware) GetAccountsWithTheirIdentities(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		allAccountIds := req.AllAccountIds()
		resp, sessionState, state, lang, err := g.jmap.GetIdentitiesForAllAccounts(allAccountIds, req.session, req.ctx, req.logger, req.language())
		if err != nil {
			return req.errorResponseFromJmap(allAccountIds, err)
		}
		list := make([]AccountWithIdAndIdentities, len(req.session.Accounts))
		i := 0
		for accountId, account := range req.session.Accounts {
			identities, ok := resp.Identities[accountId]
			if !ok {
				identities = []jmap.Identity{}
			}
			slices.SortFunc(identities, func(a, b jmap.Identity) int { return strings.Compare(a.Id, b.Id) })
			list[i] = AccountWithIdAndIdentities{
				AccountId:  accountId,
				Account:    account,
				Identities: identities,
			}
			i++
		}
		// sort on accountId to have a stable order that remains the same with every query
		slices.SortFunc(list, func(a, b AccountWithIdAndIdentities) int { return strings.Compare(a.AccountId, b.AccountId) })
		var RBODY []AccountWithIdAndIdentities = list
		return etagResponse(structs.Map(list, func(a AccountWithIdAndIdentities) string { return a.AccountId }), RBODY, sessionState, AccountResponseObjectType, state, lang)
	})
}

type AccountWithId struct {
	AccountId string `json:"accountId,omitempty"`
	jmap.Account
}

type AccountWithIdAndIdentities struct {
	AccountId string `json:"accountId,omitempty"`
	jmap.Account
	Identities []jmap.Identity `json:"identities,omitempty"`
}
