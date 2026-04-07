package jmap

import (
	"context"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

var NS_ADDRESSBOOKS = ns(JmapContacts)

type AddressBooksResponse struct {
	AddressBooks []AddressBook `json:"addressbooks"`
	NotFound     []string      `json:"notFound,omitempty"`
}

func (j *Client) GetAddressbooks(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, ids []string) (AddressBooksResponse, SessionState, State, Language, Error) {
	return get(j, "GetAddressbooks", NS_ADDRESSBOOKS,
		func(accountId string, ids []string) AddressBookGetCommand {
			return AddressBookGetCommand{AccountId: accountId, Ids: ids}
		},
		AddressBookGetResponse{},
		func(resp AddressBookGetResponse) AddressBooksResponse {
			return AddressBooksResponse{AddressBooks: resp.List, NotFound: resp.NotFound}
		},
		accountId, session, ctx, logger, acceptLanguage, ids,
	)
}

type AddressBookChanges struct {
	HasMoreChanges bool          `json:"hasMoreChanges"`
	OldState       State         `json:"oldState,omitempty"`
	NewState       State         `json:"newState"`
	Created        []AddressBook `json:"created,omitempty"`
	Updated        []AddressBook `json:"updated,omitempty"`
	Destroyed      []string      `json:"destroyed,omitempty"`
}

// Retrieve Address Book changes since a given state.
// @apidoc addressbook,changes
func (j *Client) GetAddressbookChanges(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, sinceState State, maxChanges uint) (AddressBookChanges, SessionState, State, Language, Error) {
	return changes(j, "GetAddressbookChanges", NS_ADDRESSBOOKS,
		func() AddressBookChangesCommand {
			return AddressBookChangesCommand{AccountId: accountId, SinceState: sinceState, MaxChanges: posUIntPtr(maxChanges)}
		},
		AddressBookChangesResponse{},
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
		func(resp AddressBookGetResponse) []AddressBook { return resp.List },
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
