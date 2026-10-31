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
	getallpaged(Contact, w, r, g, true,
		g.buildContactsFilter, supportedContactsFilterQueryParams,
		[]jmap.ContactCardComparator{{Property: jmap.ContactCardPropertyCreated, IsAscending: true}},
		curryQueryFunc(g.queryContactCards),
	)
}

func (g *Groupware) GetContactById(w http.ResponseWriter, r *http.Request) {
	get(Contact, w, r, g, g.listContactCards)
}

func (g *Groupware) GetAllContacts(w http.ResponseWriter, r *http.Request) {
	getallpaged(Contact, w, r, g, false,
		g.buildContactsFilter, supportedContactsFilterQueryParams,
		[]jmap.ContactCardComparator{{Property: jmap.ContactCardPropertyCreated, IsAscending: true}},
		curryQueryFunc(g.queryContactCards),
	)
}

var supportedContactsFilterQueryParams = toSupportedQueryParams(
	QueryParamContactFilterUid,
	QueryParamContactFilterUid,
	QueryParamContactFilterCreatedAfter,
	QueryParamContactFilterCreatedBefore,
	QueryParamContactFilterEmail,
	QueryParamContactFilterMember,
	QueryParamContactFilterKind,
	QueryParamContactFilterName,
	// TODO add more as they are added below
)

func (g *Groupware) buildContactsFilter(addressbookId string, req Request, _ *log.Logger) (jmap.ContactCardFilterElement, *Error) {
	filter := jmap.ContactCardFilterCondition{}
	if addressbookId != "" {
		filter.InAddressBook = addressbookId
	}
	if v, ok := req.getStringParam(QueryParamContactFilterUid, ""); ok {
		filter.Uid = v
	}
	if v, ok := req.getStringParam(QueryParamContactFilterAddress, ""); ok {
		filter.Address = v
	}
	if v, ok, err := req.parseUTCDateParam(QueryParamContactFilterCreatedAfter); err != nil {
		return filter, err
	} else if ok {
		filter.CreatedAfter = v
	}
	if v, ok, err := req.parseUTCDateParam(QueryParamContactFilterCreatedBefore); err != nil {
		return filter, err
	} else if ok {
		filter.CreatedBefore = v
	}
	if v, ok := req.getStringParam(QueryParamContactFilterEmail, ""); ok {
		filter.Email = v
	}
	if v, ok := req.getStringParam(QueryParamContactFilterMember, ""); ok {
		filter.HasMember = v
	}
	if v, ok, err := req.getValidStringParam(QueryParamContactFilterKind, "", striter(jscontact.ContactCardKinds)); err != nil {
		return filter, err
	} else if ok {
		filter.Kind = jscontact.ContactCardKind(v)
	}
	if v, ok := req.getStringParam(QueryParamContactFilterName, ""); ok {
		filter.Name = v
	}
	// TODO more filter conditions for ContactCard
	return filter, nil
}

// Get changes to Contacts since a given State
// @api:tags contact,changes
func (g *Groupware) GetContactsChanges(w http.ResponseWriter, r *http.Request) {
	changes(Contact, w, r, g, g.contactCardsChanges)
}

func (g *Groupware) CreateContact(w http.ResponseWriter, r *http.Request) {
	create(Contact, w, r, g, nil, g.createContactCard)
}

func (g *Groupware) DeleteContact(w http.ResponseWriter, r *http.Request) {
	deleteById(Contact, w, r, g, g.deleteContactCards)
}

func (g *Groupware) ModifyContact(w http.ResponseWriter, r *http.Request) {
	modify(Contact, w, r, g, g.updateContactCard)
}

func (g *Groupware) createContactCard(accountId jmap.AccountId, contact jmap.ContactCardChange, ctx jmap.Context) (jmap.Result[*jmap.ContactCard], error) {
	return screate(g.suppliers.contactCards.create, accountId, contact, ctx)
}

func (g *Groupware) updateContactCard(accountId jmap.AccountId, id string, contactCard jmap.ContactCardChange, ctx jmap.Context) (jmap.Result[jmap.ContactCard], error) {
	return supdate(g.suppliers.contactCards.update, accountId, id, contactCard, ctx)
}

func (g *Groupware) deleteContactCards(accountId jmap.AccountId, destroyIds []string, ctx jmap.Context) (jmap.Result[map[string]jmap.SetError], error) {
	return sdelete[jmap.ContactCard](g.suppliers.contactCards.delete, accountId, destroyIds, ctx)
}

func (g *Groupware) listContactCards(accountId jmap.AccountId, ids []string, ctx jmap.Context) (jmap.Result[jmap.ContactCardGetResponse], error) {
	return slist(g.suppliers.contactCards.list, accountId, ids, ctx,
		func(accountId jmap.AccountId, state jmap.State, notFound []string, list []jmap.ContactCard) jmap.ContactCardGetResponse {
			return jmap.ContactCardGetResponse{AccountId: accountId, State: state, NotFound: notFound, List: list}
		},
	)
}

func (g *Groupware) contactCardsChanges(accountId jmap.AccountId, sinceState jmap.State, maxChanges uint, ctx jmap.Context) (jmap.Result[jmap.ContactCardChanges], error) {
	return schanges(g.suppliers.contactCards.changes, accountId, sinceState, maxChanges, ctx,
		func(accountId jmap.AccountId, oldState, newState jmap.State, created, updated []jmap.ContactCard, destroyed []string, hasMoreChanges bool) jmap.ContactCardChanges {
			return jmap.ContactCardChanges{HasMoreChanges: hasMoreChanges, OldState: oldState, NewState: newState, Created: created, Updated: updated, Destroyed: destroyed}
		},
	)
}

func (g *Groupware) queryContactCards(accountIds []jmap.AccountId, qps QueryParamsSupplier, limit *uint, //NOSONAR
	filter jmap.ContactCardFilterElement, sortBy []jmap.ContactCardComparator, calculateTotal bool,
	ctx jmap.Context) (jmap.Result[*jmap.ContactCardSearchResults], NextToken, error) {
	return squery(g.suppliers.contactCards.query, accountIds, qps, limit, filter, sortBy, calculateTotal, ctx,
		func(supplier QuerySupplier[jmap.ContactCard, *jmap.ContactCardSearchResults, jmap.ContactCardFilterElement, jmap.ContactCardComparator], filter jmap.ContactCardFilterElement) bool {
			switch c := filter.(type) {
			case jmap.ContactCardFilterCondition:
				if c.InAddressBook != "" {
					if !supplier.IsMine(c.InAddressBook) {
						return false
					}
				}
			}
			return true
		},
		func(a, b jmap.ContactCard) int { return a.Created.Compare(b.Created) },
		func(canCalculateChanges jmap.ChangeCalculation, position, limit, total *uint, results []jmap.ContactCard) *jmap.ContactCardSearchResults {
			return &jmap.ContactCardSearchResults{
				Results:             results,
				CanCalculateChanges: canCalculateChanges,
				Position:            position,
				Limit:               limit,
				Total:               total,
			}
		},
	)
}
