package groupware

import (
	"net/http"

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

		addressbooks, sessionState, state, lang, jerr := g.jmap.GetAddressbooks(accountId, req.session, req.ctx, req.logger, req.language(), nil)
		if jerr != nil {
			return req.errorResponseFromJmap(single(accountId), jerr)
		}

		return etagResponse(single(accountId), addressbooks, sessionState, AddressBookResponseObjectType, state, lang)
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

		addressBookId, err := req.PathParam(UriParamAddressBookId)
		if err != nil {
			return errorResponse(single(accountId), err)
		}
		l = l.Str(UriParamAddressBookId, log.SafeString(addressBookId))

		logger := log.From(l)
		addressbooks, sessionState, state, lang, jerr := g.jmap.GetAddressbooks(accountId, req.session, req.ctx, logger, req.language(), []string{addressBookId})
		if jerr != nil {
			return req.errorResponseFromJmap(single(accountId), jerr)
		}

		if len(addressbooks.NotFound) > 0 {
			return notFoundResponse(single(accountId), sessionState)
		} else {
			return etagResponse(single(accountId), addressbooks, sessionState, AddressBookResponseObjectType, state, lang)
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

		addressBookId, err := req.PathParam(UriParamAddressBookId)
		if err != nil {
			return errorResponse(single(accountId), err)
		}
		l = l.Str(UriParamAddressBookId, log.SafeString(addressBookId))

		offset, ok, err := req.parseUIntParam(QueryParamOffset, 0)
		if err != nil {
			return errorResponse(single(accountId), err)
		}
		if ok {
			l = l.Uint(QueryParamOffset, offset)
		}

		limit, ok, err := req.parseUIntParam(QueryParamLimit, g.defaults.contactLimit)
		if err != nil {
			return errorResponse(single(accountId), err)
		}
		if ok {
			l = l.Uint(QueryParamLimit, limit)
		}

		filter := jmap.ContactCardFilterCondition{
			InAddressBook: addressBookId,
		}
		sortBy := []jmap.ContactCardComparator{{Property: jscontact.ContactCardPropertyUpdated, IsAscending: false}}

		logger := log.From(l)
		contactsByAccountId, sessionState, state, lang, jerr := g.jmap.QueryContactCards(single(accountId), req.session, req.ctx, logger, req.language(), filter, sortBy, offset, limit)
		if jerr != nil {
			return req.errorResponseFromJmap(single(accountId), jerr)
		}

		if contacts, ok := contactsByAccountId[accountId]; ok {
			return etagResponse(single(accountId), contacts, sessionState, ContactResponseObjectType, state, lang)
		} else {
			return etagNotFoundResponse(single(accountId), sessionState, ContactResponseObjectType, state, lang)
		}
	})
}

func (g *Groupware) GetContactById(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needContactWithAccount()
		if !ok {
			return resp
		}

		l := req.logger.With()

		contactId, err := req.PathParam(UriParamContactId)
		if err != nil {
			return errorResponse(single(accountId), err)
		}
		l = l.Str(UriParamContactId, log.SafeString(contactId))

		logger := log.From(l)
		contactsById, sessionState, state, lang, jerr := g.jmap.GetContactCardsById(accountId, req.session, req.ctx, logger, req.language(), []string{contactId})
		if jerr != nil {
			return req.errorResponseFromJmap(single(accountId), jerr)
		}

		if contact, ok := contactsById[contactId]; ok {
			return etagResponse(single(accountId), contact, sessionState, ContactResponseObjectType, state, lang)
		} else {
			return etagNotFoundResponse(single(accountId), sessionState, ContactResponseObjectType, state, lang)
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

		addressBookId, err := req.PathParam(UriParamAddressBookId)
		if err != nil {
			return errorResponse(single(accountId), err)
		}
		l = l.Str(UriParamAddressBookId, log.SafeString(addressBookId))

		var create jscontact.ContactCard
		err = req.bodydoc(&create, "The contact to create, which may not have its id attribute set")
		if err != nil {
			return errorResponse(single(accountId), err)
		}

		logger := log.From(l)
		created, sessionState, state, lang, jerr := g.jmap.CreateContactCard(accountId, req.session, req.ctx, logger, req.language(), create)
		if jerr != nil {
			return req.errorResponseFromJmap(single(accountId), jerr)
		}
		return etagResponse(single(accountId), created, sessionState, ContactResponseObjectType, state, lang)
	})
}

func (g *Groupware) DeleteContact(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needContactWithAccount()
		if !ok {
			return resp
		}
		l := req.logger.With().Str(accountId, log.SafeString(accountId))

		contactId, err := req.PathParam(UriParamContactId)
		if err != nil {
			return errorResponse(single(accountId), err)
		}
		l.Str(UriParamContactId, log.SafeString(contactId))

		logger := log.From(l)

		deleted, sessionState, state, _, jerr := g.jmap.DeleteContactCard(accountId, []string{contactId}, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.errorResponseFromJmap(single(accountId), jerr)
		}

		for _, e := range deleted {
			desc := e.Description
			if desc != "" {
				return errorResponseWithSessionState(single(accountId), apiError(
					req.errorId(),
					ErrorFailedToDeleteContact,
					withDetail(e.Description),
				), sessionState)
			} else {
				return errorResponseWithSessionState(single(accountId), apiError(
					req.errorId(),
					ErrorFailedToDeleteContact,
				), sessionState)
			}
		}
		return noContentResponseWithEtag(single(accountId), sessionState, ContactResponseObjectType, state)
	})
}
