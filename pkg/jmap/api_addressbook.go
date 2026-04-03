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
	logger = j.logger("GetAddressbooks", session, logger)

	cmd, err := j.request(session, logger, NS_ADDRESSBOOKS,
		invocation(CommandAddressBookGet, AddressBookGetCommand{AccountId: accountId, Ids: ids}, "0"),
	)
	if err != nil {
		return AddressBooksResponse{}, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (AddressBooksResponse, State, Error) {
		var response AddressBookGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandAddressBookGet, "0", &response)
		if err != nil {
			return AddressBooksResponse{}, response.State, err
		}
		return AddressBooksResponse{
			AddressBooks: response.List,
			NotFound:     response.NotFound,
		}, response.State, nil
	})
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
	return changesTemplate(j, "GetAddressbookChanges", NS_ADDRESSBOOKS,
		CommandAddressBookChanges, CommandAddressBookGet,
		func() AddressBookChangesCommand {
			return AddressBookChangesCommand{AccountId: accountId, SinceState: sinceState, MaxChanges: posUIntPtr(maxChanges)}
		},
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
		func(resp AddressBookChangesResponse) (State, State, bool, []string) {
			return resp.OldState, resp.NewState, resp.HasMoreChanges, resp.Destroyed
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
		func(resp AddressBookGetResponse) State { return resp.State },
		session, ctx, logger, acceptLanguage,
	)
}

func (j *Client) CreateAddressBook(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, create AddressBookChange) (*AddressBook, SessionState, State, Language, Error) {
	return createTemplate(j, "CreateAddressBook", NS_ADDRESSBOOKS, AddressBookType, CommandAddressBookSet, CommandAddressBookGet,
		func(accountId string, create map[string]AddressBookChange) AddressBookSetCommand {
			return AddressBookSetCommand{AccountId: accountId, Create: create}
		},
		func(accountId string, ref string) AddressBookGetCommand {
			return AddressBookGetCommand{AccountId: accountId, Ids: []string{ref}}
		},
		func(resp AddressBookSetResponse) map[string]*AddressBook {
			return resp.Created
		},
		func(resp AddressBookSetResponse) map[string]SetError {
			return resp.NotCreated
		},
		func(resp AddressBookGetResponse) []AddressBook {
			return resp.List
		},
		func(resp AddressBookSetResponse) State {
			return resp.NewState
		},
		accountId, session, ctx, logger, acceptLanguage, create,
	)
}

func (j *Client) DeleteAddressBook(accountId string, destroy []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string]SetError, SessionState, State, Language, Error) {
	return deleteTemplate(j, "DeleteAddressBook", NS_ADDRESSBOOKS, CommandAddressBookSet,
		func(accountId string, destroy []string) AddressBookSetCommand {
			return AddressBookSetCommand{AccountId: accountId, Destroy: destroy}
		},
		func(resp AddressBookSetResponse) map[string]SetError { return resp.NotDestroyed },
		func(resp AddressBookSetResponse) State { return resp.NewState },
		accountId, destroy, session, ctx, logger, acceptLanguage,
	)
}
