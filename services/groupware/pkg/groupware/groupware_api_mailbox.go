package groupware

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

// When the request succeeds.
// swagger:response MailboxResponse200
type SwaggerGetMailboxById200 struct {
	// in: body
	Body struct {
		*jmap.Mailbox
	}
}

// swagger:route GET /accounts/{account}/mailboxes/{id} mailboxes_by_id
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
func (g Groupware) GetMailbox(w http.ResponseWriter, r *http.Request) {
	mailboxId := chi.URLParam(r, UriParamMailboxId)
	if mailboxId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	g.respond(w, r, func(req Request) (any, string, *Error) {
		res, err := g.jmap.GetMailbox(req.GetAccountId(), req.session, req.ctx, req.logger, []string{mailboxId})
		if err != nil {
			return res, "", req.apiErrorFromJmap(err)
		}

		if len(res.List) == 1 {
			return res.List[0], res.State, req.apiErrorFromJmap(err)
		} else {
			return nil, res.State, req.apiErrorFromJmap(err)
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

// swagger:route GET /accounts/{account}/mailboxes mailboxes
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
func (g Groupware) GetMailboxes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var filter jmap.MailboxFilterCondition

	hasCriteria := false
	name := q.Get("name")
	if name != "" {
		filter.Name = name
		hasCriteria = true
	}
	role := q.Get("role")
	if role != "" {
		filter.Role = role
		hasCriteria = true
	}
	subscribed := q.Get("subscribed")
	if subscribed != "" {
		b, err := strconv.ParseBool(subscribed)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		filter.IsSubscribed = &b
		hasCriteria = true
	}

	g.respond(w, r, func(req Request) (any, string, *Error) {
		if hasCriteria {
			mailboxes, err := g.jmap.SearchMailboxes(req.GetAccountId(), req.session, req.ctx, req.logger, filter)
			if err != nil {
				return nil, "", req.apiErrorFromJmap(err)
			}
			return mailboxes.Mailboxes, mailboxes.State, nil
		} else {
			mailboxes, err := g.jmap.GetAllMailboxes(req.GetAccountId(), req.session, req.ctx, req.logger)
			if err != nil {
				return nil, "", req.apiErrorFromJmap(err)
			}
			return mailboxes.List, mailboxes.State, nil
		}
	})
}
