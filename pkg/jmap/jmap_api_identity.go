package jmap

import (
	"context"
	"strconv"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
)

type Identities struct {
	Identities []Identity `json:"identities"`
	State      State      `json:"state"`
}

// https://jmap.io/spec-mail.html#identityget
func (j *Client) GetIdentity(accountId string, session *Session, ctx context.Context, logger *log.Logger) (Identities, SessionState, Error) {
	logger = j.logger("GetIdentity", session, logger)
	cmd, err := request(invocation(CommandIdentityGet, IdentityGetCommand{AccountId: accountId}, "0"))
	if err != nil {
		logger.Error().Err(err)
		return Identities{}, "", simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (Identities, Error) {
		var response IdentityGetResponse
		err = retrieveResponseMatchParameters(body, CommandIdentityGet, "0", &response)
		if err != nil {
			logger.Error().Err(err)
			return Identities{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}
		return Identities{
			Identities: response.List,
			State:      response.State,
		}, nil
	})
}

type IdentitiesGetResponse struct {
	Identities map[string][]Identity `json:"identities,omitempty"`
	NotFound   []string              `json:"notFound,omitempty"`
	State      State                 `json:"state"`
}

func (j *Client) GetIdentities(accountIds []string, session *Session, ctx context.Context, logger *log.Logger) (IdentitiesGetResponse, SessionState, Error) {
	uniqueAccountIds := structs.Uniq(accountIds)

	logger = j.logger("GetIdentities", session, logger)

	calls := make([]Invocation, len(uniqueAccountIds))
	for i, accountId := range uniqueAccountIds {
		calls[i] = invocation(CommandIdentityGet, IdentityGetCommand{AccountId: accountId}, strconv.Itoa(i))
	}

	cmd, err := request(calls...)
	if err != nil {
		logger.Error().Err(err)
		return IdentitiesGetResponse{}, "", simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (IdentitiesGetResponse, Error) {
		identities := make(map[string][]Identity, len(uniqueAccountIds))
		var lastState State
		notFound := []string{}
		for i, accountId := range uniqueAccountIds {
			var response IdentityGetResponse
			err = retrieveResponseMatchParameters(body, CommandIdentityGet, strconv.Itoa(i), &response)
			if err != nil {
				logger.Error().Err(err)
				return IdentitiesGetResponse{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
			} else {
				identities[accountId] = response.List
			}
			lastState = response.State
			notFound = append(notFound, response.NotFound...)
		}

		return IdentitiesGetResponse{
			Identities: identities,
			NotFound:   structs.Uniq(notFound),
			State:      lastState,
		}, nil
	})
}

type IdentitiesAndMailboxesGetResponse struct {
	Identities map[string][]Identity `json:"identities,omitempty"`
	NotFound   []string              `json:"notFound,omitempty"`
	State      State                 `json:"state"`
	Mailboxes  []Mailbox             `json:"mailboxes"`
}

func (j *Client) GetIdentitiesAndMailboxes(mailboxAccountId string, accountIds []string, session *Session, ctx context.Context, logger *log.Logger) (IdentitiesAndMailboxesGetResponse, SessionState, Error) {
	uniqueAccountIds := structs.Uniq(accountIds)

	logger = j.logger("GetIdentitiesAndMailboxes", session, logger)

	calls := make([]Invocation, len(uniqueAccountIds)+1)
	calls[0] = invocation(CommandMailboxGet, MailboxGetCommand{AccountId: mailboxAccountId}, "0")
	for i, accountId := range uniqueAccountIds {
		calls[i+1] = invocation(CommandIdentityGet, IdentityGetCommand{AccountId: accountId}, strconv.Itoa(i+1))
	}

	cmd, err := request(calls...)
	if err != nil {
		logger.Error().Err(err)
		return IdentitiesAndMailboxesGetResponse{}, "", simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (IdentitiesAndMailboxesGetResponse, Error) {
		identities := make(map[string][]Identity, len(uniqueAccountIds))
		var lastState State
		notFound := []string{}
		for i, accountId := range uniqueAccountIds {
			var response IdentityGetResponse
			err = retrieveResponseMatchParameters(body, CommandIdentityGet, strconv.Itoa(i+1), &response)
			if err != nil {
				logger.Error().Err(err)
				return IdentitiesAndMailboxesGetResponse{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
			} else {
				identities[accountId] = response.List
			}
			lastState = response.State
			notFound = append(notFound, response.NotFound...)
		}

		var mailboxResponse MailboxGetResponse
		err = retrieveResponseMatchParameters(body, CommandMailboxGet, "0", &mailboxResponse)
		if err != nil {
			logger.Error().Err(err)
			return IdentitiesAndMailboxesGetResponse{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}

		return IdentitiesAndMailboxesGetResponse{
			Identities: identities,
			NotFound:   structs.Uniq(notFound),
			State:      lastState,
			Mailboxes:  mailboxResponse.List,
		}, nil
	})
}
