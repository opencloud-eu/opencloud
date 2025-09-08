package groupware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

// When the request succeeds.
// swagger:response MailboxResponse200
type SwaggerGetMailboxById200 struct {
	// in: body
	Body struct {
		*jmap.Mailbox
	}
}

// swagger:route GET /groupware/accounts/{account}/mailboxes/{mailbox} mailbox mailboxes_by_id
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

		mailboxesByAccountId, sessionState, jerr := g.jmap.GetMailbox([]string{accountId}, req.session, req.ctx, req.logger, []string{mailboxId})
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		mailboxes, ok := mailboxesByAccountId[accountId]
		if ok && len(mailboxes.Mailboxes) == 1 {
			return etagResponse(mailboxes.Mailboxes[0], sessionState, mailboxes.State)
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

// swagger:route GET /groupware/accounts/{account}/mailboxes mailbox mailboxes
// Get the list of all the mailboxes of an account, potentially filtering on the
// name and/or role of the mailbox.
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
			mailboxesByAccountId, sessionState, err := g.jmap.SearchMailboxes([]string{accountId}, req.session, req.ctx, logger, filter)
			if err != nil {
				return req.errorResponseFromJmap(err)
			}

			mailboxes, ok := mailboxesByAccountId[accountId]
			if ok {
				return etagResponse(mailboxes.Mailboxes, sessionState, mailboxes.State)
			} else {
				return notFoundResponse(sessionState)
			}
		} else {
			mailboxesByAccountId, sessionState, err := g.jmap.GetAllMailboxes([]string{accountId}, req.session, req.ctx, logger)
			if err != nil {
				return req.errorResponseFromJmap(err)
			}
			mailboxes, ok := mailboxesByAccountId[accountId]
			if ok {
				return etagResponse(mailboxes.Mailboxes, sessionState, mailboxes.State)
			} else {
				return notFoundResponse(sessionState)
			}
		}
	})
}

// When the request succeeds.
// swagger:response MailboxesForAllAccountsResponse200
type SwaggerMailboxesForAllAccountsResponse200 struct {
	// in: body
	Body map[string][]jmap.Mailbox
}

// swagger:route GET /groupware/accounts/all/mailboxes mailboxesforallaccounts mailbox
// Get the list of all the mailboxes of all accounts of a user, potentially filtering on the
// role of the mailboxes.
//
// responses:
//
//	200: MailboxesForAllAccountsResponse200
//	400: ErrorResponse400
//	500: ErrorResponse500
func (g *Groupware) GetMailboxesForAllAccounts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var filter jmap.MailboxFilterCondition

	hasCriteria := false
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

		accountIds := structs.Keys(req.session.Accounts)
		if len(accountIds) < 1 {
			return noContentResponse("")
		}
		logger := log.From(req.logger.With().Array(logAccountId, log.SafeStringArray(accountIds)))

		if hasCriteria {
			mailboxesByAccountId, sessionState, err := g.jmap.SearchMailboxes(accountIds, req.session, req.ctx, logger, filter)
			if err != nil {
				return req.errorResponseFromJmap(err)
			}
			return response(mailboxesByAccountId, sessionState)
		} else {
			mailboxesByAccountId, sessionState, err := g.jmap.GetAllMailboxes(accountIds, req.session, req.ctx, logger)
			if err != nil {
				return req.errorResponseFromJmap(err)
			}
			return response(mailboxesByAccountId, sessionState)
		}
	})
}

// When the request succeeds.
// swagger:response MailboxChangesResponse200
type SwaggerMailboxChangesResponse200 struct {
	// in: body
	Body *jmap.MailboxChanges
}

// swagger:route GET /groupware/accounts/{account}/mailboxes/{mailbox}/changes mailbox mailboxchanges
// Get the changes that occured in a given mailbox since a certain state.
//
// responses:
//
//	200: MailboxChangesResponse200
//	400: ErrorResponse400
//	500: ErrorResponse500
func (g *Groupware) GetMailboxChanges(w http.ResponseWriter, r *http.Request) {
	mailboxId := chi.URLParam(r, UriParamMailboxId)
	sinceState := r.Header.Get(HeaderSince)

	g.respond(w, r, func(req Request) Response {
		l := req.logger.With().Str(HeaderSince, sinceState)

		maxChanges, ok, err := req.parseUIntParam(QueryParamMaxChanges, 0)
		if err != nil {
			return errorResponse(err)
		}
		if ok {
			l = l.Uint(QueryParamMaxChanges, maxChanges)
		}

		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return errorResponse(err)
		}
		l = l.Str(logAccountId, accountId)

		logger := log.From(l)

		changes, sessionState, jerr := g.jmap.GetMailboxChanges(accountId, req.session, req.ctx, logger, mailboxId, sinceState, true, g.maxBodyValueBytes, maxChanges)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return etagResponse(changes, sessionState, changes.State)
	})
}

// When the request succeeds.
// swagger:response MailboxChangesForAllAccountsResponse200
type SwaggerMailboxChangesForAllAccountsResponse200 struct {
	// in: body
	Body map[string]jmap.MailboxChanges
}

// swagger:route GET /groupware/accounts/all/mailboxes/changes mailbox mailboxchangesforallaccounts
// Get the changes that occured in all the mailboxes of all accounts.
//
// responses:
//
//	200: MailboxChangesForAllAccountsResponse200
//	400: ErrorResponse400
//	500: ErrorResponse500
func (g *Groupware) GetMailboxChangesForAllAccounts(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With()

		sinceStateMap, ok, err := req.parseMapParam(QueryParamSince)
		if err != nil {
			return errorResponse(err)
		}
		if ok {
			dict := zerolog.Dict()
			for k, v := range sinceStateMap {
				dict.Str(log.SafeString(k), log.SafeString(v))
			}
			l = l.Dict(QueryParamSince, dict)
		}

		maxChanges, ok, err := req.parseUIntParam(QueryParamMaxChanges, 0)
		if err != nil {
			return errorResponse(err)
		}
		if ok {
			l = l.Uint(QueryParamMaxChanges, maxChanges)
		}

		allAccountIds := structs.Keys(req.session.Accounts) // TODO(pbleser-oc) do we need a limit for a maximum amount of accounts to query at once?
		l.Array(logAccountId, log.SafeStringArray(allAccountIds))

		logger := log.From(l)

		changesByAccountId, sessionState, jerr := g.jmap.GetMailboxChangesForMultipleAccounts(allAccountIds, req.session, req.ctx, logger, sinceStateMap, true, g.maxBodyValueBytes, maxChanges)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return response(changesByAccountId, sessionState)
	})
}
