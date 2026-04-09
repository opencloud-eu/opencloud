package jmap

var NS_CONTACTS = ns(JmapContacts)

func (j *Client) GetContactCards(accountId string, contactIds []string, ctx Context) (ContactCardGetResponse, SessionState, State, Language, Error) {
	return get(j, "GetContactCards", NS_CONTACTS,
		func(accountId string, ids []string) ContactCardGetCommand {
			return ContactCardGetCommand{AccountId: accountId, Ids: contactIds}
		},
		ContactCardGetResponse{},
		identity1,
		accountId, contactIds,
		ctx,
	)
}

type ContactCardChanges = ChangesTemplate[ContactCard]

// Retrieve the changes in Contact Cards since a given State.
// @api:tags contact,changes
func (j *Client) GetContactCardChanges(accountId string, sinceState State, maxChanges uint, ctx Context) (ContactCardChanges, SessionState, State, Language, Error) {
	return changes(j, "GetContactCardChanges", NS_CONTACTS,
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

var _ SearchResults[ContactCard] = ContactCardSearchResults{}

func (r ContactCardSearchResults) GetResults() []ContactCard    { return r.Results }
func (r ContactCardSearchResults) GetCanCalculateChanges() bool { return r.CanCalculateChanges }
func (r ContactCardSearchResults) GetPosition() uint            { return r.Position }
func (r ContactCardSearchResults) GetLimit() uint               { return r.Limit }
func (r ContactCardSearchResults) GetTotal() *uint              { return r.Total }

func (j *Client) QueryContactCards(accountIds []string,
	filter ContactCardFilterElement, sortBy []ContactCardComparator,
	position int, limit uint, calculateTotal bool,
	ctx Context) (map[string]ContactCardSearchResults, SessionState, State, Language, Error) {
	return queryN(j, "QueryContactCards", NS_CONTACTS,
		[]ContactCardComparator{{Property: ContactCardPropertyUpdated, IsAscending: false}},
		func(accountId string, filter ContactCardFilterElement, sortBy []ContactCardComparator, position int, limit uint) ContactCardQueryCommand {
			return ContactCardQueryCommand{AccountId: accountId, Filter: filter, Sort: sortBy, Position: position, Limit: uintPtr(limit), CalculateTotal: calculateTotal}
		},
		func(accountId string, cmd Command, path string, rof string) ContactCardGetRefCommand {
			return ContactCardGetRefCommand{AccountId: accountId, IdsRef: &ResultReference{Name: cmd, Path: path, ResultOf: rof}}
		},
		func(query ContactCardQueryResponse, get ContactCardGetResponse) ContactCardSearchResults {
			return ContactCardSearchResults{
				Results:             get.List,
				CanCalculateChanges: query.CanCalculateChanges,
				Position:            query.Position,
				Total:               uintPtrIf(query.Total, calculateTotal),
				Limit:               query.Limit,
			}
		},
		accountIds,
		filter, sortBy, limit, position, ctx,
	)
}

func (j *Client) CreateContactCard(accountId string, contact ContactCard, ctx Context) (*ContactCard, SessionState, State, Language, Error) {
	return create(j, "CreateContactCard", NS_CONTACTS,
		func(accountId string, create map[string]ContactCard) ContactCardSetCommand {
			return ContactCardSetCommand{AccountId: accountId, Create: create}
		},
		func(accountId string, ids string) ContactCardGetCommand {
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

func (j *Client) DeleteContactCard(accountId string, destroyIds []string, ctx Context) (map[string]SetError, SessionState, State, Language, Error) {
	return destroy(j, "DeleteContactCard", NS_CONTACTS,
		func(accountId string, destroy []string) ContactCardSetCommand {
			return ContactCardSetCommand{AccountId: accountId, Destroy: destroy}
		},
		ContactCardSetResponse{},
		accountId, destroyIds,
		ctx,
	)
}
