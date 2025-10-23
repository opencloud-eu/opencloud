package jmap

import (
	"context"
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/jscontact"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

type AddressBooksResponse struct {
	AddressBooks []AddressBook `json:"addressbooks"`
	NotFound     []string      `json:"notFound,omitempty"`
	State        State         `json:"state"`
}

func (j *Client) GetAddressbooks(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, ids []string) (AddressBooksResponse, SessionState, Language, Error) {
	logger = j.logger("GetAddressbooks", session, logger)

	cmd, err := j.request(session, logger,
		invocation(CommandAddressBookGet, AddressBookGetCommand{AccountId: accountId, Ids: ids}, "0"),
	)
	if err != nil {
		return AddressBooksResponse{}, "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (AddressBooksResponse, Error) {
		var response AddressBookGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandAddressBookGet, "0", &response)
		if err != nil {
			return AddressBooksResponse{}, err
		}
		return AddressBooksResponse{
			AddressBooks: response.List,
			NotFound:     response.NotFound,
			State:        response.State,
		}, nil
	})
}

func (j *Client) QueryContactCards(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string,
	filter ContactCardFilterElement, sortBy []ContactCardComparator,
	position uint, limit uint) (map[string][]jscontact.ContactCard, SessionState, Language, Error) {
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
	cmd, err := j.request(session, logger, invocations...)
	if err != nil {
		return nil, "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string][]jscontact.ContactCard, Error) {
		resp := map[string][]jscontact.ContactCard{}
		for _, accountId := range uniqueAccountIds {
			var response ContactCardGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandContactCardGet, mcid(accountId, "1"), &response)
			if err != nil {
				return nil, err
			}
			if len(response.NotFound) > 0 {
				// TODO what to do when there are not-found emails here? potentially nothing, they could have been deleted between query and get?
			}
			resp[accountId] = response.List
		}
		return resp, nil
	})
}

type CreatedContactCard struct {
	ContactCard *jscontact.ContactCard `json:"contactCard"`
	State       State                  `json:"state"`
}

func (j *Client) CreateContactCard(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, create jscontact.ContactCard) (CreatedContactCard, SessionState, Language, Error) {
	logger = j.logger("CreateContactCard", session, logger)

	cmd, err := j.request(session, logger,
		invocation(CommandContactCardSet, ContactCardSetCommand{
			AccountId: accountId,
			Create: map[string]jscontact.ContactCard{
				"c": create,
			},
		}, "0"),
		invocation(CommandContactCardGet, ContactCardGetRefCommand{
			AccountId: accountId,
			IdsRef: &ResultReference{
				ResultOf: "0",
				Name:     CommandContactCardSet,
				Path:     "/created/c/id",
			},
		}, "1"),
	)
	if err != nil {
		return CreatedContactCard{}, "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (CreatedContactCard, Error) {
		var setResponse ContactCardSetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandContactCardSet, "0", &setResponse)
		if err != nil {
			return CreatedContactCard{}, err
		}

		setErr, notok := setResponse.NotCreated["c"]
		if notok {
			logger.Error().Msgf("%T.NotCreated returned an error %v", setResponse, setErr)
			return CreatedContactCard{}, setErrorError(setErr, EmailType)
		}

		if created, ok := setResponse.Created["c"]; !ok || created != nil {
			berr := fmt.Errorf("failed to find %s in %s response", string(ContactCardType), string(CommandContactCardSet))
			logger.Error().Err(berr)
			return CreatedContactCard{}, simpleError(berr, JmapErrorInvalidJmapResponsePayload)
		}

		var getResponse ContactCardGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandContactCardGet, "1", &getResponse)
		if err != nil {
			return CreatedContactCard{}, err
		}

		if len(getResponse.List) < 1 {
			berr := fmt.Errorf("failed to find %s in %s response", string(ContactCardType), string(CommandContactCardSet))
			logger.Error().Err(berr)
			return CreatedContactCard{}, simpleError(berr, JmapErrorInvalidJmapResponsePayload)
		}

		return CreatedContactCard{
			ContactCard: &getResponse.List[0],
			State:       setResponse.NewState,
		}, nil
	})
}

type DeletedContactCards struct {
	State        State               `json:"state"`
	NotDestroyed map[string]SetError `json:"notDestroyed"`
}

func (j *Client) DeleteContactCard(accountId string, destroy []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (DeletedContactCards, SessionState, Language, Error) {
	logger = j.logger("DeleteContactCard", session, logger)

	cmd, err := j.request(session, logger,
		invocation(CommandContactCardSet, ContactCardSetCommand{
			AccountId: accountId,
			Destroy:   destroy,
		}, "0"),
	)
	if err != nil {
		return DeletedContactCards{}, "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (DeletedContactCards, Error) {
		var setResponse ContactCardSetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandContactCardSet, "0", &setResponse)
		if err != nil {
			return DeletedContactCards{}, err
		}
		return DeletedContactCards{
			State:        setResponse.NewState,
			NotDestroyed: setResponse.NotDestroyed,
		}, nil
	})
}
