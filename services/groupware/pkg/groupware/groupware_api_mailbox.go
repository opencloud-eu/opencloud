package groupware

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

// When the request succeeds.
// swagger:response MailboxResponse200
type SwaggerGetMailboxById200 struct {
	// in: body
	Body struct {
		*jmap.Mailbox
	}
}

// swagger:route GET /accounts/{account}/mailboxes/{id} mailboxes mailboxes_by_id
// Get a specific mailbox by its identifier.
//
// A Mailbox represents a named set of Emails.
// This is the primary mechanism for organising Emails within an account.
// It is analogous to a folder or a label in other systems.
//
// responses:
//
//	200: MailboxResponse200
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) GetMailbox(w http.ResponseWriter, r *http.Request) {
	mailboxId := chi.URLParam(r, UriParamMailboxId)
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return errorResponse(err)
		}

		res, sessionState, jerr := g.jmap.GetMailbox(accountId, req.session, req.ctx, req.logger, []string{mailboxId})
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		if len(res.Mailboxes) == 1 {
			return etagResponse(res.Mailboxes[0], sessionState, res.State)
		} else {
			return notFoundResponse(sessionState)
		}
	})
}

// swagger:parameters mailboxes
type SwaggerMailboxesParams struct {
	// The name of the mailbox, with substring matching.
	// in: query
	Name string `json:"name,omitempty"`
	// The role of the mailbox.
	// in: query
	Role string `json:"role,omitempty"`
	// Whether the mailbox is subscribed by the user or not.
	// When omitted, the subscribed and unsubscribed mailboxes are returned.
	// in: query
	Subscribed bool `json:"subscribed,omitempty"`
}

// When the request succeeds.
// swagger:response MailboxesResponse200
type SwaggerMailboxesResponse200 struct {
	// in: body
	Body []jmap.Mailbox
}

// swagger:route GET /accounts/{account}/mailboxes mailboxes mailboxes
// Get the list of all the mailboxes of an account.
//
// A Mailbox represents a named set of Emails.
// This is the primary mechanism for organising Emails within an account.
// It is analogous to a folder or a label in other systems.
//
// When none of the query parameters are specified, all the mailboxes are returned.
//
// responses:
//
//	200: MailboxesResponse200
//	400: ErrorResponse400
//	500: ErrorResponse500
func (g *Groupware) GetMailboxes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var filter jmap.MailboxFilterCondition

	hasCriteria := false
	name := q.Get(QueryParamMailboxSearchName)
	if name != "" {
		filter.Name = name
		hasCriteria = true
	}
	role := q.Get(QueryParamMailboxSearchRole)
	if role != "" {
		filter.Role = role
		hasCriteria = true
	}

	g.respond(w, r, func(req Request) Response {
		subscribed, set, err := req.parseBoolParam(QueryParamMailboxSearchSubscribed, false)
		if err != nil {
			return errorResponse(err)
		}
		if set {
			filter.IsSubscribed = &subscribed
			hasCriteria = true
		}

		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return errorResponse(err)
		}
		logger := log.From(req.logger.With().Str(logAccountId, accountId))

		if hasCriteria {
			mailboxes, sessionState, err := g.jmap.SearchMailboxes(accountId, req.session, req.ctx, logger, filter)
			if err != nil {
				return req.errorResponseFromJmap(err)
			}
			return etagResponse(mailboxes.Mailboxes, sessionState, mailboxes.State)
		} else {
			mailboxes, sessionState, err := g.jmap.GetAllMailboxes(accountId, req.session, req.ctx, logger)
			if err != nil {
				return req.errorResponseFromJmap(err)
			}
			return etagResponse(mailboxes.Mailboxes, sessionState, mailboxes.State)
		}
	})
}
