package groupware

import (
	"net/http"
	"slices"
	"strings"

	"github.com/rs/zerolog"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

// Get a specific mailbox by its identifier.
//
// A Mailbox represents a named set of Emails.
//
// This is the primary mechanism for organising Emails within an account.
// It is analogous to a folder or a label in other systems.
func (g *Groupware) GetMailbox(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return req.error(accountId, err)
		}

		mailboxId, err := req.PathParam(UriParamMailboxId)
		if err != nil {
			return req.error(accountId, err)
		}

		mailboxes, sessionState, state, lang, jerr := g.jmap.GetMailbox(accountId, req.session, req.ctx, req.logger, req.language(), []string{mailboxId})
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		if len(mailboxes.List) == 1 {
			return req.respond(accountId, mailboxes.List[0], sessionState, MailboxResponseObjectType, state)
		} else {
			return req.notFound(accountId, sessionState, MailboxResponseObjectType, state)
		}
	})
}

// Get the list of all the mailboxes of an account, potentially filtering on the
// name and/or role of the mailbox.
//
// A Mailbox represents a named set of Emails.
//
// This is the primary mechanism for organising Emails within an account.
// It is analogous to a folder or a label in other systems.
//
// When none of the query parameters are specified, all the mailboxes are returned.
func (g *Groupware) GetMailboxes(w http.ResponseWriter, r *http.Request) { //NOSONAR
	g.respond(w, r, func(req Request) Response {
		var filter jmap.MailboxFilterCondition

		hasCriteria := false
		name, ok := req.getStringParam(QueryParamMailboxSearchName, "") // the mailbox name to filter on
		if ok && name != "" {
			filter.Name = name
			hasCriteria = true
		}
		role, ok := req.getStringParam(QueryParamMailboxSearchRole, "") // the mailbox role to filter on
		if ok && role != "" {
			filter.Role = role
			hasCriteria = true
		}

		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return req.error(accountId, err)
		}

		subscribed, set, err := req.parseBoolParam(QueryParamMailboxSearchSubscribed, false)
		if err != nil {
			return req.error(accountId, err)
		}
		if set {
			filter.IsSubscribed = &subscribed
			hasCriteria = true
		}

		logger := log.From(req.logger.With().Str(logAccountId, accountId))

		if hasCriteria {
			mailboxesByAccountId, sessionState, state, lang, err := g.jmap.SearchMailboxes(single(accountId), req.session, req.ctx, logger, req.language(), filter)
			if err != nil {
				return req.jmapError(accountId, err, sessionState, lang)
			}

			if mailboxes, ok := mailboxesByAccountId[accountId]; ok {
				return req.respond(accountId, sortMailboxSlice(mailboxes), sessionState, MailboxResponseObjectType, state)
			} else {
				return req.notFound(accountId, sessionState, MailboxResponseObjectType, state)
			}
		} else {
			mailboxesByAccountId, sessionState, state, lang, err := g.jmap.GetAllMailboxes(single(accountId), req.session, req.ctx, logger, req.language())
			if err != nil {
				return req.jmapError(accountId, err, sessionState, lang)
			}
			if mailboxes, ok := mailboxesByAccountId[accountId]; ok {
				return req.respond(accountId, sortMailboxSlice(mailboxes), sessionState, MailboxResponseObjectType, state)
			} else {
				return req.notFound(accountId, sessionState, MailboxResponseObjectType, state)
			}
		}
	})
}

// Get the list of all the mailboxes of all accounts of a user, potentially filtering on the role of the mailboxes.
func (g *Groupware) GetMailboxesForAllAccounts(w http.ResponseWriter, r *http.Request) { //NOSONAR
	g.respond(w, r, func(req Request) Response {
		accountIds := req.AllAccountIds()
		if len(accountIds) < 1 {
			return req.noopN(nil) // when the user has no accounts
		}
		logger := log.From(req.logger.With().Array(logAccountId, log.SafeStringArray(accountIds)))

		var filter jmap.MailboxFilterCondition
		hasCriteria := false

		if role, set := req.getStringParam(QueryParamMailboxSearchRole, ""); set {
			filter.Role = role
			hasCriteria = true
		}

		if subscribed, set, err := req.parseBoolParam(QueryParamMailboxSearchSubscribed, false); err != nil {
			return req.errorN(accountIds, err)
		} else if set {
			filter.IsSubscribed = &subscribed
			hasCriteria = true
		}

		if hasCriteria {
			mailboxesByAccountId, sessionState, state, lang, err := g.jmap.SearchMailboxes(accountIds, req.session, req.ctx, logger, req.language(), filter)
			if err != nil {
				return req.jmapErrorN(accountIds, err, sessionState, lang)
			}
			return req.respondN(accountIds, sortMailboxesMap(mailboxesByAccountId), sessionState, MailboxResponseObjectType, state)
		} else {
			mailboxesByAccountId, sessionState, state, lang, err := g.jmap.GetAllMailboxes(accountIds, req.session, req.ctx, logger, req.language())
			if err != nil {
				return req.jmapErrorN(accountIds, err, sessionState, lang)
			}
			return req.respondN(accountIds, sortMailboxesMap(mailboxesByAccountId), sessionState, MailboxResponseObjectType, state)
		}
	})
}

// Retrieve Mailboxes by their role for all accounts.
func (g *Groupware) GetMailboxByRoleForAllAccounts(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		accountIds := req.AllAccountIds()
		if len(accountIds) < 1 {
			return req.noopN(accountIds) // when the user has no accounts
		}

		role, err := req.PathParamDoc(UriParamRole, "Role of the mailboxes to retrieve across all accounts")
		if err != nil {
			return req.errorN(accountIds, err)
		}

		logger := log.From(req.logger.With().Array(logAccountId, log.SafeStringArray(accountIds)).Str("role", role))

		filter := jmap.MailboxFilterCondition{
			Role: role,
		}

		mailboxesByAccountId, sessionState, state, lang, jerr := g.jmap.SearchMailboxes(accountIds, req.session, req.ctx, logger, req.language(), filter)
		if jerr != nil {
			return req.jmapErrorN(accountIds, jerr, sessionState, lang)
		}
		return req.respondN(accountIds, sortMailboxesMap(mailboxesByAccountId), sessionState, MailboxResponseObjectType, state)
	})
}

// Get the changes tp Mailboxes since a certain State.
// @api:tags mailbox,changes
func (g *Groupware) GetMailboxChanges(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With()

		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return req.error(accountId, err)
		}
		l = l.Str(logAccountId, accountId)

		maxChanges, ok, err := req.parseUIntParam(QueryParamMaxChanges, 0)
		if err != nil {
			return req.error(accountId, err)
		}
		if ok {
			l = l.Uint(QueryParamMaxChanges, maxChanges)
		}

		sinceState := jmap.State(req.OptHeaderParamDoc(HeaderParamSince, "Optionally specifies the state identifier from which on to list mailbox changes"))
		if sinceState != "" {
			l = l.Str(HeaderParamSince, log.SafeString(string(sinceState)))
		}

		logger := log.From(l)

		// As for Emails and Contacts, one would expect this request without any prior state to
		// be usable to list all the objects that currently exist, but that is not the case for
		// Mailbox, at least in combination with Stalwart, as those are initial objects that
		// "always existed", even with the initial State, and this the Mailbox/changes operation
		// returns nothing.
		// For this reason, when the "since" state is empty, we respond with an error.
		if sinceState == "" {
			return req.error(accountId, req.apiError(&ErrorInvalidUserRequest, withTitle(
				"Mailbox changes without state is unsupported",
				"Requesting Mailbox changes without an initial state is an unsupported operation",
			)))
		}

		changes, sessionState, state, lang, jerr := g.jmap.GetMailboxChanges(accountId, req.session, req.ctx, logger, req.language(), sinceState, maxChanges)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		return req.respond(accountId, changes, sessionState, MailboxResponseObjectType, state)
	})
}

// Get the changes that occured in all the mailboxes of all accounts.
func (g *Groupware) GetMailboxChangesForAllAccounts(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With()

		allAccountIds := req.AllAccountIds()
		l.Array(logAccountId, log.SafeStringArray(allAccountIds))

		sinceStateStrMap, ok, err := req.parseMapParam(QueryParamSince)
		if err != nil {
			return req.errorN(allAccountIds, err)
		}
		if ok {
			dict := zerolog.Dict()
			for k, v := range sinceStateStrMap {
				dict.Str(log.SafeString(k), log.SafeString(v))
			}
			l = l.Dict(QueryParamSince, dict)
		}

		maxChanges, ok, err := req.parseUIntParam(QueryParamMaxChanges, 0)
		if err != nil {
			return req.errorN(allAccountIds, err)
		}
		if ok {
			l = l.Uint(QueryParamMaxChanges, maxChanges)
		}

		logger := log.From(l)

		sinceStateMap := structs.MapValues(sinceStateStrMap, toState)
		changesByAccountId, sessionState, state, lang, jerr := g.jmap.GetMailboxChangesForMultipleAccounts(allAccountIds, req.session, req.ctx, logger, req.language(), sinceStateMap, maxChanges)
		if jerr != nil {
			return req.jmapErrorN(allAccountIds, jerr, sessionState, lang)
		}

		return req.respondN(allAccountIds, changesByAccountId, sessionState, MailboxResponseObjectType, state)
	})
}

// Retrieve the roles of all the Mailboxes of all Accounts.
// @api:example mailboxrolesbyaccount
func (g *Groupware) GetMailboxRoles(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With()
		allAccountIds := req.AllAccountIds()
		l.Array(logAccountId, log.SafeStringArray(allAccountIds))
		logger := log.From(l)

		rolesByAccountId, sessionState, state, lang, jerr := g.jmap.GetMailboxRolesForMultipleAccounts(allAccountIds, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.jmapErrorN(allAccountIds, jerr, sessionState, lang)
		}

		return req.respondN(allAccountIds, rolesByAccountId, sessionState, MailboxResponseObjectType, state)
	})
}

func (g *Groupware) UpdateMailbox(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With()

		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return req.error(accountId, err)
		}
		l = l.Str(logAccountId, accountId)

		mailboxId, err := req.PathParamDoc(UriParamMailboxId, "the identifier of the mailbox to update")
		if err != nil {
			return req.error(accountId, err)
		}
		l = l.Str(UriParamMailboxId, log.SafeString(mailboxId))

		var body jmap.MailboxChange
		err = req.body(&body)
		if err != nil {
			return req.error(accountId, err)
		}
		logger := log.From(l)

		updated, sessionState, state, lang, jerr := g.jmap.UpdateMailbox(accountId, req.session, req.ctx, logger, req.language(), mailboxId, "", body)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		return req.respond(accountId, updated, sessionState, MailboxResponseObjectType, state)
	})
}

func (g *Groupware) CreateMailbox(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With()
		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return req.error(accountId, err)
		}
		l = l.Str(logAccountId, accountId)

		var body jmap.MailboxChange
		err = req.body(&body)
		if err != nil {
			return req.error(accountId, err)
		}
		logger := log.From(l)

		created, sessionState, state, lang, jerr := g.jmap.CreateMailbox(accountId, req.session, req.ctx, logger, req.language(), "", body)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		return req.respond(accountId, created, sessionState, MailboxResponseObjectType, state)
	})
}

// Delete Mailboxes by their unique identifiers.
//
// Returns the identifiers of the Mailboxes that have successfully been deleted.
//
// @api:example deletedmailboxes
func (g *Groupware) DeleteMailbox(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With()
		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return req.error(accountId, err)
		}
		l = l.Str(logAccountId, accountId)

		mailboxIds, err := req.PathListParamDoc(UriParamMailboxId, "the identifier of the mailbox to delete")
		if err != nil {
			return req.error(accountId, err)
		}
		l = l.Array(UriParamMailboxId, log.SafeStringArray(mailboxIds))

		if len(mailboxIds) < 1 {
			return req.noop(accountId) // no mailbox identifiers were mentioned in the request
		}

		logger := log.From(l)

		deleted, sessionState, state, lang, jerr := g.jmap.DeleteMailboxes(accountId, req.session, req.ctx, logger, req.language(), "", mailboxIds)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		return req.respond(accountId, deleted, sessionState, MailboxResponseObjectType, state)
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

func sortMailboxesMap(mailboxesByAccountId map[string][]jmap.Mailbox) map[string][]jmap.Mailbox {
	sortedByAccountId := make(map[string][]jmap.Mailbox, len(mailboxesByAccountId))
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
