package groupware

import (
	"net/http"
	"slices"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
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
		return response(list, req.session.State, "")
	})
}

type AccountWithId struct {
	AccountId string `json:"accountId,omitempty"`
	jmap.Account
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
