package jmap

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

var NS_ADDRESSBOOKS = ns(JmapContacts)

func (j *Client) GetAddressbooks(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, ids []string) (AddressBookGetResponse, SessionState, State, Language, Error) {
	return get(j, "GetAddressbooks", NS_ADDRESSBOOKS,
		func(accountId string, ids []string) AddressBookGetCommand {
			return AddressBookGetCommand{AccountId: accountId, Ids: ids}
		},
		AddressBookGetResponse{},
		identity1,
		accountId, session, ctx, logger, acceptLanguage, ids,
	)
}

type AddressBookChanges = ChangesTemplate[AddressBook]

// Retrieve Address Book changes since a given state.
// @apidoc addressbook,changes
func (j *Client) GetAddressbookChanges(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, sinceState State, maxChanges uint) (AddressBookChanges, SessionState, State, Language, Error) {
	return changesA(j, "GetAddressbookChanges", NS_ADDRESSBOOKS,
		func() AddressBookChangesCommand {
			return AddressBookChangesCommand{AccountId: accountId, SinceState: sinceState, MaxChanges: posUIntPtr(maxChanges)}
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
		session, ctx, logger, acceptLanguage,
	)
}

func (j *Client) CreateAddressBook(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, addressbook AddressBookChange) (*AddressBook, SessionState, State, Language, Error) {
	return create(j, "CreateAddressBook", NS_ADDRESSBOOKS,
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
		accountId, session, ctx, logger, acceptLanguage, addressbook,
	)
}

func (j *Client) DeleteAddressBook(accountId string, destroyIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string]SetError, SessionState, State, Language, Error) {
	return destroy(j, "DeleteAddressBook", NS_ADDRESSBOOKS,
		func(accountId string, destroy []string) AddressBookSetCommand {
			return AddressBookSetCommand{AccountId: accountId, Destroy: destroy}
		},
		AddressBookSetResponse{},
		accountId, destroyIds, session, ctx, logger, acceptLanguage,
	)
}

func (j *Client) UpdateAddressBook(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, id string, changes AddressBookChange) (AddressBook, SessionState, State, Language, Error) {
	return update(j, "UpdateAddressBook", NS_ADDRESSBOOKS,
		func(update map[string]PatchObject) AddressBookSetCommand {
			return AddressBookSetCommand{AccountId: accountId, Update: update}
		},
		func(id string) AddressBookGetCommand {
			return AddressBookGetCommand{AccountId: accountId, Ids: []string{id}}
		},
		func(resp AddressBookSetResponse) map[string]SetError { return resp.NotUpdated },
		func(resp AddressBookGetResponse) AddressBook { return resp.List[0] },
		id, changes, session, ctx, logger, acceptLanguage,
	)
}
