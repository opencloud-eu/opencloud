package jmap

import (
	"context"
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/jscontact"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

var NS_CONTACTS = ns(JmapContacts)

type AddressBooksResponse struct {
	AddressBooks []AddressBook `json:"addressbooks"`
	NotFound     []string      `json:"notFound,omitempty"`
}

func (j *Client) GetAddressbooks(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, ids []string) (AddressBooksResponse, SessionState, State, Language, Error) {
	logger = j.logger("GetAddressbooks", session, logger)

	cmd, err := j.request(session, logger, NS_CONTACTS,
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
	return changesTemplate(j, "GetAddressbookChanges", NS_CONTACTS,
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

func (j *Client) GetContactCardsById(accountId string, session *Session, ctx context.Context, logger *log.Logger,
	acceptLanguage string, contactIds []string) (map[string]jscontact.ContactCard, SessionState, State, Language, Error) {
	logger = j.logger("GetContactCardsById", session, logger)

	cmd, err := j.request(session, logger, NS_CONTACTS, invocation(CommandContactCardGet, ContactCardGetCommand{
		Ids:       contactIds,
		AccountId: accountId,
	}, "0"))
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string]jscontact.ContactCard, State, Error) {
		var response ContactCardGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandContactCardGet, "0", &response)
		if err != nil {
			return nil, "", err
		}
		m := map[string]jscontact.ContactCard{}
		for _, contact := range response.List {
			m[contact.Id] = contact
		}
		return m, response.State, nil
	})
}

func (j *Client) GetContactCards(accountId string, session *Session, ctx context.Context, logger *log.Logger,
	acceptLanguage string, contactIds []string) ([]jscontact.ContactCard, SessionState, State, Language, Error) {
	return getTemplate(j, "GetContactCards", NS_CONTACTS, CommandContactCardGet,
		func(accountId string, ids []string) ContactCardGetCommand {
			return ContactCardGetCommand{AccountId: accountId, Ids: contactIds}
		},
		func(resp ContactCardGetResponse) []jscontact.ContactCard { return resp.List },
		func(resp ContactCardGetResponse) State { return resp.State },
		accountId, session, ctx, logger, acceptLanguage, contactIds,
	)
}

type ContactCardChanges struct {
	OldState       State                   `json:"oldState,omitempty"`
	NewState       State                   `json:"newState"`
	HasMoreChanges bool                    `json:"hasMoreChanges"`
	Created        []jscontact.ContactCard `json:"created,omitempty"`
	Updated        []jscontact.ContactCard `json:"updated,omitempty"`
	Destroyed      []string                `json:"destroyed,omitempty"`
}

func (j *Client) GetContactCardChanges(accountId string, session *Session, ctx context.Context, logger *log.Logger,
	acceptLanguage string, sinceState State, maxChanges uint) (ContactCardChanges, SessionState, State, Language, Error) {
	return changesTemplate(j, "GetContactCardChanges", NS_CONTACTS,
		CommandContactCardChanges, CommandContactCardGet,
		func() ContactCardChangesCommand {
			return ContactCardChangesCommand{AccountId: accountId, SinceState: sinceState, MaxChanges: posUIntPtr(maxChanges)}
		},
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
		func(resp ContactCardChangesResponse) (State, State, bool, []string) {
			return resp.OldState, resp.NewState, resp.HasMoreChanges, resp.Destroyed
		},
		func(resp ContactCardGetResponse) []jscontact.ContactCard { return resp.List },
		func(oldState, newState State, hasMoreChanges bool, created, updated []jscontact.ContactCard, destroyed []string) ContactCardChanges {
			return ContactCardChanges{
				OldState:       oldState,
				NewState:       newState,
				HasMoreChanges: hasMoreChanges,
				Created:        created,
				Updated:        updated,
				Destroyed:      destroyed,
			}
		},
		func(resp ContactCardGetResponse) State { return resp.State },
		session, ctx, logger, acceptLanguage,
	)
}

func (j *Client) QueryContactCards(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, //NOSONAR
	filter ContactCardFilterElement, sortBy []ContactCardComparator,
	position uint, limit uint) (map[string][]jscontact.ContactCard, SessionState, State, Language, Error) {
	logger = j.logger("QueryContactCards", session, logger)

	uniqueAccountIds := structs.Uniq(accountIds)

	if sortBy == nil {
		sortBy = []ContactCardComparator{{Property: jscontact.ContactCardPropertyUpdated, IsAscending: false}}
	}

	invocations := make([]Invocation, len(uniqueAccountIds)*2)
	for i, accountId := range uniqueAccountIds {
		query := ContactCardQueryCommand{
			AccountId: accountId,
			Filter:    filter,
			Sort:      sortBy,
		}
		if limit > 0 {
			query.Limit = limit
		}
		if position > 0 {
			query.Position = position
		}
		invocations[i*2+0] = invocation(CommandContactCardQuery, query, mcid(accountId, "0"))
		invocations[i*2+1] = invocation(CommandContactCardGet, ContactCardGetRefCommand{
			AccountId: accountId,
			IdsRef: &ResultReference{
				Name:     CommandContactCardQuery,
				Path:     "/ids/*",
				ResultOf: mcid(accountId, "0"),
			},
		}, mcid(accountId, "1"))
	}
	cmd, err := j.request(session, logger, NS_CONTACTS, invocations...)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string][]jscontact.ContactCard, State, Error) {
		resp := map[string][]jscontact.ContactCard{}
		stateByAccountId := map[string]State{}
		for _, accountId := range uniqueAccountIds {
			var response ContactCardGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandContactCardGet, mcid(accountId, "1"), &response)
			if err != nil {
				return nil, "", err
			}
			if len(response.NotFound) > 0 {
				// TODO what to do when there are not-found emails here? potentially nothing, they could have been deleted between query and get?
			}
			resp[accountId] = response.List
			stateByAccountId[accountId] = response.State
		}
		return resp, squashState(stateByAccountId), nil
	})
}

func (j *Client) CreateContactCard(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, create jscontact.ContactCard) (*jscontact.ContactCard, SessionState, State, Language, Error) {
	logger = j.logger("CreateContactCard", session, logger)

	cmd, err := j.request(session, logger, NS_CONTACTS,
		invocation(CommandContactCardSet, ContactCardSetCommand{
			AccountId: accountId,
			Create: map[string]jscontact.ContactCard{
				"c": create,
			},
		}, "0"),
		invocation(CommandContactCardGet, ContactCardGetCommand{
			AccountId: accountId,
			Ids:       []string{"#c"},
		}, "1"),
	)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (*jscontact.ContactCard, State, Error) {
		var setResponse ContactCardSetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandContactCardSet, "0", &setResponse)
		if err != nil {
			return nil, "", err
		}

		setErr, notok := setResponse.NotCreated["c"]
		if notok {
			logger.Error().Msgf("%T.NotCreated returned an error %v", setResponse, setErr)
			return nil, "", setErrorError(setErr, EmailType)
		}

		if created, ok := setResponse.Created["c"]; !ok || created == nil {
			berr := fmt.Errorf("failed to find %s in %s response", string(ContactCardType), string(CommandContactCardSet))
			logger.Error().Err(berr)
			return nil, "", jmapError(berr, JmapErrorInvalidJmapResponsePayload)
		}

		var getResponse ContactCardGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandContactCardGet, "1", &getResponse)
		if err != nil {
			return nil, "", err
		}

		if len(getResponse.List) < 1 {
			berr := fmt.Errorf("failed to find %s in %s response", string(ContactCardType), string(CommandContactCardSet))
			logger.Error().Err(berr)
			return nil, "", jmapError(berr, JmapErrorInvalidJmapResponsePayload)
		}

		return &getResponse.List[0], setResponse.NewState, nil
	})
}

func (j *Client) DeleteContactCard(accountId string, destroy []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string]SetError, SessionState, State, Language, Error) {
	logger = j.logger("DeleteContactCard", session, logger)

	cmd, err := j.request(session, logger, NS_CONTACTS,
		invocation(CommandContactCardSet, ContactCardSetCommand{
			AccountId: accountId,
			Destroy:   destroy,
		}, "0"),
	)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string]SetError, State, Error) {
		var setResponse ContactCardSetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandContactCardSet, "0", &setResponse)
		if err != nil {
			return nil, "", err
		}
		return setResponse.NotDestroyed, setResponse.NewState, nil
	})
}
