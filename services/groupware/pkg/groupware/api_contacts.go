package groupware

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
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
		func(addressbookId string) jmap.ContactCardFilterElement {
			return jmap.ContactCardFilterCondition{InAddressBook: addressbookId}
		},
		[]jmap.ContactCardComparator{{Property: jmap.ContactCardPropertyCreated, IsAscending: true}},
		curryQueryFunc(g.contacts),
	)
}

func (g *Groupware) GetContactById(w http.ResponseWriter, r *http.Request) {
	get(Contact, w, r, g, g.jmap.GetContactCards)
}

func (g *Groupware) GetAllContacts(w http.ResponseWriter, r *http.Request) {
	getallpaged(Contact, w, r, g, false,
		func(_ string) jmap.ContactCardFilterElement {
			return jmap.ContactCardFilterCondition{}
		},
		[]jmap.ContactCardComparator{{Property: jmap.ContactCardPropertyCreated, IsAscending: true}},
		curryQueryFunc(g.contacts),
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

func (g *Groupware) contacts(accountIds []string, qps QueryParamsSupplier, limit *uint, //NOSONAR
	filter jmap.ContactCardFilterElement, sortBy []jmap.ContactCardComparator, calculateTotal bool,
	ctx jmap.Context) (jmap.Result[*jmap.ContactCardSearchResults], NextToken, error) {
	return squery(g.contactCardQuerySuppliers, accountIds, qps, limit, filter, sortBy, calculateTotal, ctx,
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
