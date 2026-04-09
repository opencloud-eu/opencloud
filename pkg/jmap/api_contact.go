package jmap

import (
	"context"
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

var NS_CONTACTS = ns(JmapContacts)

func (j *Client) GetContactCards(accountId string, session *Session, ctx context.Context, logger *log.Logger,
	acceptLanguage string, contactIds []string) (ContactCardGetResponse, SessionState, State, Language, Error) {
	return get(j, "GetContactCards", NS_CONTACTS,
		func(accountId string, ids []string) ContactCardGetCommand {
			return ContactCardGetCommand{AccountId: accountId, Ids: contactIds}
		},
		ContactCardGetResponse{},
		identity1,
		accountId, session, ctx, logger, acceptLanguage, contactIds,
	)
}

type ContactCardChanges = ChangesTemplate[ContactCard]

// Retrieve the changes in Contact Cards since a given State.
// @api:tags contact,changes
func (j *Client) GetContactCardChanges(accountId string, session *Session, ctx context.Context, logger *log.Logger,
	acceptLanguage string, sinceState State, maxChanges uint) (ContactCardChanges, SessionState, State, Language, Error) {
	return changes(j, "GetContactCardChanges", NS_CONTACTS,
		func() ContactCardChangesCommand {
			return ContactCardChangesCommand{AccountId: accountId, SinceState: sinceState, MaxChanges: posUIntPtr(maxChanges)}
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
		session, ctx, logger, acceptLanguage,
	)
}

func (j *Client) QueryContactCards(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, //NOSONAR
	filter ContactCardFilterElement, sortBy []ContactCardComparator,
	position uint, limit uint) (map[string][]ContactCard, SessionState, State, Language, Error) {
	logger = j.logger("QueryContactCards", session, logger)

	uniqueAccountIds := structs.Uniq(accountIds)

	if sortBy == nil {
		sortBy = []ContactCardComparator{{Property: ContactCardPropertyUpdated, IsAscending: false}}
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
		invocations[i*2+0] = invocation(query, mcid(accountId, "0"))
		invocations[i*2+1] = invocation(ContactCardGetRefCommand{
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

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string][]ContactCard, State, Error) {
		resp := map[string][]ContactCard{}
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

func (j *Client) CreateContactCard(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, create ContactCard) (*ContactCard, SessionState, State, Language, Error) {
	logger = j.logger("CreateContactCard", session, logger)

	cmd, err := j.request(session, logger, NS_CONTACTS,
		invocation(ContactCardSetCommand{
			AccountId: accountId,
			Create: map[string]ContactCard{
				"c": create,
			},
		}, "0"),
		invocation(ContactCardGetCommand{
			AccountId: accountId,
			Ids:       []string{"#c"},
		}, "1"),
	)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (*ContactCard, State, Error) {
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
			berr := fmt.Errorf("failed to find %s in %s response", ContactCardType, string(CommandContactCardSet))
			logger.Error().Err(berr)
			return nil, "", jmapError(berr, JmapErrorInvalidJmapResponsePayload)
		}

		var getResponse ContactCardGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandContactCardGet, "1", &getResponse)
		if err != nil {
			return nil, "", err
		}

		if len(getResponse.List) < 1 {
			berr := fmt.Errorf("failed to find %s in %s response", ContactCardType, string(CommandContactCardSet))
			logger.Error().Err(berr)
			return nil, "", jmapError(berr, JmapErrorInvalidJmapResponsePayload)
		}

		return &getResponse.List[0], setResponse.NewState, nil
	})
}

func (j *Client) DeleteContactCard(accountId string, destroyIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string]SetError, SessionState, State, Language, Error) {
	return destroy(j, "DeleteContactCard", NS_CONTACTS,
		func(accountId string, destroy []string) ContactCardSetCommand {
			return ContactCardSetCommand{AccountId: accountId, Destroy: destroy}
		},
		ContactCardSetResponse{},
		accountId, destroyIds, session, ctx, logger, acceptLanguage,
	)
}
