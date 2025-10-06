package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
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
		account, err := req.GetAccountForMail()
		if err != nil {
			return errorResponse(err)
		}
		return response(account, req.session.State, "")
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
		return response(req.session.Accounts, req.session.State, "")
	})
}

type AccountBootstrapResponse struct {
	// The API version.
	Version string `json:"version"`

	// A list of capabilities of this API version.
	Capabilities []string `json:"capabilities"`

	// API limits.
	Limits IndexLimits `json:"limits"`

	// Accounts that are available to the user.
	//
	// The key of the mapis the identifier.
	Accounts map[string]IndexAccount `json:"accounts"`

	// Primary accounts for usage types.
	PrimaryAccounts IndexPrimaryAccounts `json:"primaryAccounts"`

	// Mailboxes.
	Mailboxes map[string][]jmap.Mailbox `json:"mailboxes"`
}

// When the request suceeds.
// swagger:response GetAccountBootstrapResponse200
type SwaggerAccountBootstrapResponse struct {
	// in: body
	Body struct {
		*AccountBootstrapResponse
	}
}

// swagger:route GET /groupware/accounts/{account}/bootstrap account accountbootstrap
// Get account bootstrapping.
//
// responses:
//
//	200: GetAccountBootstrapResponse200
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) GetAccountBootstrap(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		mailAccountId, err := req.GetAccountIdForMail()
		if err != nil {
			return errorResponse(err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, mailAccountId))
		accountIds := structs.Keys(req.session.Accounts)

		resp, sessionState, lang, jerr := g.jmap.GetIdentitiesAndMailboxes(mailAccountId, accountIds, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return response(AccountBootstrapResponse{
			Version:         Version,
			Capabilities:    Capabilities,
			Limits:          buildIndexLimits(req.session),
			Accounts:        buildIndexAccount(req.session, resp.Identities),
			PrimaryAccounts: buildIndexPrimaryAccounts(req.session),
			Mailboxes: map[string][]jmap.Mailbox{
				mailAccountId: resp.Mailboxes,
			},
		}, sessionState, lang)
	})
}
