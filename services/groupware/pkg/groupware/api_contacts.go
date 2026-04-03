package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/jscontact"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

var (
	// Ideally, we would be using this for sorting, but unfortunately, it is currently not supported by
	// Stalwart: https://github.com/stalwartlabs/stalwart/discussions/2918
	/*
		DefaultContactSort = []jmap.ContactCardComparator{
			{Property: string(jscontact.ContactCardPropertyName) + "/surname", IsAscending: true},
			{Property: string(jscontact.ContactCardPropertyName) + "/given", IsAscending: true},
		}

		SupportedContactSortingProperties = []string{
			jscontact.ContactCardPropertyUpdated,
			jscontact.ContactCardPropertyCreated,
			"surname",
			"given",
		}

	*/
	// So we have to settle for this, as only 'updated' and 'created' are supported for now:
	DefaultContactSort = []jmap.ContactCardComparator{
		{Property: jscontact.ContactCardPropertyUpdated, IsAscending: true},
	}

	SupportedContactSortingProperties = []string{
		jscontact.ContactCardPropertyUpdated,
		jscontact.ContactCardPropertyCreated,
	}

	ContactSortingPropertyMapping = map[string]string{
		"surname": string(jscontact.ContactCardPropertyName) + "/surname",
		"given":   string(jscontact.ContactCardPropertyName) + "/given",
	}
)

// Get all the contacts in an addressbook of an account by its identifier.
func (g *Groupware) GetContactsInAddressbook(w http.ResponseWriter, r *http.Request) { //NOSONAR
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needContactWithAccount()
		if !ok {
			return resp
		}
		accountIds := single(accountId)

		l := req.logger.With()

		addressBookId, err := req.PathParam(UriParamAddressBookId)
		if err != nil {
			return req.errorN(accountIds, err)
		}
		l = l.Str(UriParamAddressBookId, log.SafeString(addressBookId))

		offset, ok, err := req.parseUIntParam(QueryParamOffset, 0)
		if err != nil {
			return req.errorN(accountIds, err)
		}
		if ok {
			l = l.Uint(QueryParamOffset, offset)
		}

		limit, ok, err := req.parseUIntParam(QueryParamLimit, g.defaults.contactLimit)
		if err != nil {
			return req.errorN(accountIds, err)
		}
		if ok {
			l = l.Uint(QueryParamLimit, limit)
		}

		filter := jmap.ContactCardFilterCondition{
			InAddressBook: addressBookId,
		}
		var sortBy []jmap.ContactCardComparator
		if sort, ok, resp := mapSort(accountIds, &req, DefaultContactSort, SupportedContactSortingProperties, mapContactCardSort); !ok {
			return resp
		} else {
			sortBy = sort
		}

		logger := log.From(l)
		contactsByAccountId, sessionState, state, lang, jerr := g.jmap.QueryContactCards(accountIds, req.session, req.ctx, logger, req.language(), filter, sortBy, offset, limit)
		if jerr != nil {
			return req.jmapErrorN(accountIds, jerr, sessionState, lang)
		}

		if contacts, ok := contactsByAccountId[accountId]; ok {
			return req.respondN(accountIds, contacts, sessionState, ContactResponseObjectType, state)
		} else {
			return req.notFoundN(accountIds, sessionState, ContactResponseObjectType, state)
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
			return req.error(accountId, err)
		}
		l = l.Str(UriParamContactId, log.SafeString(contactId))

		logger := log.From(l)
		contactsById, sessionState, state, lang, jerr := g.jmap.GetContactCardsById(accountId, req.session, req.ctx, logger, req.language(), []string{contactId})
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		if contact, ok := contactsById[contactId]; ok {
			return req.respond(accountId, contact, sessionState, ContactResponseObjectType, state)
		} else {
			return req.notFound(accountId, sessionState, ContactResponseObjectType, state)
		}
	})
}

func (g *Groupware) GetAllContacts(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needContactWithAccount()
		if !ok {
			return resp
		}

		l := req.logger.With()

		logger := log.From(l)
		contacts, sessionState, state, lang, jerr := g.jmap.GetContactCards(accountId, req.session, req.ctx, logger, req.language(), []string{})
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		var body []jscontact.ContactCard = contacts

		return req.respond(accountId, body, sessionState, ContactResponseObjectType, state)
	})
}

// Get changes to Contacts since a given State
// @api:tags contact,changes
func (g *Groupware) GetContactsChanges(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, accountId, resp := req.needContactWithAccount()
		if !ok {
			return resp
		}

		l := req.logger.With()

		var maxChanges uint = 0
		if v, ok, err := req.parseUIntParam(QueryParamMaxChanges, 0); err != nil {
			return req.error(accountId, err)
		} else if ok {
			maxChanges = v
			l = l.Uint(QueryParamMaxChanges, v)
		}

		sinceState := jmap.State(req.OptHeaderParamDoc(HeaderParamSince, "Specifies the state identifier from which on to list contact changes"))
		l = l.Str(HeaderParamSince, log.SafeString(string(sinceState)))

		logger := log.From(l)
		changes, sessionState, state, lang, jerr := g.jmap.GetContactCardChanges(accountId, req.session, req.ctx, logger, req.language(), sinceState, maxChanges)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		var body jmap.ContactCardChanges = changes

		return req.respond(accountId, body, sessionState, ContactResponseObjectType, state)
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
			return req.error(accountId, err)
		}
		l = l.Str(UriParamAddressBookId, log.SafeString(addressBookId))

		var create jscontact.ContactCard
		err = req.bodydoc(&create, "The contact to create, which may not have its id attribute set")
		if err != nil {
			return req.error(accountId, err)
		}

		logger := log.From(l)
		created, sessionState, state, lang, jerr := g.jmap.CreateContactCard(accountId, req.session, req.ctx, logger, req.language(), create)
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}
		return req.respond(accountId, created, sessionState, ContactResponseObjectType, state)
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
			return req.error(accountId, err)
		}
		l.Str(UriParamContactId, log.SafeString(contactId))

		logger := log.From(l)

		deleted, sessionState, state, lang, jerr := g.jmap.DeleteContactCard(accountId, []string{contactId}, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.jmapError(accountId, jerr, sessionState, lang)
		}

		for _, e := range deleted {
			desc := e.Description
			if desc != "" {
				return req.error(accountId, apiError(
					req.errorId(),
					ErrorFailedToDeleteContact,
					withDetail(e.Description),
				))
			} else {
				return req.error(accountId, apiError(
					req.errorId(),
					ErrorFailedToDeleteContact,
				))
			}
		}
		return req.noContent(accountId, sessionState, ContactResponseObjectType, state)
	})
}

func mapContactCardSort(s SortCrit) jmap.ContactCardComparator {
	attr := s.Attribute
	if mapped, ok := ContactSortingPropertyMapping[s.Attribute]; ok {
		attr = mapped
	}
	return jmap.ContactCardComparator{Property: attr, IsAscending: s.Ascending}
}
