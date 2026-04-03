package jmap

import (
	"context"
	"strconv"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

var NS_IDENTITY = ns(JmapMail)

func (j *Client) GetAllIdentities(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) ([]Identity, SessionState, State, Language, Error) {
	return getTemplate(j, "GetAllIdentities", NS_IDENTITY, CommandIdentityGet,
		func(accountId string, ids []string) IdentityGetCommand {
			return IdentityGetCommand{AccountId: accountId}
		},
		func(resp IdentityGetResponse) []Identity { return resp.List },
		func(resp IdentityGetResponse) State { return resp.State },
		accountId, session, ctx, logger, acceptLanguage, []string{},
	)
}

func (j *Client) GetIdentities(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, identityIds []string) ([]Identity, SessionState, State, Language, Error) {
	return getTemplate(j, "GetIdentities", NS_IDENTITY, CommandIdentityGet,
		func(accountId string, ids []string) IdentityGetCommand {
			return IdentityGetCommand{AccountId: accountId, Ids: ids}
		},
		func(resp IdentityGetResponse) []Identity { return resp.List },
		func(resp IdentityGetResponse) State { return resp.State },
		accountId, session, ctx, logger, acceptLanguage, identityIds,
	)
}

func (j *Client) GetIdentitiesForAllAccounts(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string][]Identity, SessionState, State, Language, Error) {
	return getTemplateN(j, "GetIdentitiesForAllAccounts", NS_IDENTITY, CommandIdentityGet,
		func(accountId string, ids []string) IdentityGetCommand {
			return IdentityGetCommand{AccountId: accountId}
		},
		func(resp IdentityGetResponse) []Identity { return resp.List },
		identity1,
		func(resp IdentityGetResponse) State { return resp.State },
		accountIds, session, ctx, logger, acceptLanguage, []string{},
	)
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

	cmd, err := j.request(session, logger, NS_IDENTITY, calls...)
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
	cmd, err := j.request(session, logger, NS_IDENTITY, invocation(CommandIdentitySet, IdentitySetCommand{
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
			logger.Error().Msgf("%T.NotCreated returned an error %v", response, setErr) //NOSONAR
			return Identity{}, "", setErrorError(setErr, IdentityType)
		}
		return response.Created["c"], response.NewState, nil
	})
}

func (j *Client) UpdateIdentity(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, identity Identity) (Identity, SessionState, State, Language, Error) {
	logger = j.logger("UpdateIdentity", session, logger)
	cmd, err := j.request(session, logger, NS_IDENTITY, invocation(CommandIdentitySet, IdentitySetCommand{
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
	cmd, err := j.request(session, logger, NS_IDENTITY, invocation(CommandIdentitySet, IdentitySetCommand{
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

type IdentityChanges struct {
	OldState       State      `json:"oldState,omitempty"`
	NewState       State      `json:"newState"`
	HasMoreChanges bool       `json:"hasMoreChanges"`
	Created        []Identity `json:"created,omitempty"`
	Updated        []Identity `json:"updated,omitempty"`
	Destroyed      []string   `json:"destroyed,omitempty"`
}

// Retrieve the changes in Email Identities since a given State.
// @api:tags email,changes
func (j *Client) GetIdentityChanges(accountId string, session *Session, ctx context.Context, logger *log.Logger,
	acceptLanguage string, sinceState State, maxChanges uint) (IdentityChanges, SessionState, State, Language, Error) {
	return changesTemplate(j, "GetIdentityChanges", NS_IDENTITY,
		CommandIdentityChanges, CommandIdentityGet,
		func() IdentityChangesCommand {
			return IdentityChangesCommand{AccountId: accountId, SinceState: sinceState, MaxChanges: posUIntPtr(maxChanges)}
		},
		func(path string, rof string) IdentityGetRefCommand {
			return IdentityGetRefCommand{
				AccountId: accountId,
				IdsRef: &ResultReference{
					Name:     CommandIdentityChanges,
					Path:     path,
					ResultOf: rof,
				},
			}
		},
		func(resp IdentityChangesResponse) (State, State, bool, []string) {
			return resp.OldState, resp.NewState, resp.HasMoreChanges, resp.Destroyed
		},
		func(resp IdentityGetResponse) []Identity { return resp.List },
		func(oldState, newState State, hasMoreChanges bool, created, updated []Identity, destroyed []string) IdentityChanges {
			return IdentityChanges{
				OldState:       oldState,
				NewState:       newState,
				HasMoreChanges: hasMoreChanges,
				Created:        created,
				Updated:        updated,
				Destroyed:      destroyed,
			}
		},
		func(resp IdentityGetResponse) State { return resp.State },
		session, ctx, logger, acceptLanguage,
	)
}
