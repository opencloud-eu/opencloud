package groupware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/jscontact"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

// When the request succeeds.
// swagger:response GetAddressbooks200
type SwaggerGetAddressbooks200 struct {
	// in: body
	Body []jmap.AddressBook
}

// swagger:route GET /groupware/accounts/{account}/addressbooks addressbook addressbooks
// Get all addressbooks of an account.
//
// responses:
//
//	200: GetAddressbooks200
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) GetAddressbooks(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needContactWithAccount()
		if !ok {
			return resp
		}

		addressbooks, sessionState, lang, jerr := g.jmap.GetAddressbooks(accountId, req.session, req.ctx, req.logger, req.language(), nil)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return response(addressbooks, sessionState, lang)
	})
}

// When the request succeeds.
// swagger:response GetAddressbookById200
type SwaggerGetAddressbookById200 struct {
	// in: body
	Body struct {
		*jmap.AddressBook
	}
}

// swagger:route GET /groupware/accounts/{account}/addressbooks/{addressbookid} addressbook addressbook_by_id
// Get an addressbook of an account by its identifier.
//
// responses:
//
//	200: GetAddressbookById200
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) GetAddressbook(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needContactWithAccount()
		if !ok {
			return resp
		}

		l := req.logger.With()

		addressBookId := chi.URLParam(r, UriParamAddressBookId)
		l = l.Str(UriParamAddressBookId, log.SafeString(addressBookId))

		logger := log.From(l)
		addressbooks, sessionState, lang, jerr := g.jmap.GetAddressbooks(accountId, req.session, req.ctx, logger, req.language(), []string{addressBookId})
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		if len(addressbooks.NotFound) > 0 {
			return notFoundResponse(sessionState)
		} else {
			return response(addressbooks, sessionState, lang)
		}
	})
}

// When the request succeeds.
// swagger:response GetContactsInAddressbook200
type SwaggerGetContactsInAddressbook200 struct {
	// in: body
	Body []jscontact.ContactCard
}

// swagger:route GET /groupware/accounts/{account}/addressbooks/{addressbookid}/contacts contact contacts_in_addressbook
// Get all the contacts in an addressbook of an account by its identifier.
//
// responses:
//
//	200: GetContactsInAddressbook200
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) GetContactsInAddressbook(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needContactWithAccount()
		if !ok {
			return resp
		}

		l := req.logger.With()

		addressBookId := chi.URLParam(r, UriParamAddressBookId)
		l = l.Str(UriParamAddressBookId, log.SafeString(addressBookId))

		offset, ok, err := req.parseUIntParam(QueryParamOffset, 0)
		if err != nil {
			return errorResponse(err)
		}
		if ok {
			l = l.Uint(QueryParamOffset, offset)
		}

		limit, ok, err := req.parseUIntParam(QueryParamLimit, g.defaultContactLimit)
		if err != nil {
			return errorResponse(err)
		}
		if ok {
			l = l.Uint(QueryParamLimit, limit)
		}

		filter := jmap.ContactCardFilterCondition{
			InAddressBook: addressBookId,
		}
		sortBy := []jmap.ContactCardComparator{{Property: jscontact.ContactCardPropertyUpdated, IsAscending: false}}

		logger := log.From(l)
		contactsByAccountId, sessionState, lang, jerr := g.jmap.QueryContactCards([]string{accountId}, req.session, req.ctx, logger, req.language(), filter, sortBy, offset, limit)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		if contacts, ok := contactsByAccountId[accountId]; ok {
			return response(contacts, req.session.State, lang)
		} else {
			return notFoundResponse(sessionState)
		}
	})
}

func (g *Groupware) CreateContact(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needContactWithAccount()
		if !ok {
			return resp
		}

		l := req.logger.With()

		addressBookId := chi.URLParam(r, UriParamAddressBookId)
		l = l.Str(UriParamAddressBookId, log.SafeString(addressBookId))

		var create jscontact.ContactCard
		err := req.body(&create)
		if err != nil {
			return errorResponse(err)
		}

		logger := log.From(l)
		created, sessionState, lang, jerr := g.jmap.CreateContactCard(accountId, req.session, req.ctx, logger, req.language(), create)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}
		return etagResponse(created.ContactCard, sessionState, created.State, lang)
	})
}

func (g *Groupware) DeleteContact(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needContactWithAccount()
		if !ok {
			return resp
		}
		l := req.logger.With().Str(accountId, log.SafeString(accountId))

		contactId := chi.URLParam(r, UriParamContactId)
		l.Str(UriParamContactId, log.SafeString(contactId))

		logger := log.From(l)

		deleted, sessionState, _, jerr := g.jmap.DeleteContactCard(accountId, []string{contactId}, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		for _, e := range deleted.NotDestroyed {
			desc := e.Description
			if desc != "" {
				return errorResponseWithSessionState(apiError(
					req.errorId(),
					ErrorFailedToDeleteContact,
					withDetail(e.Description),
				), sessionState)
			} else {
				return errorResponseWithSessionState(apiError(
					req.errorId(),
					ErrorFailedToDeleteContact,
				), sessionState)
			}
		}
		return noContentResponseWithEtag(sessionState, deleted.State)
	})
}
