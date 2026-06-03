package jmap

import "github.com/opencloud-eu/opencloud/pkg/jscontact"

var NS_CONTACTS = ns(JmapContacts)

var DEFAULT_CONTACT_CARD_VERSION = jscontact.JSContactVersion_1_0

func (j *Client) GetContactCards(accountId AccountId, contactIds []string, ctx Context) (Result[ContactCardGetResponse], error) {
	return get(j, "GetContactCards", ContactCardType,
		func(accountId AccountId, ids []string) ContactCardGetCommand {
			return ContactCardGetCommand{AccountId: accountId, Ids: contactIds}
		},
		ContactCardGetResponse{},
		identity1,
		accountId, contactIds,
		ctx,
	)
}

type ContactCardChanges ChangesTemplate[ContactCard]

var _ Changes[ContactCard] = ContactCardChanges{}

func (c ContactCardChanges) GetHasMoreChanges() bool   { return c.HasMoreChanges }
func (c ContactCardChanges) GetOldState() State        { return c.OldState }
func (c ContactCardChanges) GetNewState() State        { return c.NewState }
func (c ContactCardChanges) GetCreated() []ContactCard { return c.Created }
func (c ContactCardChanges) GetUpdated() []ContactCard { return c.Updated }
func (c ContactCardChanges) GetDestroyed() []string    { return c.Destroyed }

// Retrieve the changes in Contact Cards since a given State.
// @api:tags contact,changes
func (j *Client) GetContactCardChanges(accountId AccountId, sinceState State, maxChanges uint, ctx Context) (Result[ContactCardChanges], error) {
	return changes(j, "GetContactCardChanges", ContactCardType,
		func() ContactCardChangesCommand {
			return ContactCardChangesCommand{AccountId: accountId, SinceState: sinceState, MaxChanges: uintPtr(maxChanges)}
		},
		ContactCardChangesResponse{},
		func(path string, rof string) ContactCardGetRefCommand {
			return ContactCardGetRefCommand{
				AccountId: accountId,
				IdsRef: &ResultReference{
					Name:     CommandContactCardChanges,
					Path:     path,
					ResultOf: rof,
				},
			}
		},
		func(resp ContactCardGetResponse) []ContactCard { return resp.List },
		func(oldState, newState State, hasMoreChanges bool, created, updated []ContactCard, destroyed []string) ContactCardChanges {
			return ContactCardChanges{
				OldState:       oldState,
				NewState:       newState,
				HasMoreChanges: hasMoreChanges,
				Created:        created,
				Updated:        updated,
				Destroyed:      destroyed,
			}
		},
		ctx,
	)
}

type ContactCardSearchResults SearchResultsTemplate[ContactCard]

var _ SearchResults[ContactCard] = &ContactCardSearchResults{}

func (r *ContactCardSearchResults) GetResults() []ContactCard { return r.Results }
func (r *ContactCardSearchResults) GetCanCalculateChanges() ChangeCalculation {
	return r.CanCalculateChanges
}
func (r *ContactCardSearchResults) GetPosition() *uint         { return r.Position }
func (r *ContactCardSearchResults) GetLimit() *uint            { return r.Limit }
func (r *ContactCardSearchResults) GetTotal() *uint            { return r.Total }
func (r *ContactCardSearchResults) RemoveResults()             { r.Results = nil }
func (r *ContactCardSearchResults) SetLimit(limit *uint)       { r.Limit = limit }
func (r *ContactCardSearchResults) SetPosition(position *uint) { r.Position = position }

func (j *Client) QueryContactCards(accountIds map[AccountId]QueryParams, //NOSONAR
	limit *uint, filter ContactCardFilterElement, sortBy []ContactCardComparator, calculateTotal bool,
	ctx Context) (Result[map[AccountId]*ContactCardSearchResults], error) {
	return queryN(j, "QueryContactCards", ContactCardType,
		[]ContactCardComparator{{Property: ContactCardPropertyUpdated, IsAscending: false}},
		func(accountId AccountId, qp QueryParams, limit *uint, filter ContactCardFilterElement, sortBy []ContactCardComparator) ContactCardQueryCommand {
			if qp.Anchor != "" {
				return ContactCardQueryCommand{AccountId: accountId, Filter: filter, Sort: sortBy, Anchor: qp.Anchor, AnchorOffset: qp.AnchorOffset, Limit: limit, CalculateTotal: calculateTotal}
			} else {
				return ContactCardQueryCommand{AccountId: accountId, Filter: filter, Sort: sortBy, Position: qp.Position, Limit: limit, CalculateTotal: calculateTotal}
			}
		},
		func(accountId AccountId, cmd Command, path string, rof string) ContactCardGetRefCommand {
			return ContactCardGetRefCommand{AccountId: accountId, IdsRef: &ResultReference{Name: cmd, Path: path, ResultOf: rof}}
		},
		func(query ContactCardQueryResponse, queryParams QueryParams, limit *uint) *ContactCardSearchResults {
			return &ContactCardSearchResults{
				Results:             []ContactCard{},
				CanCalculateChanges: ChangeCalculation(query.CanCalculateChanges),
				Position:            valueIf(query.Position, queryParams.Anchor == ""),
				Total:               valueIf(query.Total, calculateTotal),
				Limit:               valueIf(query.Limit, limit != nil),
			}
		},
		func(query ContactCardQueryResponse, get ContactCardGetResponse, queryParams QueryParams, limit *uint) *ContactCardSearchResults {
			return &ContactCardSearchResults{
				Results:             get.List,
				CanCalculateChanges: ChangeCalculation(query.CanCalculateChanges),
				Position:            valueIf(query.Position, queryParams.Anchor == ""),
				Total:               valueIf(query.Total, calculateTotal),
				Limit:               valueIf(query.Limit, limit != nil),
			}
		},
		accountIds, limit,
		filter, sortBy, ctx,
	)
}

// @api:example create
func (j *Client) CreateContactCard(accountId AccountId, contact ContactCardChange, ctx Context) (Result[*ContactCard], error) {
	if contact.Version == nil {
		contact.Version = &DEFAULT_CONTACT_CARD_VERSION
	}
	return create(j, "CreateContactCard", ContactCardType,
		func(accountId AccountId, create map[string]ContactCardChange) ContactCardSetCommand {
			return ContactCardSetCommand{AccountId: accountId, Create: create}
		},
		func(accountId AccountId, ids string) ContactCardGetCommand {
			return ContactCardGetCommand{AccountId: accountId, Ids: []string{ids}}
		},
		func(resp ContactCardSetResponse) map[string]*ContactCard {
			return resp.Created
		},
		func(resp ContactCardGetResponse) []ContactCard {
			return resp.List
		},
		accountId, contact,
		ctx,
	)
}

func (j *Client) DeleteContactCard(accountId AccountId, destroyIds []string, ctx Context) (Result[map[string]SetError], error) {
	return destroy(j, "DeleteContactCard", ContactCardType,
		func(accountId AccountId, destroy []string) ContactCardSetCommand {
			return ContactCardSetCommand{AccountId: accountId, Destroy: destroy}
		},
		ContactCardSetResponse{},
		accountId, destroyIds,
		ctx,
	)
}

// @api:example update
func (j *Client) UpdateContactCard(accountId AccountId, id string, changes ContactCardChange, ctx Context) (Result[ContactCard], error) {
	return update(j, "UpdateContactCard", ContactCardType,
		func(update map[string]PatchObject) ContactCardSetCommand {
			return ContactCardSetCommand{AccountId: accountId, Update: update}
		},
		func(id string) ContactCardGetCommand {
			return ContactCardGetCommand{AccountId: accountId, Ids: []string{id}}
		},
		func(resp ContactCardSetResponse) map[string]SetError { return resp.NotUpdated },
		func(resp ContactCardGetResponse) ContactCard { return resp.List[0] },
		id, changes,
		ctx,
	)
}
