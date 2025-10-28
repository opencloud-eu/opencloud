package groupware

import (
	"net/http"
	"slices"
	"strings"

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

		mailboxes, sessionState, state, lang, jerr := g.jmap.GetMailbox(accountId, req.session, req.ctx, req.logger, req.language(), []string{mailboxId})
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		if len(mailboxes.Mailboxes) == 1 {
			return etagResponse(mailboxes.Mailboxes[0], sessionState, state, lang)
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
			mailboxesByAccountId, sessionState, state, lang, err := g.jmap.SearchMailboxes([]string{accountId}, req.session, req.ctx, logger, req.language(), filter)
			if err != nil {
				return req.errorResponseFromJmap(err)
			}

			if mailboxes, ok := mailboxesByAccountId[accountId]; ok {
				return etagResponse(sortMailboxSlice(mailboxes), sessionState, state, lang)
			} else {
				return notFoundResponse(sessionState)
			}
		} else {
			mailboxesByAccountId, sessionState, state, lang, err := g.jmap.GetAllMailboxes([]string{accountId}, req.session, req.ctx, logger, req.language())
			if err != nil {
				return req.errorResponseFromJmap(err)
			}
			if mailboxes, ok := mailboxesByAccountId[accountId]; ok {
				return etagResponse(sortMailboxSlice(mailboxes), sessionState, state, lang)
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
			mailboxesByAccountId, sessionState, state, lang, err := g.jmap.SearchMailboxes(accountIds, req.session, req.ctx, logger, req.language(), filter)
			if err != nil {
				return req.errorResponseFromJmap(err)
			}
			return etagResponse(sortMailboxesMap(mailboxesByAccountId), sessionState, state, lang)
		} else {
			mailboxesByAccountId, sessionState, state, lang, err := g.jmap.GetAllMailboxes(accountIds, req.session, req.ctx, logger, req.language())
			if err != nil {
				return req.errorResponseFromJmap(err)
			}
			return etagResponse(sortMailboxesMap(mailboxesByAccountId), sessionState, state, lang)
		}
	})
}

func (g *Groupware) GetMailboxByRoleForAllAccounts(w http.ResponseWriter, r *http.Request) {
	role := chi.URLParam(r, UriParamRole)
	g.respond(w, r, func(req Request) Response {
		accountIds := structs.Keys(req.session.Accounts)
		if len(accountIds) < 1 {
			return noContentResponse("")
		}
		logger := log.From(req.logger.With().Array(logAccountId, log.SafeStringArray(accountIds)).Str("role", role))

		filter := jmap.MailboxFilterCondition{
			Role: role,
		}

		mailboxesByAccountId, sessionState, state, lang, err := g.jmap.SearchMailboxes(accountIds, req.session, req.ctx, logger, req.language(), filter)
		if err != nil {
			return req.errorResponseFromJmap(err)
		}
		return etagResponse(sortMailboxesMap(mailboxesByAccountId), sessionState, state, lang)
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

		changes, sessionState, state, lang, jerr := g.jmap.GetMailboxChanges(accountId, req.session, req.ctx, logger, req.language(), mailboxId, sinceState, true, g.maxBodyValueBytes, maxChanges)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return etagResponse(changes, sessionState, state, lang)
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

		changesByAccountId, sessionState, state, lang, jerr := g.jmap.GetMailboxChangesForMultipleAccounts(allAccountIds, req.session, req.ctx, logger, req.language(), sinceStateMap, true, g.maxBodyValueBytes, maxChanges)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return etagResponse(changesByAccountId, sessionState, state, lang)
	})
}

func (g *Groupware) GetMailboxRoles(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With()
		allAccountIds := structs.Keys(req.session.Accounts) // TODO(pbleser-oc) do we need a limit for a maximum amount of accounts to query at once?
		l.Array(logAccountId, log.SafeStringArray(allAccountIds))
		logger := log.From(l)

		rolesByAccountId, sessionState, state, lang, jerr := g.jmap.GetMailboxRolesForMultipleAccounts(allAccountIds, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return etagResponse(rolesByAccountId, sessionState, state, lang)
	})
}

func (g *Groupware) UpdateMailbox(w http.ResponseWriter, r *http.Request) {
	mailboxId := chi.URLParam(r, UriParamMailboxId)

	g.respond(w, r, func(req Request) Response {
		l := req.logger.With().Str(UriParamMailboxId, log.SafeString(mailboxId))

		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return errorResponse(err)
		}
		l = l.Str(logAccountId, accountId)

		var body jmap.MailboxChange
		err = req.body(&body)
		if err != nil {
			return errorResponse(err)
		}
		logger := log.From(l)

		updated, sessionState, state, lang, jerr := g.jmap.UpdateMailbox(accountId, req.session, req.ctx, logger, req.language(), mailboxId, "", body)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return etagResponse(updated, sessionState, state, lang)
	})
}

func (g *Groupware) CreateMailbox(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With()
		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return errorResponse(err)
		}
		l = l.Str(logAccountId, accountId)

		var body jmap.MailboxChange
		err = req.body(&body)
		if err != nil {
			return errorResponse(err)
		}
		logger := log.From(l)

		created, sessionState, state, lang, jerr := g.jmap.CreateMailbox(accountId, req.session, req.ctx, logger, req.language(), "", body)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return etagResponse(created, sessionState, state, lang)
	})
}

func (g *Groupware) DeleteMailbox(w http.ResponseWriter, r *http.Request) {
	mailboxId := chi.URLParam(r, UriParamMailboxId)
	mailboxIds := strings.Split(mailboxId, ",")

	g.respond(w, r, func(req Request) Response {
		if len(mailboxIds) < 1 {
			return noContentResponse(req.session.State)
		}

		l := req.logger.With()
		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return errorResponse(err)
		}
		l = l.Str(logAccountId, accountId)
		l = l.Array(UriParamMailboxId, log.SafeStringArray(mailboxIds))
		logger := log.From(l)

		deleted, sessionState, state, lang, jerr := g.jmap.DeleteMailboxes(accountId, req.session, req.ctx, logger, req.language(), "", mailboxIds)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return etagResponse(deleted, sessionState, state, lang)
	})
}

var mailboxRoleSortOrderScore = map[string]int{
	jmap.JmapMailboxRoleInbox:  100,
	jmap.JmapMailboxRoleDrafts: 200,
	jmap.JmapMailboxRoleSent:   300,
	jmap.JmapMailboxRoleJunk:   400,
	jmap.JmapMailboxRoleTrash:  500,
}

func scoreMailbox(m jmap.Mailbox) int {
	if score, ok := mailboxRoleSortOrderScore[m.Role]; ok {
		return score
	}
	return 1000
}

func sortMailboxesMap[K comparable](mailboxesByAccountId map[K][]jmap.Mailbox) map[K][]jmap.Mailbox {
	sortedByAccountId := make(map[K][]jmap.Mailbox, len(mailboxesByAccountId))
	for accountId, unsorted := range mailboxesByAccountId {
		mailboxes := make([]jmap.Mailbox, len(unsorted))
		copy(mailboxes, unsorted)
		slices.SortFunc(mailboxes, compareMailboxes)
		sortedByAccountId[accountId] = mailboxes
	}
	return sortedByAccountId
}

func sortMailboxSlice(s []jmap.Mailbox) []jmap.Mailbox {
	r := make([]jmap.Mailbox, len(s))
	copy(r, s)
	slices.SortFunc(r, compareMailboxes)
	return r
}

func compareMailboxes(a, b jmap.Mailbox) int {
	// first, use the defined order:
	// Defines the sort order of Mailboxes when presented in the client’s UI, so it is consistent between devices.
	// Default value: 0
	// The number MUST be an integer in the range 0 <= sortOrder < 2^31.
	// A Mailbox with a lower order should be displayed before a Mailbox with a higher order
	// (that has the same parent) in any Mailbox listing in the client’s UI.
	sa := 0
	if a.SortOrder != nil {
		sa = *a.SortOrder
	}
	sb := 0
	if b.SortOrder != nil {
		sb = *b.SortOrder
	}
	r := sa - sb
	if r != 0 {
		return r
	}

	// the JMAP specification says this:
	// > Mailboxes with equal order SHOULD be sorted in alphabetical order by name.
	// > The sorting should take into account locale-specific character order convention.
	// but we feel like users would rather expect standard folders to come first,
	// in an order that is common across MUAs:
	// - inbox
	// - drafts
	// - sent
	// - junk
	// - trash
	// - *everything else*
	sa = scoreMailbox(a)
	sb = scoreMailbox(b)
	r = sa - sb
	if r != 0 {
		return r
	}

	// now we have "everything else", let's use alphabetical order here:
	return strings.Compare(a.Name, b.Name)
}
