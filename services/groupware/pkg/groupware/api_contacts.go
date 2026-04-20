package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
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
		{Property: jmap.ContactCardPropertyUpdated, IsAscending: true},
	}

	SupportedContactSortingProperties = []string{
		jmap.ContactCardPropertyUpdated,
		jmap.ContactCardPropertyCreated,
	}

	ContactSortingPropertyMapping = map[string]string{
		"surname": string(jmap.ContactCardPropertyName) + "/surname",
		"given":   string(jmap.ContactCardPropertyName) + "/given",
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

		offset, ok, err := req.parseIntParam(QueryParamOffset, 0)
		if err != nil {
			return req.errorN(accountIds, err)
		}
		if ok {
			l = l.Int(QueryParamOffset, offset)
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
		ctx := req.ctx.WithLogger(logger)
		contactsByAccountId, sessionState, state, lang, jerr := g.jmap.QueryContactCards(accountIds, filter, sortBy, offset, limit, true, ctx)
		if jerr != nil {
			return req.jmapErrorN(accountIds, jerr, sessionState, lang)
		}

		if contacts, ok := contactsByAccountId[accountId]; ok {
			return req.respondN(accountIds, contacts, sessionState, ContactResponseObjectType, state, lang)
		} else {
			return req.notFoundN(accountIds, sessionState, ContactResponseObjectType, state)
		}
	})
}

func (g *Groupware) GetContactById(w http.ResponseWriter, r *http.Request) {
	get(Contact, w, r, g, g.jmap.GetContactCards)
}

func (g *Groupware) GetAllContacts(w http.ResponseWriter, r *http.Request) {
	getallpaged(Contact, w, r, g,
		g.jmap.GetContactCards,
		func(cid string) jmap.ContactCardFilterElement {
			return jmap.ContactCardFilterCondition{InAddressBook: cid}
		},
		[]jmap.ContactCardComparator{{Property: jmap.ContactCardPropertyUpdated, IsAscending: true}},
		curryMapQuery(g.jmap.QueryContactCards),
	)
}

// Get changes to Contacts since a given State
// @api:tags contact,changes
func (g *Groupware) GetContactsChanges(w http.ResponseWriter, r *http.Request) {
	changes(Contact, w, r, g, g.jmap.GetContactCardChanges)
}

func (g *Groupware) CreateContact(w http.ResponseWriter, r *http.Request) {
	create(Contact, w, r, g, nil, g.jmap.CreateContactCard)
}

func (g *Groupware) DeleteContact(w http.ResponseWriter, r *http.Request) {
	delete(Contact, w, r, g, g.jmap.DeleteContactCard)
}

func (g *Groupware) ModifyContact(w http.ResponseWriter, r *http.Request) {
	modify(Contact, w, r, g, g.jmap.UpdateContactCard)
}

func mapContactCardSort(s SortCrit) jmap.ContactCardComparator {
	attr := s.Attribute
	if mapped, ok := ContactSortingPropertyMapping[s.Attribute]; ok {
		attr = mapped
	}
	return jmap.ContactCardComparator{Property: attr, IsAscending: s.Ascending}
}
