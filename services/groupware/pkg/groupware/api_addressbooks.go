package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

// Get all addressbooks of an account.
func (g *Groupware) GetAddressbooks(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needContactWithAccount()
		if !ok {
			return resp
		}

		addressbooks, sessionState, state, lang, jerr := g.jmap.GetAddressbooks(accountId, req.session, req.ctx, req.logger, req.language(), nil)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		var body jmap.AddressBookGetResponse = addressbooks
		return req.respond(accountId, body, sessionState, AddressBookResponseObjectType, state)
	})
}

// Get an addressbook of an account by its identifier.
func (g *Groupware) GetAddressbook(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needContactWithAccount()
		if !ok {
			return resp
		}

		l := req.logger.With()

		addressBookId, err := req.PathParam(UriParamAddressBookId)
		if err != nil {
			return req.error(accountId, err)
		}
		l = l.Str(UriParamAddressBookId, log.SafeString(addressBookId))

		logger := log.From(l)
		addressbooks, sessionState, state, lang, jerr := g.jmap.GetAddressbooks(accountId, req.session, req.ctx, logger, req.language(), []string{addressBookId})
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		switch len(addressbooks.List) {
		case 0:
			return req.notFound(accountId, sessionState, ContactResponseObjectType, state)
		case 1:
			return req.respond(accountId, addressbooks.List[0], sessionState, ContactResponseObjectType, state)
		default:
			logger.Error().Msgf("found %d addressbooks matching '%s' instead of 1", len(addressbooks.List), addressBookId)
			return req.errorS(accountId, req.apiError(&ErrorMultipleIdMatches), sessionState)
		}
	})
}

// Get the changes to Address Books since a certain State.
// @api:tags addressbook,changes
func (g *Groupware) GetAddressBookChanges(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needContactWithAccount()
		if !ok {
			return resp
		}

		l := req.logger.With()

		maxChanges, ok, err := req.parseUIntParam(QueryParamMaxChanges, 0)
		if err != nil {
			return req.error(accountId, err)
		}
		if ok {
			l = l.Uint(QueryParamMaxChanges, maxChanges)
		}

		sinceState := jmap.State(req.OptHeaderParamDoc(HeaderParamSince, "Optionally specifies the state identifier from which on to list addressbook changes"))
		if sinceState != "" {
			l = l.Str(HeaderParamSince, log.SafeString(string(sinceState)))
		}

		logger := log.From(l)

		changes, sessionState, state, lang, jerr := g.jmap.GetAddressbookChanges(accountId, req.session, req.ctx, logger, req.language(), sinceState, maxChanges)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		return req.respond(accountId, changes, sessionState, AddressBookResponseObjectType, state)
	})
}

func (g *Groupware) CreateAddressBook(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needContactWithAccount()
		if !ok {
			return resp
		}

		l := req.logger.With()

		var create jmap.AddressBookChange
		err := req.bodydoc(&create, "The address book to create")
		if err != nil {
			return req.error(accountId, err)
		}

		logger := log.From(l)
		created, sessionState, state, lang, jerr := g.jmap.CreateAddressBook(accountId, req.session, req.ctx, logger, req.language(), create)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		return req.respond(accountId, created, sessionState, ContactResponseObjectType, state)
	})
}

func (g *Groupware) DeleteAddressBook(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needContactWithAccount()
		if !ok {
			return resp
		}
		l := req.logger.With().Str(accountId, log.SafeString(accountId))

		addressBookId, err := req.PathParam(UriParamAddressBookId)
		if err != nil {
			return req.error(accountId, err)
		}
		l.Str(UriParamAddressBookId, log.SafeString(addressBookId))

		logger := log.From(l)

		deleted, sessionState, state, lang, jerr := g.jmap.DeleteAddressBook(accountId, []string{addressBookId}, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		for _, e := range deleted {
			desc := e.Description
			if desc != "" {
				return req.error(accountId, apiError(
					req.errorId(),
					ErrorFailedToDeleteAddressBook,
					withDetail(e.Description),
				))
			} else {
				return req.error(accountId, apiError(
					req.errorId(),
					ErrorFailedToDeleteAddressBook,
				))
			}
		}
		return req.noContent(accountId, sessionState, AddressBookResponseObjectType, state)
	})
}
