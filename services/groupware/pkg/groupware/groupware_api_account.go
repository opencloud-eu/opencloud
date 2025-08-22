package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

func (g *Groupware) GetAccount(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		account, err := req.GetAccount()
		if err != nil {
			return errorResponse(err)
		}
		return response(account, req.session.State)
	})
}

func (g *Groupware) GetAccounts(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		return response(req.session.Accounts, req.session.State)
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
// swagger:response IndexResponse
type SwaggerAccountBootstrapResponse struct {
	// in: body
	Body struct {
		*AccountBootstrapResponse
	}
}

func (g *Groupware) GetAccountBootstrap(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		mailAccountId := req.GetAccountId()
		accountIds := structs.Keys(req.session.Accounts)

		resp, jerr := g.jmap.GetIdentitiesAndMailboxes(mailAccountId, accountIds, req.session, req.ctx, req.logger)
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
		}, resp.SessionState)
	})
}
