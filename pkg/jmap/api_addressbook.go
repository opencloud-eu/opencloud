package jmap

var NS_ADDRESSBOOKS = ns(JmapContacts)

func (j *Client) GetAddressbooks(accountId string, ids []string, ctx Context) (Result[AddressBookGetResponse], Error) {
	return get(j, "GetAddressbooks", MailboxType,
		func(accountId string, ids []string) AddressBookGetCommand {
			return AddressBookGetCommand{AccountId: accountId, Ids: ids}
		},
		AddressBookGetResponse{},
		identity1,
		accountId, ids,
		ctx,
	)
}

type AddressBookChanges ChangesTemplate[AddressBook]

var _ Changes[AddressBook] = AddressBookChanges{}

func (c AddressBookChanges) GetHasMoreChanges() bool   { return c.HasMoreChanges }
func (c AddressBookChanges) GetOldState() State        { return c.OldState }
func (c AddressBookChanges) GetNewState() State        { return c.NewState }
func (c AddressBookChanges) GetCreated() []AddressBook { return c.Created }
func (c AddressBookChanges) GetUpdated() []AddressBook { return c.Updated }
func (c AddressBookChanges) GetDestroyed() []string    { return c.Destroyed }

// Retrieve Address Book changes since a given state.
// @apidoc addressbook,changes
func (j *Client) GetAddressbookChanges(accountId string, sinceState State, maxChanges uint, ctx Context) (Result[AddressBookChanges], Error) {
	return changesA(j, "GetAddressbookChanges", MailboxType,
		func() AddressBookChangesCommand {
			return AddressBookChangesCommand{AccountId: accountId, SinceState: sinceState, MaxChanges: uintPtr(maxChanges)}
		},
		AddressBookChangesResponse{},
		AddressBookGetResponse{},
		func(path string, rof string) AddressBookGetRefCommand {
			return AddressBookGetRefCommand{
				AccountId: accountId,
				IdsRef: &ResultReference{
					Name:     CommandAddressBookChanges,
					Path:     path,
					ResultOf: rof,
				},
			}
		},
		func(oldState, newState State, hasMoreChanges bool, created, updated []AddressBook, destroyed []string) AddressBookChanges {
			return AddressBookChanges{
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

func (j *Client) CreateAddressBook(accountId string, addressbook AddressBookChange, ctx Context) (Result[*AddressBook], Error) {
	return create(j, "CreateAddressBook", MailboxType,
		func(accountId string, create map[string]AddressBookChange) AddressBookSetCommand {
			return AddressBookSetCommand{AccountId: accountId, Create: create}
		},
		func(accountId string, ids string) AddressBookGetCommand {
			return AddressBookGetCommand{AccountId: accountId, Ids: []string{ids}}
		},
		func(resp AddressBookSetResponse) map[string]*AddressBook {
			return resp.Created
		},
		func(resp AddressBookGetResponse) []AddressBook {
			return resp.List
		},
		accountId, addressbook,
		ctx,
	)
}

func (j *Client) DeleteAddressBook(accountId string, destroyIds []string, ctx Context) (Result[map[string]SetError], Error) {
	return destroy(j, "DeleteAddressBook", MailboxType,
		func(accountId string, destroy []string) AddressBookSetCommand {
			return AddressBookSetCommand{AccountId: accountId, Destroy: destroy}
		},
		AddressBookSetResponse{},
		accountId, destroyIds,
		ctx,
	)
}

func (j *Client) UpdateAddressBook(accountId string, id string, changes AddressBookChange, ctx Context) (Result[AddressBook], Error) {
	return update(j, "UpdateAddressBook", MailboxType,
		func(update map[string]PatchObject) AddressBookSetCommand {
			return AddressBookSetCommand{AccountId: accountId, Update: update}
		},
		func(id string) AddressBookGetCommand {
			return AddressBookGetCommand{AccountId: accountId, Ids: []string{id}}
		},
		func(resp AddressBookSetResponse) map[string]SetError { return resp.NotUpdated },
		func(resp AddressBookGetResponse) AddressBook { return resp.List[0] },
		id, changes,
		ctx,
	)
}
