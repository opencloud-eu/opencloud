package jmap

import (
	"context"
	"strconv"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

func (j *Client) GetAllIdentities(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) ([]Identity, SessionState, State, Language, Error) {
	logger = j.logger("GetAllIdentities", session, logger)
	cmd, err := j.request(session, logger, invocation(CommandIdentityGet, IdentityGetCommand{AccountId: accountId}, "0"))
	if err != nil {
		return nil, "", "", "", err
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) ([]Identity, State, Error) {
		var response IdentityGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandIdentityGet, "0", &response)
		if err != nil {
			return nil, "", err
		}
		return response.List, response.State, nil
	})
}

func (j *Client) GetIdentities(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, identityIds []string) ([]Identity, SessionState, State, Language, Error) {
	logger = j.logger("GetIdentities", session, logger)
	cmd, err := j.request(session, logger, invocation(CommandIdentityGet, IdentityGetCommand{AccountId: accountId, Ids: identityIds}, "0"))
	if err != nil {
		return nil, "", "", "", err
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) ([]Identity, State, Error) {
		var response IdentityGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandIdentityGet, "0", &response)
		if err != nil {
			return nil, "", err
		}
		return response.List, response.State, nil
	})
}

type IdentitiesGetResponse struct {
	Identities map[string][]Identity `json:"identities,omitempty"`
	NotFound   []string              `json:"notFound,omitempty"`
}

func (j *Client) GetIdentitiesForAllAccounts(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (IdentitiesGetResponse, SessionState, State, Language, Error) {
	logger = j.logger("GetIdentitiesForAllAccounts", session, logger)
	uniqueAccountIds := structs.Uniq(accountIds)
	calls := make([]Invocation, len(uniqueAccountIds))
	for i, accountId := range uniqueAccountIds {
		calls[i] = invocation(CommandIdentityGet, IdentityGetCommand{AccountId: accountId}, strconv.Itoa(i))
	}

	cmd, err := j.request(session, logger, calls...)
	if err != nil {
		return IdentitiesGetResponse{}, "", "", "", err
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (IdentitiesGetResponse, State, Error) {
		identities := make(map[string][]Identity, len(uniqueAccountIds))
		stateByAccountId := make(map[string]State, len(uniqueAccountIds))
		notFound := []string{}
		for i, accountId := range uniqueAccountIds {
			var response IdentityGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandIdentityGet, strconv.Itoa(i), &response)
			if err != nil {
				return IdentitiesGetResponse{}, "", err
			} else {
				identities[accountId] = response.List
			}
			stateByAccountId[accountId] = response.State
			notFound = append(notFound, response.NotFound...)
		}

		return IdentitiesGetResponse{
			Identities: identities,
			NotFound:   structs.Uniq(notFound),
		}, squashState(stateByAccountId), nil
	})
}

type IdentitiesAndMailboxesGetResponse struct {
	Identities map[string][]Identity `json:"identities,omitempty"`
	NotFound   []string              `json:"notFound,omitempty"`
	Mailboxes  []Mailbox             `json:"mailboxes"`
}

func (j *Client) GetIdentitiesAndMailboxes(mailboxAccountId string, accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (IdentitiesAndMailboxesGetResponse, SessionState, State, Language, Error) {
	uniqueAccountIds := structs.Uniq(accountIds)

	logger = j.logger("GetIdentitiesAndMailboxes", session, logger)

	calls := make([]Invocation, len(uniqueAccountIds)+1)
	calls[0] = invocation(CommandMailboxGet, MailboxGetCommand{AccountId: mailboxAccountId}, "0")
	for i, accountId := range uniqueAccountIds {
		calls[i+1] = invocation(CommandIdentityGet, IdentityGetCommand{AccountId: accountId}, strconv.Itoa(i+1))
	}

	cmd, err := j.request(session, logger, calls...)
	if err != nil {
		return IdentitiesAndMailboxesGetResponse{}, "", "", "", err
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (IdentitiesAndMailboxesGetResponse, State, Error) {
		identities := make(map[string][]Identity, len(uniqueAccountIds))
		stateByAccountId := make(map[string]State, len(uniqueAccountIds))
		notFound := []string{}
		for i, accountId := range uniqueAccountIds {
			var response IdentityGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandIdentityGet, strconv.Itoa(i+1), &response)
			if err != nil {
				return IdentitiesAndMailboxesGetResponse{}, "", err
			} else {
				identities[accountId] = response.List
			}
			stateByAccountId[accountId] = response.State
			notFound = append(notFound, response.NotFound...)
		}

		var mailboxResponse MailboxGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandMailboxGet, "0", &mailboxResponse)
		if err != nil {
			return IdentitiesAndMailboxesGetResponse{}, "", err
		}

		return IdentitiesAndMailboxesGetResponse{
			Identities: identities,
			NotFound:   structs.Uniq(notFound),
			Mailboxes:  mailboxResponse.List,
		}, squashState(stateByAccountId), nil
	})
}

func (j *Client) CreateIdentity(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, identity Identity) (Identity, SessionState, State, Language, Error) {
	logger = j.logger("CreateIdentity", session, logger)
	cmd, err := j.request(session, logger, invocation(CommandIdentitySet, IdentitySetCommand{
		AccountId: accountId,
		Create: map[string]Identity{
			"c": identity,
		},
	}, "0"))
	if err != nil {
		return Identity{}, "", "", "", err
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (Identity, State, Error) {
		var response IdentitySetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandIdentitySet, "0", &response)
		if err != nil {
			return Identity{}, response.NewState, err
		}
		setErr, notok := response.NotCreated["c"]
		if notok {
			logger.Error().Msgf("%T.NotCreated returned an error %v", response, setErr)
			return Identity{}, "", setErrorError(setErr, IdentityType)
		}
		return response.Created["c"], response.NewState, nil
	})
}

func (j *Client) UpdateIdentity(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, identity Identity) (Identity, SessionState, State, Language, Error) {
	logger = j.logger("UpdateIdentity", session, logger)
	cmd, err := j.request(session, logger, invocation(CommandIdentitySet, IdentitySetCommand{
		AccountId: accountId,
		Update: map[string]PatchObject{
			"c": identity.AsPatch(),
		},
	}, "0"))
	if err != nil {
		return Identity{}, "", "", "", err
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (Identity, State, Error) {
		var response IdentitySetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandIdentitySet, "0", &response)
		if err != nil {
			return Identity{}, response.NewState, err
		}
		setErr, notok := response.NotCreated["c"]
		if notok {
			logger.Error().Msgf("%T.NotCreated returned an error %v", response, setErr)
			return Identity{}, "", setErrorError(setErr, IdentityType)
		}
		return response.Created["c"], response.NewState, nil
	})
}

func (j *Client) DeleteIdentity(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, ids []string) ([]string, SessionState, State, Language, Error) {
	logger = j.logger("DeleteIdentity", session, logger)
	cmd, err := j.request(session, logger, invocation(CommandIdentitySet, IdentitySetCommand{
		AccountId: accountId,
		Destroy:   ids,
	}, "0"))
	if err != nil {
		return nil, "", "", "", err
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) ([]string, State, Error) {
		var response IdentitySetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandIdentitySet, "0", &response)
		if err != nil {
			return nil, "", err
		}
		for _, setErr := range response.NotDestroyed {
			// TODO only returning the first error here, we should probably aggregate them instead
			logger.Error().Msgf("%T.NotCreated returned an error %v", response, setErr)
			return nil, "", setErrorError(setErr, IdentityType)
		}
		return response.Destroyed, response.NewState, nil
	})
}
